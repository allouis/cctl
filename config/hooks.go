package config

import "encoding/json"

type hookEntry struct {
	Matcher string       `json:"matcher"`
	Hooks   []hookAction `json:"hooks"`
}

type hookAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hooksFile struct {
	Hooks map[string][]hookEntry `json:"hooks"`
}

func GenerateHooksJSON(cctlBin string) ([]byte, error) {
	action := hookAction{Type: "command", Command: cctlBin + " hook"}
	entry := hookEntry{Matcher: "", Hooks: []hookAction{action}}
	stopEntry := hookEntry{Hooks: []hookAction{action}}

	h := hooksFile{
		Hooks: map[string][]hookEntry{
			"SessionStart": {entry},
			"SessionEnd":   {entry},
			"Notification": {entry},
			"Stop":         {stopEntry},
			"PreToolUse":   {entry},
			"PostToolUse":  {entry},
			"SubagentStop": {entry},
		},
	}

	return json.MarshalIndent(h, "", "  ")
}
