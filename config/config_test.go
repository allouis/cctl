package config

import (
	"os"
	"path/filepath"
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

func TestLoadSettings(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Dir: dir}

	// No file → no error, no env
	LoadSettings(cfg)
	if cfg.SessionEnv != nil {
		t.Errorf("expected nil SessionEnv with no file, got %v", cfg.SessionEnv)
	}

	// Valid file
	data := `{"sessionEnv": {"RAM_STORE": "{{dir}}/.ram/tasks.jsonl", "DEBUG": "1"}}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(data), 0644)
	LoadSettings(cfg)
	if len(cfg.SessionEnv) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(cfg.SessionEnv))
	}
	if cfg.SessionEnv["RAM_STORE"] != "{{dir}}/.ram/tasks.jsonl" {
		t.Errorf("RAM_STORE = %q", cfg.SessionEnv["RAM_STORE"])
	}
	if cfg.SessionEnv["DEBUG"] != "1" {
		t.Errorf("DEBUG = %q", cfg.SessionEnv["DEBUG"])
	}
}

func TestLoadSettingsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Dir: dir}
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{bad json"), 0644)
	LoadSettings(cfg)
	if cfg.SessionEnv != nil {
		t.Errorf("invalid JSON should not set SessionEnv, got %v", cfg.SessionEnv)
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
