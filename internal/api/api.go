// Package api serves the control-plane HTTP interface.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ebnsina/ferrite-ship/internal/files"
	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/services"
	"github.com/ebnsina/ferrite-ship/internal/store"
	"github.com/ebnsina/ferrite-ship/internal/terminal"
)

type API struct {
	store     *store.Store
	runner    *runner.Runner
	bus       *runner.Bus
	sealer    *secret.Sealer
	terminals *terminal.Service
	files     *files.Service
	services  *services.Service
	log       *slog.Logger

	allowedOrigin string
	// allowedOriginHost is the same origin without its scheme, which is the
	// form the websocket handshake checks against.
	allowedOriginHost string
}

type Options struct {
	Store         *store.Store
	Runner        *runner.Runner
	Bus           *runner.Bus
	Sealer        *secret.Sealer
	Terminals     *terminal.Service
	Files         *files.Service
	Services      *services.Service
	Logger        *slog.Logger
	AllowedOrigin string
}

func New(opts Options) *API {
	api := &API{
		store:         opts.Store,
		runner:        opts.Runner,
		bus:           opts.Bus,
		sealer:        opts.Sealer,
		terminals:     opts.Terminals,
		files:         opts.Files,
		services:      opts.Services,
		log:           opts.Logger,
		allowedOrigin: opts.AllowedOrigin,
	}

	if parsed, err := url.Parse(opts.AllowedOrigin); err == nil {
		api.allowedOriginHost = parsed.Host
	}

	return api
}

// Routes returns the API handler. Static assets are mounted separately.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", a.handleHealth)

	mux.HandleFunc("GET /v1/servers", a.handleListServers)
	mux.HandleFunc("GET /v1/servers/{id}", a.handleGetServer)
	mux.HandleFunc("GET /v1/servers/{id}/jobs", a.handleServerJobs)
	mux.HandleFunc("POST /v1/servers", a.handleCreateServer)
	mux.HandleFunc("DELETE /v1/servers/{id}", a.handleDeleteServer)
	mux.HandleFunc("POST /v1/servers/{id}/jobs", a.handleStartJob)
	mux.HandleFunc("GET /v1/servers/{id}/terminal", a.handleTerminal)

	mux.HandleFunc("GET /v1/servers/{id}/files", a.handleListFiles)
	mux.HandleFunc("DELETE /v1/servers/{id}/files", a.handleRemoveFile)
	mux.HandleFunc("GET /v1/servers/{id}/files/content", a.handleReadFile)
	mux.HandleFunc("PUT /v1/servers/{id}/files/content", a.handleWriteFile)
	mux.HandleFunc("GET /v1/servers/{id}/files/download", a.handleDownloadFile)

	mux.HandleFunc("GET /v1/servers/{id}/services", a.handleListServices)
	mux.HandleFunc("POST /v1/servers/{id}/services/{unit}/actions", a.handleServiceAction)
	mux.HandleFunc("GET /v1/servers/{id}/services/{unit}/logs", a.handleServiceLogs)

	mux.HandleFunc("GET /v1/jobs/{id}", a.handleGetJob)
	mux.HandleFunc("GET /v1/jobs/{id}/events", a.handleJobEvents)

	mux.HandleFunc("GET /v1/activity", a.handleActivity)
	mux.HandleFunc("GET /v1/metrics", a.handleMetrics)

	return a.withCORS(mux)
}

// withCORS allows the dev frontend on its own origin to call the API.
// In production the SPA is served by this same process, so no origin is set
// and no cross-origin request is permitted.
func (a *API) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.allowedOrigin != "" && r.Header.Get("Origin") == a.allowedOrigin {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", a.allowedOrigin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			h.Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- responses --------------------------------------------------------------

// errorBody is the single error envelope every non-2xx response uses. The web
// client depends on this shape.
type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status line is already sent; all that is left is to record it.
		slog.Default().Error("could not encode response", "error", err)
	}
}

func (a *API) writeError(w http.ResponseWriter, status int, code, message string) {
	requestID := ids.New("req")
	if status >= http.StatusInternalServerError {
		a.log.Error("request failed", "code", code, "message", message, "request_id", requestID)
	}
	writeJSON(w, status, errorBody{Code: code, Message: message, RequestID: requestID})
}

// writeStoreError maps storage failures onto status codes.
func (a *API) writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		a.writeError(w, http.StatusNotFound, "not_found", "We could not find that.")
		return
	}
	a.writeError(w, http.StatusInternalServerError, "server", "Something went wrong on our side.")
}

func jsonBytes(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
