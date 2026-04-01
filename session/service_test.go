package session

import (
	"testing"
	"time"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/tmux"
)

type mockRunner struct {
	hasSession      bool
	windows         []tmux.Window
	activeWindowIDs map[string]bool
	sentKeys        []string
	killed          []string
	created         []string
	selected        []string
}

func (m *mockRunner) HasSession(name string) bool                     { return m.hasSession }
func (m *mockRunner) NewSession(session, window, command string) error { return nil }
func (m *mockRunner) NewWindow(session, window, command, dir string) (string, error) {
	m.created = append(m.created, window)
	return "@1", nil
}
func (m *mockRunner) ListWindows(session string) ([]tmux.Window, error) {
	return m.windows, nil
}
func (m *mockRunner) ActiveWindowIDs(session string) map[string]bool {
	return m.activeWindowIDs
}
func (m *mockRunner) SelectWindow(session, target string) error {
	m.selected = append(m.selected, target)
	return nil
}
func (m *mockRunner) SendKeys(session, target, keys string) error {
	m.sentKeys = append(m.sentKeys, target+":"+keys)
	return nil
}
func (m *mockRunner) CapturePane(session, target string) (string, error) {
	return "", nil
}
func (m *mockRunner) KillWindow(session, target string) error {
	m.killed = append(m.killed, target)
	return nil
}
func (m *mockRunner) KillSession(session string) error { return nil }

func setupTest(t *testing.T) (*Service, *db.DB, *mockRunner) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	runner := &mockRunner{hasSession: true}
	cfg := &config.Config{Session: "test", Cmd: "bash", Dir: t.TempDir()}
	svc := NewService(store, runner, cfg)
	return svc, store, runner
}

func TestListEmpty(t *testing.T) {
	svc, _, _ := setupTest(t)

	sessions, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

func TestListReturnsDone(t *testing.T) {
	svc, store, _ := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "finished", ExecutorState: "DONE",
		UpdatedAt: time.Now().Unix(),
	})

	sessions, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].ExecutorState != "DONE" {
		t.Errorf("state = %q, want DONE", sessions[0].ExecutorState)
	}
}

func TestListMarksOrphanedAsDead(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "orphan", ExecutorState: "WORKING",
		WindowID: "@5", UpdatedAt: time.Now().Unix(),
	})
	// No live tmux windows — @5 is gone
	runner.windows = []tmux.Window{}

	sessions, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].ExecutorState != "DEAD" {
		t.Errorf("state = %q, want DEAD", sessions[0].ExecutorState)
	}
}

func TestListSkipsDash(t *testing.T) {
	svc, store, _ := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "dash", ExecutorState: "WORKING",
		UpdatedAt: time.Now().Unix(),
	})

	sessions, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0 (dash should be filtered)", len(sessions))
	}
}

func TestListPreservesLiveSession(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "alpha", ExecutorState: "WORKING",
		WindowID: "@1", UpdatedAt: time.Now().Unix(),
	})
	runner.windows = []tmux.Window{{ID: "@1", Name: "alpha", Index: "1"}}

	sessions, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].ExecutorState != "WORKING" {
		t.Errorf("state = %q, want WORKING", sessions[0].ExecutorState)
	}
}

func TestGetEnriched(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "mytest", ExecutorState: "NEEDS_INPUT",
		ExecutorDetail: "permission", Preview: "Allow?", WindowID: "@1",
		UpdatedAt: time.Now().Unix(),
	})
	runner.windows = []tmux.Window{{ID: "@1", Name: "mytest", Index: "1"}}

	s, err := svc.Get("mytest")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if s == nil {
		t.Fatal("session not found")
	}
	if s.ExecutorState != "NEEDS_INPUT" {
		t.Errorf("state = %q, want NEEDS_INPUT", s.ExecutorState)
	}
}

