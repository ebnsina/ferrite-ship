package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// BackupDestination is where an account's backups are sent.
//
// S3-compatible, which covers AWS, Cloudflare R2, Backblaze B2, DigitalOcean
// Spaces and MinIO from one implementation. The keys are sealed exactly like
// an SSH credential: they can write to, and usually read, everything in the
// bucket.
type BackupDestination struct {
	UserID   string `json:"-"`
	Endpoint string `json:"endpoint"`
	Region   string `json:"region"`
	Bucket   string `json:"bucket"`
	// Prefix keeps several things apart in one bucket.
	Prefix string `json:"prefix"`

	SealedAccessKey string `json:"-"`
	SealedSecretKey string `json:"-"`

	UpdatedAt time.Time `json:"updatedAt"`
}

type BackupStatus string

const (
	BackupRunning BackupStatus = "running"
	BackupReady   BackupStatus = "ready"
	BackupFailed  BackupStatus = "failed"
)

// Backup is one copy taken at one moment.
type Backup struct {
	ID       string `json:"id"`
	UserID   string `json:"-"`
	ServerID string `json:"-"`
	ToolID   string `json:"toolId"`
	// ObjectKey is where it lives in the bucket.
	ObjectKey string       `json:"objectKey"`
	SizeBytes int64        `json:"sizeBytes"`
	Status    BackupStatus `json:"status"`
	JobID     string       `json:"jobId,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (s *Store) SaveBackupDestination(ctx context.Context, d BackupDestination) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO backup_destinations
			(user_id, endpoint, region, bucket, prefix, sealed_access_key, sealed_secret_key, updated_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET
			endpoint          = excluded.endpoint,
			region            = excluded.region,
			bucket            = excluded.bucket,
			prefix            = excluded.prefix,
			sealed_access_key = excluded.sealed_access_key,
			sealed_secret_key = excluded.sealed_secret_key,
			updated_at        = excluded.updated_at`,
		d.UserID, d.Endpoint, d.Region, d.Bucket, d.Prefix,
		d.SealedAccessKey, d.SealedSecretKey, formatTime(d.UpdatedAt))
	if err != nil {
		return fmt.Errorf("store: save backup destination: %w", err)
	}
	return nil
}

// GetBackupDestination returns ErrNotFound when none is configured, which is
// the signal that backups are not set up rather than an error.
func (s *Store) GetBackupDestination(ctx context.Context, userID string) (BackupDestination, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT user_id, endpoint, region, bucket, prefix,
		       sealed_access_key, sealed_secret_key, updated_at
		FROM backup_destinations WHERE user_id = ?`, userID)

	var (
		d         BackupDestination
		updatedAt string
	)
	err := row.Scan(&d.UserID, &d.Endpoint, &d.Region, &d.Bucket, &d.Prefix,
		&d.SealedAccessKey, &d.SealedSecretKey, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BackupDestination{}, ErrNotFound
	}
	if err != nil {
		return BackupDestination{}, err
	}

	d.UpdatedAt = parseTime(updatedAt)
	return d, nil
}

func (s *Store) DeleteBackupDestination(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM backup_destinations WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("store: delete backup destination: %w", err)
	}
	return nil
}

func (s *Store) CreateBackup(ctx context.Context, b Backup) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO backups (id, user_id, server_id, tool_id, object_key, size_bytes, status, job_id, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		b.ID, b.UserID, b.ServerID, b.ToolID, b.ObjectKey, b.SizeBytes,
		string(b.Status), b.JobID, formatTime(b.CreatedAt))
	if err != nil {
		return fmt.Errorf("store: create backup: %w", err)
	}
	return nil
}

// FinishBackup records the outcome once the job has run.
func (s *Store) FinishBackup(ctx context.Context, id string, status BackupStatus, size int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE backups SET status = ?, size_bytes = ? WHERE id = ?`, string(status), size, id)
	if err != nil {
		return fmt.Errorf("store: finish backup: %w", err)
	}
	return nil
}

// ListBackups returns what can be restored, newest first.
func (s *Store) ListBackups(ctx context.Context, userID, serverID, toolID string, limit int) ([]Backup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, server_id, tool_id, object_key, size_bytes, status, job_id, created_at
		FROM backups
		WHERE user_id = ? AND server_id = ? AND tool_id = ?
		ORDER BY created_at DESC
		LIMIT ?`, userID, serverID, toolID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list backups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	backups := []Backup{}
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

func (s *Store) GetBackup(ctx context.Context, userID, id string) (Backup, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, server_id, tool_id, object_key, size_bytes, status, job_id, created_at
		FROM backups WHERE id = ? AND user_id = ?`, id, userID)

	b, err := scanBackup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Backup{}, ErrNotFound
	}
	return b, err
}

func scanBackup(row rowScanner) (Backup, error) {
	var (
		b         Backup
		status    string
		createdAt string
	)
	err := row.Scan(&b.ID, &b.UserID, &b.ServerID, &b.ToolID, &b.ObjectKey,
		&b.SizeBytes, &status, &b.JobID, &createdAt)
	if err != nil {
		return Backup{}, err
	}
	b.Status = BackupStatus(status)
	b.CreatedAt = parseTime(createdAt)
	return b, nil
}
