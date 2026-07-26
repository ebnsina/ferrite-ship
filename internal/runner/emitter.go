package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/store"
)

// emitter numbers a job's events, writes them down, and hands them to watchers.
//
// Persisting before publishing matters: a client that reconnects asks for
// "everything after sequence N", and that is only answerable if the database
// is never behind the stream.
type emitter struct {
	store *store.Store
	bus   *Bus
	jobID string

	mu  sync.Mutex
	seq int
}

func newEmitter(st *store.Store, bus *Bus, jobID string) *emitter {
	return &emitter{store: st, bus: bus, jobID: jobID}
}

func (e *emitter) emit(ctx context.Context, event store.Event) {
	e.mu.Lock()
	e.seq++
	event.Seq = e.seq
	e.mu.Unlock()

	event.JobID = e.jobID
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	// Store with a background context: a cancelled run still deserves a
	// complete record of how far it got.
	id, err := e.store.AppendEvent(context.WithoutCancel(ctx), event)
	if err == nil {
		event.ID = id
	}

	e.bus.Publish(e.jobID, event)
}

// newID returns a short, sortable-enough identifier with a readable prefix.
func newID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing is not recoverable in any useful way here.
		panic("runner: could not read random bytes: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
