package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Notifications is where a person wants to be told, and about what.
//
// Off by default in the sense that an account with no row gets no mail: an
// address is asked for once rather than assumed from the login, because the
// address you sign in with and the one you read at three in the morning are
// often not the same one.
type Notifications struct {
	Email string `json:"email"`

	OnBackupFailed bool `json:"onBackupFailed"`
	OnServerDown   bool `json:"onServerDown"`
	OnDiskLow      bool `json:"onDiskLow"`

	// DiskPercent is how full a disk has to be before it is worth saying so.
	DiskPercent int `json:"diskPercent"`

	UpdatedAt *time.Time `json:"updatedAt"`
}

// Wants reports whether a kind of alert should be sent at all.
func (n Notifications) Wants(kind string) bool {
	if n.Email == "" {
		return false
	}
	switch kind {
	case "backup-failed":
		return n.OnBackupFailed
	case "server-down":
		return n.OnServerDown
	case "disk-low":
		return n.OnDiskLow
	default:
		return false
	}
}

// Alert is a condition that is currently true, or was.
//
// Stored rather than derived because the interesting question is not "is the
// disk full" but "has this already been said". A condition that is checked
// every few minutes and mailed every time it is still true is a condition
// nobody reads about twice.
type Alert struct {
	ID       string `json:"id"`
	UserID   string `json:"-"`
	ServerID string `json:"serverId"`
	Kind     string `json:"kind"`
	// Subject narrows it within the server — a tool id, for instance — and is
	// empty when the machine itself is the subject.
	Subject string `json:"subject"`
	Detail  string `json:"detail"`

	OpenedAt  time.Time  `json:"openedAt"`
	ClearedAt *time.Time `json:"clearedAt"`
}

func (s *Store) GetNotifications(ctx context.Context, userID string) (Notifications, error) {
	var (
		settings Notifications
		updated  sql.NullString
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT email, on_backup_failed, on_server_down, on_disk_low, disk_percent, updated_at
		FROM notification_settings WHERE user_id = ?`, userID).
		Scan(&settings.Email, &settings.OnBackupFailed, &settings.OnServerDown,
			&settings.OnDiskLow, &settings.DiskPercent, &updated)

	if errors.Is(err, sql.ErrNoRows) {
		// Never configured. Returned rather than erroring: "nowhere to send"
		// is a normal state, and the page needs to render it.
		return Notifications{DiskPercent: 85}, nil
	}
	if err != nil {
		return Notifications{}, fmt.Errorf("store: read notification settings: %w", err)
	}

	if updated.Valid {
		if at, err := time.Parse(time.RFC3339Nano, updated.String); err == nil {
			settings.UpdatedAt = &at
		}
	}
	return settings, nil
}

func (s *Store) SaveNotifications(ctx context.Context, userID string, settings Notifications) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_settings
			(user_id, email, on_backup_failed, on_server_down, on_disk_low, disk_percent, updated_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET
			email = excluded.email,
			on_backup_failed = excluded.on_backup_failed,
			on_server_down = excluded.on_server_down,
			on_disk_low = excluded.on_disk_low,
			disk_percent = excluded.disk_percent,
			updated_at = excluded.updated_at`,
		userID, settings.Email, settings.OnBackupFailed, settings.OnServerDown,
		settings.OnDiskLow, settings.DiskPercent, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("store: save notification settings: %w", err)
	}
	return nil
}

// OpenAlert records a condition, and reports whether it is new.
//
// False means one was already open, and the caller should stay quiet. That
// decision lives here rather than in the watcher because it is the database
// that can make it atomically — two checks overlapping must not produce two
// messages about one disk.
func (s *Store) OpenAlert(ctx context.Context, alert Alert) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO alerts (id, user_id, server_id, kind, subject, detail, opened_at, cleared_at)
		SELECT ?,?,?,?,?,?,?,NULL
		WHERE NOT EXISTS (
			SELECT 1 FROM alerts
			WHERE user_id = ? AND server_id = ? AND kind = ? AND subject = ?
			  AND cleared_at IS NULL
		)`,
		alert.ID, alert.UserID, alert.ServerID, alert.Kind, alert.Subject, alert.Detail,
		formatTime(time.Now().UTC()),
		alert.UserID, alert.ServerID, alert.Kind, alert.Subject)
	if err != nil {
		return false, fmt.Errorf("store: open alert: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: open alert: %w", err)
	}
	return affected > 0, nil
}

// ClearAlert closes a condition, and reports whether one was open.
//
// Owner-scoped like every other query here, even though a server id is already
// specific enough in practice. The rule is that the compiler refuses a call
// site which has not established who is asking — an exception "because this
// one is safe" is how the next one gets written without a userID at all.
//
// True is the signal to send the "it is back" message. Sent only when
// something was actually said in the first place, so a server that has always
// been fine never produces a recovery notice for a problem it never had.
func (s *Store) ClearAlert(
	ctx context.Context, userID, serverID, kind, subject string,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE alerts SET cleared_at = ?
		WHERE user_id = ? AND server_id = ? AND kind = ? AND subject = ?
		  AND cleared_at IS NULL`,
		formatTime(time.Now().UTC()), userID, serverID, kind, subject)
	if err != nil {
		return false, fmt.Errorf("store: clear alert: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: clear alert: %w", err)
	}
	return affected > 0, nil
}

// OpenAlerts returns what is currently wrong for one account, newest first.
func (s *Store) OpenAlerts(ctx context.Context, userID string) ([]Alert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, server_id, kind, subject, detail, opened_at, cleared_at
		FROM alerts WHERE user_id = ? AND cleared_at IS NULL
		ORDER BY opened_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list alerts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	alerts := []Alert{}
	for rows.Next() {
		var (
			alert   Alert
			opened  string
			cleared sql.NullString
		)
		if err := rows.Scan(&alert.ID, &alert.UserID, &alert.ServerID, &alert.Kind,
			&alert.Subject, &alert.Detail, &opened, &cleared); err != nil {
			return nil, fmt.Errorf("store: list alerts: %w", err)
		}

		alert.OpenedAt, _ = time.Parse(time.RFC3339Nano, opened)
		if cleared.Valid {
			if at, err := time.Parse(time.RFC3339Nano, cleared.String); err == nil {
				alert.ClearedAt = &at
			}
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

// EveryNotifiable returns the accounts that have asked to be told something.
//
// The watcher works from this rather than from the server list: a server whose
// owner wants no mail still needs its facts refreshed, but there is no reason
// to evaluate alert rules for an account that would never hear about them.
func (s *Store) EveryNotifiable(ctx context.Context) (map[string]Notifications, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, email, on_backup_failed, on_server_down, on_disk_low, disk_percent
		FROM notification_settings WHERE email <> ''`)
	if err != nil {
		return nil, fmt.Errorf("store: list notification settings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := map[string]Notifications{}
	for rows.Next() {
		var (
			userID   string
			settings Notifications
		)
		if err := rows.Scan(&userID, &settings.Email, &settings.OnBackupFailed,
			&settings.OnServerDown, &settings.OnDiskLow, &settings.DiskPercent); err != nil {
			return nil, fmt.Errorf("store: list notification settings: %w", err)
		}
		found[userID] = settings
	}
	return found, rows.Err()
}
