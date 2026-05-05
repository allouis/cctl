package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
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
	Harness   string
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
	}
	s.initSessionEnv()

	// Generate session ID and build command
	sessionID := uuid.New().String()
	safe := opts.Safe || s.cfg.Safe
	harness := opts.Harness

	log.Printf("session/create: name=%q dir=%q harness=%q cmd=%q safe=%v sessionID=%s", name, dir, harness, s.cfg.Cmd, safe, sessionID)

	if err := s.ensureHarness(harness); err != nil {
		return "", fmt.Errorf("setup harness: %w", err)
	}

	// Create a VCS workspace if the directory is in a supported repo.
	var workspaceName string
	workDir := dir
	if mgr := workspace.Detect(dir, s.cfg.Dir); mgr != nil {
		workspaceName = workspace.Name(name, sessionID)
		wd, err := mgr.Create(dir, workspaceName)
		if err != nil {
			return "", fmt.Errorf("create workspace: %w", err)
		}
		workDir = wd
	}

	win := s.buildWindow(sessionID, name, opts.Prompt, "", harness, safe, false, workspaceName != "", dir)
	log.Printf("session/create: resolved command: %s", win.command)

	windowID, err := s.runner.NewWindow(s.cfg.Session, name, win.command, workDir, win.env)
	if err != nil {
		log.Printf("session/create: window creation failed: %v", err)
		return "", fmt.Errorf("create window: %w", err)
	}
	log.Printf("session/create: window created: %s (tmux=%s)", sessionID, windowID)

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
		Harness:      harness,
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

func (s *Service) buildWindow(sessionID, name, prompt, transcriptPath, harness string, safe, resume, isWorkspace bool, dir string) windowSpec {
	env := []string{
		"CCTL_NAME=" + name,
		"AGENT_BROWSER_SESSION=" + sessionID,
	}
	for k, v := range s.cfg.SessionEnv {
		env = append(env, k+"="+expandSessionEnv(v, dir, sessionID, name))
	}

	usePi := harness == "pi" || (harness == "" && isPiCmd(s.cfg.Cmd))
	if usePi {
		piCmd := "pi"
		if harness == "" && isPiCmd(s.cfg.Cmd) {
			piCmd = s.cfg.Cmd
		}
		piCmd = resolveCmd(piCmd)
		return s.buildPiWindow(sessionID, prompt, piCmd, resume, transcriptPath, env)
	}

	cmd := s.cfg.Cmd
	if harness == "claude" && !isClaudeCmd(s.cfg.Cmd) {
		cmd = "claude"
	}
	cmd = resolveCmd(cmd)
	if !isClaudeCmd(cmd) {
		log.Printf("session/buildWindow: cmd %q is not a claude command, returning as-is", cmd)
		return windowSpec{command: cmd, env: env}
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
	cmd = cmd + " " + flags

	// Auto-dismiss startup prompts (bypass-permissions confirmation,
	// workspace trust) by sending keystrokes from a background process.
	// Down selects "Yes, I accept" on the bypass prompt, first Enter
	// confirms it, second Enter dismisses any workspace trust prompt.
	if !safe && !resume {
		cmd = fmt.Sprintf("bash -c 'sleep 3 && tmux send-keys -t \"$TMUX_PANE\" Down && sleep 0.5 && tmux send-keys -t \"$TMUX_PANE\" Enter && sleep 2 && tmux send-keys -t \"$TMUX_PANE\" Enter &'; %s", cmd)
	}

	return windowSpec{command: cmd, env: env}
}

func expandSessionEnv(tmpl, dir, uuid, name string) string {
	r := strings.NewReplacer("{{dir}}", dir, "{{uuid}}", uuid, "{{name}}", name)
	return r.Replace(tmpl)
}

// Kill terminates a session's tmux window and marks it DEAD.
// If the session's workspace is clean (no uncommitted changes, no unlanded commits),
// the workspace is pruned too — there's nothing to lose, and it keeps `jj log` tidy.
// A dirty workspace is retained so `cctl resume` can still pick up where it left off.
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

	os.Remove(bridgeSocketPath(sess.SessionID))
	s.pruneWorkspaceIfClean(sess)

	s.store.UpdateSessionState(sess.SessionID, "DEAD", "killed")
	s.store.UpdateWindowID(sess.SessionID, "")
	return nil
}

