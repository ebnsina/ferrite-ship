package store

import (
	"context"
	"fmt"
	"time"
)

// SavedQuery is a query someone kept, against one tool on one server.
type SavedQuery struct {
	ID       string    `json:"id"`
	UserID   string    `json:"-"`
	ServerID string    `json:"-"`
	ToolID   string    `json:"toolId"`
	Name     string    `json:"name"`
	Query    string    `json:"query"`
	SavedAt  time.Time `json:"savedAt"`
}

// SaveQuery stores a query, replacing one of the same name.
func (s *Store) SaveQuery(ctx context.Context, q SavedQuery) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO saved_queries (id, user_id, server_id, tool_id, name, query, created_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(user_id, server_id, tool_id, name) DO UPDATE SET
			query      = excluded.query,
			created_at = excluded.created_at`,
		q.ID, q.UserID, q.ServerID, q.ToolID, q.Name, q.Query, formatTime(q.SavedAt))
	if err != nil {
		return fmt.Errorf("store: save query: %w", err)
	}
	return nil
}

// ListSavedQueries returns what this account has kept for one tool.
//
// userID is in the WHERE clause rather than checked afterwards, as everywhere
// else here: a saved query can contain anything the author typed, and handing
// one account another's is handing over their schema.
func (s *Store) ListSavedQueries(ctx context.Context, userID, serverID, toolID string) ([]SavedQuery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, server_id, tool_id, name, query, created_at
		FROM saved_queries
		WHERE user_id = ? AND server_id = ? AND tool_id = ?
		ORDER BY name ASC`, userID, serverID, toolID)
	if err != nil {
		return nil, fmt.Errorf("store: list saved queries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	saved := []SavedQuery{}
	for rows.Next() {
		var (
			q         SavedQuery
			createdAt string
		)
		if err := rows.Scan(&q.ID, &q.UserID, &q.ServerID, &q.ToolID, &q.Name, &q.Query, &createdAt); err != nil {
			return nil, err
		}
		q.SavedAt = parseTime(createdAt)
		saved = append(saved, q)
	}
	return saved, rows.Err()
}

func (s *Store) DeleteSavedQuery(ctx context.Context, userID, serverID, id string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM saved_queries WHERE id = ? AND user_id = ? AND server_id = ?`,
		id, userID, serverID)
	if err != nil {
		return fmt.Errorf("store: delete saved query: %w", err)
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
