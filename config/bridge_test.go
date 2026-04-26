package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBridgeExtension(t *testing.T) {
	dir := t.TempDir()

	path, err := WriteBridgeExtension(dir)
	if err != nil {
		t.Fatalf("write bridge: %v", err)
	}

	if filepath.Base(path) != "pi-bridge.ts" {
		t.Errorf("path = %q, want pi-bridge.ts basename", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bridge file: %v", err)
	}

	content := string(data)
	checks := []string{
		"@mariozechner/pi-coding-agent",
		"CCTL_SESSION_ID",
		"CCTL_BIN",
		"fireHook",
		"SessionStart",
		"PreToolUse",
		"PostToolUse",
		"SessionEnd",
		"sendUserMessage",
		"createServer",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("bridge source missing %q", check)
		}
	}
}

func TestWriteBridgeExtensionOverwrites(t *testing.T) {
	dir := t.TempDir()

	path, _ := WriteBridgeExtension(dir)
	os.WriteFile(path, []byte("stale"), 0o644)

	path2, err := WriteBridgeExtension(dir)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if path != path2 {
		t.Errorf("paths differ: %q vs %q", path, path2)
	}

	data, _ := os.ReadFile(path)
	if string(data) == "stale" {
		t.Error("file was not overwritten")
	}
}
