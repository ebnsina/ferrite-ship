package api

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/console"
	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// toolView is a catalogue entry plus, where there is one, what this server has
// done with it. One shape for both means the dashboard renders a single list
// rather than reconciling two.
type toolView struct {
	catalog.Tool
	// Status is "" for a tool that has never been installed here.
	Status      string `json:"status"`
	InstalledAt string `json:"installedAt"`
	// LastJobID lets the UI offer "see what happened" after a failure.
	LastJobID string `json:"lastJobId,omitempty"`
}

func (a *API) handleListCatalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, catalog.All())
}

func (a *API) handleListTools(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	installed, err := a.store.ListInstallations(r.Context(), currentUser(r).ID, server.ID)
	if err != nil {
		a.failServer(w, err)
		return
	}

	byTool := make(map[string]store.Installation, len(installed))
	for _, in := range installed {
		byTool[in.ToolID] = in
	}

	views := make([]toolView, 0, len(catalog.All()))
	for _, tool := range catalog.All() {
		view := toolView{Tool: tool}
		if in, ok := byTool[tool.ID]; ok {
			view.Status = string(in.Status)
			view.InstalledAt = in.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
			view.LastJobID = in.LastJobID
		}
		views = append(views, view)
	}

	writeJSON(w, http.StatusOK, views)
}

type installToolRequest struct {
	ToolID string `json:"toolId"`
	Actor  string `json:"actor"`
	DryRun bool   `json:"dryRun"`
}