func TestGetMarksDead(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "gone", ExecutorState: "WORKING",
		WindowID: "@5", UpdatedAt: time.Now().Unix(),
	})
	runner.windows = []tmux.Window{}

	s, err := svc.Get("gone")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if s == nil {
		t.Fatal("session not found")
	}
	if s.ExecutorState != "DEAD" {
		t.Errorf("state = %q, want DEAD", s.ExecutorState)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, _, _ := setupTest(t)

	s, err := svc.Get("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Error("expected nil session")
	}
}

func TestKill(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "victim", ExecutorState: "WORKING",
		WindowID: "@3", UpdatedAt: 1000,
	})

	if err := svc.Kill("victim"); err != nil {
		t.Fatalf("kill: %v", err)
	}

	if len(runner.killed) != 1 || runner.killed[0] != "@3" {
		t.Errorf("killed = %v, want [@3]", runner.killed)
	}

	s, _ := store.GetSession("victim")
	if s == nil {
		t.Fatal("session should still exist in DB (soft delete)")
	}
	if s.ExecutorState != "DEAD" {
		t.Errorf("state = %q, want DEAD", s.ExecutorState)
	}
}

func TestKillNotFound(t *testing.T) {
	svc, _, _ := setupTest(t)

	err := svc.Kill("nonexistent")
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestResume(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "revive", ExecutorState: "DEAD",
		CWD: "/tmp", UpdatedAt: 1000,
	})

	if err := svc.Resume("revive"); err != nil {
		t.Fatalf("resume: %v", err)
	}

	if len(runner.created) != 1 || runner.created[0] != "revive" {
		t.Errorf("created = %v, want [revive]", runner.created)
	}

	s, _ := store.GetSession("revive")
	if s == nil {
		t.Fatal("session not found after resume")
	}
	if s.ExecutorState != "STARTING" {
		t.Errorf("state = %q, want STARTING", s.ExecutorState)
	}
	if s.WindowID != "@1" {
		t.Errorf("window_id = %q, want @1", s.WindowID)
	}
}

func TestResumeNotResumable(t *testing.T) {
	svc, store, _ := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "active", ExecutorState: "WORKING",
		WindowID: "@1", UpdatedAt: 1000,
	})

	err := svc.Resume("active")
	if err == nil {
		t.Error("expected error resuming active session")
	}
}

func TestSend(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "target", ExecutorState: "IDLE",
		WindowID: "@2", UpdatedAt: 1000,
	})

	if _, err := svc.Send("target", "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(runner.sentKeys) != 1 || runner.sentKeys[0] != "@2:hello" {
		t.Errorf("sentKeys = %v, want [@2:hello]", runner.sentKeys)
	}
}

func TestSendNoWindow(t *testing.T) {
	svc, store, _ := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "dead", ExecutorState: "DEAD",
		UpdatedAt: 1000,
	})

	_, err := svc.Send("dead", "hello")
	if err == nil {
		t.Error("expected error sending to session with no window")
	}
}

