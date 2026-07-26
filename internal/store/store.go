// Package store persists servers, jobs and job history.
//
// SQLite keeps the MVP to a single binary with no service to run. The queries
// are deliberately plain SQL behind narrow methods, so moving to Postgres when
// multi-tenancy arrives is a change to this package alone.
package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // pure-Go driver: no cgo, no system SQLite
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS servers (
	id                 TEXT PRIMARY KEY,
	user_id            TEXT NOT NULL DEFAULT '',
	name               TEXT NOT NULL,
	connection_kind    TEXT NOT NULL,
	host               TEXT NOT NULL DEFAULT '',
	port               INTEGER NOT NULL DEFAULT 22,
	username           TEXT NOT NULL DEFAULT '',
	region             TEXT NOT NULL DEFAULT '',
	status             TEXT NOT NULL DEFAULT 'unknown',
	facts_json         TEXT NOT NULL DEFAULT '{}',
	services_json      TEXT NOT NULL DEFAULT '[]',
	sealed_password    TEXT NOT NULL DEFAULT '',
	sealed_private_key TEXT NOT NULL DEFAULT '',
	public_key         TEXT NOT NULL DEFAULT '',
	host_key           TEXT NOT NULL DEFAULT '',
	created_at         TEXT NOT NULL,
	last_seen_at       TEXT
);

CREATE TABLE IF NOT EXISTS jobs (
	id          TEXT PRIMARY KEY,
	server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	kind        TEXT NOT NULL,
	title       TEXT NOT NULL,
	actor       TEXT NOT NULL DEFAULT '',
	status      TEXT NOT NULL,
	started_at  TEXT NOT NULL,
	finished_at TEXT,
	changed     INTEGER NOT NULL DEFAULT 0,
	unchanged   INTEGER NOT NULL DEFAULT 0,
	skipped     INTEGER NOT NULL DEFAULT 0,
	failed      INTEGER NOT NULL DEFAULT 0,
	error       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS jobs_started_at_idx ON jobs(started_at DESC);

CREATE TABLE IF NOT EXISTS job_events (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	job_id     TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	seq        INTEGER NOT NULL,
	type       TEXT NOT NULL,
	step_id    TEXT NOT NULL DEFAULT '',
	step_title TEXT NOT NULL DEFAULT '',
	level      TEXT NOT NULL DEFAULT '',
	message    TEXT NOT NULL DEFAULT '',
	outcome    TEXT NOT NULL DEFAULT '',
	at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS job_events_job_seq_idx ON job_events(job_id, seq);

-- Fleet-wide snapshots, written whenever a server's facts are refreshed.
-- Sparklines are drawn from these; with no history the UI shows none rather
-- than inventing a trend.
CREATE TABLE IF NOT EXISTS fleet_samples (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id          TEXT NOT NULL DEFAULT '',
	at               TEXT NOT NULL,
	server_count     INTEGER NOT NULL,
	online_count     INTEGER NOT NULL,
	cpu_usage        REAL NOT NULL,
	memory_used      INTEGER NOT NULL,
	memory_total     INTEGER NOT NULL,
	disk_used        INTEGER NOT NULL,
	disk_total       INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS fleet_samples_at_idx ON fleet_samples(at DESC);

-- Single-user for now, but a table rather than a config value: multi-tenancy
-- is on the roadmap and moving from one row to many should not be a rewrite.
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

-- Software installed on a server from the catalogue.
--
-- The generated password is sealed exactly like an SSH credential: it is what
-- someone signs in to their database with, and we hold it only so the
-- connection details can be shown again later.
CREATE TABLE IF NOT EXISTS installations (
	id              TEXT PRIMARY KEY,
	user_id         TEXT NOT NULL,
	server_id       TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	tool_id         TEXT NOT NULL,
	version         TEXT NOT NULL DEFAULT '',
	status          TEXT NOT NULL,
	sealed_password TEXT NOT NULL DEFAULT '',
	last_job_id     TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	-- One installation of a tool per server: the compose project name is fixed,
	-- so a second one would fight the first over the same containers.
	UNIQUE(server_id, tool_id)
);

CREATE INDEX IF NOT EXISTS installations_server_idx ON installations(server_id);

-- Queries someone has kept for later.
--
-- Per server and tool rather than global: "yesterday's signups" means a
-- different table on a different machine, and a saved query that silently
-- refers to somewhere else is worse than no saved query.
CREATE TABLE IF NOT EXISTS saved_queries (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	server_id  TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	tool_id    TEXT NOT NULL,
	name       TEXT NOT NULL,
	query      TEXT NOT NULL,
	created_at TEXT NOT NULL,
	-- Saving under a name you have used before replaces it, which is what
	-- someone refining a query expects rather than a second entry.
	UNIQUE(user_id, server_id, tool_id, name)
);

CREATE INDEX IF NOT EXISTS saved_queries_lookup_idx
	ON saved_queries(user_id, server_id, tool_id);

-- Where backups are sent.
--
-- One per account for now. It is deliberately somewhere else: a copy on the
-- same disk as the database survives a clumsy DROP TABLE and nothing worse,
-- and the failure people actually lose data to is the disk going away.
CREATE TABLE IF NOT EXISTS backup_destinations (
	user_id           TEXT PRIMARY KEY,
	endpoint          TEXT NOT NULL,
	region            TEXT NOT NULL DEFAULT '',
	bucket            TEXT NOT NULL,
	prefix            TEXT NOT NULL DEFAULT '',
	sealed_access_key TEXT NOT NULL,
	sealed_secret_key TEXT NOT NULL,
	updated_at        TEXT NOT NULL
);

-- Every backup taken, so there is a list to restore from.
CREATE TABLE IF NOT EXISTS backups (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	server_id  TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	tool_id    TEXT NOT NULL,
	object_key TEXT NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	status     TEXT NOT NULL,
	job_id     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS backups_lookup_idx
	ON backups(user_id, server_id, tool_id, created_at DESC);
`

func Open(path string) (*Store, error) {
	// WAL keeps the log writer from blocking dashboard reads.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}

	// SQLite tolerates one writer; more connections buys contention, not speed.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// migrations are additive statements applied to databases created before the
// column existed. CREATE TABLE IF NOT EXISTS cannot add columns, so each one
// is attempted and a "duplicate column" reply is treated as success.
var migrations = []string{
	`ALTER TABLE servers ADD COLUMN public_key TEXT NOT NULL DEFAULT ''`,
	// Ownership arrived after servers did. Existing rows are left unowned and
	// are claimed by the first account created — see ClaimUnownedServers.
	`ALTER TABLE servers ADD COLUMN user_id TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE fleet_samples ADD COLUMN user_id TEXT NOT NULL DEFAULT ''`,
	// The server's SSH identity, learned on first connection and checked on
	// every one after.
	`ALTER TABLE servers ADD COLUMN host_key TEXT NOT NULL DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS servers_user_idx ON servers(user_id)`,
}

func migrate(db *sql.DB) error {
	for _, statement := range migrations {
		if _, err := db.Exec(statement); err != nil {
			if strings.Contains(err.Error(), "duplicate column") {
				continue
			}
			return fmt.Errorf("store: migrate %q: %w", statement, err)
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }
