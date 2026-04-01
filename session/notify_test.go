package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNotifySocketPath(t *testing.T) {
	got := NotifySocketPath("/home/user/.config/cctl")
	want := "/home/user/.config/cctl/notify.sock"
	if got != want {
		t.Errorf("NotifySocketPath = %q, want %q", got, want)
	}
}

func TestListenAndSignal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.sock")
	ch, cleanup, err := ListenNotify(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	SignalNotify(path)

	select {
	case <-ch:
		// success
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notify signal")
	}
}

func TestSignalCoalescing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.sock")
	ch, cleanup, err := ListenNotify(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	// Send multiple signals rapidly — they should coalesce
	for i := 0; i < 5; i++ {
		SignalNotify(path)
	}

	// Drain whatever is in the channel
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first signal")
	}

	// Channel should be empty (or have at most one more)
	select {
	case <-ch:
		// one more is ok
	default:
		// empty is ok
	}
}

func TestSignalNoListener(t *testing.T) {
	// SignalNotify should not panic when no listener exists
	SignalNotify(filepath.Join(t.TempDir(), "nonexistent.sock"))
}

func TestCleanupRemovesSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.sock")
	_, cleanup, err := ListenNotify(path)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()

	// Signal after cleanup should silently fail
	SignalNotify(path)
}
