package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CountUsers decides whether the first-run setup is still open.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count users: %w", err)
	}
	return count, nil
}

func (s *Store) CreateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, created_at) VALUES (?,?,?,?)`,
		user.ID, strings.ToLower(user.Email), user.PasswordHash, formatTime(user.CreatedAt))
	if err != nil {
		return fmt.Errorf("store: insert user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`,
		strings.ToLower(strings.TrimSpace(email)))
	return scanUser(row)
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func scanUser(row rowScanner) (User, error) {
	var (
		user      User
		createdAt string
	)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.CreatedAt = parseTime(createdAt)
	return user, nil
}

func (s *Store) CreateSession(ctx context.Context, session Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?,?,?,?)`,
		session.ID, session.UserID, formatTime(session.CreatedAt), formatTime(session.ExpiresAt))
	if err != nil {
		return fmt.Errorf("store: insert session: %w", err)
	}
	return nil
}

// GetValidSession returns the session only if it has not expired, so an old
// cookie cannot be used simply because the row still exists.
func (s *Store) GetValidSession(ctx context.Context, id string, now time.Time) (Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?`, id)

	var (
		session   Session
		createdAt string
		expiresAt string
	)
	err := row.Scan(&session.ID, &session.UserID, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}

	session.CreatedAt = parseTime(createdAt)
	session.ExpiresAt = parseTime(expiresAt)

	if !now.Before(session.ExpiresAt) {
		return Session{}, ErrNotFound
	}
	return session, nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	return nil
}

// DeleteAllUsers removes every account and, by cascade, every session.
//
// There is no password reset over the network by design: this is a
// single-user tool, and an email-based reset would be a second way in. The
// recovery path is instead proving you control the machine, which is the same
// authority that could read the database anyway.
func (s *Store) DeleteAllUsers(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM users`); err != nil {
		return fmt.Errorf("store: delete users: %w", err)
	}
	return nil
}

// DeleteExpiredSessions keeps the table from growing without bound.
func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= ?`, formatTime(now))
	if err != nil {
		return fmt.Errorf("store: delete expired sessions: %w", err)
	}
	return nil
}
