package cmd

import (
	"fmt"
	"os"
)

const version = "0.1.0"

const usage = `cctl — manage Claude Code sessions via tmux

Usage:
  cctl [flags] <command> [args]

Flags:
  --session <name>    tmux session name (default: cc)
  --cmd <command>     command to run in windows (default: claude)
  --db <path>         database path (default: ~/.config/cctl/cctl.db)
  --safe              omit --dangerously-skip-permissions from claude

Commands:
  new <name> [dir] [-p prompt] [--safe]  Start a new session
  resume <name>          Resume a dead/done session
  ls [-a|--all]          List sessions (default: active only)
  peek <name>            Show preview text
  log <name> [n]         Show transcript entries
  go <name|number>       Switch to tmux window
  send <name> <text>     Send text to session
  kill <name>            Kill session (keeps history, workspace)
  delete <name>          Permanently remove session and workspace
  attach [name]          Attach to tmux session
  repos add <path>       Register a directory containing repos
  repos rm <path>        Unregister a directory
  repos [list]           List registered directories
  serve [--port 4141]    Start web dashboard
  hook                   Handle hook event (stdin)
  bar                    Render tmux status bar
  version                Print version

Aliases: ls|list, go|focus, delete|rm, peek|preview, log|transcript, attach|a`

func Help() {
	fmt.Fprintln(os.Stderr, usage)
}

func Version() {
	fmt.Println("cctl", version)
}
