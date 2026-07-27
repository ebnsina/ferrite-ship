// Package github talks to GitHub as an installed app.
//
// A GitHub App rather than an OAuth app, which is the same choice Vercel and
// Render made and for the same reasons. An OAuth app receives a long-lived
// token carrying every repository the person can see, including their
// employer's; a GitHub App is installed on repositories they choose, hands
// back tokens that expire in an hour, and can be revoked from GitHub's own
// settings page without asking us. It also receives webhooks, which is the
// only way deploy-on-push can exist at all.
//
// Nothing here is stored. Installation tokens are minted on demand and kept in
// memory until they expire — writing an hour-long credential to disk to save a
// round trip would be trading the whole point of short-lived tokens for
// nothing.
package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// api is GitHub's REST endpoint, overridable so tests never reach the network.
const api = "https://api.github.com"

var ErrNotConfigured = errors.New("github: no app is configured")

// App is a configured GitHub App.
type App struct {
	id   string
	slug string
	key  *rsa.PrivateKey

	baseURL string
	client  *http.Client

	// tokens caches installation tokens until shortly before they expire.
	// Keyed by installation, because one control plane serves many accounts
	// and each has its own.
	mu     sync.Mutex
	tokens map[int64]cached
}

type cached struct {
	token     string
	expiresAt time.Time
}

// New builds an App from a PEM private key.
//
// The key is GitHub's own download, unchanged. It arrives as PKCS#1 from the
// app settings page but PKCS#8 from some tooling, and telling somebody their
// valid key is invalid because of a header they never chose would be a poor
// welcome — so both are accepted.
func New(id, slug string, pemKey []byte) (*App, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("github: the private key is not PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, pkcs8Err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if pkcs8Err != nil {
			return nil, fmt.Errorf("github: the private key could not be read: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("github: the private key is not RSA")
		}
		key = rsaKey
	}

	return &App{
		id:      id,
		slug:    slug,
		key:     key,
		baseURL: api,
		client:  &http.Client{Timeout: 15 * time.Second},
		tokens:  map[int64]cached{},
	}, nil
}

// InstallURL is where somebody is sent to choose which repositories we may see.
//
// The state is checked when GitHub sends them back, so that a link somebody
// else made cannot attach their installation to this account.
func (a *App) InstallURL(state string) string {
	return "https://github.com/apps/" + a.slug + "/installations/new?state=" + state
}

// jwt signs a short-lived assertion proving we are this app.
//
// Hand-rolled rather than pulling in a JWT library: this is one algorithm with
// two fixed claims, and a dependency that parses every algorithm — including
// the "none" one that has caused real vulnerabilities — is a larger surface
// than the thirty lines it saves.
func (a *App) jwt(now time.Time) (string, error) {
	// Backdated a minute. The clock here and GitHub's are not the same clock,
	// and a token issued "in the future" is rejected outright.
	claims := map[string]any{
		"iat": now.Add(-time.Minute).Unix(),
		// GitHub refuses anything more than ten minutes out, so nine leaves
		// room for the skew above without going over.
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": a.id,
	}

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signing := encode(header) + "." + encode(body)
	digest := sha256Sum(signing)

	signature, err := rsa.SignPKCS1v15(rand.Reader, a.key, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("github: could not sign: %w", err)
	}
	return signing + "." + encode(signature), nil
}

// InstallationToken returns a token that can read the repositories this
// installation covers, minting a new one when the cached one is close to
// expiring.
//
// A minute of margin, because the token is not used here — it is handed to a
// git clone running on somebody's server, and one that expires between being
// issued and being used fails as "repository not found", which reads like the
// repository was deleted.
func (a *App) InstallationToken(ctx context.Context, installation int64) (string, error) {
	a.mu.Lock()
	if held, ok := a.tokens[installation]; ok && time.Now().Before(held.expiresAt.Add(-time.Minute)) {
		a.mu.Unlock()
		return held.token, nil
	}
	a.mu.Unlock()

	assertion, err := a.jwt(time.Now())
	if err != nil {
		return "", err
	}

	url := a.baseURL + "/app/installations/" + strconv.FormatInt(installation, 10) + "/access_tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+assertion)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github: could not reach GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		// 404 here means the installation is gone — somebody uninstalled the
		// app from GitHub's side, which we only find out by asking.
		return "", fmt.Errorf("github: GitHub refused the token request: %s", resp.Status)
	}

	var reply struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return "", fmt.Errorf("github: could not read the token: %w", err)
	}
	if reply.Token == "" {
		return "", errors.New("github: GitHub returned an empty token")
	}

	a.mu.Lock()
	a.tokens[installation] = cached{token: reply.Token, expiresAt: reply.ExpiresAt}
	a.mu.Unlock()

	return reply.Token, nil
}

// CloneURL is the https address a server can clone with, carrying the token.
//
// x-access-token is the username GitHub expects for an installation token. The
// token is in the URL, so this value never reaches a log: steps.Session
// redaction is given it before any command that uses it runs.
func CloneURL(token, owner, repo string) string {
	return "https://x-access-token:" + token + "@github.com/" + owner + "/" + repo + ".git"
}

func encode(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sha256Sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
