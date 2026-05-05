# cctl

Manage multiple Claude Code sessions from one place.

cctl runs Claude Code instances inside tmux windows and serves a web dashboard
so you can see what they're all doing without flipping between terminal tabs.

## Why

Running several Claude Code sessions at once gets messy fast. Which one is
waiting for permission? Which one finished? cctl puts everything on one screen --
a dashboard with live updates and a CLI for when you'd rather stay in the
terminal.

## Install

Requires [Nix](https://nixos.org/) with flakes enabled.

```bash
# Run directly
nix run github:allouis/cctl

# Install to your profile
nix profile install github:allouis/cctl
```

Or add it as an input to your own flake:

```nix
{
  inputs.cctl.url = "github:allouis/cctl";

  # Then use inputs.cctl.packages.${system}.default
}
```

### Home-manager module

cctl provides a home-manager module that runs the dashboard as a user service:

```nix
{
  inputs.cctl.url = "github:allouis/cctl";

  outputs = { cctl, ... }: {
    homeConfigurations.me = home-manager.lib.homeManagerConfiguration {
      modules = [
        cctl.homeModules.default
        {
          services.cctl.enable = true;

          # Optional: override the Claude Code package (defaults to pkgs.claude-code)
          # services.cctl.claudePackage = my-claude-package;

          # Optional: change the port (defaults to 4141)
          # services.cctl.port = 8080;
        }
      ];
    };
  };
}
```

The binary embeds the web frontend -- no separate install needed.

## Usage

Start the web dashboard:

```bash
cctl serve                     # start on :4141
```

Manage sessions:

```bash
cctl new my-feature ~/project  # start a new Claude Code session
cctl new fix-bug -p "fix the login redirect loop"  # with an initial prompt
cctl ls                        # list active sessions
cctl peek my-feature           # show latest preview text
cctl send my-feature "try the other approach"
cctl kill my-feature           # stop session (keeps history)
cctl resume my-feature         # pick up where it left off
```

Or use the dashboard at `http://localhost:4141` -- it shows all sessions with
live state updates, transcript viewing, and inline messaging.

## How it works

One binary, three modes:

- **CLI** (`cctl new`, `cctl ls`, ...) -- session management
- **Hook handler** (`cctl hook`) -- receives Claude Code lifecycle events via stdin
- **Web server** (`cctl serve`) -- HTTP API + SSE + embedded PWA

State flows through SQLite. Claude Code hooks fire on every lifecycle event
(tool use, permission prompt, session end, etc.) and cctl writes them to the
database. The dashboard subscribes via SSE for live updates.

Hook configuration is generated automatically whenever you create or resume a
session.

Sessions survive tmux window crashes and can be resumed. In
[jj](https://github.com/jj-vcs/jj) repositories, each session gets its own
workspace so they don't step on each other.

## Configuration

cctl reads optional settings from `~/.config/cctl/settings.json`.

### Session environment variables

Use `sessionEnv` to inject environment variables into every session window.
Values support template variables that expand at window creation time:

| Template | Expands to |
|----------|------------|
| `{{dir}}` | Parent repository directory |
| `{{uuid}}` | Session UUID |
| `{{name}}` | Session name |

```json
{
  "sessionEnv": {
    "RAM_STORE": "{{dir}}/.ram/tasks.jsonl",
    "PROJECT_ROOT": "{{dir}}",
    "SESSION_LOG": "/tmp/logs/{{uuid}}.log"
  }
}
```

This is useful when sessions run in isolated workspaces (e.g. jj worktrees) and
you want external tools to share state with the parent repo instead of each
workspace maintaining its own copy. Values without templates are passed through
as-is.

## Development

```bash
nix develop                    # enter dev shell
just dev                       # Air (Go hot reload) + Vite (frontend HMR)
just check                     # tests + lint + build
```

## License

GPLv3. See [LICENSE](LICENSE).
