// Package apierr is the single source of truth for everything the API tells a
// person went wrong.
//
// Every failure the browser can see is defined here: its code, its HTTP
// status, what happened in plain language, and what to do next. Handlers pick
// an entry rather than writing a sentence, and the web client renders what it
// is given rather than keeping a second copy of the same wording. One place to
// change the words, one vocabulary on both sides.
package apierr

import (
	"errors"
	"fmt"
	"strings"
)

// Code is the machine-readable half. The web client switches on these, so the
// set is a shared contract — see web/src/lib/errors/app-error.ts.
type Code string

const (
	// CodeInvalid means the request itself was wrong.
	CodeInvalid Code = "invalid"
	// CodeUnauthorized means no valid session.
	CodeUnauthorized Code = "unauthorized"
	// CodeForbidden means signed in, but not allowed.
	CodeForbidden Code = "forbidden"
	// CodeNotFound covers "does not exist" and "not yours" alike — telling them
	// apart would reveal that somebody else's resource exists.
	CodeNotFound Code = "not_found"
	// CodeConflict means the current state does not allow this.
	CodeConflict Code = "conflict"
	// CodeTooLarge means the payload exceeded a limit.
	CodeTooLarge Code = "too_large"
	// CodeUnsupported means we understood it but cannot do it with this input.
	CodeUnsupported Code = "unsupported"
	// CodeUpstream means the managed server misbehaved, not us.
	CodeUpstream Code = "upstream"
	// CodeInternal means our fault.
	CodeInternal Code = "internal"
)

