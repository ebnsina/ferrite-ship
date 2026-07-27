// Package watch looks at servers when nobody has asked it to.
//
// Until this existed, a server's facts were only refreshed as a side effect of
// running a job, so a machine that stopped answering stayed "online" on the
// dashboard indefinitely — the last thing anyone knew about it was whatever was
// true the last time they pressed a button. Nothing can be reported that is
// never looked at.
package watch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/alerts"
	"github.com/ebnsina/ferrite-ship/internal/dialer"
	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/facts"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

const (
	// interval is how often every server is checked.
	//
	// Five minutes is a compromise. More often costs an SSH connection per
	// server per round for a number that rarely changes; less often means a
	// machine can be down for a quarter of an hour before anyone hears.
	interval = 5 * time.Minute

	// perServer bounds one check. Generous, because a server under real load
	// answers slowly, and calling that "down" would be its own false alarm.
	perServer = 30 * time.Second

	// failuresBeforeDown is how many rounds in a row must fail before a server
	// is called down.
	//
	// Two, not one: a single missed connection is a network hiccup far more
	// often than it is a dead machine, and an alert that cries wolf is worse
	// than one that arrives five minutes later.
	failuresBeforeDown = 2
)

type Watcher struct {
	store  *store.Store
	dialer *dialer.Dialer
	alerts *alerts.Reporter
	log    *slog.Logger

	// failures counts consecutive unsuccessful checks per server.
	//
	// In memory on purpose. It is a debounce, not a record: after a restart the
	// count starts again, which delays an alert by one round and never invents
	// one. Persisting it would mean writing to the database every five minutes
	// to store something nobody reads.
	failures map[string]int
}

func New(st *store.Store, d *dialer.Dialer, reporter *alerts.Reporter, log *slog.Logger) *Watcher {
	return &Watcher{
		store:    st,
		dialer:   d,
		alerts:   reporter,
		log:      log,
		failures: map[string]int{},
	}
}

// Run checks every server until the context is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	w.log.Info("server watch started", "interval", interval.String())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.round(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("server watch stopped")
			return
		case <-ticker.C:
			w.round(ctx)
		}
	}
}

func (w *Watcher) round(ctx context.Context) {
	accounts, err := w.store.EveryNotifiable(ctx)
	if err != nil {
		w.log.Error("could not read notification settings", "error", err)
		return
	}

	for userID, settings := range accounts {
		servers, err := w.store.ListServers(ctx, userID)
		if err != nil {
			w.log.Error("could not list servers to check", "user", userID, "error", err)
			continue
		}

		for _, server := range servers {
			if ctx.Err() != nil {
				return
			}
			w.check(ctx, server, settings)
		}
	}
}

// check reaches one server and acts on what it finds.
func (w *Watcher) check(ctx context.Context, server store.Server, settings store.Notifications) {
	if server.Kind != store.ConnectionSSH {
		// Simulated servers have nothing to reach and would otherwise report
		// themselves down for ever.
		return
	}

	ctx, cancel := context.WithTimeout(ctx, perServer)
	defer cancel()

	client, _, err := w.dialer.Dial(ctx, server.UserID, server.ID)
	if err != nil {
		w.unreachable(ctx, server, settings, err)
		return
	}
	defer func() { _ = client.Close() }()

	gathered, err := facts.Gather(ctx, w.session(client))
	if err != nil {
		w.unreachable(ctx, server, settings, err)
		return
	}

	w.answered(server.ID)

	if err := w.store.UpdateServerState(ctx, server.ID, store.StatusOnline, gathered, time.Now().UTC()); err != nil {
		w.log.Warn("could not record what a server reported", "server", server.ID, "error", err)
	}

	w.resolve(ctx, server, settings, notify.Alert{
		Kind:   notify.KindServerDown,
		Server: server.Name,
		Link:   "/dashboard/servers/" + server.ID,
	})
	w.disk(ctx, server, settings, gathered)
}

// session wraps a connection so facts can read from it.
//
// The log callback drops everything: a scheduled health check has no job to
// write to, and facts.Gather says nothing worth keeping anyway.
func (w *Watcher) session(client *sshexec.Client) *steps.Session {
	return steps.NewSession(client, func(steps.Level, string) {})
}

// answered forgets a server's run of failures, so a machine that blips and
// recovers never accumulates its way to an alert. Flapping is a different
// problem from being down, and calling it "down" would be wrong.
func (w *Watcher) answered(serverID string) {
	delete(w.failures, serverID)
}

func (w *Watcher) unreachable(
	ctx context.Context, server store.Server, settings store.Notifications, cause error,
) {
	w.failures[server.ID]++
	if w.failures[server.ID] < failuresBeforeDown {
		w.log.Info("a server did not answer, will try again",
			"server", server.Name, "attempt", w.failures[server.ID])
		return
	}

	if err := w.store.SetServerStatus(ctx, server.ID, store.StatusOffline); err != nil {
		w.log.Warn("could not mark a server offline", "server", server.ID, "error", err)
	}

	w.raise(ctx, server, settings, notify.Alert{
		Kind:   notify.KindServerDown,
		Server: server.Name,
		Detail: cause.Error(),
		Link:   "/dashboard/servers/" + server.ID,
	})
}

// disk raises or clears the "nearly full" alert.
//
// Both directions are handled here rather than only the raise, because a disk
// that was cleaned up and never reported as recovered leaves a warning on the
// dashboard that outlives the problem.
func (w *Watcher) disk(
	ctx context.Context, server store.Server, settings store.Notifications, gathered facts.Facts,
) {
	if gathered.DiskTotalBytes <= 0 {
		return
	}

	threshold := settings.DiskPercent
	if threshold <= 0 || threshold > 100 {
		threshold = 85
	}

	percent := int(float64(gathered.DiskUsedBytes) / float64(gathered.DiskTotalBytes) * 100)
	if percent < threshold {
		w.resolve(ctx, server, settings, notify.Alert{
			Kind:   notify.KindDiskLow,
			Server: server.Name,
			Link:   "/dashboard/servers/" + server.ID + "/storage",
		})
		return
	}

	free := gathered.DiskTotalBytes - gathered.DiskUsedBytes
	w.raise(ctx, server, settings, notify.Alert{
		Kind:   notify.KindDiskLow,
		Server: server.Name,
		Detail: fmt.Sprintf("The disk is %d%% full, with %s left of %s.",
			percent, humanBytes(free), humanBytes(gathered.DiskTotalBytes)),
		Link: "/dashboard/servers/" + server.ID + "/storage",
	})
}

// raise records a condition and, if it is new, says so.
func (w *Watcher) raise(
	ctx context.Context, server store.Server, settings store.Notifications, alert notify.Alert,
) {
	w.alerts.Raise(ctx, server.UserID, server.ID, settings, alert)
}

func (w *Watcher) resolve(
	ctx context.Context, server store.Server, settings store.Notifications, alert notify.Alert,
) {
	w.alerts.Resolve(ctx, server.UserID, server.ID, settings, alert)
}

// humanBytes is for message bodies only.
//
// Everything a browser renders goes through Intl; this is a plain-text email
// that never reaches one, so the alternative is shipping a byte count.
func humanBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}

	div, exp := int64(unit), 0
	for n := value / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTP"[exp])
}
