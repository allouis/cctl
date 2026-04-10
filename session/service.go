package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/tmux"
	"github.com/allouis/cctl/transcript"
	"github.com/allouis/cctl/workspace"
	"github.com/google/uuid"
)

type Service struct {
	store  *db.DB
	runner tmux.Runner
	cfg    *config.Config
}

func NewService(store *db.DB, runner tmux.Runner, cfg *config.Config) *Service {
	return &Service{store: store, runner: runner, cfg: cfg}
}

// List returns sessions with enrichment: inferIdle, tmux cross-ref, DEAD marking.
// If activeOnly is true, DONE and DEAD sessions are excluded.
func (s *Service) List() ([]db.Session, error) {
	sessions, err := s.store.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	live := s.liveWindowIDs()
	attached := s.runner.ActiveWindowIDs(s.cfg.Session)

	var result []db.Session
	for _, sess := range sessions {
		if sess.Name == "dash" {
			continue
		}
		s.enrichSession(&sess, live, attached)
		result = append(result, sess)
	}

	return result, nil
}

// resolveSession looks up a session by session_id first, then falls back to name.
func (s *Service) resolveSession(nameOrID string) (*db.Session, error) {
	sess, err := s.store.GetSessionByID(nameOrID)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		return sess, nil
	}
	return s.store.GetSession(nameOrID)
}

// Get returns a single session with enrichment applied.
func (s *Service) Get(nameOrID string) (*db.Session, error) {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}

	live := s.liveWindowIDs()
	attached := s.runner.ActiveWindowIDs(s.cfg.Session)
	s.enrichSession(sess, live, attached)

	return sess, nil
}

// enrichSession applies runtime state corrections to a session.
// live==nil means we couldn't query tmux — skip DEAD inference to avoid false positives.
func (s *Service) enrichSession(sess *db.Session, live, attached map[string]bool) {
	InferIdle(sess)
	if live != nil && sess.WindowID != "" && !live[sess.WindowID] && sess.ExecutorState != "DONE" {
		sess.ExecutorState = "DEAD"
		sess.ExecutorDetail = "window lost"
		sess.WindowID = ""
	}
	sess.Attached = sess.WindowID != "" && attached[sess.WindowID]
}

// Transcript returns parsed transcript entries for a session.
func (s *Service) Transcript(nameOrID string, limit int) ([]transcript.Entry, error) {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session '%s' not found", nameOrID)
	}
	if sess.TranscriptPath == "" {
		return []transcript.Entry{}, nil
	}
	return transcript.Parse(sess.TranscriptPath, limit)
}

// CreateOpts holds parameters for creating a new session.
type CreateOpts struct {
	Name      string
	Dir       string
	Prompt    string
	Safe      bool
	ParentID  *string
	ProjectID *string
}