// Error carries everything a response needs. Message says what happened;
// Action says what to do about it. Both are written for someone who does not
// know how any of this works.
type Error struct {
	Code    Code
	Status  int
	Message string
	Action  string

	// cause is kept for logs and never sent to the browser.
	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// WithCause attaches the underlying error for logging without changing what
// the person is told.
func (e *Error) WithCause(cause error) *Error {
	clone := *e
	clone.cause = cause
	return &clone
}

// WithMessage replaces the wording for a case that needs to be more specific
// than the catalogue entry, keeping the code, status and action.
func (e *Error) WithMessage(message string) *Error {
	clone := *e
	clone.Message = message
	return &clone
}

func newError(code Code, status int, message, action string) *Error {
	return &Error{Code: code, Status: status, Message: message, Action: action}
}

// --- the catalogue ----------------------------------------------------------

var (
	// Requests
	BadRequest = newError(CodeInvalid, 400,
		"We could not read that request.",
		"Check the details and try again.")

	NotSignedIn = newError(CodeUnauthorized, 401,
		"You are not signed in.",
		"Sign in to continue.")

	WrongCredentials = newError(CodeUnauthorized, 401,
		"That email and password do not match.",
		"Check both and try again.")

	WeakPassword = newError(CodeInvalid, 400,
		"That password is too short.",
		"Use at least 10 characters. Length matters more than symbols.")

	InvalidEmail = newError(CodeInvalid, 400,
		"That does not look like an email address.",
		"Check it and try again.")

	SetupClosed = newError(CodeConflict, 409,
		"An account already exists on this installation.",
		"Sign in with it, or run `ferrite-ship reset-account` on the server.")

	Internal = newError(CodeInternal, 500,
		"Something went wrong on our side.",
		"Try again in a moment. If it keeps happening, the reference below will help.")

	// Servers
	ServerNotFound = newError(CodeNotFound, 404,
		"We could not find that server.",
		"It may have been removed. Go back to your list of servers.")

	NotFound = newError(CodeNotFound, 404,
		"We could not find that.",
		"It may have been removed.")

	ServerBusy = newError(CodeConflict, 409,
		"Something is already running on this server.",
		"Wait for it to finish, then try again.")

	NeedsRealServer = newError(CodeUnsupported, 400,
		"This is a simulated server, so there is nothing real behind it.",
		"Connect a real server to use this.")

	NameRequired = newError(CodeInvalid, 400,
		"That server needs a name.",
		"Give it something you will recognise later.")

	AddressRequired = newError(CodeInvalid, 400,
		"We need the server's address.",
		"Enter the IP address or hostname you connect to.")

	UsernameRequired = newError(CodeInvalid, 400,
		"We need a username to sign in with.",
		"Enter the account you use to connect, such as root.")

	CredentialRequired = newError(CodeInvalid, 400,
		"We need a way to sign in to that server.",
		"Provide either a password or a private key.")

	InvalidPort = newError(CodeInvalid, 400,
		"That port number is not valid.",
		"Use a number between 1 and 65535. SSH is usually 22.")

	InvalidConnectionKind = newError(CodeInvalid, 400,
		"We do not recognise that kind of connection.",
		`Choose either a simulated server or a real one.`)

	CredentialNotStored = newError(CodeInternal, 500,
		"We could not store those sign-in details safely.",
		"Try again. If it keeps happening, check the server's secret key is set.")

	// Jobs
	UnknownJobKind = newError(CodeInvalid, 400,
		"We do not know how to run that.",
		`The only job available right now is the setup check.`)

	// Files
	PathNotAbsolute = newError(CodeInvalid, 400,
		"That path does not look right.",
		"Paths must start with a slash, like /etc/hosts.")

	FileTooLarge = newError(CodeTooLarge, 413,
		"That file is too big to open here.",
		"Download it instead.")

	FileNotText = newError(CodeUnsupported, 415,
		"That does not look like a text file.",
		"Download it instead — showing it here would be gibberish.")

	FileGone = newError(CodeNotFound, 404,
		"That file or folder is not there any more.",
		"Refresh the folder to see what is there now.")

	FolderNotEmpty = newError(CodeConflict, 409,
		"That folder still has things in it.",
		"Empty it first. We do not delete folders and their contents in one go.")

	NoPermission = newError(CodeForbidden, 403,
		"You do not have permission to do that on this server.",
		"Sign in to the server with an account that can, or use sudo.")

	// Services
	UnknownService = newError(CodeInvalid, 400,
		"That is not a service name we recognise.",
		"Pick one from the list.")

	UnknownServiceAction = newError(CodeInvalid, 400,
		"That is not something we can do to a service.",
		"You can start, stop, restart, turn on or turn off a service.")

	ServiceProtected = newError(CodeConflict, 409,
		"That service is protected.",
		"Stopping it would cut off access to this server. Restarting it is allowed.")

	// Reaching the managed server
	ServerTimedOut = newError(CodeUpstream, 502,
		"That server did not answer in time.",
		"Check it is running and that its firewall allows you in.")

	ServerRefused = newError(CodeUpstream, 502,
		"That server refused the connection.",
		"Check the address and port are right, and that SSH is running.")

	ServerUnknownHost = newError(CodeUpstream, 502,
		"We could not find that address.",
		"Check it is spelled correctly.")

	ServerUnreachable = newError(CodeUpstream, 502,
		"That server cannot be reached from here.",
		"Check your network and the server's firewall.")

	ServerRejectedLogin = newError(CodeUpstream, 502,
		"That server did not accept the sign-in details.",
		"Check the username and the key or password you saved for it.")

	ServerFailed = newError(CodeUpstream, 502,
		"Something went wrong talking to that server.",
		"Try again. If it keeps happening, open a terminal on it and look for yourself.")
)

// From classifies any error into a catalogue entry.
//
// This is the only place infrastructure wording is interpreted. SSH, SFTP and
// systemd all produce terse strings meant for engineers; matching on them once
// here keeps that guesswork out of every handler.
func From(err error) *Error {
	if err == nil {
		return nil
	}

	var known *Error
	if errors.As(err, &known) {
		return known
	}

	message := strings.ToLower(err.Error())

	switch {
	// Connection problems first: they explain everything after them.
	case containsAny(message, "i/o timeout", "connection timed out", "deadline exceeded"):
		return ServerTimedOut.WithCause(err)
	case containsAny(message, "connection refused"):
		return ServerRefused.WithCause(err)
	case containsAny(message, "no such host", "lookup "):
		return ServerUnknownHost.WithCause(err)
	case containsAny(message, "network is unreachable", "no route to host"):
		return ServerUnreachable.WithCause(err)
	case containsAny(message, "unable to authenticate", "handshake failed", "permission denied (publickey"):
		return ServerRejectedLogin.WithCause(err)

	// Filesystem
	case containsAny(message, "permission denied"):
		return NoPermission.WithCause(err)
	case containsAny(message, "does not exist", "no such file", "file not found"):
		return FileGone.WithCause(err)
	case containsAny(message, "directory not empty"):
		return FolderNotEmpty.WithCause(err)

	default:
		return ServerFailed.WithCause(err)
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
