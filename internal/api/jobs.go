package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

type startJobRequest struct {
	Kind  string `json:"kind"`
	Actor string `json:"actor"`
	// DryRun runs every check and reports what would change, altering nothing.
	DryRun bool `json:"dryRun"`
}

func (a *API) handleStartJob(w http.ResponseWriter, r *http.Request) {
	var req startJobRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &req); err != nil {
			a.writeError(w, http.StatusBadRequest, "parse", "We could not read that request.")
			return
		}
	}

	if req.Kind != "" && req.Kind != "baseline" {
		a.writeError(w, http.StatusBadRequest, "parse",
			`The only job available right now is "baseline".`)
		return
	}
	if req.Actor == "" {
		req.Actor = "You"
	}

	job, err := a.runner.StartBaseline(r.Context(), r.PathValue("id"), req.Actor, req.DryRun)
	switch {
	case errors.Is(err, runner.ErrAlreadyRunning):
		a.writeError(w, http.StatusConflict, "conflict",
			"Something is already running on this server. Wait for it to finish.")
		return
	case err != nil:
		a.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (a *API) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := a.store.GetJob(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// handleJobEvents streams a job's history as Server-Sent Events.
//
// A client reconnecting sends Last-Event-ID (or ?after=), and gets everything
// after that sequence number followed by the live tail — so a dropped
// connection never loses a log line.
func (a *API) handleJobEvents(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")

	job, err := a.store.GetJob(r.Context(), jobID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		a.writeError(w, http.StatusInternalServerError, "server",
			"Live updates are not available on this connection.")
		return
	}

	after := resumeFrom(r)

	// Subscribe before replaying, so events produced during the replay are
	// buffered rather than lost in the gap between the two.
	live, unsubscribe := a.bus.Subscribe(jobID)
	defer unsubscribe()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// Without this, a reverse proxy may hold the stream in a buffer forever.
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	history, err := a.store.ListEvents(r.Context(), jobID, after)
	if err != nil {
		a.log.Error("could not replay job events", "job", jobID, "error", err)
		return
	}

	highest := after
	for _, event := range history {
		writeEvent(w, event)
		highest = event.Seq
	}
	flusher.Flush()

	// A finished job has nothing more to say.
	if job.Status == store.JobSucceeded || job.Status == store.JobFailed {
		writeDone(w)
		flusher.Flush()
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return

		case event, open := <-live:
			if !open {
				return
			}
			// Skip anything the replay already delivered.
			if event.Seq <= highest {
				continue
			}
			highest = event.Seq

			writeEvent(w, event)
			flusher.Flush()

			if event.Type == runner.EventJobEnded {
				writeDone(w)
				flusher.Flush()
				return
			}
		}
	}
}

func resumeFrom(r *http.Request) int {
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		if seq, err := strconv.Atoi(raw); err == nil {
			return seq
		}
	}
	if raw := r.URL.Query().Get("after"); raw != "" {
		if seq, err := strconv.Atoi(raw); err == nil {
			return seq
		}
	}
	return 0
}

func writeEvent(w http.ResponseWriter, event store.Event) {
	payload, err := jsonBytes(event)
	if err != nil {
		return
	}
	// The SSE id is the sequence number, which is what Last-Event-ID resumes from.
	fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.Seq, event.Type, payload)
}

func writeDone(w http.ResponseWriter) {
	fmt.Fprint(w, "event: done\ndata: {}\n\n")
}

// --- activity ---------------------------------------------------------------

type activityView struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ServerName string `json:"serverName"`
	Actor      string `json:"actor"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	DurationMs *int64 `json:"durationMs"`
}

func (a *API) handleActivity(w http.ResponseWriter, r *http.Request) {
	jobs, err := a.store.ListRecentJobs(r.Context(), 25)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	names := make(map[string]string, len(servers))
	for _, s := range servers {
		names[s.ID] = s.Name
	}

	views := make([]activityView, 0, len(jobs))
	for _, job := range jobs {
		views = append(views, activityView{
			ID:         job.ID,
			Title:      job.Title,
			ServerName: orDefault(names[job.ServerID], "a removed server"),
			Actor:      job.Actor,
			Status:     string(job.Status),
			StartedAt:  job.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
			DurationMs: job.DurationMs(),
		})
	}

	writeJSON(w, http.StatusOK, views)
}