// Create starts a new Claude Code session in a tmux window.
// Returns the session_id of the created session.
func (s *Service) Create(opts CreateOpts) (string, error) {
	name := opts.Name
	dir := opts.Dir

	// If name looks like a path, treat it as dir and derive name from basename
	if strings.Contains(name, "/") {
		dir = name
		name = filepath.Base(dir)
	}

	// Expand ~ in dir
	if strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, dir[2:])
		}
	}

	// Resolve to absolute path
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
	}

	// Ensure tmux session exists
	if !s.runner.HasSession(s.cfg.Session) {
		if err := s.runner.NewSession(s.cfg.Session, "dash", ""); err != nil {
			return "", fmt.Errorf("create tmux session: %w", err)
		}
		s.initSessionEnv()
	}

	// Generate session ID and build command
	sessionID := uuid.New().String()
	safe := opts.Safe || s.cfg.Safe

	if err := s.ensureHooksJSON(); err != nil {
		return "", fmt.Errorf("generate hooks.json: %w", err)
	}

	// Create a VCS workspace if the directory is in a supported repo.
	var workspaceName string
	workDir := dir
	if mgr := workspace.Detect(dir, s.cfg.Dir); mgr != nil {
		wd, err := mgr.Create(dir, sessionID)
		if err != nil {
			return "", fmt.Errorf("create workspace: %w", err)
		}
		workDir = wd
		workspaceName = sessionID
	}

	win := s.buildWindow(sessionID, name, opts.Prompt, safe, false, workspaceName != "")

	windowID, err := s.runner.NewWindow(s.cfg.Session, name, win.command, workDir, win.env)
	if err != nil {
		return "", fmt.Errorf("create window: %w", err)
	}

	// Write DB row immediately — no more gap before first hook.
	now := time.Now().Unix()
	if err := s.store.CreateSession(db.Session{
		SessionID: sessionID,
		Name:      name,
		ParentID:      opts.ParentID,
		ProjectID:     opts.ProjectID,
		WorkState:     "running",
		ExecutorState: "STARTING",
		Dir:          dir,
		CWD:          dir,
		WindowID:     windowID,
		Workspace:    workspaceName,
		Prompt:       opts.Prompt,
		Safe:         safe,
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		return "", fmt.Errorf("create session record: %w", err)
	}

	return sessionID, nil
}

// windowSpec holds the command and per-window environment for a tmux window.
type windowSpec struct {
	command string
	env     []string // KEY=VALUE pairs passed via tmux new-window -e
}

func (s *Service) buildWindow(sessionID, name, prompt string, safe, resume, isWorkspace bool) windowSpec {
	env := []string{
		"CCTL_NAME=" + name,
		"AGENT_BROWSER_SESSION=" + sessionID,
	}

	if s.cfg.Cmd != "claude" {
		return windowSpec{command: s.cfg.Cmd, env: env}
	}

	hooksPath := filepath.Join(s.cfg.Dir, "hooks.json")
	var flags string
	if resume {
		flags = fmt.Sprintf("--settings %s --resume %s", shellQuote(hooksPath), sessionID)
	} else {
		flags = fmt.Sprintf("--settings %s --session-id %s", shellQuote(hooksPath), sessionID)
	}
	if !safe {
		flags += " --dangerously-skip-permissions"
	}
	if sp := s.cfg.SystemPromptPath(); sp != "" {
		if info, err := os.Stat(sp); err == nil && info.Size() > 0 {
			flags += fmt.Sprintf(" --append-system-prompt-file %s", shellQuote(sp))
		}
	}
	if prompt != "" {
		flags += fmt.Sprintf(" -p %s", shellQuote(prompt))
	}
	cmd := "claude " + flags

	// Claude shows a workspace trust prompt for untrusted directories.
	// Auto-approve by sending Enter from a background process when
	// running without --safe (permissions are already skipped, so the
	// user trusts the directory). Harmless if no prompt appears — Enter
	// at the idle prompt is ignored.
	// Wrapped in bash -c because tmux may use fish as default shell.
	if !safe && !resume {
		cmd = fmt.Sprintf("bash -c 'sleep 3 && tmux send-keys -t \"$TMUX_PANE\" Enter &'; %s", cmd)
	}

	return windowSpec{command: cmd, env: env}
}

// Kill terminates a session's tmux window and marks it DEAD.
func (s *Service) Kill(nameOrID string) error {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session '%s' not found", nameOrID)
	}

	if sess.WindowID != "" {
		s.runner.KillWindow(s.cfg.Session, sess.WindowID)
	}

	s.store.UpdateSessionState(sess.SessionID, "DEAD", "killed")
	s.store.UpdateWindowID(sess.SessionID, "")
	return nil
}

