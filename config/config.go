package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Session string // tmux session name
	Cmd     string // command to run in windows
	DBPath  string // path to SQLite database
	Dir     string // config directory
	Port    int    // web server port
	Safe    bool   // when true, omit --dangerously-skip-permissions
}

// Defaults returns a Config with default values.
func Defaults() *Config {
	dir := DefaultDir()
	return &Config{
		Session: "cc",
		Cmd:     "claude",
		DBPath:  filepath.Join(dir, "cctl.db"),
		Dir:     dir,
		Port:    4141,
	}
}

// DefaultDir returns the cctl config directory (~/.config/cctl).
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "cctl")
	}
	return filepath.Join(home, ".config", "cctl")
}

// SystemPromptPath returns the path to the global system prompt file.
func (c *Config) SystemPromptPath() string {
	return filepath.Join(c.Dir, "system-prompt.md")
}
