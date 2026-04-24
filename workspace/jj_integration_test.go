package workspace

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func jjAvailable() bool {
	_, err := exec.LookPath("jj")
	return err == nil
}

// setupJJRepo creates a jj repo with one described commit on main and returns its root.
func setupJJRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("jj", args...)
		cmd.Dir = root
		cmd.Env = append(cmd.Environ(),
			"JJ_USER=test",
			"JJ_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("jj %s: %s: %v", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
		}
	}
	run("git", "init", "--colocate")
	run("bookmark", "create", "main")
	run("describe", "-m", "root")
	run("new", "main")
	run("bookmark", "move", "main", "--to", "@-")
	return root
}

func TestIsCleanNewWorkspace(t *testing.T) {
	if !jjAvailable() {
		t.Skip("jj not available")
	}
	repo := setupJJRepo(t)
	configDir := t.TempDir()
	m := &JJManager{configDir: configDir}

	if _, err := m.Create(repo, "ws-new"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	clean, err := m.IsClean(repo, "ws-new")
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Error("expected new workspace to be clean")
	}
}

func TestIsCleanWithUncommittedChanges(t *testing.T) {
	if !jjAvailable() {
		t.Skip("jj not available")
	}
	repo := setupJJRepo(t)
	configDir := t.TempDir()
	m := &JJManager{configDir: configDir}

	workDir, err := m.Create(repo, "ws-dirty")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Write a file in the workspace -> working copy becomes non-empty.
	if err := exec.Command("sh", "-c", "echo hi > "+filepath.Join(workDir, "file.txt")).Run(); err != nil {
		t.Fatal(err)
	}

	clean, err := m.IsClean(repo, "ws-dirty")
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if clean {
		t.Error("expected workspace with uncommitted changes to be dirty")
	}
}

func TestIsCleanWithUnlandedCommit(t *testing.T) {
	if !jjAvailable() {
		t.Skip("jj not available")
	}
	repo := setupJJRepo(t)
	configDir := t.TempDir()
	m := &JJManager{configDir: configDir}

	workDir, err := m.Create(repo, "ws-work")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Make a file, describe the commit, create a fresh empty @. Now the
	// workspace has an ancestor commit that is non-empty and not in main.
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("jj", args...)
		cmd.Dir = workDir
		cmd.Env = append(cmd.Environ(), "JJ_USER=test", "JJ_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("jj %s: %s: %v", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
		}
	}
	if err := exec.Command("sh", "-c", "echo work > "+filepath.Join(workDir, "work.txt")).Run(); err != nil {
		t.Fatal(err)
	}
	run("commit", "-m", "in-progress")

	clean, err := m.IsClean(repo, "ws-work")
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if clean {
		t.Error("expected workspace with unlanded commits to be dirty")
	}
}
