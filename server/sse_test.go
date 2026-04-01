package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/session"
	"github.com/allouis/cctl/tmux"
)

func setupTestHub(t *testing.T) (*Hub, *db.DB, *mockRunner) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	runner := &mockRunner{hasSession: true}
	cfg := &config.Config{Session: "test", Cmd: "bash", Dir: t.TempDir()}
	svc := session.NewService(store, runner, cfg)
	notify := make(chan struct{}, 1)
	return NewHub(svc, notify), store, runner
}

func TestSSEInitialState(t *testing.T) {
	hub, store, runner := setupTestHub(t)

	store.InsertEvent(db.Event{
		SessionID: "s1", Event: "Notification", State: "NEEDS_INPUT",
		Name: "test", Detail: "permission", Timestamp: 1000,
	})
	runner.windows = []tmux.Window{{Name: "test", Index: "1"}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		hub.ServeSSE(w, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		cancel()
		<-done
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: sessions") {
		t.Error("missing sessions event in SSE output")
	}
	if !strings.Contains(body, "test") {
		t.Error("missing session name in SSE data")
	}
}

func TestSSEBroadcast(t *testing.T) {
	hub, _, _ := setupTestHub(t)

	ch := hub.subscribe()
	defer hub.unsubscribe(ch)

	hub.broadcast([]byte(`[{"Name":"test","State":"WORKING"}]`))

	select {
	case data := <-ch:
		if !strings.Contains(string(data), "test") {
			t.Errorf("data = %q, missing test", string(data))
		}
	case <-time.After(time.Second):
		t.Error("no broadcast received")
	}
}

func TestSSEHeaders(t *testing.T) {
	hub, _, _ := setupTestHub(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so SSE returns quickly

	req := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	hub.ServeSSE(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestSSENoFlusher(t *testing.T) {
	hub, _, _ := setupTestHub(t)

	req := httptest.NewRequest("GET", "/api/events", nil)
	w := &noFlushWriter{code: 0, header: http.Header{}}
	hub.ServeSSE(w, req)

	if w.code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.code)
	}
}

// noFlushWriter is an http.ResponseWriter that does NOT implement http.Flusher.
type noFlushWriter struct {
	code   int
	header http.Header
	body   []byte
}

func (n *noFlushWriter) Header() http.Header         { return n.header }
func (n *noFlushWriter) WriteHeader(code int)         { n.code = code }
func (n *noFlushWriter) Write(b []byte) (int, error)  { n.body = append(n.body, b...); return len(b), nil }
