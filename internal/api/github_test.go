package api

import (
	"testing"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/secret"
)

func testAPI(t *testing.T) *API {
	t.Helper()

	key, err := secret.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sealer, err := secret.NewSealer(key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	return &API{sealer: sealer}
}

// The state is what stops one person's install being attached to another
// person's account. Everything here is a way that could happen.
func TestInstallStateOnlyOpensForWhoStartedIt(t *testing.T) {
	api := testAPI(t)

	state, err := api.signState("usr_alice")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	got, err := api.openState(state)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if got != "usr_alice" {
		t.Errorf("state opened as %q, want usr_alice", got)
	}

	// It must not be readable, or writable, without the key.
	if state == "usr_alice" || len(state) < 20 {
		t.Errorf("the state is not sealed: %q", state)
	}
}

func TestForgedOrMissingStatesAreRefused(t *testing.T) {
	api := testAPI(t)

	for name, state := range map[string]string{
		"empty":              "",
		"plain text":         "usr_alice|2099-01-01T00:00:00Z",
		"not base64":         "!!!!",
		"base64 of nonsense": "aGVsbG8gd29ybGQ=",
	} {
		if _, err := api.openState(state); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// A state sealed by a different installation must not open here, or a link
// from one control plane would work against another.
func TestAStateFromAnotherInstallationIsRefused(t *testing.T) {
	mine := testAPI(t)
	theirs := testAPI(t)

	state, err := theirs.signState("usr_alice")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := mine.openState(state); err == nil {
		t.Error("a state sealed with another key was accepted")
	}
}

// A link left in a browser's history should stop working. The deadline is
// inside the sealed payload, so it cannot be extended without the key.
func TestAnExpiredStateIsRefused(t *testing.T) {
	api := testAPI(t)

	stale, err := api.sealer.Seal("usr_alice|" + time.Now().Add(-time.Minute).UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := api.openState(stale); err == nil {
		t.Error("an expired state was accepted")
	}

	fresh, err := api.sealer.Seal("usr_alice|" + time.Now().Add(time.Minute).UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := api.openState(fresh); err != nil {
		t.Errorf("a state a minute from expiry was refused: %v", err)
	}
}

// A payload with no deadline at all must not be treated as one that never
// expires.
func TestAStateWithoutADeadlineIsRefused(t *testing.T) {
	api := testAPI(t)

	sealed, err := api.sealer.Seal("usr_alice")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := api.openState(sealed); err == nil {
		t.Error("a state with no deadline was accepted")
	}
}
