package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectJJ(t *testing.T) {
	dir := t.TempDir()
	// No VCS — should return nil
	if mgr := Detect(dir, t.TempDir()); mgr != nil {
		t.Error("expected nil for non-VCS directory")
	}

	// Create .jj dir — should detect jj
	os.Mkdir(filepath.Join(dir, ".jj"), 0o755)
	mgr := Detect(dir, t.TempDir())
	if mgr == nil {
		t.Fatal("expected JJManager for directory with .jj")
	}
	if _, ok := mgr.(*JJManager); !ok {
		t.Errorf("expected *JJManager, got %T", mgr)
	}
}

func TestDetectWalksUp(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".jj"), 0o755)

	sub := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(sub, 0o755)

	mgr := Detect(sub, t.TempDir())
	if mgr == nil {
		t.Fatal("expected to find .jj by walking up")
	}
}

func TestFindRepoRoot(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".jj"), 0o755)

	sub := filepath.Join(root, "src", "pkg")
	os.MkdirAll(sub, 0o755)

	got := findRepoRoot(sub, ".jj")
	if got != root {
		t.Errorf("findRepoRoot = %q, want %q", got, root)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	dir := t.TempDir()
	got := findRepoRoot(dir, ".jj")
	if got != "" {
		t.Errorf("findRepoRoot = %q, want empty", got)
	}
}

func TestWorkspaceDir(t *testing.T) {
	got := WorkspaceDir("/home/user/.config/cctl", "abc-123")
	want := "/home/user/.config/cctl/workspaces/abc-123"
	if got != want {
		t.Errorf("WorkspaceDir = %q, want %q", got, want)
	}
}
