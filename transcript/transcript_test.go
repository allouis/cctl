package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseUserAndAssistant(t *testing.T) {
	path := writeTestFile(t, `{"type":"system","message":{"content":"System init"}}
{"type":"user","message":{"content":"Hello, can you help me?"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Sure, I can help with that."}]}}
{"type":"progress","message":{"content":"running tool"}}
{"type":"user","message":{"content":"Thanks!"}}
{"type":"assistant","message":{"content":[{"type":"thinking","text":"let me think"},{"type":"text","text":"You're welcome!"}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	expected := []Entry{
		{Role: "user", Text: "Hello, can you help me?"},
		{Role: "assistant", Text: "Sure, I can help with that."},
		{Role: "user", Text: "Thanks!"},
		{Role: "assistant", Text: "You're welcome!"},
	}

	for i, e := range expected {
		if entries[i].Role != e.Role {
			t.Errorf("entry[%d].Role = %q, want %q", i, entries[i].Role, e.Role)
		}
		if entries[i].Text != e.Text {
			t.Errorf("entry[%d].Text = %q, want %q", i, entries[i].Text, e.Text)
		}
	}
}

func TestParseToolUse(t *testing.T) {
	path := writeTestFile(t, `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls -la"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"file1.txt\nfile2.txt","tool_use_id":"123"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"I can see two files."}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	if entries[0].Role != "tool_use" {
		t.Errorf("entry[0].Role = %q, want %q", entries[0].Role, "tool_use")
	}
	if entries[0].Text != "$ ls -la" {
		t.Errorf("entry[0].Text = %q, want %q", entries[0].Text, "$ ls -la")
	}

	if entries[1].Role != "tool_result" {
		t.Errorf("entry[1].Role = %q, want %q", entries[1].Role, "tool_result")
	}
	if entries[1].Text != "file1.txt\nfile2.txt" {
		t.Errorf("entry[1].Text = %q, want %q", entries[1].Text, "file1.txt\nfile2.txt")
	}

	if entries[2].Role != "assistant" {
		t.Errorf("entry[2].Role = %q, want %q", entries[2].Role, "assistant")
	}
}

