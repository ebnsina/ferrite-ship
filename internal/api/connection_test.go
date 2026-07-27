package api

import (
	"strings"
	"testing"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/catalog"
)

// A web tool's address must not carry the password.
//
// Browsers have spent years stripping user:password out of URLs and phishing
// filters treat what is left as hostile, so a Grafana address built the way a
// PostgreSQL one is would be both leaky and broken. This is also the response
// that is never cached, so a regression here puts a password somewhere it can
// be pasted into a chat window without anyone noticing.
func TestWebConnectionURLCarriesNoCredential(t *testing.T) {
	tool := catalog.Tool{
		ID:     "grafana",
		Web:    true,
		Access: &catalog.Access{Scheme: "https", Username: "ferrite", Port: 3000},
	}

	routed := connectionURL(tool, connectionView{
		Host: "grafana.example.com", Port: 443,
		Username: "ferrite", Password: "hunter2",
	})
	if want := "https://grafana.example.com/"; routed != want {
		t.Errorf("routed url = %q, want %q", routed, want)
	}

	tunnelled := connectionURL(tool, connectionView{
		Host: "127.0.0.1", Port: 3000,
		Username: "ferrite", Password: "hunter2",
	})
	if want := "http://127.0.0.1:3000/"; tunnelled != want {
		t.Errorf("tunnelled url = %q, want %q", tunnelled, want)
	}

	for _, url := range []string{routed, tunnelled} {
		if strings.Contains(url, "hunter2") || strings.Contains(url, "ferrite:") {
			t.Errorf("%q has a credential in it", url)
		}
	}
}

// The database shape is unchanged, and has to stay that way: the whole string
// is what a client accepts, password included.
func TestDatabaseConnectionURLStillCarriesTheCredential(t *testing.T) {
	tool := catalog.Tool{
		ID:     "postgres",
		Access: &catalog.Access{Scheme: "postgresql", Username: "ferrite", Database: "app", Port: 5432},
	}

	got := connectionURL(tool, connectionView{
		Host: "127.0.0.1", Port: 5432,
		Username: "ferrite", Password: "hunter2", Database: "app",
	})
	if want := "postgresql://ferrite:hunter2@127.0.0.1:5432/app"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

// Grafana is the tool that proves the routing works, so its own wiring is
// pinned: routed through Traefik, never published itself, and reached at its
// own id under the server's domain.
func TestGrafanaIsRoutedRatherThanPublished(t *testing.T) {
	tool, err := catalog.Find("grafana")
	if err != nil {
		t.Fatalf("find grafana: %v", err)
	}

	if !tool.Web {
		t.Error("grafana is opened in a browser, so it should be marked Web")
	}
	if len(tool.PublicPorts()) != 0 {
		t.Error("grafana should be reachable only through Traefik")
	}
	if got := tool.Subdomain("example.com"); got != "grafana.example.com" {
		t.Errorf("subdomain = %q, want grafana.example.com", got)
	}
	// Without a domain there is nothing to route to, and the tunnel has to
	// remain the answer rather than an address that resolves nowhere.
	if got := tool.Subdomain(""); got != "" {
		t.Errorf("subdomain without a domain = %q, want empty", got)
	}
}

// "There is nothing to back up" and "we cannot back this up yet" must not be
// swapped.
//
// The first is reassuring and the second is an admission, so telling someone
// their search index has nothing worth keeping — when in fact we simply have
// not built the copy for it — is the kind of wrong answer that gets believed.
func TestNoBackupMessageMatchesWhetherTheToolStoresAnything(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want string
	}{
		// Keeps data, no backup built yet.
		{"meilisearch", apierr.BackupNotSupported.Message},
		{"qdrant", apierr.BackupNotSupported.Message},
		{"grafana", apierr.BackupNotSupported.Message},
		{"nats", apierr.BackupNotSupported.Message},
		// Stores nothing at all.
		{"mediamtx", apierr.BackupNotNeeded.Message},
	} {
		tool, err := catalog.Find(tc.id)
		if err != nil {
			t.Fatalf("find %s: %v", tc.id, err)
		}
		if tool.Supported() {
			t.Fatalf("%s can be backed up now; this case is stale", tc.id)
		}
		if got := noBackupFor(tool).Message; got != tc.want {
			t.Errorf("%s: %q, want %q", tc.id, got, tc.want)
		}
	}
}
