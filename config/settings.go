package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type settings struct {
	SessionEnv map[string]string `json:"sessionEnv"`
}

// LoadSettings reads ~/.config/cctl/settings.json and applies it to cfg.
// Missing file is not an error.
func LoadSettings(cfg *Config) {
	path := filepath.Join(cfg.Dir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return
	}
	if s.SessionEnv != nil {
		cfg.SessionEnv = s.SessionEnv
	}
}
