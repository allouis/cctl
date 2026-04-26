package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Event struct {
	SessionID      string `json:"session_id"`
	Event          string `json:"event"`
	State          string `json:"state"`
	Detail         string `json:"detail"`
	Tool           string `json:"tool"`
	Preview        string `json:"preview"`
	CWD            string `json:"cwd"`
	Name           string `json:"name"`
	TranscriptPath string `json:"transcript_path"`
	Timestamp      int64  `json:"timestamp"`
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
}

type Session struct {
	SessionID      string  `json:"session_id"`
	Name           string  `json:"name"`
	ParentID       *string `json:"parent_id"`
	ProjectID      *string `json:"project_id"`
	WorkState      string  `json:"work_state"`
	ExecutorState  string `json:"executor_state"`
	ExecutorDetail string `json:"executor_detail"`
	Tool           string `json:"tool"`
	Preview        string `json:"preview"`
	Dir            string `json:"dir"`
	CWD            string `json:"cwd"`
	LastEvent      string `json:"last_event"`
	TranscriptPath string `json:"transcript_path"`
	ConversationID string `json:"conversation_id"`
	WindowID       string `json:"window_id"`
	Workspace      string `json:"workspace"`
	Prompt         string `json:"prompt"`
	Safe           bool   `json:"safe"`
	Harness        string `json:"harness"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	Attached       bool   `json:"attached"`
}

const schema = `
CREATE TABLE IF NOT EXISTS events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT    NOT NULL,
    event           TEXT    NOT NULL,
    state           TEXT    NOT NULL,
    detail          TEXT    NOT NULL DEFAULT '',
    tool            TEXT    NOT NULL DEFAULT '',
    preview         TEXT    NOT NULL DEFAULT '',
    cwd             TEXT    NOT NULL DEFAULT '',
    name            TEXT    NOT NULL DEFAULT '',
    transcript_path TEXT    NOT NULL DEFAULT '',
    ts              INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts DESC);

