package config

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Session != "cc" {
		t.Errorf("Session = %q, want %q", cfg.Session, "cc")
	}
	if cfg.Cmd != "claude" {
		t.Errorf("Cmd = %q, want %q", cfg.Cmd, "claude")
	}
	if cfg.Port != 4141 {
		t.Errorf("Port = %d, want %d", cfg.Port, 4141)
	}
	if !strings.HasSuffix(cfg.DBPath, "cctl.db") {
		t.Errorf("DBPath = %q, want suffix %q", cfg.DBPath, "cctl.db")
	}
	if cfg.Safe != false {
		t.Errorf("Safe = %v, want false", cfg.Safe)
	}
}

func TestGenerateHooksJSON(t *testing.T) {
	data, err := GenerateHooksJSON("/usr/local/bin/cctl")
	if err != nil {
		t.Fatalf("GenerateHooksJSON failed: %v", err)
	}

	s := string(data)
	if len(s) == 0 {
		t.Fatal("empty output")
	}

	for _, event := range []string{"SessionStart", "SessionEnd", "Notification", "Stop", "PreToolUse", "PostToolUse"} {
		if !strings.Contains(s, event) {
			t.Errorf("missing hook event %q", event)
		}
	}

	if !strings.Contains(s, "/usr/local/bin/cctl hook") {
		t.Error("missing cctl hook command")
	}
}
