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
	// Create makes a new workspace for the given session.
	// repoDir is the root of the repository, sessionID is used as the workspace name.
	// Returns the path to the new working copy.
	Create(repoDir, sessionID string) (workDir string, err error)

	// Remove tears down the workspace and cleans up the directory.
	Remove(repoDir, sessionID string) error
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
// config directory and session ID.
func WorkspaceDir(configDir, sessionID string) string {
	return filepath.Join(configDir, "workspaces", sessionID)
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

func (m *JJManager) Create(repoDir, sessionID string) (string, error) {
	workDir := WorkspaceDir(m.configDir, sessionID)

	if err := os.MkdirAll(filepath.Dir(workDir), 0o755); err != nil {
		return "", fmt.Errorf("create workspace parent: %w", err)
	}

	// Ensure the repo's working copy is not stale before adding a workspace.
	// This is a no-op when fresh, but fixes staleness caused by concurrent
	// workspace operations or external jj commands.
	update := exec.Command("jj", "workspace", "update-stale")
	update.Dir = repoDir
	update.CombinedOutput() // best-effort

	cmd := exec.Command("jj", "workspace", "add", workDir, "--name", sessionID)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj workspace add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return workDir, nil
}

func (m *JJManager) Remove(repoDir, sessionID string) error {
	// Forget the workspace in jj (safe even if dir is already gone)
	cmd := exec.Command("jj", "workspace", "forget", sessionID)
	cmd.Dir = repoDir
	cmd.CombinedOutput() // best-effort

	// Remove the directory
	workDir := WorkspaceDir(m.configDir, sessionID)
	return os.RemoveAll(workDir)
}
