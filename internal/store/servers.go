package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/facts"
)

var ErrNotFound = errors.New("not found")

const serverColumns = `id, name, connection_kind, host, port, username, region, status,
	facts_json, services_json, sealed_password, sealed_private_key, public_key,
	created_at, last_seen_at`

func (s *Store) CreateServer(ctx context.Context, srv Server) error {
	factsJSON, err := json.Marshal(srv.Facts)
	if err != nil {
		return fmt.Errorf("store: encode facts: %w", err)
	}
	servicesJSON, err := json.Marshal(orEmpty(srv.Services))
	if err != nil {
		return fmt.Errorf("store: encode services: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO servers (`+serverColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		srv.ID, srv.Name, string(srv.Kind), srv.Host, srv.Port, srv.User, srv.Region,
		string(srv.Status), string(factsJSON), string(servicesJSON),
		srv.SealedPassword, srv.SealedPrivateKey, srv.PublicKey,
		formatTime(srv.CreatedAt), formatTimePtr(srv.LastSeenAt))
	if err != nil {
		return fmt.Errorf("store: insert server: %w", err)
	}
	return nil
}

func (s *Store) ListServers(ctx context.Context) ([]Server, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+serverColumns+` FROM servers ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("store: list servers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	servers := []Server{}
	for rows.Next() {
		srv, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	return servers, rows.Err()
}

func (s *Store) GetServer(ctx context.Context, id string) (Server, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+serverColumns+` FROM servers WHERE id = ?`, id)

	srv, err := scanServer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Server{}, ErrNotFound
	}
	return srv, err
}

func (s *Store) DeleteServer(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM servers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete server: %w", err)
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

// UpdateServerState records what a probe or a job run just learned.
func (s *Store) UpdateServerState(
	ctx context.Context, id string, status ServerStatus, f facts.Facts, seenAt time.Time,
) error {
	factsJSON, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("store: encode facts: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE servers SET status = ?, facts_json = ?, last_seen_at = ? WHERE id = ?`,
		string(status), string(factsJSON), formatTime(seenAt), id)
	if err != nil {
		return fmt.Errorf("store: update server state: %w", err)
	}
	return nil
}

func (s *Store) SetServerStatus(ctx context.Context, id string, status ServerStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE servers SET status = ? WHERE id = ?`, string(status), id)
	if err != nil {
		return fmt.Errorf("store: set server status: %w", err)
	}
	return nil
}

// rowScanner covers both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanServer(row rowScanner) (Server, error) {
	var (
		srv           Server
		kind, status  string
		factsJSON     string
		servicesJSON  string
		createdAt     string
		lastSeenAtRaw sql.NullString
	)

	err := row.Scan(&srv.ID, &srv.Name, &kind, &srv.Host, &srv.Port, &srv.User, &srv.Region,
		&status, &factsJSON, &servicesJSON, &srv.SealedPassword, &srv.SealedPrivateKey,
		&srv.PublicKey, &createdAt, &lastSeenAtRaw)
	if err != nil {
		return Server{}, err
	}

	srv.Kind = ConnectionKind(kind)
	srv.Status = ServerStatus(status)

	if err := json.Unmarshal([]byte(factsJSON), &srv.Facts); err != nil {
		return Server{}, fmt.Errorf("store: decode facts for %s: %w", srv.ID, err)
	}
	if err := json.Unmarshal([]byte(servicesJSON), &srv.Services); err != nil {
		return Server{}, fmt.Errorf("store: decode services for %s: %w", srv.ID, err)
	}
	srv.Services = orEmpty(srv.Services)

	srv.CreatedAt = parseTime(createdAt)
	if lastSeenAtRaw.Valid {
		seen := parseTime(lastSeenAtRaw.String)
		srv.LastSeenAt = &seen
	}

	return srv, nil
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