// pruneWorkspaceIfClean removes the session's workspace (and clears the DB field)
// when nothing would be lost. Best-effort — errors are swallowed because Kill must
// always succeed at marking the session dead.
func (s *Service) pruneWorkspaceIfClean(sess *db.Session) {
	if sess.Workspace == "" {
		return
	}
	mgr := workspace.Detect(sess.Dir, s.cfg.Dir)
	if mgr == nil {
		return
	}
	clean, err := mgr.IsClean(sess.Dir, sess.Workspace)
	if err != nil || !clean {
		return
	}
	mgr.Remove(sess.Dir, sess.Workspace)
	s.store.UpdateSessionWorkspace(sess.SessionID, "")
	sess.Workspace = ""
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

	log.Printf("session/resume: id=%s harness=%q cmd=%q", sess.SessionID, sess.Harness, s.cfg.Cmd)

	// Ensure tmux session exists
	if !s.runner.HasSession(s.cfg.Session) {
		if err := s.runner.NewSession(s.cfg.Session, "dash", ""); err != nil {
			return fmt.Errorf("create tmux session: %w", err)
		}
	}
	s.initSessionEnv()

	if err := s.ensureHarness(sess.Harness); err != nil {
		return fmt.Errorf("setup harness: %w", err)
	}

	// Kill any orphaned process still holding this session ID.
	killOrphanedProcess(sess.SessionID)
	os.Remove(bridgeSocketPath(sess.SessionID))

	win := s.buildWindow(sess.SessionID, sess.Name, "", sess.TranscriptPath, sess.Harness, sess.Safe, true, sess.Workspace != "", sess.Dir)
	log.Printf("session/resume: resolved command: %s", win.command)

	// Use workspace dir if the session has one, otherwise the original CWD.
	// If the workspace dir is missing (pruned by Kill or an earlier `workspace prune`),
	// recreate it so Resume always lands in a real isolated working copy.
	dir := sess.CWD
	if sess.Workspace != "" {
		workDir := workspace.WorkspaceDir(s.cfg.Dir, sess.Workspace)
		if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
			mgr := workspace.Detect(sess.Dir, s.cfg.Dir)
			if mgr == nil {
				return fmt.Errorf("cannot recreate workspace: no VCS at %s", sess.Dir)
			}
			if _, err := mgr.Create(sess.Dir, sess.Workspace); err != nil {
				return fmt.Errorf("recreate workspace: %w", err)
			}
		}
		dir = workDir
	}

	windowID, err := s.runner.NewWindow(s.cfg.Session, sess.Name, win.command, dir, win.env)
	if err != nil {
		log.Printf("session/resume: window creation failed: %v", err)
		return fmt.Errorf("create window: %w", err)
	}
	log.Printf("session/resume: window created: %s (tmux=%s)", sess.SessionID, windowID)

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

