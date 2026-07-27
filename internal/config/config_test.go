package config

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/ebnsina/ferrite-ship/internal/catalog"
)

// A mail server that is half-configured is the state worth failing on: it
// looks set up, and it sends nothing.
func TestSMTPConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		from    string
		wantErr bool
		check   func(*testing.T, SMTP)
	}{
		{
			name: "unset is an error, not a quiet no",
			url:  "", from: "", wantErr: true,
		},
		{
			name: "none turns it off",
			url:  "none", from: "",
			check: func(t *testing.T, got SMTP) {
				if got.Enabled() {
					t.Error(`"none" must mean no mail server`)
				}
			},
		},
		{
			name: "a server with no sender is refused",
			url:  "smtp://user:pass@mail.example.com:587", from: "",
			wantErr: true,
		},
		{
			name: "a sender that is not an address is refused",
			url:  "smtp://user:pass@mail.example.com:587", from: "ferrite",
			wantErr: true,
		},
		{
			name: "a port is required, because 25 is rarely what anyone means",
			url:  "smtp://user:pass@mail.example.com", from: "a@b.com",
			wantErr: true,
		},
		{
			name: "an unknown scheme is refused rather than assumed",
			url:  "https://mail.example.com:587", from: "a@b.com",
			wantErr: true,
		},
		{
			name: "starttls",
			url:  "smtp://user:p%40ss@mail.example.com:587", from: "alerts@example.com",
			check: func(t *testing.T, got SMTP) {
				if got.Implicit {
					t.Error("smtp:// starts in the clear and upgrades")
				}
				if got.Host != "mail.example.com" || got.Port != 587 {
					t.Errorf("wrong destination: %s:%d", got.Host, got.Port)
				}
				if got.User != "user" {
					t.Errorf("wrong user: %q", got.User)
				}
				// Escaped in the URL, and a password that arrived still
				// percent-encoded would fail authentication with no clue why.
				if got.Password != "p@ss" {
					t.Errorf("password not decoded: %q", got.Password)
				}
			},
		},
		{
			name: "implicit tls",
			url:  "smtps://user:pass@mail.example.com:465", from: "alerts@example.com",
			check: func(t *testing.T, got SMTP) {
				if !got.Implicit {
					t.Error("smtps:// is TLS from the first byte")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := requireSMTPOrDisabled(test.url, test.from)

			if test.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.check != nil {
				test.check(t, got)
			}
		})
	}
}

// Three of the four set is the state worth failing on: it looks configured,
// and it fails at the first clone of a private repository — which is both the
// least convenient moment and the hardest to attribute to a missing variable.
func TestGitHubConfiguration(t *testing.T) {
	// A real PEM is not needed: what is checked here is the shape, and using a
	// generated key would make this test about RSA rather than about config.
	pem := base64.StdEncoding.EncodeToString(
		[]byte("-----BEGIN RSA PRIVATE KEY-----\nx\n-----END RSA PRIVATE KEY-----\n"))

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(*testing.T, GitHub)
	}{
		{
			name:    "unset is an error, not a quiet no",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "none turns it off, and asks for nothing else",
			env:  map[string]string{"FERRITE_GITHUB_APP_ID": "none"},
			check: func(t *testing.T, got GitHub) {
				if got.Enabled() {
					t.Error(`"none" must mean no GitHub app`)
				}
			},
		},
		{
			name: "an app id that is not a number is refused",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "ferrite-ship", "FERRITE_GITHUB_APP_SLUG": "s",
				"FERRITE_GITHUB_PRIVATE_KEY": pem, "FERRITE_GITHUB_WEBHOOK_SECRET": "shh",
			},
			wantErr: true,
		},
		{
			name: "a missing slug is refused, because nobody could install it",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "123", "FERRITE_GITHUB_PRIVATE_KEY": pem,
				"FERRITE_GITHUB_WEBHOOK_SECRET": "shh",
			},
			wantErr: true,
		},
		{
			name: "a key that is not base64 is refused",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "123", "FERRITE_GITHUB_APP_SLUG": "s",
				"FERRITE_GITHUB_PRIVATE_KEY":    "-----BEGIN RSA PRIVATE KEY-----",
				"FERRITE_GITHUB_WEBHOOK_SECRET": "shh",
			},
			wantErr: true,
		},
		{
			name: "base64 of something that is not a key is refused",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "123", "FERRITE_GITHUB_APP_SLUG": "s",
				"FERRITE_GITHUB_PRIVATE_KEY":    base64.StdEncoding.EncodeToString([]byte("hello")),
				"FERRITE_GITHUB_WEBHOOK_SECRET": "shh",
			},
			wantErr: true,
		},
		{
			// The one that would otherwise be discovered by an open endpoint
			// accepting a deploy request from anybody.
			name: "a missing webhook secret is refused",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "123", "FERRITE_GITHUB_APP_SLUG": "s",
				"FERRITE_GITHUB_PRIVATE_KEY": pem,
			},
			wantErr: true,
		},
		{
			name: "all four together",
			env: map[string]string{
				"FERRITE_GITHUB_APP_ID": "123", "FERRITE_GITHUB_APP_SLUG": "ferrite-ship",
				"FERRITE_GITHUB_PRIVATE_KEY": pem, "FERRITE_GITHUB_WEBHOOK_SECRET": "shh",
			},
			check: func(t *testing.T, got GitHub) {
				if !got.Enabled() {
					t.Fatal("a fully configured app reports itself disabled")
				}
				if got.Slug != "ferrite-ship" {
					t.Errorf("slug is %q", got.Slug)
				}
				if !bytes.Contains(got.PrivateKey, []byte("PRIVATE KEY")) {
					t.Error("the key was not decoded from base64")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{
				"FERRITE_GITHUB_APP_ID", "FERRITE_GITHUB_APP_SLUG",
				"FERRITE_GITHUB_PRIVATE_KEY", "FERRITE_GITHUB_WEBHOOK_SECRET",
			} {
				t.Setenv(name, tc.env[name])
			}

			got, err := requireGitHubOrDisabled()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected this to be refused")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// Production limits duplicate certificates to five per week, so getting a new
// setup wrong twice locks you out of the fix for a week. There is no default
// for that reason: silently choosing production for somebody who is still
// testing is the expensive direction to be wrong in.
func TestACMEEndpointMustBeChosen(t *testing.T) {
	if _, err := requireACMEEndpoint(""); err == nil {
		t.Error("an unset endpoint quietly picked one")
	}
	if _, err := requireACMEEndpoint("https://acme-v02.api.letsencrypt.org/directory"); err == nil {
		t.Error("a URL was accepted; the two words exist to stop a typo becoming a failed issuance")
	}
	if _, err := requireACMEEndpoint("prod"); err == nil {
		t.Error(`"prod" was accepted`)
	}

	staging, err := requireACMEEndpoint("staging")
	if err != nil {
		t.Fatalf("staging: %v", err)
	}
	if staging != catalog.ACMEStaging {
		t.Errorf("staging mapped to %q", staging)
	}

	production, err := requireACMEEndpoint("  production  ")
	if err != nil {
		t.Fatalf("production: %v", err)
	}
	if production != catalog.ACMEProduction {
		t.Errorf("production mapped to %q", production)
	}
	if staging == production {
		t.Error("both words map to the same endpoint, so staging would rate limit too")
	}
}
