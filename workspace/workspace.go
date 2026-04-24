package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles creating and removing VCS workspaces/worktrees
// so each session gets an isolated working copy.
type Manager interface {
	// Create makes a new workspace under the given name.
	// repoDir is any path inside the repository. Returns the path to the new working copy.
	Create(repoDir, name string) (workDir string, err error)

	// Remove tears down the workspace and cleans up the directory.
	Remove(repoDir, name string) error

	// IsClean reports whether the workspace has no work that would be lost on removal.
	// "Clean" means: no uncommitted changes in the working copy AND no non-empty
	// ancestor commits that aren't already reachable from trunk().
	IsClean(repoDir, name string) (bool, error)

	// ListWorkspaces returns the VCS-tracked workspace names in the given repo.
	// Used to discover workspaces that cctl no longer tracks (orphans).
	ListWorkspaces(repoDir string) ([]string, error)
}

// Detect returns a Manager if the given directory is inside a supported VCS repo.
// configDir is used as the base for workspace directories.
// Returns nil if no supported VCS is detected.
func Detect(dir, configDir string) Manager {
	if findRepoRoot(dir, ".jj") != "" {
		return &JJManager{configDir: configDir}
	}
	// Future: git worktree support
	// if findRepoRoot(dir, ".git") != "" {
	//     return &GitManager{configDir: configDir}
	// }
	return nil
}

// WorkspaceDir returns the on-disk path for a workspace given the base
// config directory and workspace name.
func WorkspaceDir(configDir, name string) string {
	return filepath.Join(configDir, "workspaces", name)
}

// findRepoRoot walks up from dir looking for a directory containing marker
// (e.g. ".jj" or ".git"). Returns the repo root or empty string.
func findRepoRoot(dir, marker string) string {
	for {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// JJManager creates jujutsu workspaces.
type JJManager struct {
	configDir string
}

func (m *JJManager) Create(repoDir, name string) (string, error) {
	workDir := WorkspaceDir(m.configDir, name)

	if err := os.MkdirAll(filepath.Dir(workDir), 0o755); err != nil {
		return "", fmt.Errorf("create workspace parent: %w", err)
	}

	// Ensure the repo's working copy is not stale before adding a workspace.
	// This is a no-op when fresh, but fixes staleness caused by concurrent
	// workspace operations or external jj commands.
	update := exec.Command("jj", "workspace", "update-stale")
	update.Dir = repoDir
	update.CombinedOutput() // best-effort

	cmd := exec.Command("jj", "workspace", "add", workDir, "--name", name)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj workspace add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return workDir, nil
}

func (m *JJManager) Remove(repoDir, name string) error {
	// Forget the workspace in jj (safe even if dir is already gone)
	cmd := exec.Command("jj", "workspace", "forget", name)
	cmd.Dir = repoDir
	cmd.CombinedOutput() // best-effort

	// Remove the directory
	workDir := WorkspaceDir(m.configDir, name)
	return os.RemoveAll(workDir)
}

// ListWorkspaces returns all jj workspace names in the given repo, excluding "default".
// Used to find jj-known workspaces that cctl has forgotten about (orphans).
func (m *JJManager) ListWorkspaces(repoDir string) ([]string, error) {
	cmd := exec.Command("jj", "workspace", "list", "-T", `name ++ "\n"`)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("jj workspace list: %s: %w", strings.TrimSpace(string(out)), err)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || name == "default" {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// IsClean returns true iff the workspace has no work that would be lost on removal.
// Clean = working copy has no uncommitted changes AND every non-empty ancestor
// commit is already reachable from trunk().
//
// The check runs from the workspace dir so jj snapshots its working copy first;
// running from the main repo would not see uncommitted edits in this workspace.
func (m *JJManager) IsClean(repoDir, name string) (bool, error) {
	workDir := WorkspaceDir(m.configDir, name)
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		// No directory — nothing could be lost.
		return true, nil
	}

	revset := fmt.Sprintf("(::%s@ ~ ::trunk()) ~ empty()", name)
	cmd := exec.Command("jj", "log", "-r", revset, "--no-graph", "-T", `change_id ++ "\n"`)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("jj log: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)) == "", nil
}
