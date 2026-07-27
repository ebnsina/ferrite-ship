package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// notificationsView says what this installation can do as well as what the
// account asked for.
//
// CanSend is the honest part: without a mail server the settings still save,
// and a page that offered no explanation for the silence would leave somebody
// believing they had turned alerts on.
type notificationsView struct {
	CanSend bool `json:"canSend"`

	Email          string `json:"email"`
	OnBackupFailed bool   `json:"onBackupFailed"`
	OnServerDown   bool   `json:"onServerDown"`
	OnDiskLow      bool   `json:"onDiskLow"`
	DiskPercent    int    `json:"diskPercent"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}

func (a *API) viewOf(settings store.Notifications) notificationsView {
	view := notificationsView{
		CanSend:        a.alerts != nil && a.alerts.Enabled(),
		Email:          settings.Email,
		OnBackupFailed: settings.OnBackupFailed,
		OnServerDown:   settings.OnServerDown,
		OnDiskLow:      settings.OnDiskLow,
		DiskPercent:    settings.DiskPercent,
	}
	if settings.UpdatedAt != nil {
		view.UpdatedAt = settings.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return view
}

func (a *API) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	settings, err := a.store.GetNotifications(r.Context(), currentUser(r).ID)
	if err != nil {
		a.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, a.viewOf(settings))
}

type notificationsRequest struct {
	Email          string `json:"email"`
	OnBackupFailed bool   `json:"onBackupFailed"`
	OnServerDown   bool   `json:"onServerDown"`
	OnDiskLow      bool   `json:"onDiskLow"`
	DiskPercent    int    `json:"diskPercent"`
}

func (a *API) handleSaveNotifications(w http.ResponseWriter, r *http.Request) {
	var req notificationsRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	email := strings.TrimSpace(req.Email)
	// An empty address is how alerts are turned off, and is not an error. A
	// non-empty one that cannot possibly work is: an address with a typo fails
	// silently at exactly the moment it matters.
	if email != "" && !plausibleEmail(email) {
		a.fail(w, apierr.InvalidNotificationEmail)
		return
	}

	percent := req.DiskPercent
	if percent < 50 || percent > 99 {
		// Below half is noise on any machine that is being used; at 100 there
		// is nothing left to warn about.
		percent = 85
	}

	settings := store.Notifications{
		Email:          email,
		OnBackupFailed: req.OnBackupFailed,
		OnServerDown:   req.OnServerDown,
		OnDiskLow:      req.OnDiskLow,
		DiskPercent:    percent,
	}

	if err := a.store.SaveNotifications(r.Context(), currentUser(r).ID, settings); err != nil {
		a.fail(w, err)
		return
	}

	stored, err := a.store.GetNotifications(r.Context(), currentUser(r).ID)
	if err != nil {
		a.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, a.viewOf(stored))
}

// handleTestNotification proves the path end to end.
//
// The one thing worse than no alerts is alerts nobody has ever seen arrive.
// The reply is the mail server's own refusal where there is one, because
// "check your settings" is useless next to "the server rejected the login".
func (a *API) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if a.alerts == nil || !a.alerts.Enabled() {
		a.fail(w, apierr.NoMailServer)
		return
	}

	settings, err := a.store.GetNotifications(r.Context(), currentUser(r).ID)
	if err != nil {
		a.fail(w, err)
		return
	}
	if settings.Email == "" {
		a.fail(w, apierr.InvalidNotificationEmail)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	err = a.alerts.Send(ctx, notify.Message{
		To:      settings.Email,
		Subject: "Ferrite Ship can reach you",
		Body: "This is the test message from your notification settings.\n\n" +
			"Nothing is wrong. It means that when something is — a scheduled " +
			"backup that did not finish, a server that stopped answering, a " +
			"disk filling up — the message will arrive here.\n",
	})
	if err != nil {
		a.log.Error("test notification failed", "error", err)
		a.fail(w, apierr.MailNotDelivered.WithCause(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"sentTo": settings.Email})
}

// handleListAlerts returns what is currently wrong, for the dashboard.
//
// Read from the same rows the emails are sent from, so the page and the inbox
// cannot disagree about whether something is still a problem.
func (a *API) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	open, err := a.store.OpenAlerts(r.Context(), currentUser(r).ID)
	if err != nil {
		a.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, open)
}

// plausibleEmail is deliberately not a full validator.
//
// The only address that matters is one a mail server will accept, and no
// regular expression settles that — the test send does. This rejects what is
// certainly wrong and gets out of the way.
func plausibleEmail(value string) bool {
	local, domain, found := strings.Cut(value, "@")
	if !found || local == "" || domain == "" {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n,") {
		return false
	}
	return strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".")
}
