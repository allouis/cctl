package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Entry struct {
	Role      string `json:"role"`
	Text      string `json:"text"`
	FullText  string `json:"full_text,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
}

type rawMessage struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type messageContent struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type toolResultBlock struct {
	Type      string          `json:"type"`
	Content   json.RawMessage `json:"content,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Parse reads a Claude Code JSONL transcript and returns entries.
// Entries include user text, assistant text, tool_use, and tool_result.
func Parse(path string, limit int) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		var raw rawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		switch raw.Type {
		case "user":
			entries = append(entries, parseUserMessage(raw.Message)...)
		case "assistant":
			entries = append(entries, parseAssistantMessage(raw.Message)...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan transcript: %w", err)
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	if entries == nil {
		entries = []Entry{}
	}

	return entries, nil
}

func parseUserMessage(data json.RawMessage) []Entry {
	var msg messageContent
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	// Content can be a plain string
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		if text == "" || isSystemMessage(text) {
			return nil
		}
		return []Entry{{Role: "user", Text: text}}
	}

	// Content can be an array of blocks (tool_result, text, etc.)
	var blocks []toolResultBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}

	var entries []Entry
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		text := extractToolResultContent(b.Content)
		if text != "" {
			entries = append(entries, Entry{Role: "tool_result", Text: truncate(text, 300), IsError: b.IsError, ToolUseID: b.ToolUseID})
		}
	}
	return entries
}

func parseAssistantMessage(data json.RawMessage) []Entry {
	var msg messageContent
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}

	var entries []Entry
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				entries = append(entries, Entry{Role: "assistant", Text: b.Text})
			}
		case "tool_use":
			summary, full := summarizeToolInput(b.Name, b.Input)
			e := Entry{Role: "tool_use", Text: summary, ToolUseID: b.ID}
			if full != summary {
				e.FullText = full
			}
			entries = append(entries, e)
		}
	}
	return entries
}

func summarizeToolInput(name string, input json.RawMessage) (summary string, full string) {
	if len(input) == 0 {
		return name, name
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return name, name
	}

	switch name {
	case "Bash":
		if cmd, ok := extractString(fields["command"]); ok {
			full = "$ " + cmd
			return "$ " + truncate(cmd, 200), full
		}
	case "Read":
		if fp, ok := extractString(fields["file_path"]); ok {
			return "Read " + fp, "Read " + fp
		}
	case "Write":
		if fp, ok := extractString(fields["file_path"]); ok {
			return "Write " + fp, "Write " + fp
		}
	case "Edit", "MultiEdit":
		if fp, ok := extractString(fields["file_path"]); ok {
			return name + " " + fp, name + " " + fp
		}
	case "Glob":
		if pat, ok := extractString(fields["pattern"]); ok {
			return "Glob " + pat, "Glob " + pat
		}
	case "Grep":
		if pat, ok := extractString(fields["pattern"]); ok {
			return "Grep " + pat, "Grep " + pat
		}
	case "Agent":
		if desc, ok := extractString(fields["description"]); ok {
			full = "Agent: " + desc
			return "Agent: " + truncate(desc, 100), full
		}
	case "Task":
		if desc, ok := extractString(fields["description"]); ok {
			full = "Task: " + desc
			return "Task: " + truncate(desc, 100), full
		}
	case "TodoWrite":
		return "TodoWrite", "TodoWrite"
	}

	return name, name
}

func extractString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Content can be a plain string
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	// Content can be an array of {type, text} blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

func isSystemMessage(text string) bool {
	return strings.HasPrefix(text, "<task-notification>") ||
		strings.HasPrefix(text, "<system-reminder>")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
