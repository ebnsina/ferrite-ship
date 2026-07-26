package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Sign-in throttling.
//
// argon2id already makes each guess expensive, but expensive is not the same
// as limited: without a cap an attacker can simply keep going. This bounds
// attempts per client and, once tripped, refuses without hashing at all — so
// the defence cannot itself be used to burn CPU.
const (
	maxAttempts   = 8
	attemptWindow = 15 * time.Minute
	lockoutFor    = 15 * time.Minute
)

type attemptRecord struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

// throttle counts failed sign-ins per client address.
//
// In memory rather than in the database: this is a single-process control
// plane, the state is worthless after a restart, and a restart clearing it is
// an acceptable trade for not writing on every failed attempt.
type throttle struct {
	mu      sync.Mutex
	records map[string]*attemptRecord
	now     func() time.Time
}

func newThrottle() *throttle {
	return &throttle{records: map[string]*attemptRecord{}, now: time.Now}
}

// retryAfter reports how long a client must wait, or zero if it may proceed.
func (t *throttle) retryAfter(key string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	record, ok := t.records[key]
	if !ok {
		return 0
	}

	now := t.now()
	if now.Before(record.lockedUntil) {
		return record.lockedUntil.Sub(now)
	}

	// The window has passed without reaching the limit; forget it entirely so
	// the map does not grow with clients that simply mistyped once.
	if now.Sub(record.windowStart) > attemptWindow {
		delete(t.records, key)
	}
	return 0
}

// recordFailure counts one failed attempt and locks out at the limit.
func (t *throttle) recordFailure(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	record, ok := t.records[key]
	if !ok || now.Sub(record.windowStart) > attemptWindow {
		t.records[key] = &attemptRecord{count: 1, windowStart: now}
		return
	}

	record.count++
	if record.count >= maxAttempts {
		record.lockedUntil = now.Add(lockoutFor)
		record.count = 0
		record.windowStart = now
	}
}

// recordSuccess clears the history: a correct password proves the client is
// not the attacker the counter was there to stop.
func (t *throttle) recordSuccess(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, key)
}

// clientKey identifies the caller for throttling.
//
// RemoteAddr only, deliberately: X-Forwarded-For is attacker-controlled unless
// a trusted proxy sets it, and trusting it blindly would let anyone reset
// their own counter by changing a header. Behind a reverse proxy this
// throttles the proxy, which is the honest behaviour until there is a
// configured list of trusted forwarders.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
