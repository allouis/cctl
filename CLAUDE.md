# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cctl (Crowd Control) is a single Go binary that manages Claude Code sessions via tmux and serves a PWA web dashboard.

## Build and test

```bash
just check                     # Run all checks (tests + lint + build)
just test                      # Run all tests (Go + frontend)
just test-go                   # Run Go tests only
just test-frontend             # Run frontend tests only
just lint                      # Run all linters (go vet + tsc)
just build                     # Full Nix build
```

Nix is the only build system. `flake.nix` builds the React frontend via `buildNpmPackage` and injects it into the Go build via `preBuild`. `nix build` runs both Go and frontend tests.

Use `just` commands for fast iteration, `nix build` as the final gate before committing.

### Commit verification

Always run `nix build` before committing. This is the only build that catches:
- `npmDepsHash` drift (after any change to `package-lock.json`)
- TypeScript errors that Vite dev mode doesn't surface (it skips type-checking)

Do not use `go build`, `npm run build`, or `tsc` as substitutes — they don't verify the Nix derivation.

If `nix build` fails with "npmDepsHash is out of date":
1. Set `npmDepsHash = pkgs.lib.fakeHash;` in `flake.nix`
2. Run `nix build`, copy the `got: sha256-...` hash from the error
3. Replace `fakeHash` with the real hash

### Frontend dev workflow

```bash
just dev
# Open http://localhost:5173
```

This starts both:
- **Air** — watches `.go` files, auto-rebuilds and restarts the Go server on `:4141`
- **Vite** — serves the frontend on `:5173` with HMR, proxies `/api/*` to Go

Edit React components for instant HMR. Edit Go files for automatic rebuild (~1s).

The Go binary uses build tags to skip `go:embed` in dev — no `nix build` needed to start developing. Production builds pass `-tags embed` via `flake.nix`.

### Frontend quality gates

UI work is not complete until:
1. Visually verified with `agent-browser` (take screenshots, check layout at different sizes)
2. Tests written for any new components or hooks

`agent-browser` is a locally-installed CLI for headless browser automation (not in the Nix shell). Use it to verify layout:
```bash
agent-browser set viewport 1280 720
agent-browser open http://localhost:4141
agent-browser screenshot /tmp/page.png
agent-browser snapshot -i              # list interactive elements
agent-browser click @e2                # click by ref
agent-browser eval 'document.querySelector("aside").offsetHeight'
```

## Architecture

One binary, three modes:
- `cctl new/ls/peek/...` — CLI commands
- `cctl hook` — Hook handler (reads JSON from stdin, writes to SQLite)
- `cctl serve` — Web server (HTTP + SSE + embedded PWA)

State flows through SQLite at `~/.config/cctl/cctl.db`.

### Data flow

```
Claude Code hooks → stdin JSON → hook/handler.go (parse + state mapping)
    → db.InsertEvent (SQLite upsert)
    → session.SignalNotify (Unix socket fire-and-forget)
    → server/sse.go Hub (listens on socket, queries DB, broadcasts)
    → SSE → browser dashboard
```

The hook process is fire-and-forget: it writes to the DB and pokes the Unix socket, then exits. The server's SSE hub wakes on the socket signal, fetches current state from the DB, and broadcasts to connected clients.

### HTTP API

Response types: `web/app/src/types.ts`. Full API reference: `docs/API.md` (gitignored, local only).

When adding or modifying API endpoints, update both `docs/API.md` and
`web/app/src/types.ts`.

### Key packages

```
main.go              Entrypoint + subcommand dispatch (manual switch on os.Args[1])
cmd/                 CLI command implementations
session/             Central coordinator: ties tmux, DB, and notify together
hook/                Event → state mapping (pure functions, no I/O)
db/                  SQLite setup, migrations, queries (WAL mode, modernc.org/sqlite)
server/              HTTP routes + SSE hub with hash-based deduplication
tmux/                Runner interface + exec wrappers
transcript/          JSONL transcript parser
web/                 Embedded React PWA (Vite + React 19 + TypeScript + Tailwind v4)
config/              Paths, defaults, hooks.json generation, settings.json
```

### Frontend architecture

The React app follows a layered architecture (Presentation-Domain-Data):

```
web/app/src/
  domain/            Pure TypeScript — NO React imports
    session.ts       State predicates, color mappings, formatting
    transcript.ts    Tool result pairing logic
    notifications.ts NEEDS_INPUT transition detection
  api/               Data/gateway layer — typed fetch wrappers
    client.ts
  hooks/             React adapters — thin wrappers over domain functions
  components/        Presentation — pure rendering, no business logic
  context/           React context providers
```

#### Where does new code go?

