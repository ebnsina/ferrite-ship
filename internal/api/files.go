package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ebnsina/ferrite-ship/internal/files"
)

// writeFilesError maps filesystem failures onto messages a person can act on.
// SFTP errors are terse ("permission denied", "file does not exist") and worth
// translating rather than passing through raw.
func (a *API) writeFilesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrNotSupported):
		a.writeError(w, http.StatusBadRequest, "parse",
			"This is a simulated server, so there are no files to browse. Connect a real server to use this.")
	case errors.Is(err, files.ErrBadPath):
		a.writeError(w, http.StatusBadRequest, "parse", "That path does not look right.")
	case errors.Is(err, files.ErrTooLarge):
		a.writeError(w, http.StatusRequestEntityTooLarge, "parse",
			"That file is too big to open here. Download it instead.")
	case errors.Is(err, files.ErrNotText):
		a.writeError(w, http.StatusUnsupportedMediaType, "parse",
			"That does not look like a text file, so there is nothing sensible to show. Download it instead.")
	default:
		a.writeError(w, http.StatusBadGateway, "network", friendlyFileError(err))
	}
}

func friendlyFileError(err error) string {
	message := err.Error()
	switch {
	case containsAny(message, "permission denied"):
		return "You do not have permission to do that on this server."
	case containsAny(message, "does not exist", "no such file"):
		return "That file or folder is not there any more."
	case containsAny(message, "directory not empty"):
		return "That folder still has things in it. Empty it first."
	default:
		return "Could not reach the server's files: " + message
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && len(haystack) >= len(needle) &&
			indexFold(haystack, needle) >= 0 {
			return true
		}
	}
	return false
}

func (a *API) handleListFiles(w http.ResponseWriter, r *http.Request) {
	listing, err := a.files.List(r.Context(), r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		a.writeFilesError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (a *API) handleReadFile(w http.ResponseWriter, r *http.Request) {
	content, err := a.files.Read(r.Context(), r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		a.writeFilesError(w, err)
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
		a.writeError(w, http.StatusBadRequest, "parse", "We could not read that request.")
		return
	}

	if err := a.files.Write(r.Context(), r.PathValue("id"), req.Path, req.Text); err != nil {
		a.writeFilesError(w, err)
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

	if _, err := a.files.Download(r.Context(), r.PathValue("id"), path, w); err != nil {
		// The status line may already be sent; log rather than pretend.
		a.log.Warn("file download failed", "path", path, "error", err)
	}
}

func (a *API) handleRemoveFile(w http.ResponseWriter, r *http.Request) {
	if err := a.files.Remove(r.Context(), r.PathValue("id"), r.URL.Query().Get("path")); err != nil {
		a.writeFilesError(w, err)
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

// indexFold is a case-insensitive substring search, kept local to avoid
// pulling strings into this file for one call.
func indexFold(haystack, needle string) int {
	h, n := toLowerASCII(haystack), toLowerASCII(needle)
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

func toLowerASCII(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + 32
		}
	}
	return string(out)
}