func TestParseToolUseSummary(t *testing.T) {
	tests := []struct {
		name     string
		jsonl    string
		wantText string
	}{
		{
			name:     "Read file",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/foo.go"}}]}}`,
			wantText: "Read /tmp/foo.go",
		},
		{
			name:     "Write file",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/tmp/bar.go"}}]}}`,
			wantText: "Write /tmp/bar.go",
		},
		{
			name:     "Edit file",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/tmp/baz.go"}}]}}`,
			wantText: "Edit /tmp/baz.go",
		},
		{
			name:     "Glob pattern",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Glob","input":{"pattern":"**/*.go"}}]}}`,
			wantText: "Glob **/*.go",
		},
		{
			name:     "Grep pattern",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Grep","input":{"pattern":"func main"}}]}}`,
			wantText: "Grep func main",
		},
		{
			name:     "Task description",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Task","input":{"description":"Search for files"}}]}}`,
			wantText: "Task: Search for files",
		},
		{
			name:     "Unknown tool",
			jsonl:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"CustomTool","input":{"x":1}}]}}`,
			wantText: "CustomTool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, tt.jsonl+"\n")
			entries, err := Parse(path, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			if entries[0].Role != "tool_use" {
				t.Errorf("Role = %q, want %q", entries[0].Role, "tool_use")
			}
			if entries[0].Text != tt.wantText {
				t.Errorf("Text = %q, want %q", entries[0].Text, tt.wantText)
			}
		})
	}
}

func TestParseToolResultArray(t *testing.T) {
	// tool_result content can be an array of {type, text} blocks
	path := writeTestFile(t, `{"type":"user","message":{"content":[{"type":"tool_result","content":[{"type":"text","text":"output line 1"},{"type":"text","text":"output line 2"}],"tool_use_id":"456"}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if entries[0].Role != "tool_result" {
		t.Errorf("Role = %q, want %q", entries[0].Role, "tool_result")
	}
	if entries[0].Text != "output line 1\noutput line 2" {
		t.Errorf("Text = %q, want %q", entries[0].Text, "output line 1\noutput line 2")
	}
}

func TestParseToolResultTruncated(t *testing.T) {
	long := ""
	for i := 0; i < 400; i++ {
		long += "x"
	}
	path := writeTestFile(t, `{"type":"user","message":{"content":[{"type":"tool_result","content":"`+long+`","tool_use_id":"789"}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if len(entries[0].Text) != 303 { // 300 + "..."
		t.Errorf("Text length = %d, want 303", len(entries[0].Text))
	}
}

func TestParseMixedAssistantBlocks(t *testing.T) {
	// An assistant message with both text and tool_use produces two entries
	path := writeTestFile(t, `{"type":"assistant","message":{"content":[{"type":"text","text":"Let me check."},{"type":"tool_use","name":"Bash","input":{"command":"cat file.txt"}}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Role != "assistant" || entries[0].Text != "Let me check." {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[1].Role != "tool_use" || entries[1].Text != "$ cat file.txt" {
		t.Errorf("entry[1] = %+v", entries[1])
	}
}

func TestParseToolUseID(t *testing.T) {
	path := writeTestFile(t, `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_abc123","name":"Bash","input":{"command":"echo hi"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"hi","tool_use_id":"toolu_abc123"}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].ToolUseID != "toolu_abc123" {
		t.Errorf("tool_use ToolUseID = %q, want %q", entries[0].ToolUseID, "toolu_abc123")
	}
	if entries[1].ToolUseID != "toolu_abc123" {
		t.Errorf("tool_result ToolUseID = %q, want %q", entries[1].ToolUseID, "toolu_abc123")
	}
}

func TestParseToolResultIsError(t *testing.T) {
	path := writeTestFile(t, `{"type":"user","message":{"content":[{"type":"tool_result","content":"command failed","tool_use_id":"err1","is_error":true}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"success output","tool_use_id":"ok1","is_error":false}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"no error field","tool_use_id":"none1"}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	if !entries[0].IsError {
		t.Error("entry[0].IsError = false, want true")
	}
	if entries[1].IsError {
		t.Error("entry[1].IsError = true, want false")
	}
	if entries[2].IsError {
		t.Error("entry[2].IsError = true, want false")
	}
}

func TestParseSystemMessagesFiltered(t *testing.T) {
	path := writeTestFile(t, `{"type":"user","message":{"content":"Hello"}}
{"type":"user","message":{"content":"<task-notification>\n<task-id>abc123</task-id>\n<result>some output</result>\n</task-notification>"}}
{"type":"user","message":{"content":"<system-reminder>\nRemember to do X\n</system-reminder>"}}
{"type":"user","message":{"content":"Goodbye"}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (system messages should be filtered)", len(entries))
	}
	if entries[0].Text != "Hello" {
		t.Errorf("entry[0].Text = %q, want %q", entries[0].Text, "Hello")
	}
	if entries[1].Text != "Goodbye" {
		t.Errorf("entry[1].Text = %q, want %q", entries[1].Text, "Goodbye")
	}
}

func TestParseAgentToolSummary(t *testing.T) {
	path := writeTestFile(t, `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Agent","input":{"description":"Research composer libs","prompt":"Find the best..."}}]}}
`)

	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Text != "Agent: Research composer libs" {
		t.Errorf("Text = %q, want %q", entries[0].Text, "Agent: Research composer libs")
	}
}

func TestParseLimit(t *testing.T) {
	path := writeTestFile(t, `{"type":"user","message":{"content":"msg1"}}
{"type":"user","message":{"content":"msg2"}}
{"type":"user","message":{"content":"msg3"}}
{"type":"user","message":{"content":"msg4"}}
{"type":"user","message":{"content":"msg5"}}
`)

	entries, err := Parse(path, 2)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Text != "msg4" {
		t.Errorf("first entry = %q, want %q", entries[0].Text, "msg4")
	}
}

func TestParseMissingFile(t *testing.T) {
	entries, err := Parse("/nonexistent/file.jsonl", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestParseEmptyFile(t *testing.T) {
	path := writeTestFile(t, "")
	entries, err := Parse(path, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}
