// Package config reads the process environment.
//
// Every value is required and validated here. There are no defaults for
// anything that decides where data goes or how it is protected: a missing
// variable stops the process at boot rather than producing a server that runs
// against the wrong database or with an unprotected credential store.
package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/secret"
)

type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DatabasePath is the SQLite file.
	DatabasePath string
	// SecretKey is base64 of 32 bytes, used to seal stored credentials.
	SecretKey string
	// AllowedOrigin enables CORS for the dev frontend, or is empty when the
	// operator set the literal "none".
	AllowedOrigin string
	// WebDir is the built dashboard to serve, or empty when the operator set
	// the literal "none" to run API-only.
	WebDir string
	// PublicURL is where this dashboard is reachable, used to put a link in an
	// alert email, or empty when the operator set the literal "none".
	PublicURL string
	// SMTP is where outgoing mail goes, or the zero value when the operator
	// set the literal "none". With no mail server nothing is sent, and the
	// settings page says so rather than pretending an address was saved.
	SMTP SMTP
	// ACMEDirectory is which Let's Encrypt endpoint issues certificates.
	// Chosen explicitly rather than defaulted: production rate limits
	// duplicates to five per week, so a new setup that gets it wrong twice is
	// locked out of the fix for a week.
	ACMEDirectory string
	// GitHub is the app used to read private repositories, or the zero value
	// when the operator set the literal "none". Without it the manual path —
	// a git URL and a deploy key — is the only way to deploy, which is how
	// every existing installation already works.
	GitHub GitHub
}

// GitHub is a registered GitHub App this control plane acts as.
type GitHub struct {
	// AppID is the numeric id from the app's settings page, used as the JWT
	// issuer.
	AppID string
	// Slug is the name in the app's public URL, which is where somebody is
	// sent to install it. Not derivable from the id, so it is asked for.
	Slug string
	// PrivateKey is the PEM GitHub generated, decoded from base64. Base64
	// because a PEM has newlines in it and an environment variable carrying
	// them is a thing people get wrong once each.
	PrivateKey []byte
	// WebhookSecret verifies that a delivery came from GitHub. Required
	// alongside the rest rather than when webhooks are first switched on: an
	// unverified webhook endpoint accepts a deploy request from anybody, and
	// discovering that later means it was open in between.
	WebhookSecret string
}

// Enabled reports whether repositories can be reached through GitHub.
func (g GitHub) Enabled() bool { return g.AppID != "" }

// SMTP is a mail server to hand messages to.
type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	// From is the address messages are sent as. Mail servers reject a From
	// they do not recognise, so this is asked for rather than guessed.
	From string
	// Implicit is TLS from the first byte (port 465). Otherwise the connection
	// starts in the clear and is upgraded with STARTTLS, which is required —
	// a password is being sent.
	Implicit bool
}

// Enabled reports whether mail can be sent at all.
func (s SMTP) Enabled() bool { return s.Host != "" }

// Disabled is the value a variable must carry to switch its feature off.
//
// Turning something off has to be written down. An unset variable is always a
// mistake here — never a quiet "no".
const Disabled = "none"

type Error struct {
	Variable string
	Reason   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("configuration error: %s %s (see .env.example)", e.Variable, e.Reason)
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          os.Getenv("FERRITE_ADDR"),
		DatabasePath:  os.Getenv("FERRITE_DATABASE_PATH"),
		SecretKey:     os.Getenv("FERRITE_SECRET_KEY"),
		AllowedOrigin: os.Getenv("FERRITE_ALLOWED_ORIGIN"),
		WebDir:        os.Getenv("FERRITE_WEB_DIR"),
	}

	if err := requireNonEmpty("FERRITE_ADDR", cfg.Addr); err != nil {
		return Config{}, err
	}
	if err := validateAddr(cfg.Addr); err != nil {
		return Config{}, err
	}
	if err := requireNonEmpty("FERRITE_DATABASE_PATH", cfg.DatabasePath); err != nil {
		return Config{}, err
	}
	if err := requireNonEmpty("FERRITE_SECRET_KEY", cfg.SecretKey); err != nil {
		return Config{}, err
	}
	if err := validateSecretKey(cfg.SecretKey); err != nil {
		return Config{}, err
	}

	origin, err := requireOriginOrDisabled("FERRITE_ALLOWED_ORIGIN", cfg.AllowedOrigin)
	if err != nil {
		return Config{}, err
	}
	cfg.AllowedOrigin = origin

	webDir, err := requireDirOrDisabled("FERRITE_WEB_DIR", cfg.WebDir)
	if err != nil {
		return Config{}, err
	}
	cfg.WebDir = webDir

	publicURL, err := requireOriginOrDisabled("FERRITE_PUBLIC_URL", os.Getenv("FERRITE_PUBLIC_URL"))
	if err != nil {
		return Config{}, err
	}
	cfg.PublicURL = publicURL

	smtp, err := requireSMTPOrDisabled(os.Getenv("FERRITE_SMTP_URL"), os.Getenv("FERRITE_MAIL_FROM"))
	if err != nil {
		return Config{}, err
	}
	cfg.SMTP = smtp

	acme, err := requireACMEEndpoint(os.Getenv("FERRITE_ACME_ENDPOINT"))
	if err != nil {
		return Config{}, err
	}
	cfg.ACMEDirectory = acme

	gh, err := requireGitHubOrDisabled()
	if err != nil {
		return Config{}, err
	}
	cfg.GitHub = gh

	return cfg, nil
}

