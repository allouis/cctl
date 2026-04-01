package main

import (
	"testing"
)

func TestParseArgsDefaults(t *testing.T) {
	cfg, subcmd, args := parseArgs([]string{})
	if cfg.Session != "cc" {
		t.Errorf("Session = %q, want cc", cfg.Session)
	}
	if cfg.Cmd != "claude" {
		t.Errorf("Cmd = %q, want claude", cfg.Cmd)
	}
	if cfg.Safe {
		t.Error("Safe should be false by default")
	}
	if subcmd != "" {
		t.Errorf("subcmd = %q, want empty", subcmd)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestParseArgsFlags(t *testing.T) {
	cfg, subcmd, args := parseArgs([]string{
		"--session", "mysess", "--safe", "--cmd", "bash", "new", "proj", "/tmp",
	})
	if cfg.Session != "mysess" {
		t.Errorf("Session = %q, want mysess", cfg.Session)
	}
	if cfg.Cmd != "bash" {
		t.Errorf("Cmd = %q, want bash", cfg.Cmd)
	}
	if !cfg.Safe {
		t.Error("Safe should be true")
	}
	if subcmd != "new" {
		t.Errorf("subcmd = %q, want new", subcmd)
	}
	if len(args) != 2 || args[0] != "proj" || args[1] != "/tmp" {
		t.Errorf("args = %v, want [proj /tmp]", args)
	}
}

func TestParseArgsNoFlags(t *testing.T) {
	cfg, subcmd, args := parseArgs([]string{"ls"})
	if cfg.Session != "cc" {
		t.Errorf("Session = %q, want cc", cfg.Session)
	}
	if subcmd != "ls" {
		t.Errorf("subcmd = %q, want ls", subcmd)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestParseArgsDBFlag(t *testing.T) {
	cfg, _, _ := parseArgs([]string{"--db", "/tmp/test.db", "ls"})
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want /tmp/test.db", cfg.DBPath)
	}
	if cfg.Dir != "/tmp" {
		t.Errorf("Dir = %q, want /tmp", cfg.Dir)
	}
}