func (a *API) handleInstallTool(w http.ResponseWriter, r *http.Request) {
	var req installToolRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}
	if req.Actor == "" {
		req.Actor = "You"
	}

	// Resolve the server here, so that a missing row from this point on means
	// a missing installation. Without it, asking about a server you do not own
	// would answer "that tool is not set up" — which is both wrong and a hint
	// that the server exists.
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	job, err := a.runner.StartInstall(
		r.Context(), currentUser(r).ID, server.ID, req.ToolID, req.Actor, req.DryRun)
	if err != nil {
		a.failTool(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (a *API) handleRemoveTool(w http.ResponseWriter, r *http.Request) {
	// Deleting the data is opt-in and explicit. Anything other than a literal
	// "true" keeps it, so a malformed query string cannot destroy a database.
	purge, _ := strconv.ParseBool(r.URL.Query().Get("purge"))

	actor := r.URL.Query().Get("actor")
	if actor == "" {
		actor = "You"
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	job, err := a.runner.StartRemove(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"), actor, purge)
	if err != nil {
		a.failTool(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

// connectionView is what someone needs to reach a running tool.
type connectionView struct {
	ToolID string `json:"toolId"`
	Name   string `json:"name"`
	// URL is the full connection string, password included. This is the one
	// response that carries a credential, and it is only ever built for a tool
	// belonging to the account asking.
	URL      string `json:"url"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database,omitempty"`
	// Public says whether this can be reached from the internet. When it
	// cannot, Tunnel is the command that makes it reachable from a laptop.
	Public bool   `json:"public"`
	Tunnel string `json:"tunnel,omitempty"`
}

func (a *API) handleToolConnection(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	toolID := r.PathValue("tool")
	tool, err := catalog.Find(toolID)
	if err != nil {
		a.failTool(w, err)
		return
	}
	if tool.Access == nil {
		a.fail(w, apierr.UnknownTool)
		return
	}

	installation, err := a.store.GetInstallation(r.Context(), currentUser(r).ID, server.ID, toolID)
	if err != nil {
		a.failTool(w, err)
		return
	}
	if installation.Status != store.InstallReady {
		a.fail(w, apierr.ToolNotReady)
		return
	}

	password, err := a.sealer.Open(installation.SealedPassword)
	if err != nil {
		a.fail(w, apierr.CredentialNotStored.WithCause(err))
		return
	}

	view := connectionView{
		ToolID:   tool.ID,
		Name:     tool.Name,
		Port:     tool.Access.Port,
		Username: tool.Access.Username,
		Password: password,
		Database: tool.Access.Database,
		Public:   isPublic(tool, tool.Access.Port),
	}

	// A private tool is only reachable from the server itself, so the address
	// that works is localhost — after a tunnel. Handing back the server's
	// public address here would produce a connection string that always fails.
	view.Host = "127.0.0.1"
	if view.Public {
		view.Host = server.Host
	} else {
		view.Tunnel = tunnelCommand(server, tool.Access.Port)
	}

	view.URL = connectionURL(tool, view)

	// Never cached: this response contains a password, and a shared cache
	// holding it would outlive the session that was allowed to see it.
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, view)
}

func isPublic(tool catalog.Tool, port int) bool {
	for _, candidate := range tool.Ports {
		if candidate.Number == port {
			return candidate.Public
		}
	}
	return false
}

// connectionURL assembles the string someone pastes into a client.
//
// Built with net/url rather than concatenated: it escapes the credential
// correctly and brackets an IPv6 host, both of which produce a URL that fails
// in a confusing way when done by hand.
func connectionURL(tool catalog.Tool, view connectionView) string {
	address := url.URL{
		Scheme: tool.Access.Scheme,
		User:   url.UserPassword(view.Username, view.Password),
		Host:   net.JoinHostPort(view.Host, strconv.Itoa(view.Port)),
	}
	if view.Database != "" {
		address.Path = "/" + view.Database
	}
	return address.String()
}

// tunnelCommand is the ssh invocation that forwards a private port to the
// person's own machine, which is how a loopback-only database is reached.
func tunnelCommand(server store.Server, port int) string {
	command := "ssh -N -L " + strconv.Itoa(port) + ":127.0.0.1:" + strconv.Itoa(port)
	if server.Port != 0 && server.Port != 22 {
		command += " -p " + strconv.Itoa(server.Port)
	}
	return command + " " + server.User + "@" + server.Host
}

// failTool maps the ways a tool request can go wrong onto the catalogue.
func (a *API) failTool(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalog.ErrUnknownTool):
		a.fail(w, apierr.UnknownTool.WithCause(err))
	case errors.Is(err, runner.ErrNoAddress):
		a.fail(w, apierr.ToolNeedsAddress.WithCause(err))
	case errors.Is(err, runner.ErrAlreadyRunning):
		a.fail(w, apierr.ServerBusy.WithCause(err))
	case errors.Is(err, store.ErrNotFound):
		// Reached through a server the caller owns, so a missing row here is a
		// missing installation rather than a missing server.
		a.fail(w, apierr.ToolNotInstalled.WithCause(err))
	default:
		a.failServer(w, err)
	}
}

// --- console ----------------------------------------------------------------

type queryRequest struct {
	Query string `json:"query"`
}

// handleToolQuery runs one query against an installed tool.
//
// The request body is not logged and the response is not cached: a query is
// the owner's own text and the rows it returns are their own data.
func (a *API) handleToolQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	toolID := r.PathValue("tool")

	// The tool has to be installed and finished installing. Querying one that
	// is still starting produces a connection error the person cannot act on.
	installation, err := a.store.GetInstallation(r.Context(), currentUser(r).ID, server.ID, toolID)
	if err != nil {
		a.failTool(w, err)
		return
	}
	if installation.Status != store.InstallReady {
		a.fail(w, apierr.ToolNotReady)
		return
	}

	result, err := a.console.Run(r.Context(), currentUser(r).ID, server.ID, toolID, req.Query)
	switch {
	case errors.Is(err, console.ErrNoConsole):
		a.fail(w, apierr.ToolHasNoConsole.WithCause(err))
		return
	case errors.Is(err, console.ErrEmptyQuery):
		a.fail(w, apierr.EmptyQuery.WithCause(err))
		return
	case err != nil:
		a.failTool(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

// --- saved queries ------------------------------------------------------------

type saveQueryRequest struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

func (a *API) handleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	saved, err := a.store.ListSavedQueries(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	// Saved queries describe someone's schema, so they are no more cacheable
	// than the rows they return.
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, saved)
}

func (a *API) handleSaveQuery(w http.ResponseWriter, r *http.Request) {
	var req saveQueryRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Query = strings.TrimSpace(req.Query)

	if req.Name == "" {
		a.fail(w, apierr.QueryNameRequired)
		return
	}
	if req.Query == "" {
		a.fail(w, apierr.EmptyQuery)
		return
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	toolID := r.PathValue("tool")
	if _, err := catalog.Find(toolID); err != nil {
		a.failTool(w, err)
		return
	}

	saved := store.SavedQuery{
		ID:       ids.New("qry"),
		UserID:   currentUser(r).ID,
		ServerID: server.ID,
		ToolID:   toolID,
		Name:     req.Name,
		Query:    req.Query,
		SavedAt:  time.Now().UTC(),
	}
	if err := a.store.SaveQuery(r.Context(), saved); err != nil {
		a.failServer(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, saved)
}

func (a *API) handleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	err = a.store.DeleteSavedQuery(r.Context(), currentUser(r).ID, server.ID, r.PathValue("query"))
	if err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
