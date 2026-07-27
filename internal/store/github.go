package store

import (
	"context"
	"fmt"
	"time"
)

// GitHubInstallation is one place the app has been installed — a person's
// account or an organisation — connected to one of ours.
type GitHubInstallation struct {
	// ID is GitHub's installation id, which is what tokens are minted against.
	ID int64 `json:"id"`
	// UserID is the owner. Never serialised.
	UserID string `json:"-"`
	// Account is the login it was installed on, stored so the dashboard can
	// say "connected to acme" rather than a number.
	Account string `json:"account"`
	// Selection is "all" or "selected", so somebody wondering why a repository
	// is missing is told where to change it.
	Selection   string    `json:"selection"`
	ConnectedAt time.Time `json:"connectedAt"`
}

// SaveGitHubInstallation records a connection, replacing any earlier one.
//
// The installation id is the primary key rather than a pair with the account:
// GitHub issues one per place the app is installed, and the same one arriving
// again is the same connection being re-confirmed — which happens whenever
// somebody changes which repositories are shared.
func (s *Store) SaveGitHubInstallation(ctx context.Context, in GitHubInstallation) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO github_installations (installation_id, user_id, account, selection, connected_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(installation_id) DO UPDATE SET
			user_id = excluded.user_id,
			account = excluded.account,
			selection = excluded.selection`,
		in.ID, in.UserID, in.Account, in.Selection, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("store: save github installation: %w", err)
	}
	return nil
}

// ListGitHubInstallations returns what one account has connected.
func (s *Store) ListGitHubInstallations(ctx context.Context, userID string) ([]GitHubInstallation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT installation_id, user_id, account, selection, connected_at
		FROM github_installations WHERE user_id = ? ORDER BY connected_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list github installations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	installations := []GitHubInstallation{}
	for rows.Next() {
		var (
			in          GitHubInstallation
			connectedAt string
		)
		if err := rows.Scan(&in.ID, &in.UserID, &in.Account, &in.Selection, &connectedAt); err != nil {
			return nil, err
		}
		in.ConnectedAt = parseTime(connectedAt)
		installations = append(installations, in)
	}
	return installations, rows.Err()
}

// GetGitHubInstallation reaches one, for the account that owns it.
func (s *Store) GetGitHubInstallation(
	ctx context.Context, userID string, id int64,
) (GitHubInstallation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT installation_id, user_id, account, selection, connected_at
		FROM github_installations WHERE installation_id = ? AND user_id = ?`, id, userID)

	var (
		in          GitHubInstallation
		connectedAt string
	)
	if err := row.Scan(&in.ID, &in.UserID, &in.Account, &in.Selection, &connectedAt); err != nil {
		return GitHubInstallation{}, ErrNotFound
	}
	in.ConnectedAt = parseTime(connectedAt)
	return in, nil
}

// DeleteGitHubInstallation forgets a connection on our side only.
//
// The app stays installed on GitHub. Removing it there is deliberately the
// owner's own action on their own settings page — a control plane that could
// silently uninstall itself from an organisation would be a worse thing to
// hand somebody than one that asks.
func (s *Store) DeleteGitHubInstallation(ctx context.Context, userID string, id int64) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM github_installations WHERE installation_id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("store: delete github installation: %w", err)
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
