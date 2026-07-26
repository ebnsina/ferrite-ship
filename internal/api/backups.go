package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// destinationView never carries the keys back out.
//
// Write-only by design: the account holder pasted them once, and a response
// that returns them turns every future XSS into a storage credential leak.
type destinationView struct {
	Configured bool   `json:"configured"`
	Endpoint   string `json:"endpoint,omitempty"`
	Region     string `json:"region,omitempty"`
	Bucket     string `json:"bucket,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

func (a *API) handleGetBackupDestination(w http.ResponseWriter, r *http.Request) {
	stored, err := a.store.GetBackupDestination(r.Context(), currentUser(r).ID)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, destinationView{Configured: false})
		return
	}
	if err != nil {
		a.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, destinationView{
		Configured: true,
		Endpoint:   stored.Endpoint,
		Region:     stored.Region,
		Bucket:     stored.Bucket,
		Prefix:     stored.Prefix,
		UpdatedAt:  stored.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

type destinationRequest struct {
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

func (a *API) handleSaveBackupDestination(w http.ResponseWriter, r *http.Request) {
	var req destinationRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	req.Endpoint = strings.TrimSpace(req.Endpoint)
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.Prefix = strings.Trim(strings.TrimSpace(req.Prefix), "/")

	switch {
	case req.Endpoint == "":
		a.fail(w, apierr.StorageEndpointRequired)
		return
	case req.Bucket == "":
		a.fail(w, apierr.StorageBucketRequired)
		return
	case req.AccessKey == "" || req.SecretKey == "":
		a.fail(w, apierr.StorageKeysRequired)
		return
	}

	accessKey, err := a.sealer.Seal(req.AccessKey)
	if err != nil {
		a.fail(w, apierr.CredentialNotStored.WithCause(err))
		return
	}
	secretKey, err := a.sealer.Seal(req.SecretKey)
	if err != nil {
		a.fail(w, apierr.CredentialNotStored.WithCause(err))
		return
	}

	destination := store.BackupDestination{
		UserID:          currentUser(r).ID,
		Endpoint:        req.Endpoint,
		Region:          strings.TrimSpace(req.Region),
		Bucket:          req.Bucket,
		Prefix:          req.Prefix,
		SealedAccessKey: accessKey,
		SealedSecretKey: secretKey,
		UpdatedAt:       time.Now().UTC(),
	}
	if err := a.store.SaveBackupDestination(r.Context(), destination); err != nil {
		a.fail(w, err)
		return
	}

	a.handleGetBackupDestination(w, r)
}

func (a *API) handleDeleteBackupDestination(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteBackupDestination(r.Context(), currentUser(r).ID); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- backups ------------------------------------------------------------------

func (a *API) handleListBackups(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	backups, err := a.store.ListBackups(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"), 50)
	if err != nil {
		a.failServer(w, err)
		return
	}
	writeJSON(w, http.StatusOK, backups)
}

func (a *API) handleStartBackup(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	job, err := a.runner.StartBackup(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"), actorOf(r))
	if err != nil {
		a.failBackup(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (a *API) handleStartRestore(w http.ResponseWriter, r *http.Request) {
	job, err := a.runner.StartRestore(
		r.Context(), currentUser(r).ID, r.PathValue("backup"), actorOf(r))
	if err != nil {
		a.failBackup(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func actorOf(r *http.Request) string {
	if actor := r.URL.Query().Get("actor"); actor != "" {
		return actor
	}
	return "You"
}

// failBackup maps backup-specific problems onto the catalogue.
func (a *API) failBackup(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runner.ErrNoDestination):
		a.fail(w, apierr.NoBackupDestination.WithCause(err))
	case errors.Is(err, runner.ErrNoBackupSupport):
		a.fail(w, apierr.BackupNotSupported.WithCause(err))
	default:
		a.failTool(w, err)
	}
}
