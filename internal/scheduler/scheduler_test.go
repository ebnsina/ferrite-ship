package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// fakeStarter records what it was asked to do and answers with whatever the
// test needs, so the decisions here can be checked without a server behind
// them.
type fakeStarter struct {
	err   error
	calls []string
}

func (f *fakeStarter) StartBackup(
	_ context.Context, _, serverID, toolID, actor string,
) (store.Job, error) {
	f.calls = append(f.calls, serverID+"/"+toolID+" by "+actor)
	return store.Job{}, f.err
}

func testScheduler(t *testing.T, starter *fakeStarter) (*Scheduler, *store.Store) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	return New(st, starter, slog.New(slog.DiscardHandler)), st
}

// seedSchedule writes a schedule already due, and the server it belongs to.
func seedSchedule(t *testing.T, st *store.Store, id string, due time.Time) store.BackupSchedule {
	t.Helper()

	ctx := context.Background()
	server := store.Server{
		ID: "srv_" + id, UserID: "usr_alice", Name: "box", Kind: store.ConnectionDemo,
		Status: store.StatusOnline, Services: []string{}, CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(ctx, server); err != nil {
		t.Fatalf("create server: %v", err)
	}

	schedule := store.BackupSchedule{
		ID: id, UserID: "usr_alice", ServerID: server.ID, ToolID: "postgres",
		Cadence: store.Daily, Hour: 3, Keep: 7, NextRunAt: due,
	}
	if err := st.SaveBackupSchedule(ctx, schedule); err != nil {
		t.Fatalf("save schedule: %v", err)
	}
	return schedule
}

func reload(t *testing.T, st *store.Store, serverID, toolID string) store.BackupSchedule {
	t.Helper()

	got, err := st.GetBackupSchedule(context.Background(), "usr_alice", serverID, toolID)
	if err != nil {
		t.Fatalf("reload schedule: %v", err)
	}
	return got
}

// The ordinary case: a due schedule runs, and is moved on to its next
// occurrence so the same minute does not start it again.
func TestADueBackupRunsAndMovesOn(t *testing.T) {
	starter := &fakeStarter{}
	sched, st := testScheduler(t, starter)

	now := time.Now().UTC()
	seedSchedule(t, st, "sch_due", now.Add(-time.Minute))

	sched.runDue(context.Background())

	if len(starter.calls) != 1 {
		t.Fatalf("started %d backups, want 1", len(starter.calls))
	}
	// Started as the scheduler, not as a person: that is what decides whether
	// a failure is worth emailing about.
	if want := "srv_sch_due/postgres by " + store.ActorScheduled; starter.calls[0] != want {
		t.Errorf("started %q, want %q", starter.calls[0], want)
	}

	after := reload(t, st, "srv_sch_due", "postgres")
	if !after.NextRunAt.After(now) {
		t.Errorf("next run is %v, which is not ahead of now — it would start again next tick",
			after.NextRunAt)
	}
	if after.LastRunAt == nil {
		t.Error("a run that happened was not recorded")
	}
}

// A schedule that is not yet due must be left alone, or every tick would take
// a backup.
func TestABackupThatIsNotDueIsLeftAlone(t *testing.T) {
	starter := &fakeStarter{}
	sched, st := testScheduler(t, starter)

	later := time.Now().UTC().Add(2 * time.Hour)
	seedSchedule(t, st, "sch_later", later)

	sched.runDue(context.Background())

	if len(starter.calls) != 0 {
		t.Errorf("started %d backups that were not due", len(starter.calls))
	}
	if after := reload(t, st, "srv_sch_later", "postgres"); !after.NextRunAt.Equal(later.Truncate(time.Second)) &&
		after.NextRunAt.Sub(later).Abs() > time.Second {
		t.Errorf("an untouched schedule moved to %v, want %v", after.NextRunAt, later)
	}
}

// A server busy with something else is retried shortly rather than skipped: a
// deploy that happened to overlap should not cost somebody a day's backup.
func TestABusyServerIsRetriedSoonRatherThanSkipped(t *testing.T) {
	starter := &fakeStarter{err: runner.ErrAlreadyRunning}
	sched, st := testScheduler(t, starter)

	now := time.Now().UTC()
	seedSchedule(t, st, "sch_busy", now.Add(-time.Minute))

	sched.runDue(context.Background())

	after := reload(t, st, "srv_sch_busy", "postgres")
	gap := after.NextRunAt.Sub(now)
	if gap > retryAfter+time.Minute || gap < retryAfter-time.Minute {
		t.Errorf("retry is %v away, want about %v — a day's wait would lose the backup",
			gap, retryAfter)
	}

	// And it must not claim a backup happened. "When did this last run" is
	// read by a person deciding whether they still have a copy.
	if after.LastRunAt != nil && after.LastRunAt.After(now) {
		t.Error("a deferral was recorded as a run that never happened")
	}
}

// A tool that cannot be backed up at all must not be retried every minute for
// ever. It moves to its next occurrence; the failure is already visible.
func TestABrokenBackupMovesToItsNextOccurrence(t *testing.T) {
	starter := &fakeStarter{err: errors.New("no destination configured")}
	sched, st := testScheduler(t, starter)

	now := time.Now().UTC()
	seedSchedule(t, st, "sch_broken", now.Add(-time.Minute))

	sched.runDue(context.Background())

	after := reload(t, st, "srv_sch_broken", "postgres")
	if gap := after.NextRunAt.Sub(now); gap < time.Hour {
		t.Errorf("a failing backup is retried in %v; it would run every tick for ever", gap)
	}
}

// Deferring a schedule that HAS run before must not disturb that record
// either — a server busy all afternoon should still report this morning's
// backup, not a string of deferrals.
func TestDeferringDoesNotDisturbAnEarlierRun(t *testing.T) {
	starter := &fakeStarter{err: runner.ErrAlreadyRunning}
	sched, st := testScheduler(t, starter)

	ctx := context.Background()
	now := time.Now().UTC()
	schedule := seedSchedule(t, st, "sch_had_run", now.Add(-time.Minute))

	// It ran this morning.
	morning := now.Add(-6 * time.Hour)
	if err := st.MarkScheduleRun(ctx, schedule.ID, morning, now.Add(-time.Minute)); err != nil {
		t.Fatalf("mark run: %v", err)
	}

	sched.runDue(ctx)

	after := reload(t, st, "srv_sch_had_run", "postgres")
	if after.LastRunAt == nil {
		t.Fatal("the earlier run was forgotten")
	}
	if after.LastRunAt.Sub(morning).Abs() > time.Second {
		t.Errorf("last run moved to %v; it should still be this morning at %v",
			after.LastRunAt, morning)
	}
}
