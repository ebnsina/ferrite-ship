package api

import (
	"net/http"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/store"
)

// problemsView is everything currently worth somebody's attention.
//
// One response rather than two, because the question being asked is "what is
// wrong?" and answering it from two endpoints means a page that can render
// half an answer — alerts arriving while failures are still loading, or one of
// them failing and the page quietly showing fewer problems than there are.
// Under-reporting problems is the one failure mode this page must not have.
type problemsView struct {
	// Alerts are conditions that are true right now: a server not answering, a
	// disk filling up. They clear themselves when the condition ends.
	Alerts []alertView `json:"alerts"`
	// Failures are runs that did not finish. They do not clear — a failed
	// backup stays failed however long ago it was — so they are capped.
	Failures []failureView `json:"failures"`
}

type alertView struct {
	ID         string `json:"id"`
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName"`
	Kind       string `json:"kind"`
	Subject    string `json:"subject"`
	Detail     string `json:"detail"`
	OpenedAt   string `json:"openedAt"`
}

type failureView struct {
	JobID      string `json:"jobId"`
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	// Actor is who asked for it, or "Scheduled" where nobody did. The
	// difference matters on this page more than anywhere else: a run somebody
	// watched fail is already known about, and one that failed overnight is
	// the reason this page exists.
	Actor string `json:"actor"`
	// Error is what actually went wrong, already redacted on its way into the
	// row. Shown in full rather than summarised: a person reading this page has
	// come looking for exactly this sentence.
	Error      string `json:"error"`
	FinishedAt string `json:"finishedAt"`
}

// handleListProblems answers "what is wrong, and where?".
func (a *API) handleListProblems(w http.ResponseWriter, r *http.Request) {
	userID := currentUser(r).ID

	servers, err := a.store.ListServers(r.Context(), userID)
	if err != nil {
		a.failServer(w, err)
		return
	}
	// So a problem can name the machine rather than its id. Built here rather
	// than joined in SQL because both queries need it and neither owns it.
	names := make(map[string]string, len(servers))
	for _, srv := range servers {
		names[srv.ID] = srv.Name
	}

	open, err := a.store.OpenAlerts(r.Context(), userID)
	if err != nil {
		a.failServer(w, err)
		return
	}

	failed, err := a.store.ListFailedJobs(r.Context(), userID, 20)
	if err != nil {
		a.failServer(w, err)
		return
	}

	view := problemsView{
		Alerts:   make([]alertView, 0, len(open)),
		Failures: make([]failureView, 0, len(failed)),
	}

	for _, alert := range open {
		view.Alerts = append(view.Alerts, alertView{
			ID:         alert.ID,
			ServerID:   alert.ServerID,
			ServerName: orDefault(names[alert.ServerID], "A server you have removed"),
			Kind:       alert.Kind,
			Subject:    alert.Subject,
			Detail:     alert.Detail,
			OpenedAt:   alert.OpenedAt.UTC().Format(time.RFC3339),
		})
	}

	for _, job := range failed {
		view.Failures = append(view.Failures, failureView{
			JobID:      job.ID,
			ServerID:   job.ServerID,
			ServerName: orDefault(names[job.ServerID], "A server you have removed"),
			Title:      job.Title,
			Kind:       job.Kind,
			Actor:      orDefault(job.Actor, store.ActorScheduled),
			Error:      job.Error,
			FinishedAt: formatOptionalTime(finishedAt(job)),
		})
	}

	writeJSON(w, http.StatusOK, view)
}

// finishedAt prefers when the run ended, falling back to when it started.
//
// A job can be marked failed without a finish time if the process died between
// the two, and an empty date on this page reads as "this never happened".
func finishedAt(job store.Job) time.Time {
	if job.FinishedAt != nil {
		return *job.FinishedAt
	}
	return job.StartedAt
}
