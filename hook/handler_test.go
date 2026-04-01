package hook

import (
	"encoding/json"
	"testing"
)

func TestProcessSessionStart(t *testing.T) {
	tests := []struct {
		name   string
		input  Input
		state  string
		detail string
	}{
		{
			name:   "default source",
			input:  Input{SessionID: "s1", HookEventName: "SessionStart"},
			state:  "WORKING",
			detail: "startup",
		},
		{
			name:   "explicit source",
			input:  Input{SessionID: "s1", HookEventName: "SessionStart", Source: "resume"},
			state:  "WORKING",
			detail: "resume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Process(&tt.input)
			if r == nil {
				t.Fatal("nil result")
			}
			if r.State != tt.state {
				t.Errorf("state = %q, want %q", r.State, tt.state)
			}
			if r.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", r.Detail, tt.detail)
			}
		})
	}
}

func TestProcessPreToolUse(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolInput map[string]interface{}
		detail    string
		tool      string
	}{
		{
			name:      "Bash command",
			toolName:  "Bash",
			toolInput: map[string]interface{}{"command": "npm test"},
			detail:    "$ npm test",
			tool:      "Bash",
		},
		{
			name:      "Write file",
			toolName:  "Write",
			toolInput: map[string]interface{}{"file_path": "/home/user/app/main.go"},
			detail:    "Write: main.go",
			tool:      "Write",
		},
		{
			name:      "Edit file",
			toolName:  "Edit",
			toolInput: map[string]interface{}{"file_path": "/src/utils.ts"},
			detail:    "Edit: utils.ts",
			tool:      "Edit",
		},
		{
			name:      "Read file",
			toolName:  "Read",
			toolInput: map[string]interface{}{"file_path": "/etc/config.yaml"},
			detail:    "reading: config.yaml",
			tool:      "Read",
		},
		{
			name:      "Glob pattern",
			toolName:  "Glob",
			toolInput: map[string]interface{}{"pattern": "**/*.go"},
			detail:    "Glob: **/*.go",
			tool:      "Glob",
		},
		{
			name:      "Grep pattern",
			toolName:  "Grep",
			toolInput: map[string]interface{}{"pattern": "TODO"},
			detail:    "Grep: TODO",
			tool:      "Grep",
		},
		{
			name:      "Task subagent",
			toolName:  "Task",
			toolInput: map[string]interface{}{"prompt": "investigate the build failure"},
			detail:    "subagent: investigate the build failure",
			tool:      "Task",
		},
		{
			name:      "unknown tool",
			toolName:  "WebSearch",
			toolInput: map[string]interface{}{},
			detail:    "WebSearch",
			tool:      "WebSearch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tiJSON, _ := json.Marshal(tt.toolInput)
			input := &Input{
				SessionID:     "s1",
				HookEventName: "PreToolUse",
				ToolName:      tt.toolName,
				ToolInput:     tiJSON,
			}
			r := Process(input)
			if r == nil {
				t.Fatal("nil result")
			}
			if r.State != "WORKING" {
				t.Errorf("state = %q, want WORKING", r.State)
			}
			if r.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", r.Detail, tt.detail)
			}
			if r.Tool != tt.tool {
				t.Errorf("tool = %q, want %q", r.Tool, tt.tool)
			}
		})
	}
}

func TestProcessPostToolUse(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		response map[string]interface{}
		detail   string
	}{
		{
			name:     "success",
			toolName: "Bash",
			response: map[string]interface{}{"success": true},
			detail:   "done:Bash ✓",
		},
		{
			name:     "failure",
			toolName: "Bash",
			response: map[string]interface{}{"success": false},
			detail:   "done:Bash ✗",
		},
		{
			name:     "no response",
			toolName: "Read",
			response: nil,
			detail:   "done:Read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var trJSON json.RawMessage
			if tt.response != nil {
				trJSON, _ = json.Marshal(tt.response)
			}
			input := &Input{
				SessionID:     "s1",
				HookEventName: "PostToolUse",
				ToolName:      tt.toolName,
				ToolResponse:  trJSON,
			}
			r := Process(input)
			if r == nil {
				t.Fatal("nil result")
			}
			if r.State != "WORKING" {
				t.Errorf("state = %q, want WORKING", r.State)
			}
			if r.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", r.Detail, tt.detail)
			}
		})
	}
}

