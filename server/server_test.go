package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/session"
	"github.com/allouis/cctl/tmux"
)

type mockRunner struct {
	hasSession bool
	windows    []tmux.Window
	sentKeys   []string
	killed     []string
	selected   []string
}

func (m *mockRunner) HasSession(name string) bool                        { return m.hasSession }
func (m *mockRunner) NewSession(session, window, command string) error    { return nil }
func (m *mockRunner) SetEnv(session, key, value string) error            { return nil }
func (m *mockRunner) UnsetEnv(session, key string) error                 { return nil }
func (m *mockRunner) NewWindow(session, window, command, dir string, env []string) (string, error) {
	return "@1", nil
}
func (m *mockRunner) ListWindows(session string) ([]tmux.Window, error) {
	return m.windows, nil
}
func (m *mockRunner) ActiveWindowIDs(session string) map[string]bool { return nil }
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

func setupTestServer(t *testing.T) (*Server, *db.DB, *mockRunner) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	runner := &mockRunner{hasSession: true}
	cfg := &config.Config{Session: "test", Cmd: "bash", Dir: t.TempDir()}
	svc := session.NewService(store, runner, cfg)
	notify := make(chan struct{}, 1)
	srv := New(svc, cfg, notify)
	return srv, store, runner
}

func TestGetSessions(t *testing.T) {
	srv, store, runner := setupTestServer(t)

	store.InsertEvent(db.Event{
		SessionID: "s1", Event: "Notification", State: "NEEDS_INPUT",
		Name: "alpha", Detail: "permission", Timestamp: 1000,
	})
	// Provide a live window so the session isn't marked DONE
	runner.windows = []tmux.Window{{Name: "alpha", Index: "1"}}

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var sessions []db.Session
	json.Unmarshal(w.Body.Bytes(), &sessions)
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Name != "alpha" {
		t.Errorf("name = %q, want %q", sessions[0].Name, "alpha")
	}
	// Verify enrichment: NEEDS_INPUT state should be preserved
	if sessions[0].ExecutorState != "NEEDS_INPUT" {
		t.Errorf("state = %q, want NEEDS_INPUT (enriched)", sessions[0].ExecutorState)
	}
}

func TestGetSessionByName(t *testing.T) {
	srv, store, runner := setupTestServer(t)

	store.InsertEvent(db.Event{
		SessionID: "s1", Event: "Notification", State: "NEEDS_INPUT",
		Name: "mytest", Detail: "permission", Preview: "hello", Timestamp: 1000,
	})
	runner.windows = []tmux.Window{{Name: "mytest", Index: "1"}}

	req := httptest.NewRequest("GET", "/api/sessions/mytest", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var s db.Session
	json.Unmarshal(w.Body.Bytes(), &s)
	if s.Name != "mytest" {
		t.Errorf("name = %q, want %q", s.Name, "mytest")
	}
}

func TestGetSessionNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/sessions/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestDeleteSession(t *testing.T) {
	srv, store, runner := setupTestServer(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "killme", ExecutorState: "WORKING",
		WindowID: "@3", UpdatedAt: 1000,
	})

	req := httptest.NewRequest("DELETE", "/api/sessions/killme", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if len(runner.killed) != 1 || runner.killed[0] != "@3" {
		t.Errorf("killed = %v, want [@3]", runner.killed)
	}
}

func TestSendText(t *testing.T) {
	srv, store, runner := setupTestServer(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "mywin", ExecutorState: "IDLE",
		WindowID: "@2", UpdatedAt: 1000,
	})

	body := strings.NewReader(`{"text":"hello world"}`)
	req := httptest.NewRequest("POST", "/api/send/mywin", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if len(runner.sentKeys) != 1 {
		t.Fatalf("sentKeys = %v, want 1 entry", runner.sentKeys)
	}
	if runner.sentKeys[0] != "@2:hello world" {
		t.Errorf("sentKeys[0] = %q, want %q", runner.sentKeys[0], "@2:hello world")
	}
}

func TestTranscriptNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/transcript/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestTranscriptEndpoint(t *testing.T) {
	srv, store, _ := setupTestServer(t)

	// Create a test transcript file
	dir := t.TempDir()
	path := dir + "/transcript.jsonl"
	content := `{"type":"user","message":{"content":"Hello"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Hi there!"}]}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	store.InsertEvent(db.Event{
		SessionID: "s1", Event: "SessionStart", State: "WORKING",
		Name: "withlog", TranscriptPath: path, Timestamp: 1000,
	})

	req := httptest.NewRequest("GET", "/api/transcript/withlog?limit=10", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var entries []struct {
		Role string `json:"role"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Role != "user" || entries[0].Text != "Hello" {
		t.Errorf("entry[0] = %+v, want user/Hello", entries[0])
	}
	if entries[1].Role != "assistant" || entries[1].Text != "Hi there!" {
		t.Errorf("entry[1] = %+v, want assistant/Hi there!", entries[1])
	}
}

func TestTakeoverEndpoint(t *testing.T) {
	srv, store, _ := setupTestServer(t)

	store.CreateSession(db.Session{
		SessionID: "s1", Name: "attached", ExecutorState: "IDLE",
		WindowID: "@2", UpdatedAt: 1000,
	})

	req := httptest.NewRequest("POST", "/api/takeover/attached", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestCreateSession(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body := strings.NewReader(`{"name":"newsession","dir":"/tmp"}`)
	req := httptest.NewRequest("POST", "/api/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}
