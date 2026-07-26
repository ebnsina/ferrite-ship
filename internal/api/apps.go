package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

type appRequest struct {
	Name       string            `json:"name"`
	Repository string            `json:"repository"`
	Branch     string            `json:"branch"`
	Domain     string            `json:"domain"`
	Port       int               `json:"port"`
	Env        map[string]string `json:"env"`
	// DeployKey is a private SSH key for a repository that is not public.
	// Write-only: it is sealed on arrival and never sent back.
	DeployKey string `json:"deployKey"`
}

func (a *API) handleListApps(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	apps, err := a.store.ListApps(r.Context(), currentUser(r).ID, server.ID)
	if err != nil {
		a.failServer(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (a *API) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var req appRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	app, err := a.appFrom(req, store.App{
		ID:        ids.New("app"),
		UserID:    currentUser(r).ID,
		ServerID:  server.ID,
		Status:    store.AppNew,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		a.fail(w, err)
		return
	}

	if err := a.store.CreateApp(r.Context(), app); err != nil {
		a.failServer(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (a *API) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	var req appRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	existing, err := a.store.GetApp(r.Context(), currentUser(r).ID, r.PathValue("app"))
	if err != nil {
		a.fail(w, err)
		return
	}

	app, err := a.appFrom(req, existing)
	if err != nil {
		a.fail(w, err)
		return
	}

	if err := a.store.UpdateApp(r.Context(), app); err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

// appFrom validates a request onto an app, keeping the fields the request does
// not own.
func (a *API) appFrom(req appRequest, base store.App) (store.App, error) {
	base.Name = strings.TrimSpace(req.Name)
	base.Repository = strings.TrimSpace(req.Repository)
	base.Branch = strings.TrimSpace(req.Branch)
	base.Domain = strings.TrimSpace(strings.ToLower(req.Domain))
	base.Port = req.Port

	if base.Name == "" {
		return store.App{}, apierr.NameRequired
	}
	if base.Repository == "" {
		return store.App{}, apierr.RepositoryRequired
	}
	// A URL rather than a local path, and http(s) or ssh rather than anything
	// the shell might find interesting.
	if !strings.HasPrefix(base.Repository, "https://") &&
		!strings.HasPrefix(base.Repository, "http://") &&
		!strings.HasPrefix(base.Repository, "git@") {
		return store.App{}, apierr.RepositoryNotSupported
	}
	if base.Branch == "" {
		base.Branch = "main"
	}
	if base.Port == 0 {
		base.Port = 3000
	}
	if base.Port < 1 || base.Port > 65535 {
		return store.App{}, apierr.InvalidPort
	}

	// The whole environment is sealed as one blob: it is nearly always where a
	// database password ends up, and guessing which keys are sensitive is a
	// worse bet than treating all of them as such.
	if req.Env == nil {
		req.Env = map[string]string{}
	}
	encoded, err := json.Marshal(req.Env)
	if err != nil {
		return store.App{}, apierr.BadRequest.WithCause(err)
	}
	sealed, err := a.sealer.Seal(string(encoded))
	if err != nil {
		return store.App{}, apierr.CredentialNotStored.WithCause(err)
	}
	base.SealedEnv = sealed

	// Only set when supplied. base already carries the stored key on an update
	// and nothing on a create, so leaving it alone gives both the behaviour
	// they want: changing a port does not mean pasting the key again.
	if key := strings.TrimSpace(req.DeployKey); key != "" {
		sealedKey, err := a.sealer.Seal(key)
		if err != nil {
			return store.App{}, apierr.CredentialNotStored.WithCause(err)
		}
		base.SealedDeployKey = sealedKey
	}

	// An ssh:// repository cannot be read without one, and finding that out as
	// a git failure three steps into a deploy is a poor way to learn it.
	if strings.HasPrefix(base.Repository, "git@") && base.SealedDeployKey == "" {
		return store.App{}, apierr.DeployKeyRequired
	}

	// Derived on read by the store, but this value is about to be written
	// straight back as the response — without it, saving a key and being told
	// none is stored is the first thing anyone would see.
	base.HasDeployKey = base.SealedDeployKey != ""

	return base, nil
}

func (a *API) handleDeployApp(w http.ResponseWriter, r *http.Request) {
	job, err := a.runner.StartDeploy(r.Context(), currentUser(r).ID, r.PathValue("app"), actorOf(r))
	if err != nil {
		a.failTool(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (a *API) handleRemoveApp(w http.ResponseWriter, r *http.Request) {
	job, err := a.runner.StartUndeploy(r.Context(), currentUser(r).ID, r.PathValue("app"), actorOf(r))
	if err != nil {
		a.failTool(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