// Resume recreates a tmux window for a DEAD or DONE session.
func (s *Service) Resume(nameOrID string) error {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session '%s' not found", nameOrID)
	}
	if sess.ExecutorState != "DEAD" && sess.ExecutorState != "DONE" {
		return fmt.Errorf("session '%s' is %s, not resumable", nameOrID, sess.ExecutorState)
	}

	// Ensure tmux session exists
	if !s.runner.HasSession(s.cfg.Session) {
		if err := s.runner.NewSession(s.cfg.Session, "dash", ""); err != nil {
			return fmt.Errorf("create tmux session: %w", err)
		}
		s.initSessionEnv()
	}

	if err := s.ensureHooksJSON(); err != nil {
		return fmt.Errorf("generate hooks.json: %w", err)
	}

	// Kill any orphaned claude process still holding this session ID.
	// This can happen when a tmux window dies but the process survives.
	killOrphanedProcess(sess.SessionID)

	win := s.buildWindow(sess.SessionID, sess.Name, "", sess.Safe, true, sess.Workspace != "")

	// Use workspace dir if the session has one, otherwise the original CWD.
	dir := sess.CWD
	if sess.Workspace != "" {
		workDir := workspace.WorkspaceDir(s.cfg.Dir, sess.Workspace)
		if info, err := os.Stat(workDir); err == nil && info.IsDir() {
			dir = workDir
		}
	}

	windowID, err := s.runner.NewWindow(s.cfg.Session, sess.Name, win.command, dir, win.env)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}

	s.store.UpdateWindowID(sess.SessionID, windowID)
	s.store.UpdateSessionState(sess.SessionID, "STARTING", "resumed")
	return nil
}

// Delete permanently removes a session: kills the window, removes the
// workspace, and deletes the DB row.
func (s *Service) Delete(nameOrID string) error {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session '%s' not found", nameOrID)
	}

	if sess.WindowID != "" {
		s.runner.KillWindow(s.cfg.Session, sess.WindowID)
	}

	if sess.Workspace != "" {
		if mgr := workspace.Detect(sess.CWD, s.cfg.Dir); mgr != nil {
			mgr.Remove(sess.CWD, sess.Workspace)
		}
	}

	s.store.DeleteSession(sess.Name)
	return nil
}

// SendResult holds the result of a send operation.
type SendResult struct {
	Confirmed bool `json:"confirmed"`
}

// ErrSessionAttached is returned when trying to send to a session
// that has a tmux client attached (human is interacting directly).
var ErrSessionAttached = fmt.Errorf("session has an attached tmux client")

// Send sends text to a session's tmux window and verifies delivery
// by polling the transcript for the user message.
func (s *Service) Send(nameOrID, text string) (*SendResult, error) {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return nil, err
	}
	if sess == nil || sess.WindowID == "" {
		return nil, fmt.Errorf("session '%s' has no active window", nameOrID)
	}
	attached := s.runner.ActiveWindowIDs(s.cfg.Session)
	if attached[sess.WindowID] {
		return nil, ErrSessionAttached
	}
	if err := s.runner.SendKeys(s.cfg.Session, sess.WindowID, text); err != nil {
		return nil, err
	}

	confirmed := s.waitForTranscriptEntry(sess.TranscriptPath, text)
	if confirmed {
		s.store.UpdateSessionState(sess.SessionID, "WORKING", "processing")
	}
	return &SendResult{Confirmed: confirmed}, nil
}

// waitForTranscriptEntry polls the transcript file for a user message
// containing the sent text. Returns true if found within timeout.
func (s *Service) waitForTranscriptEntry(transcriptPath, text string) bool {
	if transcriptPath == "" {
		return false
	}

	// Poll every 300ms for up to 3 seconds
	for i := 0; i < 10; i++ {
		time.Sleep(300 * time.Millisecond)
		entries, err := transcript.Parse(transcriptPath, 5)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Role == "user" && strings.Contains(e.Text, text) {
				return true
			}
		}
	}
	return false
}

// Focus selects a tmux window.
func (s *Service) Focus(nameOrID string) error {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return err
	}
	if sess == nil || sess.WindowID == "" {
		return fmt.Errorf("session '%s' has no active window", nameOrID)
	}
	return s.runner.SelectWindow(s.cfg.Session, sess.WindowID)
}

