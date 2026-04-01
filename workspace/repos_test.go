package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRepos(t *testing.T) {
	root := t.TempDir()

	// Create a mix of dirs: two repos, one plain dir, one file
	mkRepo(t, root, "alpha", ".git")
	mkRepo(t, root, "beta", ".jj")
	os.MkdirAll(filepath.Join(root, "plain"), 0o755)
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("hi"), 0o644)

	got := ScanRepos([]string{root})

	want := []string{
		filepath.Join(root, "alpha"),
		filepath.Join(root, "beta"),
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestScanReposEmpty(t *testing.T) {
	root := t.TempDir()
	got := ScanRepos([]string{root})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestScanReposNonexistent(t *testing.T) {
	got := ScanRepos([]string{"/nonexistent/path/that/does/not/exist"})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestScanReposDeduplicate(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, root, "repo", ".git")

	got := ScanRepos([]string{root, root})
	if len(got) != 1 {
		t.Fatalf("expected 1 repo, got %v", got)
	}
}

func mkRepo(t *testing.T, root, name, marker string) {
	t.Helper()
	os.MkdirAll(filepath.Join(root, name, marker), 0o755)
}