CREATE TABLE IF NOT EXISTS projects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id      TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    parent_id       TEXT,
    project_id      TEXT,
    work_state      TEXT NOT NULL DEFAULT 'running',
    executor_state  TEXT NOT NULL,
    executor_detail TEXT NOT NULL DEFAULT '',
    tool            TEXT NOT NULL DEFAULT '',
    preview         TEXT NOT NULL DEFAULT '',
    dir             TEXT NOT NULL DEFAULT '',
    cwd             TEXT NOT NULL DEFAULT '',
    last_event      TEXT NOT NULL DEFAULT '',
    transcript_path TEXT NOT NULL DEFAULT '',
    conversation_id TEXT NOT NULL DEFAULT '',
    window_id       TEXT NOT NULL DEFAULT '',
    workspace       TEXT NOT NULL DEFAULT '',
    prompt          TEXT NOT NULL DEFAULT '',
    safe            INTEGER NOT NULL DEFAULT 0,
    harness         TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL DEFAULT 0,
    updated_at      INTEGER NOT NULL
);
`

const migration1 = `
ALTER TABLE sessions ADD COLUMN window_id   TEXT    NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN workspace   TEXT    NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN prompt      TEXT    NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN safe        INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN created_at  INTEGER NOT NULL DEFAULT 0;
`

const migration2 = `
ALTER TABLE sessions RENAME COLUMN state TO executor_state;
ALTER TABLE sessions RENAME COLUMN detail TO executor_detail;
ALTER TABLE sessions RENAME COLUMN ts TO updated_at;
`

const migration3 = `
ALTER TABLE sessions ADD COLUMN dir TEXT NOT NULL DEFAULT '';
`

const migration4 = `
ALTER TABLE sessions ADD COLUMN work_state TEXT NOT NULL DEFAULT 'running';
`

const migration5 = `
ALTER TABLE sessions ADD COLUMN parent_id TEXT;
`

const migration6 = `
CREATE TABLE IF NOT EXISTS projects (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
ALTER TABLE sessions ADD COLUMN project_id TEXT;
`

const migration7 = `
CREATE TABLE IF NOT EXISTS repos (
    path TEXT PRIMARY KEY
);
`

const migration8 = `
ALTER TABLE sessions ADD COLUMN harness TEXT NOT NULL DEFAULT '';
`

func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// Force connection pool to discard idle connections so each query
	// gets a fresh WAL snapshot instead of seeing stale data.
	conn.SetConnMaxIdleTime(time.Second)

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	// Migrate existing databases: add new columns if missing.
	// ALTER TABLE ADD COLUMN is a no-op error if the column exists.
	conn.Exec(migration1)
	// Rename state→executor_state, detail→executor_detail, ts→updated_at.
	// RENAME COLUMN errors if the column doesn't exist (already migrated).
	conn.Exec(migration2)
	conn.Exec(migration3)
	// Backfill: set dir=cwd for existing sessions that have no dir set.
	conn.Exec(`UPDATE sessions SET dir = cwd WHERE dir = '' AND cwd != ''`)
	conn.Exec(migration4)
	conn.Exec(migration5)
	conn.Exec(migration6)
	conn.Exec(migration7)
	conn.Exec(migration8)
	// Backfill: DONE executor sessions → done work_state.
	conn.Exec(`UPDATE sessions SET work_state = 'done' WHERE executor_state = 'DONE' AND work_state = 'running'`)

	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) InsertEvent(e Event) error {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}

	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO events (session_id, event, state, detail, tool, preview, cwd, name, transcript_path, ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.SessionID, e.Event, e.State, e.Detail, e.Tool, e.Preview, e.CWD, e.Name, e.TranscriptPath, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Upsert sessions table. Don't let SubagentStop/PostToolUse regress
	// from IDLE/DONE back to WORKING (events can arrive out of order).
	// Session-level fields (window_id, workspace, prompt, safe, created_at)
	// are preserved on update — they're set by Create, not by hook events.
	_, err = tx.Exec(`
		INSERT INTO sessions (session_id, name, parent_id, project_id, work_state, executor_state, executor_detail, tool, preview, dir, cwd, last_event, transcript_path, conversation_id, window_id, workspace, prompt, safe, harness, created_at, updated_at)
		VALUES (?, ?, NULL, NULL, 'running', ?, ?, ?, ?, '', ?, ?, ?, '', '', '', '', 0, '', 0, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			name = excluded.name,
			executor_state = CASE
				WHEN sessions.executor_state IN ('IDLE', 'DONE') AND excluded.executor_state = 'WORKING'
					AND excluded.last_event NOT IN ('SessionStart', 'PreToolUse')
				THEN sessions.executor_state
				ELSE excluded.executor_state
			END,
			executor_detail = CASE
				WHEN sessions.executor_state IN ('IDLE', 'DONE') AND excluded.executor_state = 'WORKING'
					AND excluded.last_event NOT IN ('SessionStart', 'PreToolUse')
				THEN sessions.executor_detail
				ELSE excluded.executor_detail
			END,
			tool = CASE
				WHEN sessions.executor_state IN ('IDLE', 'DONE') AND excluded.executor_state = 'WORKING'
					AND excluded.last_event NOT IN ('SessionStart', 'PreToolUse')
				THEN sessions.tool
				ELSE excluded.tool
			END,
			preview = CASE WHEN excluded.preview != '' THEN excluded.preview ELSE sessions.preview END,
			cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE sessions.cwd END,
			last_event = excluded.last_event,
			transcript_path = CASE WHEN excluded.transcript_path != '' THEN excluded.transcript_path ELSE sessions.transcript_path END,
			updated_at = excluded.updated_at`,
		e.SessionID, e.Name, e.State, e.Detail, e.Tool, e.Preview, e.CWD, e.Event, e.TranscriptPath, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	return tx.Commit()
}

const sessionCols = `session_id, name, parent_id, project_id, work_state, executor_state, executor_detail, tool, preview, dir, cwd, last_event,
	transcript_path, conversation_id, window_id, workspace, prompt, safe, harness, created_at, updated_at`

func scanSession(scanner interface{ Scan(...any) error }) (*Session, error) {
	var s Session
	err := scanner.Scan(
		&s.SessionID, &s.Name, &s.ParentID, &s.ProjectID, &s.WorkState, &s.ExecutorState, &s.ExecutorDetail, &s.Tool, &s.Preview,
		&s.Dir, &s.CWD, &s.LastEvent, &s.TranscriptPath, &s.ConversationID,
		&s.WindowID, &s.Workspace, &s.Prompt, &s.Safe, &s.Harness, &s.CreatedAt, &s.UpdatedAt,
	)
	return &s, err
}

func (d *DB) ListSessions() ([]Session, error) {
	rows, err := d.conn.Query(`SELECT ` + sessionCols + ` FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, *s)
	}
	return sessions, rows.Err()
}

