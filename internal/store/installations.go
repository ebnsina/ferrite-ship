package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// InstallStatus is where an installation is in its life.
type InstallStatus string

const (
	// InstallPending is recorded before the job starts, so a page refresh
	// during a five-minute install still shows the tool as being worked on.
	InstallPending InstallStatus = "installing"
	InstallReady   InstallStatus = "ready"
	InstallFailed  InstallStatus = "failed"
	// InstallRemoving is set while the removal job runs. The row is deleted
	// when it succeeds; a failure leaves it here rather than pretending the
	// tool is gone while its containers are still running.
	InstallRemoving InstallStatus = "removing"
)

// Installation is one tool on one server.
type Installation struct {
	ID       string
	UserID   string
	ServerID string
	ToolID   string
	Version  string
	Status   InstallStatus
	// SealedPassword is the generated credential, encrypted at rest. Never
	// serialised to the API except through the connection-details endpoint.
	SealedPassword string
	// LastJobID links to the run that last changed this, so the UI can offer
	// "see what happened" on a failure.
	LastJobID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const installationColumns = `id, user_id, server_id, tool_id, version, status,
	sealed_password, last_job_id, created_at, updated_at`

// SaveInstallation records an install, replacing any earlier attempt for the
// same tool on the same server.
//
// The password is only overwritten when a new one is supplied: re-running an
// install to repair it must keep the credential the owner is already using,
// otherwise their application stops being able to connect.
func (s *Store) SaveInstallation(ctx context.Context, in Installation) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO installations (`+installationColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(server_id, tool_id) DO UPDATE SET
			version         = excluded.version,
			status          = excluded.status,
			sealed_password = CASE WHEN excluded.sealed_password = ''
			                       THEN installations.sealed_password
			                       ELSE excluded.sealed_password END,
			last_job_id     = excluded.last_job_id,
			updated_at      = excluded.updated_at`,
		in.ID, in.UserID, in.ServerID, in.ToolID, in.Version, string(in.Status),
		in.SealedPassword, in.LastJobID, formatTime(in.CreatedAt), formatTime(in.UpdatedAt))
	if err != nil {
		return fmt.Errorf("store: save installation: %w", err)
	}
	return nil
}

// ListInstallations returns the tools on one server.
//
// userID is in the WHERE clause rather than checked afterwards, for the same
// reason it is everywhere else here: a call site that has not established who
// is asking then does not compile.
func (s *Store) ListInstallations(ctx context.Context, userID, serverID string) ([]Installation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+installationColumns+` FROM installations
		 WHERE user_id = ? AND server_id = ? ORDER BY tool_id ASC`, userID, serverID)
	if err != nil {
		return nil, fmt.Errorf("store: list installations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	installations := []Installation{}
	for rows.Next() {
		in, err := scanInstallation(rows)
		if err != nil {
			return nil, err
		}
		installations = append(installations, in)
	}
	return installations, rows.Err()
}

func (s *Store) GetInstallation(ctx context.Context, userID, serverID, toolID string) (Installation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+installationColumns+` FROM installations
		 WHERE user_id = ? AND server_id = ? AND tool_id = ?`, userID, serverID, toolID)

	in, err := scanInstallation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Installation{}, ErrNotFound
	}
	return in, err
}

// SetInstallationStatus records the outcome of a job against an installation.
func (s *Store) SetInstallationStatus(
	ctx context.Context, serverID, toolID string, status InstallStatus,
) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE installations SET status = ?, updated_at = ?
		 WHERE server_id = ? AND tool_id = ?`,
		string(status), formatTime(time.Now().UTC()), serverID, toolID)
	if err != nil {
		return fmt.Errorf("store: set installation status: %w", err)
	}
	return nil
}

func (s *Store) DeleteInstallation(ctx context.Context, userID, serverID, toolID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM installations WHERE user_id = ? AND server_id = ? AND tool_id = ?`,
		userID, serverID, toolID)
	if err != nil {
		return fmt.Errorf("store: delete installation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func scanInstallation(row rowScanner) (Installation, error) {
	var (
		in                   Installation
		status               string
		createdAt, updatedAt string
	)

	err := row.Scan(&in.ID, &in.UserID, &in.ServerID, &in.ToolID, &in.Version, &status,
		&in.SealedPassword, &in.LastJobID, &createdAt, &updatedAt)
	if err != nil {
		return Installation{}, err
	}

	in.Status = InstallStatus(status)
	in.CreatedAt = parseTime(createdAt)
	in.UpdatedAt = parseTime(updatedAt)
	return in, nil
}