Before writing code, ask: "Does this need React to work?"

- **If no** → `domain/`. Pure functions, plain TypeScript, no React imports. This includes: data transformations, validation, formatting, business rules, state predicates, anything that answers a question about data. Test with plain `describe`/`it`/`expect`.
- **If it fetches data from the server** → `api/client.ts`. Typed fetch wrappers only. No state, no React.
- **If it manages React state or lifecycle** → `hooks/`. But keep hooks thin — they should call domain functions for any logic, not contain it. The hook's job is: when does state change (lifecycle), not how to compute the new state (domain). Test with `renderHook`.
- **If it renders UI** → `components/`. Components receive data as props and render. No `fetch`, no business logic, no data transformations. If you're writing a `useMemo` that doesn't touch the DOM, it probably belongs in `domain/`.

#### Smell tests

- **Importing from `react` in `domain/`?** Wrong layer. Domain must be framework-independent.
- **A `useMemo` in a component that doesn't reference refs or DOM?** Extract to a domain function.
- **A hook with more than ~5 lines of non-React logic?** Extract the logic to domain, keep the hook as an adapter.
- **A component with conditional logic based on data (not UI state)?** The condition is domain logic — extract a predicate.
- **Can't test something without `renderHook` or a DOM?** If the logic being tested doesn't inherently need React, it's in the wrong layer.

## State model

Sessions have two independent state layers.

### Work state (application-level, executor-independent)

| State     | Meaning |
|-----------|---------|
| `pending` | Queued, not yet started |
| `running` | Active (an executor may or may not be alive) |
| `review`  | PR open, waiting for human/CI |
| `done`    | Objective achieved |
| `failed`  | Unrecoverable error |

### Executor state (Claude-Code-specific, hook-driven)

| State         | Trigger |
|---------------|---------|
| `STARTING`    | Create or Resume (DB row written before first hook) |
| `WORKING`     | SessionStart, PreToolUse, PostToolUse |
| `NEEDS_INPUT` | Notification (permission_prompt, elicitation_dialog) |
| `IDLE`        | Notification (idle_prompt), Stop |
| `DEAD`        | Window lost (crash, manual close, tmux restart) or Kill |
| `DONE`        | SessionEnd |

Sessions are durable — they survive window death. A session in DEAD or DONE executor state can be resumed with `Resume()`, which creates a new tmux window with the same `--session-id`. Work state is unchanged by executor death.

### Session identity

- **SessionID** (UUID): Primary key, passed to Claude via `--session-id`, stable across Resume.
- **Name**: Human-readable, used as tmux window display name. Not unique.
- **WindowID** (`@N`): tmux-assigned, ephemeral. Used to target SendKeys/KillWindow. Empty when no window exists.

### Non-obvious state behaviors

- **Idle inference**: Claude doesn't always send `idle_prompt`. The session service infers IDLE if state=WORKING, last event=SessionStart, and age > 5 seconds.
- **DEAD detection**: If a session has a window_id but that window no longer exists in tmux, enrichSession marks it DEAD. The SSE hub's 5-second fallback ticker catches these.
- **Event ordering resilience**: Hook processes may fire out of order. The DB upsert uses CASE statements to prevent regressions (e.g., PostToolUse won't overwrite IDLE→WORKING).
- **Immediate DB row**: Create() writes the session to the DB before the first hook fires, so there's no gap where the session is invisible.

## CLI flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--session <name>` | `cc` | tmux session name |
| `--cmd <command>` | `claude` | Command to run in windows |
| `--db <path>` | `~/.config/cctl/cctl.db` | Database path |
| `--safe` | off | Omit `--dangerously-skip-permissions` from claude |

`serve` subcommand flags:

| Flag | Default | Purpose |
|------|---------|---------|
| `--port <n>` | `4141` | Web server port |
| `--dev` | off | Frontend development mode (serves from disk instead of embedded) |

Internal env var (not user-facing): `CCTL_NAME` — set by Create() in the tmux command string, read by `hook`.

### Settings file

`~/.config/cctl/settings.json` — optional, loaded by `config.LoadSettings()` at startup.

Currently supports `sessionEnv`: a map of env vars injected into every session window via tmux `-e`. Values support `{{dir}}`, `{{uuid}}`, and `{{name}}` template expansion. See README for details.

## Testing patterns

- **tmux mocking**: `tmux.Runner` is an interface; tests use a mock that tracks created windows, sent keys, and killed sessions — no tmux required.
- **In-memory DB**: All tests use `:memory:` SQLite via `db.Open(":memory:")`.
- **Hook tests**: Table-driven, covering all event types and edge cases.
- **SSE tests**: Use `httptest` + context-based timeouts.
