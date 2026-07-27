package watch

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/alerts"
	"github.com/ebnsina/ferrite-ship/internal/config"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

func testWatcher(t *testing.T) (*Watcher, *store.Store, store.Server) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	server := store.Server{
		ID: "srv_1", UserID: "usr_alice", Name: "web-1", Kind: store.ConnectionSSH,
		Status: store.StatusOnline, Services: []string{}, CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(context.Background(), server); err != nil {
		t.Fatalf("create server: %v", err)
	}

	// A reporter with no mail server still records alerts, which is exactly
	// what is being checked: whether the condition was announced, not whether
	// an email left the building.
	reporter := alerts.New(st, notify.New(config.SMTP{}), "", slog.New(slog.DiscardHandler))

	// No dialer: nothing reached here opens a connection.
	return &Watcher{
		store: st, alerts: reporter, log: slog.New(slog.DiscardHandler),
		failures: map[string]int{},
	}, st, server
}

func openAlerts(t *testing.T, st *store.Store) []store.Alert {
	t.Helper()

	found, err := st.OpenAlerts(context.Background(), "usr_alice")
	if err != nil {
		t.Fatalf("open alerts: %v", err)
	}
	return found
}

func statusOf(t *testing.T, st *store.Store) store.ServerStatus {
	t.Helper()

	server, err := st.GetServer(context.Background(), "usr_alice", "srv_1")
	if err != nil {
		t.Fatalf("get server: %v", err)
	}
	return server.Status
}

// One missed connection is a network hiccup far more often than a dead
// machine. Alerting on the first would cry wolf, and an alert nobody believes
// is worse than one that arrives five minutes later.
func TestOneMissedCheckIsNotEnoughToCallAServerDown(t *testing.T) {
	watcher, st, server := testWatcher(t)
	ctx := context.Background()

	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))

	if got := statusOf(t, st); got == store.StatusOffline {
		t.Error("a single failure marked the server offline")
	}
	if alerts := openAlerts(t, st); len(alerts) != 0 {
		t.Errorf("a single failure raised %d alerts", len(alerts))
	}
}

// The second consecutive one is.
func TestTheSecondConsecutiveMissDoesCallItDown(t *testing.T) {
	watcher, st, server := testWatcher(t)
	ctx := context.Background()

	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))
	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))

	if got := statusOf(t, st); got != store.StatusOffline {
		t.Errorf("status is %q after two failures, want offline", got)
	}

	raised := openAlerts(t, st)
	if len(raised) != 1 {
		t.Fatalf("two failures raised %d alerts, want 1", len(raised))
	}
	if raised[0].ServerID != server.ID {
		t.Errorf("the alert is about %q", raised[0].ServerID)
	}
}

// A server that is down stays down without a second alert every five minutes.
// Twelve messages an hour is how somebody learns to filter them.
func TestStayingDownDoesNotKeepRaisingAlerts(t *testing.T) {
	watcher, st, server := testWatcher(t)
	ctx := context.Background()

	for range 6 {
		watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))
	}

	if raised := openAlerts(t, st); len(raised) != 1 {
		t.Errorf("six failed rounds produced %d alerts, want 1", len(raised))
	}
}

// A blip that recovers must not accumulate. Failing, answering, then failing
// again is two separate first failures, not a run of two — otherwise a flaky
// network would eventually be reported as an outage that never happened.
func TestARecoveryResetsTheRunOfFailures(t *testing.T) {
	watcher, st, server := testWatcher(t)
	ctx := context.Background()

	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))
	watcher.answered(server.ID)
	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))

	if got := statusOf(t, st); got == store.StatusOffline {
		t.Error("two failures either side of a success were treated as consecutive")
	}
	if raised := openAlerts(t, st); len(raised) != 0 {
		t.Errorf("a blip that recovered raised %d alerts", len(raised))
	}

	// And the count really did start again: one more failure is now the
	// second in a row, and should call it down.
	watcher.unreachable(ctx, server, store.Notifications{}, errors.New("i/o timeout"))
	if got := statusOf(t, st); got != store.StatusOffline {
		t.Errorf("status is %q; the count did not resume correctly", got)
	}
}

// The counter is per server, so one machine being down must not bring another
// closer to being called down.
func TestFailuresAreCountedPerServer(t *testing.T) {
	watcher, st, first := testWatcher(t)
	ctx := context.Background()

	second := store.Server{
		ID: "srv_2", UserID: "usr_alice", Name: "web-2", Kind: store.ConnectionSSH,
		Status: store.StatusOnline, Services: []string{}, CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(ctx, second); err != nil {
		t.Fatalf("create second server: %v", err)
	}

	watcher.unreachable(ctx, first, store.Notifications{}, errors.New("i/o timeout"))
	watcher.unreachable(ctx, second, store.Notifications{}, errors.New("i/o timeout"))

	if raised := openAlerts(t, st); len(raised) != 0 {
		t.Errorf("one failure each on two servers raised %d alerts", len(raised))
	}
}
