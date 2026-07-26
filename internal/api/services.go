package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/services"
)

// failServices maps the service layer's sentinels onto catalogue entries. The
// unit name is added where it helps, since the catalogue cannot know it.
func (a *API) failServices(w http.ResponseWriter, err error, unit string) {
	switch {
	case errors.Is(err, services.ErrNotSupported):
		a.fail(w, apierr.NeedsRealServer)
	case errors.Is(err, services.ErrBadUnit):
		a.fail(w, apierr.UnknownService)
	case errors.Is(err, services.ErrBadAction):
		a.fail(w, apierr.UnknownServiceAction)
	case errors.Is(err, services.ErrProtected):
		a.fail(w, apierr.ServiceProtected.WithMessage(unit+" is protected."))
	default:
		a.failServer(w, err)
	}
}

func (a *API) handleListServices(w http.ResponseWriter, r *http.Request) {
	units, err := a.services.List(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServices(w, err, "")
		return
	}
	writeJSON(w, http.StatusOK, units)
}

type serviceActionRequest struct {
	Action string `json:"action"`
}

func (a *API) handleServiceAction(w http.ResponseWriter, r *http.Request) {
	var req serviceActionRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	err := a.services.Perform(
		r.Context(), currentUser(r).ID, r.PathValue("id"), r.PathValue("unit"), services.Action(req.Action))
	if err != nil {
		a.failServices(w, err, r.PathValue("unit"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))

	text, err := a.services.Logs(r.Context(), currentUser(r).ID, r.PathValue("id"), r.PathValue("unit"), lines)
	if err != nil {
		a.failServices(w, err, r.PathValue("unit"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
