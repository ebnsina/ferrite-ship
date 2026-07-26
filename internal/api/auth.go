package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/auth"
)

const sessionCookie = "ferrite_session"

// setSessionCookie writes the session cookie.
//
// HttpOnly keeps it away from scripts, so an XSS bug cannot read it. SameSite
// Lax stops another site making authenticated requests on your behalf while
// still allowing ordinary navigation. Secure is set only over HTTPS, because a
// Secure cookie on plain http would simply never be stored — and this runs on
// loopback during development.
func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  expires,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
}

func sessionID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// requireSession wraps the routes that touch a server. Everything behind it
// can assume there is an account, so no handler has to remember to check.
func (a *API) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := a.auth.UserForSession(r.Context(), sessionID(r)); err != nil {
			a.writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type authStatusResponse struct {
	NeedsSetup    bool   `json:"needsSetup"`
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email,omitempty"`
}

func (a *API) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	needsSetup, err := a.auth.NeedsSetup(r.Context())
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "server", "Something went wrong on our side.")
		return
	}

	response := authStatusResponse{NeedsSetup: needsSetup}
	if user, err := a.auth.UserForSession(r.Context(), sessionID(r)); err == nil {
		response.Authenticated = true
		response.Email = user.Email
	}

	writeJSON(w, http.StatusOK, response)
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "parse", "We could not read that request.")
		return
	}

	user, err := a.auth.Setup(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, auth.ErrSetupClosed):
		a.writeError(w, http.StatusConflict, "conflict",
			"An account already exists on this installation.")
		return
	case errors.Is(err, auth.ErrWeakPassword):
		a.writeError(w, http.StatusBadRequest, "parse",
			"Choose a password of at least 10 characters. Length matters more than symbols.")
		return
	case err != nil:
		a.writeError(w, http.StatusBadRequest, "parse", err.Error())
		return
	}

	// Sign the new account straight in; making someone type it twice in a row
	// serves nobody.
	session, err := a.auth.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "server", "Account created, but signing in failed.")
		return
	}

	setSessionCookie(w, r, session.ID, session.ExpiresAt)
	a.log.Info("first account created", "email", user.Email)
	writeJSON(w, http.StatusCreated, authStatusResponse{Authenticated: true, Email: user.Email})
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		a.writeError(w, http.StatusBadRequest, "parse", "We could not read that request.")
		return
	}

	session, err := a.auth.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		// One message for both a wrong password and an unknown account, so the
		// response cannot be used to discover which emails exist.
		a.writeError(w, http.StatusUnauthorized, "unauthorized",
			"That email and password do not match.")
		return
	}

	setSessionCookie(w, r, session.ID, session.ExpiresAt)
	writeJSON(w, http.StatusOK, authStatusResponse{
		Authenticated: true,
		Email:         strings.ToLower(strings.TrimSpace(req.Email)),
	})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.SignOut(r.Context(), sessionID(r)); err != nil {
		a.log.Warn("could not delete session", "error", err)
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}
