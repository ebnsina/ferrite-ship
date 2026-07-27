package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/insight"
	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// handleStorage reports what is using a server's disk.
//
// Live rather than from the last job's facts: someone opens this because a bar
// went red, and an answer from six hours ago is the wrong answer.
func (a *API) handleStorage(w http.ResponseWriter, r *http.Request) {
	// Walking a disk is slow, and on a very large or very slow one it can be
	// slow enough to be indistinguishable from hung. Bounded here so it fails
	// with an answer rather than holding a connection open indefinitely.
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	server, err := a.store.GetServer(ctx, currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	client, _, err := a.dialer.Dial(ctx, currentUser(r).ID, server.ID)
	if err != nil {
		a.failServer(w, err)
		return
	}
	defer func() { _ = client.Close() }()

	privilege, err := steps.DetectPrivilege(ctx, client)
	if err != nil {
		a.failServer(w, err)
		return
	}

	report, err := insight.Gather(ctx, steps.NewSession(client, nil).WithPrivilege(privilege))
	if err != nil {
		a.failServer(w, err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

type reclaimRequest struct {
	// Items are ids from the report. Anything unrecognised is refused rather
	// than ignored — this runs as root.
	Items []string `json:"items"`
}

func (a *API) handleReclaim(w http.ResponseWriter, r *http.Request) {
	var req reclaimRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}
	if len(req.Items) == 0 {
		a.fail(w, apierr.NothingToReclaim)
		return
	}

	for _, item := range req.Items {
		if insight.Commands(item) == nil {
			a.fail(w, apierr.NothingToReclaim)
			return
		}
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	job, err := a.runner.StartReclaim(r.Context(), currentUser(r).ID, server.ID, req.Items, actorOf(r))
	if err != nil {
		a.failServer(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
