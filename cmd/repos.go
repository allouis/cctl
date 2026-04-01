package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/allouis/cctl/db"
)

func Repos(store *db.DB, args []string) error {
	if len(args) == 0 {
		return reposList(store)
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: cctl repos add <path>")
		}
		return reposAdd(store, args[1])
	case "rm", "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: cctl repos rm <path>")
		}
		return reposRemove(store, args[1])
	case "list", "ls":
		return reposList(store)
	default:
		return fmt.Errorf("unknown repos subcommand: %s", args[0])
	}
}

func reposAdd(store *db.DB, path string) error {
	path = expandTilde(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if err := store.AddRepo(abs); err != nil {
		return fmt.Errorf("add repo: %w", err)
	}
	fmt.Println(abs)
	return nil
}

func reposRemove(store *db.DB, path string) error {
	path = expandTilde(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	return store.RemoveRepo(abs)
}

func reposList(store *db.DB) error {
	paths, err := store.ListRepos()
	if err != nil {
		return err
	}
	for _, p := range paths {
		fmt.Println(p)
	}
	return nil
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
