package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/ids"
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

// noBackupFor picks which of the two "no backup here" answers is true.
//
// Whether the tool stores anything is the difference. A media server keeping
// nothing is finished business; a search index we cannot copy yet is an
// admission, and dressing it up as the former would tell someone their data
// does not matter.
func noBackupFor(tool catalog.Tool) *apierr.Error {
	if tool.KeepsData {
		return apierr.BackupNotSupported
	}
	return apierr.BackupNotNeeded
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

// --- schedules ----------------------------------------------------------------

type scheduleRequest struct {
	Cadence string `json:"cadence"`
	Hour    int    `json:"hour"`
	Weekday int    `json:"weekday"`
	Keep    int    `json:"keep"`
}

func (a *API) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	schedule, err := a.store.GetBackupSchedule(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"))
	if errors.Is(err, store.ErrNotFound) {
		// Not an error: most tools have no schedule, and the UI needs to say
		// "off" rather than show a failure.
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		a.failServer(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schedule)
}

func (a *API) handleSaveSchedule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	cadence := store.Cadence(strings.TrimSpace(req.Cadence))
	if cadence != store.Daily && cadence != store.Weekly {
		a.fail(w, apierr.InvalidCadence)
		return
	}
	if req.Hour < 0 || req.Hour > 23 {
		a.fail(w, apierr.InvalidCadence)
		return
	}
	if req.Weekday < 0 || req.Weekday > 6 {
		a.fail(w, apierr.InvalidCadence)
		return
	}
	// At least one, or the first successful backup would delete itself.
	if req.Keep < 1 {
		req.Keep = 1
	}

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
	if !tool.Supported() {
		a.fail(w, noBackupFor(tool))
		return
	}

	// Refuse a schedule with nowhere to send the result, rather than letting
	// it fail quietly at three in the morning.
	if _, err := a.store.GetBackupDestination(r.Context(), currentUser(r).ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			a.fail(w, apierr.NoBackupDestination)
			return
		}
		a.fail(w, err)
		return
	}

	schedule := store.BackupSchedule{
		ID:       ids.New("sch"),
		UserID:   currentUser(r).ID,
		ServerID: server.ID,
		ToolID:   toolID,
		Cadence:  cadence,
		Hour:     req.Hour,
		Weekday:  req.Weekday,
		Keep:     req.Keep,
	}
	schedule.NextRunAt = schedule.NextRun(time.Now().UTC())

	if err := a.store.SaveBackupSchedule(r.Context(), schedule); err != nil {
		a.failServer(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schedule)
}

func (a *API) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	err = a.store.DeleteBackupSchedule(
		r.Context(), currentUser(r).ID, server.ID, r.PathValue("tool"))
	if err != nil {
		a.failServer(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