func TestProcessNotification(t *testing.T) {
	tests := []struct {
		name   string
		ntype  string
		state  string
		detail string
	}{
		{"permission prompt", "permission_prompt", "NEEDS_INPUT", "permission"},
		{"idle prompt", "idle_prompt", "IDLE", "waiting"},
		{"auth success", "auth_success", "WORKING", "auth_ok"},
		{"elicitation dialog", "elicitation_dialog", "NEEDS_INPUT", "elicitation"},
		{"unknown notification", "custom_type", "NEEDS_INPUT", "notification:custom_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &Input{
				SessionID:        "s1",
				HookEventName:    "Notification",
				NotificationType: tt.ntype,
				Message:          "test message",
			}
			r := Process(input)
			if r == nil {
				t.Fatal("nil result")
			}
			if r.State != tt.state {
				t.Errorf("state = %q, want %q", r.State, tt.state)
			}
			if r.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", r.Detail, tt.detail)
			}
		})
	}
}

func TestProcessStop(t *testing.T) {
	t.Run("with last message", func(t *testing.T) {
		input := &Input{
			SessionID:     "s1",
			HookEventName: "Stop",
			LastAssistant: "I've finished the task.",
		}
		r := Process(input)
		if r.State != "IDLE" {
			t.Errorf("state = %q, want IDLE", r.State)
		}
		if r.Preview != "I've finished the task." {
			t.Errorf("preview = %q, want %q", r.Preview, "I've finished the task.")
		}
	})

	t.Run("without last message", func(t *testing.T) {
		input := &Input{SessionID: "s1", HookEventName: "Stop"}
		r := Process(input)
		if r.State != "IDLE" {
			t.Errorf("state = %q, want IDLE", r.State)
		}
		if r.Preview != "" {
			t.Errorf("preview = %q, want empty", r.Preview)
		}
	})
}

func TestProcessSubagentStop(t *testing.T) {
	input := &Input{SessionID: "s1", HookEventName: "SubagentStop"}
	r := Process(input)
	if r.State != "WORKING" {
		t.Errorf("state = %q, want WORKING", r.State)
	}
	if r.Detail != "subagent_done" {
		t.Errorf("detail = %q, want %q", r.Detail, "subagent_done")
	}
}

func TestProcessSessionEnd(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		input := &Input{SessionID: "s1", HookEventName: "SessionEnd", Reason: "user_exit"}
		r := Process(input)
		if r.State != "DONE" {
			t.Errorf("state = %q, want DONE", r.State)
		}
		if r.Detail != "user_exit" {
			t.Errorf("detail = %q, want %q", r.Detail, "user_exit")
		}
	})

	t.Run("without reason", func(t *testing.T) {
		input := &Input{SessionID: "s1", HookEventName: "SessionEnd"}
		r := Process(input)
		if r.Detail != "unknown" {
			t.Errorf("detail = %q, want %q", r.Detail, "unknown")
		}
	})
}

func TestProcessUnknownEvent(t *testing.T) {
	input := &Input{SessionID: "s1", HookEventName: "SomeFutureEvent"}
	r := Process(input)
	if r != nil {
		t.Error("expected nil result for unknown event")
	}
}

func TestParseInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		data := []byte(`{"session_id":"abc","hook_event_name":"SessionStart","cwd":"/tmp"}`)
		input, err := ParseInput(data)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if input.SessionID != "abc" {
			t.Errorf("session_id = %q, want %q", input.SessionID, "abc")
		}
	})

	t.Run("empty session id", func(t *testing.T) {
		data := []byte(`{"session_id":"","hook_event_name":"SessionStart"}`)
		input, err := ParseInput(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if input != nil {
			t.Error("expected nil for empty session id")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		data := []byte(`not json`)
		_, err := ParseInput(data)
		if err == nil {
			t.Error("expected error for invalid json")
		}
	})
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("hello world", 5); got != "hello" {
		t.Errorf("truncate long = %q, want %q", got, "hello")
	}
	if got := truncate("  spaces  ", 20); got != "spaces" {
		t.Errorf("truncate trimmed = %q, want %q", got, "spaces")
	}
}
