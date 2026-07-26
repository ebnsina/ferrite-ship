package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/files"
)

// failFiles maps the file service's sentinels onto catalogue entries.
// Everything else — SSH timeouts, SFTP's terse wording — is classified by
// apierr.From, so the interpreting happens in one place rather than here.
func (a *API) failFiles(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrNotSupported):
		a.fail(w, apierr.NeedsRealServer)
	case errors.Is(err, files.ErrBadPath):
		a.fail(w, apierr.PathNotAbsolute)
	case errors.Is(err, files.ErrTooLarge):
		a.fail(w, apierr.FileTooLarge)
	case errors.Is(err, files.ErrNotText):
		a.fail(w, apierr.FileNotText)
	default:
		a.failServer(w, err)
	}
}

func (a *API) handleListFiles(w http.ResponseWriter, r *http.Request) {
	listing, err := a.files.List(r.Context(), currentUser(r).ID, r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		a.failFiles(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (a *API) handleReadFile(w http.ResponseWriter, r *http.Request) {
	content, err := a.files.Read(r.Context(), currentUser(r).ID, r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		a.failFiles(w, err)
		return
	}
	writeJSON(w, http.StatusOK, content)
}

type writeFileRequest struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

func (a *API) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	var req writeFileRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	if err := a.files.Write(r.Context(), currentUser(r).ID, r.PathValue("id"), req.Path, req.Text); err != nil {
		a.failFiles(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	// Headers must go out before the body, and the filename is only known once
	// the transfer starts, so derive it from the request rather than the file.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", baseName(path)))

	if _, err := a.files.Download(r.Context(), currentUser(r).ID, r.PathValue("id"), path, w); err != nil {
		// The status line may already be sent; log rather than pretend.
		a.log.Warn("file download failed", "path", path, "error", err)
	}
}

func (a *API) handleRemoveFile(w http.ResponseWriter, r *http.Request) {
	if err := a.files.Remove(r.Context(), currentUser(r).ID, r.PathValue("id"), r.URL.Query().Get("path")); err != nil {
		a.failFiles(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	if p == "" {
		return "download"
	}
	return p
}
