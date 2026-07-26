package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const jobColumns = `id, server_id, kind, title, actor, status, started_at, finished_at,
	changed, unchanged, skipped, failed, error`

func (s *Store) CreateJob(ctx context.Context, job Job) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (`+jobColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		job.ID, job.ServerID, job.Kind, job.Title, job.Actor, string(job.Status),
		formatTime(job.StartedAt), formatTimePtr(job.FinishedAt),
		job.Changed, job.Unchanged, job.Skipped, job.Failed, job.Error)
	if err != nil {
		return fmt.Errorf("store: insert job: %w", err)
	}
	return nil
}

func (s *Store) FinishJob(ctx context.Context, job Job) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, finished_at = ?, changed = ?, unchanged = ?,
		                skipped = ?, failed = ?, error = ?
		WHERE id = ?`,
		string(job.Status), formatTimePtr(job.FinishedAt),
		job.Changed, job.Unchanged, job.Skipped, job.Failed, job.Error, job.ID)
	if err != nil {
		return fmt.Errorf("store: finish job: %w", err)
	}
	return nil
}

func (s *Store) SetJobStatus(ctx context.Context, id string, status JobStatus) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET status = ? WHERE id = ?`, string(status), id)
	if err != nil {
		return fmt.Errorf("store: set job status: %w", err)
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, id string) (Job, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

// ListRecentJobs powers the activity feed, newest first.
func (s *Store) ListRecentJobs(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+jobColumns+` FROM jobs ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	jobs := []Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func scanJob(row rowScanner) (Job, error) {
	var (
		job           Job
		status        string
		startedAt     string
		finishedAtRaw sql.NullString
	)

	err := row.Scan(&job.ID, &job.ServerID, &job.Kind, &job.Title, &job.Actor, &status,
		&startedAt, &finishedAtRaw, &job.Changed, &job.Unchanged, &job.Skipped,
		&job.Failed, &job.Error)
	if err != nil {
		return Job{}, err
	}

	job.Status = JobStatus(status)
	job.StartedAt = parseTime(startedAt)
	if finishedAtRaw.Valid {
		finished := parseTime(finishedAtRaw.String)
		job.FinishedAt = &finished
	}
	return job, nil
}

func (s *Store) AppendEvent(ctx context.Context, event Event) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO job_events (job_id, seq, type, step_id, step_title, level, message, outcome, at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		event.JobID, event.Seq, event.Type, event.StepID, event.StepTitle,
		event.Level, event.Message, event.Outcome, formatTime(event.At))
	if err != nil {
		return 0, fmt.Errorf("store: append event: %w", err)
	}
	return result.LastInsertId()
}

// ListEvents returns a job's history after the given sequence number, which is
// what lets a reconnecting log viewer resume without replaying everything.
func (s *Store) ListEvents(ctx context.Context, jobID string, afterSeq int) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, job_id, seq, type, step_id, step_title, level, message, outcome, at
		FROM job_events WHERE job_id = ? AND seq > ? ORDER BY seq ASC`, jobID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	events := []Event{}
	for rows.Next() {
		var (
			event Event
			at    string
		)
		if err := rows.Scan(&event.ID, &event.JobID, &event.Seq, &event.Type, &event.StepID,
			&event.StepTitle, &event.Level, &event.Message, &event.Outcome, &at); err != nil {
			return nil, err
		}
		event.At = parseTime(at)
		events = append(events, event)
	}
	return events, rows.Err()
}

// CountJobsForServer is used to decide whether a server has ever been set up.
func (s *Store) CountJobsForServer(ctx context.Context, serverID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE server_id = ?`, serverID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: count jobs: %w", err)
	}
	return count, nil
}