// requireSMTPOrDisabled accepts smtp://user:password@host:port, or "none".
//
// A URL rather than five variables: the five would each need their own "or
// none" rule, and four of them set with one missing is exactly the state that
// looks configured and silently sends nothing.
func requireSMTPOrDisabled(value, from string) (SMTP, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SMTP{}, &Error{Variable: "FERRITE_SMTP_URL",
			Reason: `is required — set smtp://user:password@host:port, or "none" to send no mail`}
	}
	if value == Disabled {
		return SMTP{}, nil
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return SMTP{}, &Error{Variable: "FERRITE_SMTP_URL", Reason: "is not a URL"}
	}
	if parsed.Scheme != "smtp" && parsed.Scheme != "smtps" {
		return SMTP{}, &Error{Variable: "FERRITE_SMTP_URL",
			Reason: `must start with smtp:// (STARTTLS) or smtps:// (TLS from the start)`}
	}
	if parsed.Hostname() == "" {
		return SMTP{}, &Error{Variable: "FERRITE_SMTP_URL", Reason: "has no host"}
	}

	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port <= 0 {
		return SMTP{}, &Error{Variable: "FERRITE_SMTP_URL",
			Reason: "must name a port, such as :587 for STARTTLS or :465 for TLS"}
	}

	password, _ := parsed.User.Password()
	settings := SMTP{
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		From:     strings.TrimSpace(from),
		Implicit: parsed.Scheme == "smtps",
	}

	// Checked here rather than at the first send: a mail server that rejects
	// the sender does so at the worst possible moment, which is the one time
	// something has gone wrong and a message actually matters.
	if settings.From == "" {
		return SMTP{}, &Error{Variable: "FERRITE_MAIL_FROM",
			Reason: "is required when a mail server is configured — the address alerts are sent from"}
	}
	if !strings.Contains(settings.From, "@") {
		return SMTP{}, &Error{Variable: "FERRITE_MAIL_FROM", Reason: "must be an email address"}
	}

	return settings, nil
}

// requireACMEEndpoint accepts "production" or "staging".
//
// Two words rather than a URL: a URL invites a typo that Traefik reports as a
// connection failure hours later, and there are exactly two answers. No
// default either — production is the one with the rate limit, so silently
// choosing it for somebody testing a new setup is the expensive direction to
// be wrong in.
func requireACMEEndpoint(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "production":
		return catalog.ACMEProduction, nil
	case "staging":
		return catalog.ACMEStaging, nil
	case "":
		return "", &Error{Variable: "FERRITE_ACME_ENDPOINT",
			Reason: `is required — "staging" while setting a server up, "production" once it works`}
	default:
		return "", &Error{Variable: "FERRITE_ACME_ENDPOINT",
			Reason: `must be "staging" or "production"`}
	}
}

