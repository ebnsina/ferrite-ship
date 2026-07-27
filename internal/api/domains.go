package api

import (
	"net/http"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
)

type domainRequest struct {
	Domain string `json:"domain"`
	Email  string `json:"email"`
}

// handleSaveDomain records the name whose wildcard record points at a server.
//
// Nothing is verified here. Checking that the DNS record exists before
// accepting it sounds careful and is the wrong order: the record is usually
// added *after* being told which server to point at, so a check would reject
// every first attempt and teach people the feature is broken. Traefik finds
// out when it asks for a certificate, and that failure is the one worth
// reporting because it is the one that is real.
func (a *API) handleSaveDomain(w http.ResponseWriter, r *http.Request) {
	var req domainRequest
	if err := decodeJSON(r, &req); err != nil {
		a.fail(w, apierr.BadRequest.WithCause(err))
		return
	}

	domain, err := cleanDomain(req.Domain)
	if err != nil {
		a.fail(w, apierr.InvalidDomain.WithCause(err))
		return
	}

	email := strings.TrimSpace(req.Email)
	// Clearing the domain clears the address with it, so an empty pair is
	// allowed and means "route nothing here".
	if domain != "" {
		if email == "" {
			a.fail(w, apierr.DomainNeedsEmail)
			return
		}
		if !strings.Contains(email, "@") {
			a.fail(w, apierr.InvalidNotificationEmail)
			return
		}
	} else {
		email = ""
	}

	server, err := a.store.GetServer(r.Context(), currentUser(r).ID, r.PathValue("id"))
	if err != nil {
		a.failServer(w, err)
		return
	}

	if err := a.store.SetServerDomain(
		r.Context(), currentUser(r).ID, server.ID, domain, email); err != nil {
		a.failServer(w, err)
		return
	}

	setUp, err := a.store.LastSetupByServer(r.Context(), currentUser(r).ID)
	if err != nil {
		a.failServer(w, err)
		return
	}

	server.Domain = domain
	server.ACMEEmail = email
	writeJSON(w, http.StatusOK, toServerView(server, setUp[server.ID]))
}

// cleanDomain reduces what someone pasted to a bare domain name.
//
// People paste what they see in the address bar, so a scheme and a trailing
// slash are the normal input rather than a mistake, and rejecting them would
// be pedantry. A wildcard prefix is stripped for the same reason: "*.x.com" is
// exactly what they were told to create in DNS, so it is the obvious thing to
// type here.
func cleanDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	if domain == "" {
		return "", nil
	}

	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	domain = strings.TrimSuffix(domain, "/")
	domain = strings.TrimPrefix(domain, "*.")
	domain = strings.Trim(domain, ".")

	// A path or a port means this is a URL, not a domain, and guessing which
	// part was meant would be inventing an answer.
	if strings.ContainsAny(domain, "/:? ") {
		return "", errDomainNotBare
	}
	// Every name a certificate can be issued for has at least one dot.
	// "localhost" and a bare hostname are the common near-misses, and both
	// would fail at Let's Encrypt instead, much later.
	if !strings.Contains(domain, ".") {
		return "", errDomainNotBare
	}

	for _, label := range strings.Split(domain, ".") {
		if label == "" {
			return "", errDomainNotBare
		}
		for _, char := range label {
			isLetter := char >= 'a' && char <= 'z'
			isDigit := char >= '0' && char <= '9'
			if !isLetter && !isDigit && char != '-' {
				return "", errDomainNotBare
			}
		}
	}

	return domain, nil
}

type domainError string

func (e domainError) Error() string { return string(e) }

const errDomainNotBare = domainError("a domain is a bare name such as example.com")
