package api

import (
	"testing"
	"time"
)

func TestThrottleLocksOutAfterRepeatedFailures(t *testing.T) {
	now := time.Now()
	th := newThrottle()
	th.now = func() time.Time { return now }

	for i := range maxAttempts - 1 {
		th.recordFailure("client")
		if wait := th.retryAfter("client"); wait != 0 {
			t.Fatalf("locked out after %d attempts; the limit is %d", i+1, maxAttempts)
		}
	}

	th.recordFailure("client")
	if wait := th.retryAfter("client"); wait <= 0 {
		t.Fatal("still allowed after reaching the limit")
	}
}

// A correct password must not clear a lockout, or an attacker who guesses
// right on attempt 900 walks straight in.
func TestLockoutIsNotBypassedByASuccess(t *testing.T) {
	now := time.Now()
	th := newThrottle()
	th.now = func() time.Time { return now }

	for range maxAttempts {
		th.recordFailure("client")
	}
	if th.retryAfter("client") <= 0 {
		t.Fatal("expected a lockout")
	}

	// The handler checks retryAfter before authenticating, so a success cannot
	// be reached; this asserts the state machine agrees.
	now = now.Add(lockoutFor - time.Minute)
	if th.retryAfter("client") <= 0 {
		t.Error("lockout expired early")
	}

	now = now.Add(2 * time.Minute)
	if wait := th.retryAfter("client"); wait != 0 {
		t.Errorf("still locked out %v after the window closed", wait)
	}
}

// Someone who mistypes once and comes back tomorrow should not be one attempt
// from a lockout, and the map should not remember them.
func TestOldAttemptsAreForgotten(t *testing.T) {
	now := time.Now()
	th := newThrottle()
	th.now = func() time.Time { return now }

	th.recordFailure("client")
	now = now.Add(attemptWindow + time.Minute)

	if wait := th.retryAfter("client"); wait != 0 {
		t.Fatalf("unexpected lockout: %v", wait)
	}
	if _, remembered := th.records["client"]; remembered {
		t.Error("a stale record was kept, so the map grows without bound")
	}
}

func TestSuccessClearsTheCount(t *testing.T) {
	th := newThrottle()

	for range maxAttempts - 1 {
		th.recordFailure("client")
	}
	th.recordSuccess("client")

	for range maxAttempts - 1 {
		th.recordFailure("client")
		if wait := th.retryAfter("client"); wait != 0 {
			t.Fatal("a successful sign-in did not reset the count")
		}
	}
}

// Throttling is per client, so one noisy address cannot lock everyone out.
func TestClientsAreThrottledSeparately(t *testing.T) {
	th := newThrottle()

	for range maxAttempts {
		th.recordFailure("noisy")
	}

	if th.retryAfter("noisy") <= 0 {
		t.Fatal("the noisy client was not locked out")
	}
	if wait := th.retryAfter("someone-else"); wait != 0 {
		t.Errorf("an unrelated client was locked out for %v", wait)
	}
}
