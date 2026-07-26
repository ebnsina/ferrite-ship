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
