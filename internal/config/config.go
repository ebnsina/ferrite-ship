// Package config reads the process environment.
//
// Every value is required and validated here. There are no defaults for
// anything that decides where data goes or how it is protected: a missing
// variable stops the process at boot rather than producing a server that runs
// against the wrong database or with an unprotected credential store.
package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
}

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

	return cfg, nil
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