// requireGitHubOrDisabled reads the four values a GitHub App needs, or "none".
//
// One switch for the set rather than four independent "or none" rules: three
// set and one missing is exactly the state that looks configured and fails at
// the first clone of a private repository, which is the least convenient
// moment to discover it.
func requireGitHubOrDisabled() (GitHub, error) {
	appID := strings.TrimSpace(os.Getenv("FERRITE_GITHUB_APP_ID"))
	if appID == "" {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_APP_ID",
			Reason: `is required — set the app's numeric id, or "none" to deploy only from a pasted git URL`}
	}
	if appID == Disabled {
		return GitHub{}, nil
	}
	if _, err := strconv.Atoi(appID); err != nil {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_APP_ID",
			Reason: "must be the app's numeric id, which is on its settings page"}
	}

	slug := strings.TrimSpace(os.Getenv("FERRITE_GITHUB_APP_SLUG"))
	if slug == "" {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_APP_SLUG",
			Reason: "is required when a GitHub app is configured — it is the name in the app's URL"}
	}

	encoded := strings.TrimSpace(os.Getenv("FERRITE_GITHUB_PRIVATE_KEY"))
	if encoded == "" {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_PRIVATE_KEY",
			Reason: "is required when a GitHub app is configured — base64 of the .pem GitHub gave you"}
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_PRIVATE_KEY",
			Reason: "must be base64. Run: base64 < your-app.private-key.pem"}
	}
	// Checked here rather than at the first clone, for the same reason the
	// mail sender is: the one moment this matters is the moment it is needed.
	if !bytes.Contains(key, []byte("PRIVATE KEY")) {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_PRIVATE_KEY",
			Reason: "does not decode to a PEM private key"}
	}

	secret := strings.TrimSpace(os.Getenv("FERRITE_GITHUB_WEBHOOK_SECRET"))
	if secret == "" {
		return GitHub{}, &Error{Variable: "FERRITE_GITHUB_WEBHOOK_SECRET",
			Reason: "is required when a GitHub app is configured — without it anybody could ask for a deploy"}
	}

	return GitHub{AppID: appID, Slug: slug, PrivateKey: key, WebhookSecret: secret}, nil
}

// requireOriginOrDisabled accepts an absolute origin, or "none" to allow no
// cross-origin requests at all.
func requireOriginOrDisabled(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", &Error{Variable: name, Reason: `is required — set an origin, or "none" to allow none`}
	}
	if value == Disabled {
		return "", nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", &Error{Variable: name,
			Reason: `must be an absolute origin like "http://localhost:5173", or "none"`}
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", &Error{Variable: name, Reason: "must be an origin with no path"}
	}

	// Compare against the Origin header, which never carries a trailing slash.
	return parsed.Scheme + "://" + parsed.Host, nil
}

// requireDirOrDisabled accepts a directory holding a built dashboard, or
// "none" to run API-only. The directory is checked now so a bad path fails at
// boot rather than on the first page load.
func requireDirOrDisabled(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", &Error{Variable: name,
			Reason: `is required — set a path to the built dashboard, or "none" to serve API only`}
	}
	if value == Disabled {
		return "", nil
	}

	info, err := os.Stat(value)
	if err != nil {
		return "", &Error{Variable: name, Reason: fmt.Sprintf("points at %q, which cannot be read: %v", value, err)}
	}
	if !info.IsDir() {
		return "", &Error{Variable: name, Reason: fmt.Sprintf("points at %q, which is not a directory", value)}
	}
	if _, err := os.Stat(filepath.Join(value, "200.html")); err != nil {
		return "", &Error{Variable: name,
			Reason: fmt.Sprintf("points at %q, which has no 200.html — run `pnpm build` in web/ first", value)}
	}

	return value, nil
}

func requireNonEmpty(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return &Error{Variable: name, Reason: "is required but was not set"}
	}
	return nil
}

func validateAddr(addr string) error {
	_, port, found := strings.Cut(addr, ":")
	if !found {
		return &Error{Variable: "FERRITE_ADDR", Reason: `must look like ":8080" or "127.0.0.1:8080"`}
	}
	if _, err := strconv.Atoi(port); err != nil {
		return &Error{Variable: "FERRITE_ADDR", Reason: "has a port that is not a number"}
	}
	return nil
}

func validateSecretKey(key string) error {
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return &Error{Variable: "FERRITE_SECRET_KEY", Reason: "is not valid base64"}
	}
	if len(raw) != secret.KeyBytes {
		return &Error{
			Variable: "FERRITE_SECRET_KEY",
			Reason: fmt.Sprintf("must decode to %d bytes, got %d — generate one with `ferrite-ship genkey`",
				secret.KeyBytes, len(raw)),
		}
	}
	return nil
}
