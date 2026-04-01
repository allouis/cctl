package db

import (
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestInsertAndListSessions(t *testing.T) {
	d := openTestDB(t)

	err := d.InsertEvent(Event{
		SessionID: "s1",
		Event:     "SessionStart",
		State:     "WORKING",
		Detail:    "startup",
		Name:      "session-one",
		CWD:       "/tmp",
		Timestamp: 1000,
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	sessions, err := d.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Name != "session-one" {
		t.Errorf("name = %q, want %q", sessions[0].Name, "session-one")
	}
	if sessions[0].ExecutorState != "WORKING" {
		t.Errorf("state = %q, want %q", sessions[0].ExecutorState, "WORKING")
	}
}

func TestUpsertSession(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{
		SessionID: "s1", Event: "SessionStart", State: "WORKING",
		Name: "test", Preview: "started", Timestamp: 1000,
	})
	d.InsertEvent(Event{
		SessionID: "s1", Event: "Stop", State: "IDLE",
		Name: "test", Detail: "stopped", Preview: "final msg", Timestamp: 2000,
	})

	s, err := d.GetSession("test")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if s == nil {
		t.Fatal("session not found")
	}
	if s.ExecutorState != "IDLE" {
		t.Errorf("state = %q, want %q", s.ExecutorState, "IDLE")
	}
	if s.ExecutorDetail != "stopped" {
		t.Errorf("detail = %q, want %q", s.ExecutorDetail, "stopped")
	}
	if s.Preview != "final msg" {
		t.Errorf("preview = %q, want %q", s.Preview, "final msg")
	}
}

func TestPreviewPreservation(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{
		SessionID: "s1", Event: "Stop", State: "IDLE",
		Name: "test", Preview: "important message", Timestamp: 1000,
	})
	// Event with empty preview should not overwrite
	d.InsertEvent(Event{
		SessionID: "s1", Event: "PreToolUse", State: "WORKING",
		Name: "test", Preview: "", Timestamp: 2000,
	})

	s, err := d.GetSession("test")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if s.Preview != "important message" {
		t.Errorf("preview = %q, want %q", s.Preview, "important message")
	}
}

func TestGetSessionByID(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{
		SessionID: "abc-123", Event: "SessionStart", State: "WORKING",
		Name: "test", Timestamp: 1000,
	})

	s, err := d.GetSessionByID("abc-123")
	if err != nil {
		t.Fatalf("get session by id: %v", err)
	}
	if s == nil {
		t.Fatal("session not found")
	}
	if s.SessionID != "abc-123" {
		t.Errorf("session_id = %q, want %q", s.SessionID, "abc-123")
	}
}

func TestGetSessionNotFound(t *testing.T) {
	d := openTestDB(t)

	s, err := d.GetSession("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Error("expected nil session")
	}
}

func TestGetEvents(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{SessionID: "s1", Event: "SessionStart", State: "WORKING", Name: "test", Timestamp: 1000})
	d.InsertEvent(Event{SessionID: "s1", Event: "PreToolUse", State: "WORKING", Name: "test", Tool: "Bash", Timestamp: 2000})
	d.InsertEvent(Event{SessionID: "s1", Event: "Stop", State: "IDLE", Name: "test", Timestamp: 3000})

	events, err := d.GetEvents("s1", 10)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	// Ordered by ts DESC
	if events[0].Event != "Stop" {
		t.Errorf("first event = %q, want %q", events[0].Event, "Stop")
	}
}

func TestDeleteSession(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{SessionID: "s1", Event: "SessionStart", State: "WORKING", Name: "test", Timestamp: 1000})

	err := d.DeleteSession("test")
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}

	s, _ := d.GetSession("test")
	if s != nil {
		t.Error("session should be deleted")
	}
}

func TestMultipleSessions(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{SessionID: "s1", Event: "SessionStart", State: "WORKING", Name: "alpha", Timestamp: 1000})
	d.InsertEvent(Event{SessionID: "s2", Event: "SessionStart", State: "WORKING", Name: "beta", Timestamp: 2000})
	d.InsertEvent(Event{SessionID: "s3", Event: "SessionStart", State: "WORKING", Name: "gamma", Timestamp: 3000})

	sessions, err := d.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}
	// Ordered by ts DESC
	if sessions[0].Name != "gamma" {
		t.Errorf("first session = %q, want %q", sessions[0].Name, "gamma")
	}
}

func TestSetConversationID(t *testing.T) {
	d := openTestDB(t)

	d.InsertEvent(Event{SessionID: "s1", Event: "SessionStart", State: "WORKING", Name: "test", Timestamp: 1000})

	err := d.SetConversationID("s1", "conv-uuid-123")
	if err != nil {
		t.Fatalf("set conversation id: %v", err)
	}

	s, _ := d.GetSessionByID("s1")
	if s.ConversationID != "conv-uuid-123" {
		t.Errorf("conversation_id = %q, want %q", s.ConversationID, "conv-uuid-123")
	}
}
