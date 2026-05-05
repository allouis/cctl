package cmd

import (
	"fmt"
	"os"
)

const version = "0.1.0"

const usage = `cctl — manage Claude Code sessions via tmux

Usage:
  cctl [flags] <command> [args]

Global flags:
  --session <name>    tmux session name (default: cc)
  --cmd <command>     command to run in windows (default: claude)
  --db <path>         database path (default: ~/.config/cctl/cctl.db)
  --safe              omit --dangerously-skip-permissions from claude

Session management:
  new <name> [dir]    Start a new session in dir (default: cwd)
        -p, --prompt <text>       initial prompt to send
        --harness <claude|pi>     executor type (default: auto-detect from --cmd)
        --safe                    omit --dangerously-skip-permissions
  resume <name>       Resume a dead/done session
  kill <name>         Kill session (keeps history and workspace)
  delete <name>       Permanently remove session and workspace

Session interaction:
  ls [-a|--all]       List sessions (default: active only)
  peek <name>         Show session state and preview text
  log <name> [n]      Show last n transcript entries (default: 20)
  go <name|number>    Switch to session's tmux window
  send <name> <text>  Send text to session
  attach [name]       Attach to tmux session

Repository management:
  repos add <path>    Register a directory containing repos
  repos rm <path>     Unregister a directory
  repos [list]        List registered directories

Maintenance:
  workspace prune     Remove workspaces for dead/done sessions with no unlanded work
  serve [--port n]    Start web dashboard (default: 4141)
  hook                Handle hook event (stdin, internal)
  bar                 Render tmux status bar (internal)
  version             Print version

Aliases: ls|list, go|focus, delete|rm, peek|preview, log|transcript, attach|a
Config:  ~/.config/cctl/settings.json (see README for sessionEnv docs)`

func Help() {
	fmt.Fprintln(os.Stderr, usage)
}

type helpError struct{}

func (helpError) Error() string { return "" }

// checkHelp prints usage and returns a helpError if args contain -h or --help.
func checkHelp(args []string, usage string) error {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Fprintln(os.Stderr, usage)
			return helpError{}
		}
	}
	return nil
}

func Version() {
	fmt.Println("cctl", version)
}
