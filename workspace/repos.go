package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanRepos walks each parent directory one level deep and returns
// absolute paths of subdirectories that contain .git or .jj.
func ScanRepos(parents []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, parent := range parents {
		parent = expandTilde(parent)
		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			abs := filepath.Join(parent, e.Name())
			if seen[abs] {
				continue
			}
			if isVCSDir(abs) {
				seen[abs] = true
				result = append(result, abs)
			}
		}
	}

	sort.Strings(result)
	return result
}

func isVCSDir(dir string) bool {
	for _, marker := range []string{".git", ".jj"} {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
