package runner

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/dialer"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// The property everything else rests on: running the baseline twice must
// change the machine once. If this breaks, re-running a playbook stops being
// safe and the product's central promise is gone.
func TestBaselineIsIdempotent(t *testing.T) {
	runner, st := newTestRunner(t)
	serverID := seedDemoServer(t, st, "idempotent")

	first := runToCompletion(t, runner, st, serverID)
	if first.Status != store.JobSucceeded {
		t.Fatalf("first run failed: %s", first.Error)
	}
	if first.Changed == 0 {
		t.Fatal("first run changed nothing on a fresh machine")
	}

	second := runToCompletion(t, runner, st, serverID)
	if second.Status != store.JobSucceeded {
		t.Fatalf("second run failed: %s", second.Error)
	}
	if second.Changed != 0 {
		t.Errorf("second run changed %d steps; a converged playbook must change none", second.Changed)
	}
	if second.Unchanged == 0 {
		t.Error("second run reported nothing as already-satisfied, so the checks are not reading state")
	}
}

// A dry run must report what is outstanding and touch nothing, so it is safe
// as a first contact with a server somebody cares about.
func TestDryRunChangesNothing(t *testing.T) {
	runner, st := newTestRunner(t)
	serverID := seedDemoServer(t, st, "dry")

	first := runDryToCompletion(t, runner, st, serverID)
	if first.Status != store.JobSucceeded {
		t.Fatalf("dry run failed: %s", first.Error)
	}
	if first.Changed != 0 {
		t.Errorf("a dry run applied %d changes", first.Changed)
	}

	// Repeating it must give the same answer — proof the first altered nothing.
	second := runDryToCompletion(t, runner, st, serverID)
	if second.Skipped != first.Skipped {
		t.Errorf("second dry run differs from the first (%d vs %d outstanding); the first changed something",
			second.Skipped, first.Skipped)
	}

	// And a real run afterwards should still have work to do.
	real := runToCompletion(t, runner, st, serverID)
	if real.Changed == 0 {
		t.Error("nothing left to change after two dry runs, so a dry run applied the work")
	}
}

func TestRefusesConcurrentJobsOnOneServer(t *testing.T) {
	runner, st := newTestRunner(t)
	runner.DemoLatency = 20 * time.Millisecond // keep the first job in flight
	serverID := seedDemoServer(t, st, "busy")

	if _, err := runner.StartBaseline(context.Background(), testUserID, serverID, "test", false); err != nil {
		t.Fatalf("first start: %v", err)
	}

	_, err := runner.StartBaseline(context.Background(), testUserID, serverID, "test", false)
	if err != ErrAlreadyRunning {
		t.Errorf("second concurrent start returned %v, want ErrAlreadyRunning", err)
	}
}

// --- helpers ---------------------------------------------------------------

// Every test server belongs to the same owner; scoping itself is covered in
// the store's own tests.
const testUserID = "usr_test"

func newTestRunner(t *testing.T) (*Runner, *store.Store) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	key, err := secret.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sealer, err := secret.NewSealer(key)
	if err != nil {
		t.Fatalf("new sealer: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := New(st, dialer.New(st, sealer), NewBus(), sealer, log)
	runner.DemoLatency = 0 // no need to pace logs for a machine nobody watches

	return runner, st
}

func seedDemoServer(t *testing.T, st *store.Store, name string) string {
	t.Helper()

	server := store.Server{
		ID:        "srv_" + name,
		UserID:    testUserID,
		Name:      name,
		Kind:      store.ConnectionDemo,
		Status:    store.StatusUnknown,
		Services:  []string{},
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(context.Background(), server); err != nil {
		t.Fatalf("create server: %v", err)
	}
	return server.ID
}

func runToCompletion(t *testing.T, r *Runner, st *store.Store, serverID string) store.Job {
	t.Helper()
	return start(t, r, st, serverID, false)
}

func runDryToCompletion(t *testing.T, r *Runner, st *store.Store, serverID string) store.Job {
	t.Helper()
	return start(t, r, st, serverID, true)
}

func start(t *testing.T, r *Runner, st *store.Store, serverID string, dryRun bool) store.Job {
	t.Helper()

	job, err := r.StartBaseline(context.Background(), testUserID, serverID, "test", dryRun)
	if err != nil {
		t.Fatalf("start job: %v", err)
	}

	// The run is detached, so wait for the record to settle rather than
	// reaching into the runner's internals.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		current, err := st.GetJob(context.Background(), testUserID, job.ID)
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		if current.Status != store.JobRunning {
			return current
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("job %s did not finish within the deadline", job.ID)
	return store.Job{}
}
