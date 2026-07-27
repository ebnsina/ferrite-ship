// Package alerts decides whether something is worth saying, and says it once.
//
// Two very different things produce an alert — a health check that found a
// server unreachable, and a scheduled job that failed — and both need the same
// three questions answered: is this new, does this account want to hear it, and
// what words does it get. Answering them in one place is what keeps a message
// from being sent twice, or in two voices.
package alerts

import (
	"context"
	"log/slog"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// sendTimeout bounds one delivery. Mail servers are slow often enough that a
// send without a deadline becomes a goroutine that never returns.
const sendTimeout = 30 * time.Second

type Reporter struct {
	store  *store.Store
	sender notify.Sender
	// origin is where the dashboard is reachable, for the link in a message.
	origin string
	log    *slog.Logger
}

func New(st *store.Store, sender notify.Sender, origin string, log *slog.Logger) *Reporter {
	return &Reporter{store: st, sender: sender, origin: origin, log: log}
}

// Raise records a condition and, if it is new, tells somebody.
func (r *Reporter) Raise(
	ctx context.Context, userID, serverID string, settings store.Notifications, alert notify.Alert,
) {
	fresh, err := r.store.OpenAlert(ctx, store.Alert{
		ID:       ids.New("alr"),
		UserID:   userID,
		ServerID: serverID,
		Kind:     string(alert.Kind),
		Subject:  key(alert),
		Detail:   alert.Detail,
	})
	if err != nil {
		r.log.Error("could not record an alert", "kind", alert.Kind, "error", err)
		return
	}
	if !fresh {
		// Already open, and already said. Staying quiet is the whole point: a
		// condition checked every five minutes would otherwise be mailed every
		// five minutes, and the next real alert would arrive in a folder
		// nobody opens.
		return
	}

	r.log.Warn("alert raised", "kind", alert.Kind, "server", alert.Server, "detail", alert.Detail)

	if !settings.Wants(string(alert.Kind)) || !r.sender.Enabled() {
		return
	}

	message := notify.Render(alert, r.origin)
	message.To = settings.Email
	r.send(ctx, message)
}

// Resolve closes a condition, and reports the recovery if the problem was
// reported in the first place.
//
// A message that never gets a second half trains people to ignore the first.
func (r *Reporter) Resolve(
	ctx context.Context, userID, serverID string,
	settings store.Notifications, alert notify.Alert,
) {
	was, err := r.store.ClearAlert(ctx, userID, serverID, string(alert.Kind), key(alert))
	if err != nil {
		r.log.Error("could not clear an alert", "kind", alert.Kind, "error", err)
		return
	}
	if !was {
		return
	}

	r.log.Info("alert cleared", "kind", alert.Kind, "server", alert.Server)

	if !settings.Wants(string(alert.Kind)) || !r.sender.Enabled() {
		return
	}

	message := notify.Cleared(alert, r.origin)
	message.To = settings.Email
	r.send(ctx, message)
}

// key is what the database de-duplicates on: stable, and not the words.
func key(alert notify.Alert) string {
	if alert.Key != "" {
		return alert.Key
	}
	return alert.Subject
}

// Settings is a convenience for callers that hold a user id and nothing else.
func (r *Reporter) Settings(ctx context.Context, userID string) store.Notifications {
	settings, err := r.store.GetNotifications(ctx, userID)
	if err != nil {
		r.log.Error("could not read notification settings", "user", userID, "error", err)
		return store.Notifications{}
	}
	return settings
}

// Send delivers a message that is not an alert — the test from the settings
// page — and reports what went wrong rather than logging it, because somebody
// is watching that button and needs the reason.
func (r *Reporter) Send(ctx context.Context, message notify.Message) error {
	return r.sender.Send(ctx, message)
}

// Enabled reports whether a mail server is configured at all.
func (r *Reporter) Enabled() bool { return r.sender.Enabled() }

// send delivers on its own deadline.
//
// Detached from the caller's context: a check being cancelled during shutdown
// has usually just found the thing most worth sending, and the alert is
// already recorded as said — abandoning the message mid-flight would mean it
// is never sent at all.
func (r *Reporter) send(ctx context.Context, message notify.Message) {
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sendTimeout)
	defer cancel()

	if err := r.sender.Send(sendCtx, message); err != nil {
		r.log.Error("could not send a notification", "to", message.To, "error", err)
	}
}