// Send sends text to a session. Uses the bridge socket for pi sessions
// (direct API injection) or tmux send-keys for Claude sessions.
func (s *Service) Send(nameOrID, text string) (*SendResult, error) {
	sess, err := s.resolveSession(nameOrID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session '%s' not found", nameOrID)
	}

	// Try bridge socket (pi harness) — no tmux attachment check needed
	// because messages go through pi's API, not the terminal.
	sockPath := bridgeSocketPath(sess.SessionID)
	if _, err := os.Stat(sockPath); err == nil {
		return s.sendViaBridge(sockPath, text)
	}

	// Fall back to tmux send-keys (Claude harness)
	if sess.WindowID == "" {
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

// ensureHarness writes the configuration file for the active harness:
// hooks.json for Claude, pi-bridge.ts for pi.
func (s *Service) ensureHarness(harness string) error {
	if harness == "pi" || (harness == "" && isPiCmd(s.cfg.Cmd)) {
		log.Printf("session/harness: writing pi bridge extension (harness=%q cmd=%q)", harness, s.cfg.Cmd)
		if err := os.MkdirAll(s.cfg.Dir, 0o755); err != nil {
			return err
		}
		_, err := config.WriteBridgeExtension(s.cfg.Dir)
		return err
	}
	if harness == "claude" || (harness == "" && isClaudeCmd(s.cfg.Cmd)) {
		log.Printf("session/harness: writing hooks.json (harness=%q cmd=%q)", harness, s.cfg.Cmd)
		return s.ensureHooksJSON()
	}
	log.Printf("session/harness: no harness matched (harness=%q cmd=%q) — skipping hooks.json", harness, s.cfg.Cmd)
	return nil
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

// PruneResult reports the outcome of PruneWorkspaces.
type PruneResult struct {
	Pruned   []string // workspaces removed because their DEAD/DONE session was clean
	Retained []string // workspaces kept because they had unlanded work
	Orphans  []string // workspaces removed because no session referenced them
}

// PruneWorkspaces removes workspaces that are safe to delete:
//   - DEAD/DONE sessions whose workspace is clean (no uncommitted changes, no unlanded commits)
//   - VCS-tracked workspaces that no cctl session references at all (orphans)
//
// Live sessions are never touched, and DEAD/DONE sessions with unlanded work are
// retained so they remain resumable.
func (s *Service) PruneWorkspaces() (PruneResult, error) {
	var res PruneResult
	sessions, err := s.store.ListSessions()
	if err != nil {
		return res, fmt.Errorf("list sessions: %w", err)
	}

	referenced := make(map[string]bool)
	repoDirs := make(map[string]bool)
	for _, sess := range sessions {
		if sess.Workspace != "" {
			referenced[sess.Workspace] = true
		}
		if sess.Dir != "" {
			repoDirs[sess.Dir] = true
		}
	}

	for _, sess := range sessions {
		if sess.Workspace == "" {
			continue
		}
		if sess.ExecutorState != "DEAD" && sess.ExecutorState != "DONE" {
			continue
		}
		mgr := workspace.Detect(sess.Dir, s.cfg.Dir)
		if mgr == nil {
			continue
		}
		clean, err := mgr.IsClean(sess.Dir, sess.Workspace)
		if err != nil || !clean {
			res.Retained = append(res.Retained, sess.Workspace)
			continue
		}
		mgr.Remove(sess.Dir, sess.Workspace)
		s.store.UpdateSessionWorkspace(sess.SessionID, "")
		delete(referenced, sess.Workspace)
		res.Pruned = append(res.Pruned, sess.Workspace)
	}

	// Orphan sweep: for every repo that any session lives in, any VCS workspace
	// not in `referenced` is a leftover from a session cctl has forgotten about.
	seen := make(map[string]bool)
	for dir := range repoDirs {
		mgr := workspace.Detect(dir, s.cfg.Dir)
		if mgr == nil {
			continue
		}
		names, err := mgr.ListWorkspaces(dir)
		if err != nil {
			continue
		}
		for _, name := range names {
			if referenced[name] || seen[name] {
				continue
			}
			seen[name] = true
			mgr.Remove(dir, name)
			res.Orphans = append(res.Orphans, name)
		}
	}
	return res, nil
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

func (s *Service) buildPiWindow(sessionID, prompt, piCmd string, resume bool, transcriptPath string, env []string) windowSpec {
	bridgePath := filepath.Join(s.cfg.Dir, "pi-bridge.ts")

	env = append(env,
		"CCTL_SESSION_ID="+sessionID,
		"CCTL_BIN="+ResolveBinaryPath(),
	)

	var flags string
	if resume && transcriptPath != "" {
		flags = fmt.Sprintf("--session %s -e %s", shellQuote(transcriptPath), shellQuote(bridgePath))
	} else {
		flags = fmt.Sprintf("-e %s", shellQuote(bridgePath))
	}
	if prompt != "" {
		flags += fmt.Sprintf(" -p %s", shellQuote(prompt))
	}

	return windowSpec{command: piCmd + " " + flags, env: env}
}

// sendViaBridge delivers a prompt to a pi session via its Unix socket.
func (s *Service) sendViaBridge(sockPath, text string) (*SendResult, error) {
	conn, err := net.DialTimeout("unix", sockPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect to bridge: %w", err)
	}
	defer conn.Close()

	cmd := struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "prompt", Text: text}
	data, _ := json.Marshal(cmd)
	data = append(data, '\n')

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write to bridge: %w", err)
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read bridge response: %w", err)
	}

	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parse bridge response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("bridge: %s", resp.Error)
	}

	return &SendResult{Confirmed: resp.OK}, nil
}

func bridgeSocketPath(sessionID string) string {
	return filepath.Join(os.TempDir(), "cctl-"+sessionID+".sock")
}

// resolveCmd returns the absolute path for a command name.
// Falls back to the original name if lookup fails.
func resolveCmd(cmd string) string {
	if filepath.IsAbs(cmd) {
		return cmd
	}
	if abs, err := exec.LookPath(cmd); err == nil {
		return abs
	}
	return cmd
}

func isPiCmd(cmd string) bool {
	return cmd == "pi" || strings.HasSuffix(cmd, "/pi")
}

func isClaudeCmd(cmd string) bool {
	return cmd == "claude" || strings.HasSuffix(cmd, "/claude")
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