// Takeover moves any attached tmux client away from this session's window
// so the web UI can send input. Moves the active window to dash (index 0).
func (s *Service) Takeover(nameOrID string) error {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session '%s' not found", nameOrID)
	}
	if sess.WindowID == "" {
		return fmt.Errorf("session '%s' has no active window", nameOrID)
	}
	attached := s.runner.ActiveWindowIDs(s.cfg.Session)
	if !attached[sess.WindowID] {
		return nil
	}
	return s.runner.SelectWindow(s.cfg.Session, "0")
}

// liveWindowIDs returns the set of tmux window IDs (@N) that currently exist.
// Returns nil if the tmux session doesn't exist (so callers skip DEAD marking).
func (s *Service) liveWindowIDs() map[string]bool {
	windows, err := s.runner.ListWindows(s.cfg.Session)
	if err != nil {
		// Session doesn't exist or tmux unreachable — return nil so
		// enrichSession doesn't incorrectly mark everything DEAD.
		return nil
	}
	ids := make(map[string]bool)
	for _, w := range windows {
		ids[w.ID] = true
	}
	return ids
}

// initSessionEnv sets the tmux session environment so windows can
// find tools and don't inherit unwanted vars from a parent Claude.
func (s *Service) initSessionEnv() {
	s.runner.SetEnv(s.cfg.Session, "PATH", os.Getenv("PATH"))
	s.runner.UnsetEnv(s.cfg.Session, "CLAUDECODE")
}

func (s *Service) ensureHooksJSON() error {
	binPath := ResolveBinaryPath()
	data, err := config.GenerateHooksJSON(binPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.cfg.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.cfg.Dir, "hooks.json"), data, 0o644)
}

// InferIdle adjusts the session state when Claude Code is at the idle prompt
// but no idle_prompt notification has been sent. This happens after SessionStart
// (Claude doesn't send idle_prompt until after the first interaction) and after
// Stop events (where the idle_prompt may not always fire).
func InferIdle(s *db.Session) {
	if s.ExecutorState != "WORKING" {
		return
	}
	age := time.Now().Unix() - s.UpdatedAt
	if s.LastEvent == "SessionStart" && age > 5 {
		s.ExecutorState = "IDLE"
		s.ExecutorDetail = "ready"
	}
}

// Project operations

func (s *Service) ListProjects() ([]db.Project, error) {
	return s.store.ListProjects()
}

func (s *Service) CreateProject(name string) (*db.Project, error) {
	p := db.Project{
		ID:   uuid.New().String(),
		Name: name,
	}
	if err := s.store.CreateProject(p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) DeleteProject(id string) error {
	return s.store.DeleteProject(id)
}

func (s *Service) SetSessionProject(sessionID string, projectID *string) error {
	return s.store.UpdateSessionProject(sessionID, projectID)
}

// ListRepoDirs returns all VCS directories found under registered repo paths.
func (s *Service) ListRepoDirs() ([]string, error) {
	parents, err := s.store.ListRepos()
	if err != nil {
		return nil, err
	}
	return workspace.ScanRepos(parents), nil
}

// killOrphanedProcess finds and kills any claude process still running
// with the given session ID. This handles cases where a tmux window died
// but the process survived (e.g., tmux session restart, manual window close).
func killOrphanedProcess(sessionID string) {
	exec.Command("pkill", "-f", "--session-id "+sessionID).Run()
	// Brief pause to let the process release its lock.
	time.Sleep(100 * time.Millisecond)
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t\n'\"\\$`!") {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}
	return s
}

// ResolveBinaryPath returns an absolute path to the current binary.
func ResolveBinaryPath() string {
	if abs, err := os.Executable(); err == nil {
		return abs
	}
	arg0 := os.Args[0]
	if filepath.IsAbs(arg0) {
		return arg0
	}
	if abs, err := filepath.Abs(arg0); err == nil {
		return abs
	}
	return arg0
}
