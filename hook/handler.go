package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allouis/cctl/db"
)

// Input represents the JSON payload from Claude Code lifecycle hooks.
type Input struct {
	SessionID        string          `json:"session_id"`
	HookEventName    string          `json:"hook_event_name"`
	CWD              string          `json:"cwd"`
	TranscriptPath   string          `json:"transcript_path"`
	ToolName         string          `json:"tool_name"`
	ToolInput        json.RawMessage `json:"tool_input"`
	ToolResponse     json.RawMessage `json:"tool_response"`
	NotificationType string          `json:"notification_type"`
	Message          string          `json:"message"`
	Source           string          `json:"source"`
	Reason           string          `json:"reason"`
	LastAssistant    string          `json:"last_assistant_message"`
}

// Result holds the mapped state from a hook event.
type Result struct {
	State   string
	Detail  string
	Tool    string
	Preview string
}

// Process maps a hook event to a state Result.
func Process(input *Input) *Result {
	switch input.HookEventName {
	case "SessionStart":
		source := input.Source
		if source == "" {
			source = "startup"
		}
		return &Result{
			State:   "WORKING",
			Detail:  source,
			Preview: fmt.Sprintf("[Session started: %s]", source),
		}

	case "PreToolUse":
		return processPreToolUse(input)

	case "PostToolUse":
		return processPostToolUse(input)

	case "Notification":
		return processNotification(input)

	case "Stop":
		r := &Result{State: "IDLE", Detail: "stopped"}
		if input.LastAssistant != "" {
			r.Preview = truncate(input.LastAssistant, 500)
		}
		return r

	case "SubagentStop":
		return &Result{State: "WORKING", Detail: "subagent_done"}

	case "SessionEnd":
		reason := input.Reason
		if reason == "" {
			reason = "unknown"
		}
		return &Result{
			State:   "DONE",
			Detail:  reason,
			Preview: fmt.Sprintf("[Session ended: %s]", reason),
		}

	default:
		return nil
	}
}

func processPreToolUse(input *Input) *Result {
	tool := input.ToolName
	if tool == "" {
		tool = "unknown"
	}

	var ti map[string]interface{}
	if len(input.ToolInput) > 0 {
		json.Unmarshal(input.ToolInput, &ti)
	}

	getString := func(key string) string {
		if v, ok := ti[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	var detail string
	switch tool {
	case "Bash":
		cmd := truncate(getString("command"), 120)
		detail = "$ " + cmd
	case "Write", "Edit":
		fp := getString("file_path")
		detail = tool + ": " + filepath.Base(fp)
	case "Read":
		fp := getString("file_path")
		detail = "reading: " + filepath.Base(fp)
	case "Glob", "Grep":
		pattern := getString("pattern")
		if pattern == "" {
			pattern = getString("regex")
		}
		detail = tool + ": " + truncate(pattern, 60)
	case "Task":
		prompt := truncate(getString("prompt"), 80)
		detail = "subagent: " + prompt
	default:
		detail = tool
	}

	return &Result{State: "WORKING", Detail: detail, Tool: tool}
}

func processPostToolUse(input *Input) *Result {
	tool := input.ToolName
	if tool == "" {
		tool = "unknown"
	}

	suffix := ""
	if len(input.ToolResponse) > 0 {
		var resp map[string]interface{}
		if json.Unmarshal(input.ToolResponse, &resp) == nil {
			if v, ok := resp["success"]; ok {
				if b, ok := v.(bool); ok {
					if b {
						suffix = " ✓"
					} else {
						suffix = " ✗"
					}
				}
			}
		}
	}

	return &Result{
		State:  "WORKING",
		Detail: "done:" + tool + suffix,
		Tool:   tool,
	}
}

func processNotification(input *Input) *Result {
	msg := truncate(input.Message, 200)

	switch input.NotificationType {
	case "permission_prompt":
		return &Result{State: "NEEDS_INPUT", Detail: "permission", Preview: msg}
	case "idle_prompt":
		return &Result{State: "IDLE", Detail: "waiting"}
	case "auth_success":
		return &Result{State: "WORKING", Detail: "auth_ok"}
	case "elicitation_dialog":
		return &Result{State: "NEEDS_INPUT", Detail: "elicitation", Preview: msg}
	default:
		return &Result{State: "NEEDS_INPUT", Detail: "notification:" + input.NotificationType, Preview: msg}
	}
}

// ToEvent converts a hook Input and Result into a db.Event.
func ToEvent(input *Input, result *Result) db.Event {
	name := os.Getenv("CCTL_NAME")
	if name == "" {
		name = filepath.Base(input.CWD)
	}

	return db.Event{
		SessionID:      input.SessionID,
		Event:          input.HookEventName,
		State:          result.State,
		Detail:         result.Detail,
		Tool:           result.Tool,
		Preview:        result.Preview,
		CWD:            input.CWD,
		Name:           name,
		TranscriptPath: input.TranscriptPath,
		Timestamp:      time.Now().Unix(),
	}
}

// ParseInput reads and parses the hook JSON from stdin.
func ParseInput(data []byte) (*Input, error) {
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parse hook input: %w", err)
	}
	if input.SessionID == "" {
		return nil, nil
	}
	return &input, nil
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
