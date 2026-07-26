package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Cadence is how often a backup runs.
type Cadence string

const (
	Daily  Cadence = "daily"
	Weekly Cadence = "weekly"
)

// BackupSchedule is one tool's automatic backup.
//
// Times are UTC throughout. A server's own clock is set by the baseline and a
// schedule that drifted with a timezone would be very hard to explain.
type BackupSchedule struct {
	ID       string `json:"id"`
	UserID   string `json:"-"`
	ServerID string `json:"-"`
	ToolID   string `json:"toolId"`

	Cadence Cadence `json:"cadence"`
	// Hour is 0–23, UTC.
	Hour int `json:"hour"`
	// Weekday is 0 (Sunday) to 6, used only for a weekly cadence.
	Weekday int `json:"weekday"`
	// Keep is how many backups to retain; older ones are deleted after a run.
	Keep int `json:"keep"`

	LastRunAt *time.Time `json:"lastRunAt"`
	NextRunAt time.Time  `json:"nextRunAt"`
}

// NextRun returns the first occurrence strictly after `after`.
//
// Exported and pure so the scheduler and the tests agree on what "next" means
// without either having to reimplement it.
func (s BackupSchedule) NextRun(after time.Time) time.Time {
	after = after.UTC()
	candidate := time.Date(after.Year(), after.Month(), after.Day(), s.Hour, 0, 0, 0, time.UTC)

	switch s.Cadence {
	case Weekly:
		// Walk forward to the right weekday, then forward a week if that
		// moment has already passed.
		delta := (s.Weekday - int(candidate.Weekday()) + 7) % 7
		candidate = candidate.AddDate(0, 0, delta)
		if !candidate.After(after) {
			candidate = candidate.AddDate(0, 0, 7)
		}
	default:
		if !candidate.After(after) {
			candidate = candidate.AddDate(0, 0, 1)
		}
	}

	return candidate
}

const scheduleColumns = `id, user_id, server_id, tool_id, cadence, hour, weekday, keep,
	last_run_at, next_run_at`

func (s *Store) SaveBackupSchedule(ctx context.Context, sc BackupSchedule) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO backup_schedules (`+scheduleColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(server_id, tool_id) DO UPDATE SET
			cadence     = excluded.cadence,
			hour        = excluded.hour,
			weekday     = excluded.weekday,
			keep        = excluded.keep,
			next_run_at = excluded.next_run_at`,
		sc.ID, sc.UserID, sc.ServerID, sc.ToolID, string(sc.Cadence), sc.Hour, sc.Weekday,
		sc.Keep, formatTimePtr(sc.LastRunAt), formatTime(sc.NextRunAt))
	if err != nil {
		return fmt.Errorf("store: save backup schedule: %w", err)
	}
	return nil
}

func (s *Store) GetBackupSchedule(ctx context.Context, userID, serverID, toolID string) (BackupSchedule, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+scheduleColumns+` FROM backup_schedules
		 WHERE user_id = ? AND server_id = ? AND tool_id = ?`, userID, serverID, toolID)

	sc, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return BackupSchedule{}, ErrNotFound
	}
	return sc, err
}

func (s *Store) DeleteBackupSchedule(ctx context.Context, userID, serverID, toolID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM backup_schedules WHERE user_id = ? AND server_id = ? AND tool_id = ?`,
		userID, serverID, toolID)
	if err != nil {
		return fmt.Errorf("store: delete backup schedule: %w", err)
	}
	return nil
}

// DueSchedules returns everything that should have run by now.
//
// Not scoped by user, and called only by the scheduler: it runs on behalf of
// every account. The ownership carried on each row is what the work it starts
// is then attributed to.
func (s *Store) DueSchedules(ctx context.Context, now time.Time) ([]BackupSchedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+scheduleColumns+` FROM backup_schedules WHERE next_run_at <= ? ORDER BY next_run_at ASC`,
		formatTime(now.UTC()))
	if err != nil {
		return nil, fmt.Errorf("store: due schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	schedules := []BackupSchedule{}
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, sc)
	}
	return schedules, rows.Err()
}

// MarkScheduleRun records an attempt and moves the schedule forward.
//
// Called whether or not the backup succeeded: a failing tool must not have its
// schedule retried every minute for ever, and the failure is already visible
// in the backup list and the job history.
func (s *Store) MarkScheduleRun(ctx context.Context, id string, ranAt, next time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE backup_schedules SET last_run_at = ?, next_run_at = ? WHERE id = ?`,
		formatTime(ranAt.UTC()), formatTime(next.UTC()), id)
	if err != nil {
		return fmt.Errorf("store: mark schedule run: %w", err)
	}
	return nil
}

// ExpiredBackups returns the ready backups beyond the newest `keep`.
func (s *Store) ExpiredBackups(ctx context.Context, serverID, toolID string, keep int) ([]Backup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, server_id, tool_id, object_key, size_bytes, status, job_id, created_at
		FROM backups
		WHERE server_id = ? AND tool_id = ? AND status = 'ready'
		ORDER BY created_at DESC
		LIMIT -1 OFFSET ?`, serverID, toolID, keep)
	if err != nil {
		return nil, fmt.Errorf("store: expired backups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	expired := []Backup{}
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		expired = append(expired, b)
	}
	return expired, rows.Err()
}

func (s *Store) DeleteBackup(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM backups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete backup: %w", err)
	}
	return nil
}

func scanSchedule(row rowScanner) (BackupSchedule, error) {
	var (
		sc        BackupSchedule
		cadence   string
		lastRun   sql.NullString
		nextRunAt string
	)
	err := row.Scan(&sc.ID, &sc.UserID, &sc.ServerID, &sc.ToolID, &cadence, &sc.Hour,
		&sc.Weekday, &sc.Keep, &lastRun, &nextRunAt)
	if err != nil {
		return BackupSchedule{}, err
	}

	sc.Cadence = Cadence(cadence)
	sc.NextRunAt = parseTime(nextRunAt)
	if lastRun.Valid {
		at := parseTime(lastRun.String)
		sc.LastRunAt = &at
	}
	return sc, nil
}
