// Package scheduler runs the work nobody pressed a button for.
//
// A backup that depends on somebody remembering is the one that turns out to
// be missing. This is the piece that makes the promise in the UI true without
// anyone being at a keyboard.
package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// tick is how often the schedule is consulted.
//
// A minute is far more often than anything here needs — the finest cadence is
// hourly — but it makes the query cheap and keeps a restart from delaying a
// due backup by more than a minute.
const tick = time.Minute

// retryAfter is how long to wait when a server is busy with something else.
//
// Short enough to catch the backup within the hour, long enough that a job
// which runs for an hour does not have the scheduler knocking every minute.
const retryAfter = 10 * time.Minute

type Scheduler struct {
	store  *store.Store
	runner *runner.Runner
	log    *slog.Logger
}

func New(st *store.Store, r *runner.Runner, log *slog.Logger) *Scheduler {
	return &Scheduler{store: st, runner: r, log: log}
}

// Run consults the schedule until the context is cancelled.
//
// Blocking, and started in its own goroutine by main. It returns on shutdown
// rather than being killed, so a backup that has already started is not
// abandoned halfway.
func (s *Scheduler) Run(ctx context.Context) {
	s.log.Info("backup scheduler started", "interval", tick.String())

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	// Once immediately: a process that was down over a due time should not
	// wait another full minute to notice.
	s.runDue(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("backup scheduler stopped")
			return
		case <-ticker.C:
			s.runDue(ctx)
		}
	}
}

func (s *Scheduler) runDue(ctx context.Context) {
	now := time.Now().UTC()

	due, err := s.store.DueSchedules(ctx, now)
	if err != nil {
		// Logged and dropped. The next tick tries again, and a database that
		// is briefly unavailable should not stop the scheduler for good.
		s.log.Error("could not read the backup schedule", "error", err)
		return
	}

	for _, schedule := range due {
		s.start(ctx, schedule, now)
	}
}

func (s *Scheduler) start(ctx context.Context, schedule store.BackupSchedule, now time.Time) {
	_, err := s.runner.StartBackup(ctx, schedule.UserID, schedule.ServerID, schedule.ToolID, store.ActorScheduled)

	switch {
	case errors.Is(err, runner.ErrAlreadyRunning):
		// Something else is using that server. Come back shortly rather than
		// skipping the whole cadence — a deploy that happened to overlap
		// should not cost somebody a day's backup.
		next := now.Add(retryAfter)
		if err := s.store.MarkScheduleRun(ctx, schedule.ID, *lastRunOf(schedule, now), next); err != nil {
			s.log.Error("could not defer a schedule", "schedule", schedule.ID, "error", err)
		}
		s.log.Info("backup deferred, the server is busy",
			"tool", schedule.ToolID, "retry_at", next.Format(time.RFC3339))
		return

	case err != nil:
		// A failing tool must not be retried every minute for ever. Move on to
		// the next occurrence; the failure is already visible in the backup
		// list and in the job history.
		s.log.Error("scheduled backup could not start",
			"tool", schedule.ToolID, "server", schedule.ServerID, "error", err)
	}

	if err := s.store.MarkScheduleRun(ctx, schedule.ID, now, schedule.NextRun(now)); err != nil {
		s.log.Error("could not advance a schedule", "schedule", schedule.ID, "error", err)
	}
}

// lastRunOf keeps the recorded last run when deferring, so "when did this last
// happen" does not report an attempt that never ran.
func lastRunOf(schedule store.BackupSchedule, fallback time.Time) *time.Time {
	if schedule.LastRunAt != nil {
		return schedule.LastRunAt
	}
	// Never run before: MarkScheduleRun needs a value, and the deferral is not
	// a run, so record the moment we looked rather than claiming a backup.
	return &fallback
}
