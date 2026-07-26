package runner

import (
	"sync"

	"github.com/ebnsina/ferrite-ship/internal/store"
)

// Bus fans job events out to whoever is watching a job's logs.
//
// Subscribers get a buffered channel; a subscriber that cannot keep up is
// dropped rather than allowed to stall the run. Nothing is lost by that — every
// event is already in the database, and a reconnecting client resumes by
// sequence number.
type Bus struct {
	mu   sync.RWMutex
	subs map[string]map[chan store.Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: make(map[string]map[chan store.Event]struct{})}
}

const subscriberBuffer = 256

// Subscribe returns a channel of events for jobID and a function to release it.
func (b *Bus) Subscribe(jobID string) (<-chan store.Event, func()) {
	ch := make(chan store.Event, subscriberBuffer)

	b.mu.Lock()
	if b.subs[jobID] == nil {
		b.subs[jobID] = make(map[chan store.Event]struct{})
	}
	b.subs[jobID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if subs, ok := b.subs[jobID]; ok {
				delete(subs, ch)
				if len(subs) == 0 {
					delete(b.subs, jobID)
				}
			}
			close(ch)
		})
	}

	return ch, cancel
}

func (b *Bus) Publish(jobID string, event store.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subs[jobID] {
		select {
		case ch <- event:
		default:
			// Slow consumer: skip it. It can catch up from the database.
		}
	}
}
