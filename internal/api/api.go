// Package api serves the control-plane HTTP interface.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/auth"
	"github.com/ebnsina/ferrite-ship/internal/console"
	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
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
	console   *console.Service
	auth      *auth.Service
	log       *slog.Logger

	allowedOrigin string
	// allowedOriginHost is the same origin without its scheme, which is the
	// form the websocket handshake checks against.
	allowedOriginHost string

	// signIns bounds password guessing.
	signIns *throttle
}

type Options struct {
	Store         *store.Store
	Runner        *runner.Runner
	Bus           *runner.Bus
	Sealer        *secret.Sealer
	Terminals     *terminal.Service
	Files         *files.Service
	Services      *services.Service
	Console       *console.Service
	Auth          *auth.Service
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
		console:       opts.Console,
		auth:          opts.Auth,
		log:           opts.Logger,
		allowedOrigin: opts.AllowedOrigin,
	}

	api.signIns = newThrottle()

	if parsed, err := url.Parse(opts.AllowedOrigin); err == nil {
		api.allowedOriginHost = parsed.Host
	}

	return api
}

// Routes returns the API handler. Static assets are mounted separately.
func (a *API) Routes() http.Handler {
	// Open: liveness, and the endpoints you need before you have a session.
	open := http.NewServeMux()
	open.HandleFunc("GET /v1/health", a.handleHealth)
	open.HandleFunc("GET /v1/auth/status", a.handleAuthStatus)
	open.HandleFunc("POST /v1/auth/setup", a.handleSetup)
	open.HandleFunc("POST /v1/auth/login", a.handleLogin)
	open.HandleFunc("POST /v1/auth/logout", a.handleLogout)

	// Everything that touches a server sits behind a session, so no individual
	// handler has to remember to check.
	mux := http.NewServeMux()

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

	mux.HandleFunc("GET /v1/catalog", a.handleListCatalog)
	mux.HandleFunc("GET /v1/servers/{id}/tools", a.handleListTools)
	mux.HandleFunc("POST /v1/servers/{id}/tools", a.handleInstallTool)
	mux.HandleFunc("DELETE /v1/servers/{id}/tools/{tool}", a.handleRemoveTool)
	mux.HandleFunc("GET /v1/servers/{id}/tools/{tool}/connection", a.handleToolConnection)
	mux.HandleFunc("POST /v1/servers/{id}/tools/{tool}/query", a.handleToolQuery)

	mux.HandleFunc("GET /v1/jobs/{id}", a.handleGetJob)
	mux.HandleFunc("GET /v1/jobs/{id}/events", a.handleJobEvents)

	mux.HandleFunc("GET /v1/activity", a.handleActivity)
	mux.HandleFunc("GET /v1/metrics", a.handleMetrics)

	open.Handle("/v1/", a.requireSession(mux))

	return a.withCORS(open)
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

// errorBody is the one shape every non-2xx response uses. Message says what
// happened; Action says what to do next. Both come from the catalogue in
// internal/apierr, so the wording lives in exactly one place and the web
// client renders it rather than keeping a second copy.
type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Action    string `json:"action,omitempty"`
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

// fail is the only way a handler reports a problem. Anything catchable can be
// passed in: the catalogue classifies it, so no handler decides on a status
// code or writes a sentence of its own.
func (a *API) fail(w http.ResponseWriter, err error) {
	a.failAs(w, err, apierr.NotFound)
}

// failServer is the same, but a missing row means a missing server — the
// wording every route under /v1/servers/{id} wants.
func (a *API) failServer(w http.ResponseWriter, err error) {
	a.failAs(w, err, apierr.ServerNotFound)
}

func (a *API) failAs(w http.ResponseWriter, err error, missing *apierr.Error) {
	// Only classify what has not already been classified. A handler that has
	// picked an entry itself knows more about the request than this does, and
	// re-deriving one from the cause would replace "that tool is not set up on
	// this server" with a vague "we could not find that".
	var chosen *apierr.Error
	if !errors.As(err, &chosen) {
		if errors.Is(err, store.ErrNotFound) {
			err = missing.WithCause(err)
		}
		if errors.Is(err, sshexec.ErrHostKeyChanged) {
			err = apierr.HostKeyChanged.WithCause(err)
		}
	}

	problem := apierr.From(err)
	requestID := ids.New("req")

	// Only our own failures are worth logging as errors; a wrong password or a
	// missing file is the system working.
	if problem.Status >= http.StatusInternalServerError {
		a.log.Error("request failed",
			"code", problem.Code, "message", problem.Message,
			"cause", problem.Unwrap(), "request_id", requestID)
	}

	writeJSON(w, problem.Status, errorBody{
		Code:      string(problem.Code),
		Message:   problem.Message,
		Action:    problem.Action,
		RequestID: requestID,
	})
}

func jsonBytes(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