func TestCreate(t *testing.T) {
	svc, store, runner := setupTest(t)

	if _, err := svc.Create(CreateOpts{Name: "newproject", Dir: "/tmp"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if len(runner.created) != 1 || runner.created[0] != "newproject" {
		t.Errorf("created = %v, want [newproject]", runner.created)
	}

	// DB row should exist immediately
	s, _ := store.GetSession("newproject")
	if s == nil {
		t.Fatal("session not found in DB after create")
	}
	if s.ExecutorState != "STARTING" {
		t.Errorf("state = %q, want STARTING", s.ExecutorState)
	}
	if s.WindowID != "@1" {
		t.Errorf("window_id = %q, want @1", s.WindowID)
	}
}

func TestCreateWithPrompt(t *testing.T) {
	svc, store, _ := setupTest(t)

	if _, err := svc.Create(CreateOpts{Name: "prompted", Dir: "/tmp", Prompt: "Fix the bugs"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	s, _ := store.GetSession("prompted")
	if s == nil {
		t.Fatal("session not found")
	}
	if s.Prompt != "Fix the bugs" {
		t.Errorf("prompt = %q, want 'Fix the bugs'", s.Prompt)
	}
}

func TestCreateSafe(t *testing.T) {
	svc, store, _ := setupTest(t)

	if _, err := svc.Create(CreateOpts{Name: "safesession", Dir: "/tmp", Safe: true}); err != nil {
		t.Fatalf("create: %v", err)
	}

	s, _ := store.GetSession("safesession")
	if s == nil {
		t.Fatal("session not found")
	}
	if !s.Safe {
		t.Error("expected safe=true")
	}
}

func TestTakeover(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "owned", ExecutorState: "IDLE",
		WindowID: "@2", UpdatedAt: 1000,
	})
	runner.activeWindowIDs = map[string]bool{"@2": true}
	runner.windows = []tmux.Window{{ID: "@2", Name: "owned", Index: "2"}}

	if err := svc.Takeover("owned"); err != nil {
		t.Fatalf("takeover: %v", err)
	}
	if len(runner.selected) != 1 || runner.selected[0] != "0" {
		t.Errorf("selected = %v, want [0]", runner.selected)
	}
}

func TestTakeoverNotAttached(t *testing.T) {
	svc, store, runner := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "free", ExecutorState: "IDLE",
		WindowID: "@2", UpdatedAt: 1000,
	})
	runner.windows = []tmux.Window{{ID: "@2", Name: "free", Index: "2"}}

	if err := svc.Takeover("free"); err != nil {
		t.Fatalf("takeover: %v", err)
	}
	if len(runner.selected) != 0 {
		t.Errorf("expected no SelectWindow call, got %v", runner.selected)
	}
}

func TestTakeoverNoWindow(t *testing.T) {
	svc, store, _ := setupTest(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "dead", ExecutorState: "DEAD",
		UpdatedAt: 1000,
	})

	err := svc.Takeover("dead")
	if err == nil {
		t.Error("expected error for session with no window")
	}
}

func TestTranscriptEmptyPath(t *testing.T) {
	svc, store, _ := setupTest(t)
	store.CreateSession(db.Session{
		SessionID: "s1", Name: "new", ExecutorState: "STARTING",
		UpdatedAt: time.Now().Unix(),
	})

	entries, err := svc.Transcript("new", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestInferIdle(t *testing.T) {
	t.Run("infers idle after SessionStart timeout", func(t *testing.T) {
		s := &db.Session{
			ExecutorState:     "WORKING",
			LastEvent: "SessionStart",
			UpdatedAt: time.Now().Unix() - 10,
		}
		InferIdle(s)
		if s.ExecutorState != "IDLE" {
			t.Errorf("state = %q, want IDLE", s.ExecutorState)
		}
		if s.ExecutorDetail != "ready" {
			t.Errorf("detail = %q, want ready", s.ExecutorDetail)
		}
	})

	t.Run("does not infer idle for recent SessionStart", func(t *testing.T) {
		s := &db.Session{
			ExecutorState:     "WORKING",
			LastEvent: "SessionStart",
			UpdatedAt: time.Now().Unix(),
		}
		InferIdle(s)
		if s.ExecutorState != "WORKING" {
			t.Errorf("state = %q, want WORKING", s.ExecutorState)
		}
	})

	t.Run("does not affect non-WORKING state", func(t *testing.T) {
		s := &db.Session{
			ExecutorState:     "NEEDS_INPUT",
			LastEvent: "SessionStart",
			UpdatedAt: time.Now().Unix() - 10,
		}
		InferIdle(s)
		if s.ExecutorState != "NEEDS_INPUT" {
			t.Errorf("state = %q, want NEEDS_INPUT", s.ExecutorState)
		}
	})

	t.Run("does not affect non-SessionStart events", func(t *testing.T) {
		s := &db.Session{
			ExecutorState:     "WORKING",
			LastEvent: "PreToolUse",
			UpdatedAt: time.Now().Unix() - 10,
		}
		InferIdle(s)
		if s.ExecutorState != "WORKING" {
			t.Errorf("state = %q, want WORKING", s.ExecutorState)
		}
	})
}