func (d *DB) GetSession(name string) (*Session, error) {
	s, err := scanSession(d.conn.QueryRow(
		`SELECT `+sessionCols+` FROM sessions WHERE name = ? ORDER BY updated_at DESC LIMIT 1`, name))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return s, nil
}

func (d *DB) GetSessionByID(sessionID string) (*Session, error) {
	s, err := scanSession(d.conn.QueryRow(
		`SELECT `+sessionCols+` FROM sessions WHERE session_id = ?`, sessionID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}
	return s, nil
}

func (d *DB) GetEvents(sessionID string, limit int) ([]Event, error) {
	rows, err := d.conn.Query(`
		SELECT session_id, event, state, detail, tool, preview, cwd, name, transcript_path, ts
		FROM events WHERE session_id = ? ORDER BY ts DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.SessionID, &e.Event, &e.State, &e.Detail, &e.Tool, &e.Preview, &e.CWD, &e.Name, &e.TranscriptPath, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// CreateSession inserts a new session row (called at session creation time,
// before any hook events arrive).
func (d *DB) CreateSession(s Session) error {
	if s.CreatedAt == 0 {
		s.CreatedAt = time.Now().Unix()
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = s.CreatedAt
	}
	_, err := d.conn.Exec(`
		INSERT INTO sessions (`+sessionCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SessionID, s.Name, s.ParentID, s.ProjectID, s.WorkState, s.ExecutorState, s.ExecutorDetail, s.Tool, s.Preview,
		s.Dir, s.CWD, s.LastEvent, s.TranscriptPath, s.ConversationID,
		s.WindowID, s.Workspace, s.Prompt, s.Safe, s.Harness, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (d *DB) UpdateSessionState(sessionID, state, detail string) error {
	_, err := d.conn.Exec(
		`UPDATE sessions SET executor_state = ?, executor_detail = ?, updated_at = ? WHERE session_id = ?`,
		state, detail, time.Now().Unix(), sessionID)
	return err
}

func (d *DB) UpdateWindowID(sessionID, windowID string) error {
	_, err := d.conn.Exec(
		`UPDATE sessions SET window_id = ? WHERE session_id = ?`,
		windowID, sessionID)
	return err
}

func (d *DB) UpdateSessionWorkspace(sessionID, workspace string) error {
	_, err := d.conn.Exec(
		`UPDATE sessions SET workspace = ? WHERE session_id = ?`,
		workspace, sessionID)
	return err
}

func (d *DB) DeleteSession(name string) error {
	_, err := d.conn.Exec(`DELETE FROM sessions WHERE name = ?`, name)
	return err
}

func (d *DB) UpdateSessionProject(sessionID string, projectID *string) error {
	_, err := d.conn.Exec(
		`UPDATE sessions SET project_id = ? WHERE session_id = ?`,
		projectID, sessionID)
	return err
}

func (d *DB) SetConversationID(sessionID, conversationID string) error {
	_, err := d.conn.Exec(`UPDATE sessions SET conversation_id = ? WHERE session_id = ?`, conversationID, sessionID)
	return err
}

// Project CRUD

func (d *DB) CreateProject(p Project) error {
	if p.CreatedAt == 0 {
		p.CreatedAt = time.Now().Unix()
	}
	_, err := d.conn.Exec(`INSERT INTO projects (id, name, created_at) VALUES (?, ?, ?)`,
		p.ID, p.Name, p.CreatedAt)
	return err
}

func (d *DB) ListProjects() ([]Project, error) {
	rows, err := d.conn.Query(`SELECT id, name, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) DeleteProject(id string) error {
	_, err := d.conn.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

// Repo CRUD

func (d *DB) AddRepo(path string) error {
	_, err := d.conn.Exec(`INSERT OR IGNORE INTO repos (path) VALUES (?)`, path)
	return err
}

func (d *DB) ListRepos() ([]string, error) {
	rows, err := d.conn.Query(`SELECT path FROM repos ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("query repos: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func (d *DB) RemoveRepo(path string) error {
	_, err := d.conn.Exec(`DELETE FROM repos WHERE path = ?`, path)
	return err
}
