package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AppStatus is where a deployment stands.
type AppStatus string

const (
	// AppNew has never been deployed.
	AppNew AppStatus = "new"
	// AppDeploying is mid-build.
	AppDeploying AppStatus = "deploying"
	AppRunning   AppStatus = "running"
	AppFailed    AppStatus = "failed"
)

// App is one application on one server.
type App struct {
	ID       string `json:"id"`
	UserID   string `json:"-"`
	ServerID string `json:"serverId"`

	Name       string `json:"name"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	// Domain is empty for an application that runs but is not published.
	Domain string `json:"domain"`
	Port   int    `json:"port"`

	// SealedEnv is the whole environment as one sealed blob. Never serialised.
	SealedEnv string `json:"-"`
	// SealedDeployKey is a private SSH key with read access to the repository,
	// for repositories that are not public. Never serialised.
	SealedDeployKey string `json:"-"`
	// HasDeployKey lets the UI say whether one is set without revealing it.
	HasDeployKey bool `json:"hasDeployKey"`

	Status    AppStatus `json:"status"`
	LastJobID string    `json:"lastJobId,omitempty"`

	CreatedAt  time.Time  `json:"createdAt"`
	DeployedAt *time.Time `json:"deployedAt"`
}

const appColumns = `id, user_id, server_id, name, repository, branch, domain, port,
	sealed_env, sealed_deploy_key, status, last_job_id, created_at, deployed_at`

func (s *Store) CreateApp(ctx context.Context, a App) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO apps (`+appColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.UserID, a.ServerID, a.Name, a.Repository, a.Branch, a.Domain, a.Port,
		a.SealedEnv, a.SealedDeployKey, string(a.Status), a.LastJobID,
		formatTime(a.CreatedAt), formatTimePtr(a.DeployedAt))
	if err != nil {
		return fmt.Errorf("store: create app: %w", err)
	}
	return nil
}

func (s *Store) UpdateApp(ctx context.Context, a App) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE apps SET name = ?, repository = ?, branch = ?, domain = ?, port = ?,
		                sealed_env = ?,
		                -- Keep the existing key when none is supplied, the same
		                -- as a repair keeps a generated password.
		                sealed_deploy_key = CASE WHEN ? = '' THEN sealed_deploy_key ELSE ? END
		WHERE id = ? AND user_id = ?`,
		a.Name, a.Repository, a.Branch, a.Domain, a.Port, a.SealedEnv,
		a.SealedDeployKey, a.SealedDeployKey, a.ID, a.UserID)
	if err != nil {
		return fmt.Errorf("store: update app: %w", err)
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

// SetAppStatus records where a deployment got to.
func (s *Store) SetAppStatus(ctx context.Context, id string, status AppStatus, jobID string, deployed *time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE apps SET status = ?, last_job_id = ?,
		                deployed_at = COALESCE(?, deployed_at)
		WHERE id = ?`,
		string(status), jobID, formatTimePtr(deployed), id)
	if err != nil {
		return fmt.Errorf("store: set app status: %w", err)
	}
	return nil
}

// ListApps returns the applications on one server.
func (s *Store) ListApps(ctx context.Context, userID, serverID string) ([]App, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+appColumns+` FROM apps WHERE user_id = ? AND server_id = ? ORDER BY name ASC`,
		userID, serverID)
	if err != nil {
		return nil, fmt.Errorf("store: list apps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	apps := []App{}
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// PublishedApps returns every application on a server that has a domain, which
// is what the proxy's configuration is built from.
//
// Not scoped by user on purpose, and called only from the runner: the proxy
// serves one machine, and a route missing because it belongs to another
// account would be a server quietly failing to serve half its traffic. The
// ownership check happens where the deployment is authorised.
func (s *Store) PublishedApps(ctx context.Context, serverID string) ([]App, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+appColumns+` FROM apps WHERE server_id = ? AND domain <> '' ORDER BY name ASC`,
		serverID)
	if err != nil {
		return nil, fmt.Errorf("store: published apps: %w", err)
	}
	defer func() { _ = rows.Close() }()

	apps := []App{}
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *Store) GetApp(ctx context.Context, userID, id string) (App, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+appColumns+` FROM apps WHERE id = ? AND user_id = ?`, id, userID)

	a, err := scanApp(row)
	if errors.Is(err, sql.ErrNoRows) {
		return App{}, ErrNotFound
	}
	return a, err
}

func (s *Store) DeleteApp(ctx context.Context, userID, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM apps WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("store: delete app: %w", err)
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

func scanApp(row rowScanner) (App, error) {
	var (
		a          App
		status     string
		createdAt  string
		deployedAt sql.NullString
	)
	err := row.Scan(&a.ID, &a.UserID, &a.ServerID, &a.Name, &a.Repository, &a.Branch,
		&a.Domain, &a.Port, &a.SealedEnv, &a.SealedDeployKey, &status, &a.LastJobID,
		&createdAt, &deployedAt)
	if err != nil {
		return App{}, err
	}

	a.HasDeployKey = a.SealedDeployKey != ""

	a.Status = AppStatus(status)
	a.CreatedAt = parseTime(createdAt)
	if deployedAt.Valid {
		at := parseTime(deployedAt.String)
		a.DeployedAt = &at
	}
	return a, nil
}
