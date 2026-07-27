package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// stateLifetime is how long the link to GitHub stays usable.
//
// Long enough to read the permissions page properly and short enough that a
// link left in a browser's history cannot be followed a week later.
const stateLifetime = 15 * time.Minute

type githubStatusView struct {
	// Configured is whether this installation has an app at all. Separate from
	// Connected on purpose: "we cannot do this" and "you have not done this
	// yet" need different sentences, and one flag would produce a button that
	// leads nowhere.
	Configured    bool                       `json:"configured"`
	Installations []store.GitHubInstallation `json:"installations"`
}

func (a *API) handleGitHubStatus(w http.ResponseWriter, r *http.Request) {
	view := githubStatusView{
		Configured:    a.github != nil,
		Installations: []store.GitHubInstallation{},
	}

	if view.Configured {
		installations, err := a.store.ListGitHubInstallations(r.Context(), currentUser(r).ID)
		if err != nil {
			a.failServer(w, err)
			return
		}
		view.Installations = installations
	}

	writeJSON(w, http.StatusOK, view)
}

// handleGitHubConnect hands back the link that starts the install.
//
// A link rather than a redirect, because the dashboard is a single page served
// from somewhere else: it opens this itself, so the browser leaves for GitHub
// from a page the person is looking at rather than from a fetch.
func (a *API) handleGitHubConnect(w http.ResponseWriter, r *http.Request) {
	if a.github == nil {
		a.fail(w, apierr.GitHubNotConfigured)
		return
	}

	state, err := a.signState(currentUser(r).ID)
	if err != nil {
		a.fail(w, apierr.Internal.WithCause(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": a.github.InstallURL(state)})
}

// handleGitHubCallback is where GitHub sends somebody back to.
//
// A browser navigation rather than an API call, so it answers with a redirect
// into the dashboard rather than with JSON — landing on a page of JSON after
// installing something is the kind of thing that makes people think it failed.
func (a *API) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if a.github == nil {
		a.redirectToSettings(w, r, "github=unavailable")
		return
	}

	// The state proves this callback belongs to the person who started it. A
	// link somebody else made would otherwise attach their installation — and
	// therefore their repositories — to whichever account happened to be
	// signed in here.
	userID, err := a.openState(r.URL.Query().Get("state"))
	if err != nil {
		a.log.Warn("github callback with a bad state", "error", err)
		a.redirectToSettings(w, r, "github=expired")
		return
	}
	if userID != currentUser(r).ID {
		a.log.Warn("github callback for a different account")
		a.redirectToSettings(w, r, "github=mismatch")
		return
	}

	installation, err := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	if err != nil {
		a.redirectToSettings(w, r, "github=incomplete")
		return
	}

	// Ask GitHub who this belongs to. A failure here is not fatal: the
	// connection works without a label, and refusing it because a name could
	// not be fetched would throw away the part that matters.
	account, err := a.github.Installation(r.Context(), installation)
	if err != nil {
		a.log.Warn("could not describe the github installation", "error", err)
	}

	err = a.store.SaveGitHubInstallation(r.Context(), store.GitHubInstallation{
		ID:      installation,
		UserID:  userID,
		Account: account.Login,
		// Both empty where the lookup failed, which reads as "not known yet"
		// rather than as a wrong answer.
		Selection: account.Repositories,
	})
	if err != nil {
		a.log.Error("could not save the github installation", "error", err)
		a.redirectToSettings(w, r, "github=failed")
		return
	}

	a.redirectToSettings(w, r, "github=connected")
}

func (a *API) handleGitHubDisconnect(w http.ResponseWriter, r *http.Request) {
	installation, err := strconv.ParseInt(r.PathValue("installation"), 10, 64)
	if err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	err = a.store.DeleteGitHubInstallation(r.Context(), currentUser(r).ID, installation)
	if errors.Is(err, store.ErrNotFound) {
		a.fail(w, apierr.NotFound)
		return
	}
	if err != nil {
		a.failServer(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// signState seals who started an install, and when.
//
// Sealed rather than stored in a table: the state is single-use and expires in
// minutes, so a table would exist only to be cleaned up. The same key that
// protects SSH credentials protects this, which means a forged state needs the
// key that would already be enough to read everything else.
func (a *API) signState(userID string) (string, error) {
	deadline := time.Now().Add(stateLifetime).UTC().Format(time.RFC3339)
	return a.sealer.Seal(userID + "|" + deadline)
}

func (a *API) openState(state string) (string, error) {
	if state == "" {
		return "", errors.New("no state")
	}

	opened, err := a.sealer.Open(state)
	if err != nil {
		return "", err
	}

	userID, deadline, found := strings.Cut(opened, "|")
	if !found {
		return "", errors.New("malformed state")
	}
	expiry, err := time.Parse(time.RFC3339, deadline)
	if err != nil {
		return "", err
	}
	if time.Now().After(expiry) {
		return "", errors.New("the link has expired")
	}
	return userID, nil
}

// redirectToSettings sends somebody back into the dashboard with a word about
// how it went.
//
// Where the dashboard is is not something this process always knows — it is
// often served by something else entirely — so with no public URL configured
// this says so plainly rather than redirecting somewhere that does not exist.
func (a *API) redirectToSettings(w http.ResponseWriter, r *http.Request, outcome string) {
	if a.publicURL == "" {
		writeJSON(w, http.StatusOK, map[string]string{
			"outcome": outcome,
			"message": "GitHub is connected. Go back to Settings in the dashboard.",
		})
		return
	}
	http.Redirect(w, r, a.publicURL+"/dashboard/settings?"+outcome, http.StatusSeeOther)
}
