package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ebnsina/ferrite-ship/internal/services"
)

func (a *API) writeServicesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrNotSupported):
		a.writeError(w, http.StatusBadRequest, "parse",
			"This is a simulated server, so there are no services to manage. Connect a real server to use this.")
	case errors.Is(err, services.ErrBadUnit):
		a.writeError(w, http.StatusBadRequest, "parse", "That is not a service name we recognise.")
	case errors.Is(err, services.ErrBadAction):
		a.writeError(w, http.StatusBadRequest, "parse",
			"You can start, stop, restart, turn on or turn off a service — nothing else.")
	case errors.Is(err, services.ErrProtected):
		a.writeError(w, http.StatusConflict, "conflict", err.Error())
	default:
		a.writeError(w, http.StatusBadGateway, "network", friendlyFileError(err))
	}
}

func (a *API) handleListServices(w http.ResponseWriter, r *http.Request) {
	units, err := a.services.List(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeServicesError(w, err)
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
		a.writeError(w, http.StatusBadRequest, "parse", "We could not read that request.")
		return
	}

	err := a.services.Perform(
		r.Context(), r.PathValue("id"), r.PathValue("unit"), services.Action(req.Action))
	if err != nil {
		a.writeServicesError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))

	text, err := a.services.Logs(r.Context(), r.PathValue("id"), r.PathValue("unit"), lines)
	if err != nil {
		a.writeServicesError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
