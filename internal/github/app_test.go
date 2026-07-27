package github

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testApp(t *testing.T) (*App, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	encoded := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	app, err := New("12345", "ferrite-ship", encoded)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app, key
}

// The assertion has to be one GitHub will accept, and the only way to know
// that without asking GitHub is to verify it exactly as GitHub would.
func TestJWTIsSignedOverTheHeaderAndClaims(t *testing.T) {
	app, key := testApp(t)

	token, err := app.jwt(time.Now())
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := sha256Sum(parts[0] + "." + parts[1])
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest, signature); err != nil {
		t.Errorf("GitHub would reject this signature: %v", err)
	}

	var header map[string]string
	decodePart(t, parts[0], &header)
	if header["alg"] != "RS256" {
		t.Errorf("alg is %q, want RS256", header["alg"])
	}
}

// GitHub rejects an assertion more than ten minutes in the future outright, and
// rejects one issued in the future at all — which is why it is backdated, since
// our clock and theirs are not the same clock.
func TestJWTFitsInsideGitHubsWindow(t *testing.T) {
	app, _ := testApp(t)

	now := time.Now()
	token, err := app.jwt(now)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}

	var claims struct {
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
		Iss string `json:"iss"`
	}
	decodePart(t, strings.Split(token, ".")[1], &claims)

	if claims.Iat >= now.Unix() {
		t.Errorf("iat is not backdated (%d vs now %d); clock skew would reject it",
			claims.Iat, now.Unix())
	}
	if life := claims.Exp - claims.Iat; life > 600 {
		t.Errorf("the token lives %ds, and GitHub refuses anything over 600", life)
	}
	if claims.Iss != "12345" {
		t.Errorf("iss is %q, want the app id", claims.Iss)
	}
}

// GitHub's own download is PKCS#1, but some tooling re-encodes it as PKCS#8.
// Refusing a valid key over a header nobody chose would be a poor welcome.
func TestBothPrivateKeyEncodingsAreAccepted(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}

	for name, block := range map[string]*pem.Block{
		"pkcs1": {Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
		"pkcs8": {Type: "PRIVATE KEY", Bytes: pkcs8},
	} {
		if _, err := New("1", "slug", pem.EncodeToMemory(block)); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}

func TestGarbageKeysAreRefused(t *testing.T) {
	for name, key := range map[string][]byte{
		"empty":   {},
		"not pem": []byte("hello"),
		"pem with rubbish inside": pem.EncodeToMemory(
			&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("not a key")}),
	} {
		if _, err := New("1", "slug", key); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// A token is minted per installation and reused until it is nearly expired.
// Minting one per clone would be a round trip to GitHub before every deploy,
// and GitHub rate limits them.
func TestInstallationTokensAreCachedPerInstallation(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("no app assertion was sent")
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "ghs_" + r.URL.Path,
			"expires_at": time.Now().Add(time.Hour),
		})
	}))
	defer server.Close()

	app, _ := testApp(t)
	app.baseURL = server.URL

	ctx := context.Background()
	first, err := app.InstallationToken(ctx, 1)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	again, err := app.InstallationToken(ctx, 1)
	if err != nil {
		t.Fatalf("again: %v", err)
	}
	if first != again {
		t.Error("the second call did not reuse the cached token")
	}
	if calls.Load() != 1 {
		t.Errorf("asked GitHub %d times for one installation's token", calls.Load())
	}

	// A different installation is a different account's repositories, and must
	// never be served the first one's token.
	other, err := app.InstallationToken(ctx, 2)
	if err != nil {
		t.Fatalf("other installation: %v", err)
	}
	if other == first {
		t.Fatal("two installations were given the same token")
	}
	if calls.Load() != 2 {
		t.Errorf("expected a second mint, got %d calls", calls.Load())
	}
}

// A token within a minute of expiring is replaced rather than handed out. It is
// used by a git clone on somebody else's machine, and one that dies in flight
// fails as "repository not found" — which reads like the repository was
// deleted rather than like a credential timing out.
func TestNearlyExpiredTokensAreReplaced(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token": "ghs_fresh",
			// The first one is already inside the margin.
			"expires_at": time.Now().Add(time.Duration(n) * time.Hour),
		})
	}))
	defer server.Close()

	app, _ := testApp(t)
	app.baseURL = server.URL
	app.tokens[7] = cached{token: "ghs_stale", expiresAt: time.Now().Add(30 * time.Second)}

	token, err := app.InstallationToken(context.Background(), 7)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token == "ghs_stale" {
		t.Error("a token 30 seconds from expiry was handed out")
	}
	if calls.Load() != 1 {
		t.Errorf("expected one mint, got %d", calls.Load())
	}
}

// An installation somebody removed from GitHub's side answers 404, and the
// only way we learn about it is by asking.
func TestARefusedTokenRequestIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	app, _ := testApp(t)
	app.baseURL = server.URL

	if _, err := app.InstallationToken(context.Background(), 9); err == nil {
		t.Fatal("a 404 was treated as success")
	}
	if _, cached := app.tokens[9]; cached {
		t.Error("a failed request left something in the cache")
	}
}

func TestCloneURLCarriesTheToken(t *testing.T) {
	got := CloneURL("ghs_secret", "ebnsina", "ferrite-ship")
	want := "https://x-access-token:ghs_secret@github.com/ebnsina/ferrite-ship.git"
	if got != want {
		t.Errorf("clone url = %q, want %q", got, want)
	}
}

func TestInstallURLCarriesTheState(t *testing.T) {
	app, _ := testApp(t)
	got := app.InstallURL("abc123")
	if !strings.Contains(got, "/apps/ferrite-ship/installations/new") {
		t.Errorf("install url does not point at the app: %q", got)
	}
	if !strings.Contains(got, "state=abc123") {
		t.Errorf("install url drops the state, so the callback cannot be checked: %q", got)
	}
}

func decodePart(t *testing.T, part string, into any) {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
