package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allouis/cctl/cmd"
	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/db"
	"github.com/allouis/cctl/session"
	"github.com/allouis/cctl/tmux"
)

func main() {
	cfg, subcmd, args := parseArgs(os.Args[1:])

	if err := run(cfg, subcmd, args); err != nil {
		if err.Error() == "" {
			return
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// parseArgs extracts global flags before the subcommand.
// Usage: cctl [--session name] [--cmd command] [--db path] [--safe] <subcommand> [args...]
func parseArgs(raw []string) (*config.Config, string, []string) {
	cfg := config.Defaults()

	i := 0
	for i < len(raw) {
		switch raw[i] {
		case "--session":
			if i+1 < len(raw) {
				cfg.Session = raw[i+1]
				i += 2
				continue
			}
		case "--cmd":
			if i+1 < len(raw) {
				cfg.Cmd = raw[i+1]
				i += 2
				continue
			}
		case "--db":
			if i+1 < len(raw) {
				cfg.DBPath = raw[i+1]
				// Update Dir to match the directory containing the DB
				cfg.Dir = filepath.Dir(raw[i+1])
				i += 2
				continue
			}
		case "--safe":
			cfg.Safe = true
			i++
			continue
		}
		break
	}

	subcmd := ""
	var args []string
	if i < len(raw) {
		subcmd = raw[i]
		args = raw[i+1:]
	}

	return cfg, subcmd, args
}

func run(cfg *config.Config, subcmd string, args []string) error {
	switch subcmd {
	case "help", "-h", "--help":
		cmd.Help()
		return nil
	case "version", "-v", "--version":
		cmd.Version()
		return nil
	}

	// Commands that need the database
	store, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	// Commands that only need the database (no tmux, no service)
	switch subcmd {
	case "repos":
		return cmd.Repos(store, args)
	}

	runner := &tmux.RealRunner{}
	svc := session.NewService(store, runner, cfg)

	switch subcmd {
	case "", "ls", "list":
		return cmd.List(svc, args)
	case "new":
		return cmd.New(svc, args)
	case "resume":
		return cmd.Resume(svc, args)
	case "peek", "preview":
		return cmd.Peek(svc, args)
	case "log", "transcript":
		return cmd.Log(svc, args)
	case "go", "focus":
		return cmd.Go(svc, args)
	case "send":
		return cmd.Send(svc, args)
	case "kill":
		return cmd.Kill(svc, args)
	case "delete", "rm":
		return cmd.Delete(svc, args)
	case "attach", "a":
		return cmd.Attach(runner, cfg.Session, args)
	case "hook":
		return cmd.Hook(cfg, store)
	case "bar":
		return cmd.Bar(svc)
	case "serve":
		return cmd.Serve(cfg, svc, args)
	case "workspace":
		return cmd.Workspace(svc, args)
	default:
		cmd.Help()
		return fmt.Errorf("unknown command: %s", subcmd)
	}
}
