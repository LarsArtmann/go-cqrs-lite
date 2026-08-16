package metaengine

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
)

// poisonTracker tracks collections that have been poisoned by fold panics.
// Once poisoned, a collection refuses reads until the store is recreated.
// Typed map replaces the untyped sync.Map for compile-time safety.
type poisonTracker struct {
	mu sync.RWMutex
	m  map[string]error
}

func newPoisonTracker() *poisonTracker {
	return &poisonTracker{m: make(map[string]error)}
}

// Poison marks a collection as poisoned with the given error.
func (p *poisonTracker) Poison(collection string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.m[collection] = err
}

// Check returns the poison error for a collection, or nil if healthy.
func (p *poisonTracker) Check(collection string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.m[collection]
}

// idempotencyTracker deduplicates event application by event ID.
// Used by ApplyIdempotent for at-least-once delivery scenarios.
//
// Memory is bounded by capacity: tracked IDs live in a fixed-capacity ring
// (dedup.Ring) and the oldest is evicted when full. Dedup is best-effort — a
// duplicate arriving after its ID was evicted re-applies, which stays within
// the at-least-once contract ApplyIdempotent serves (it narrows duplicate
// delivery, it does not eliminate it; durable dedup requires an external
// idempotency store). capacity <= 0 keeps the legacy unbounded behavior.
type idempotencyTracker struct {
	mu   sync.Mutex
	ring *dedup.Ring // nil → unbounded legacy mode
	seen sync.Map    // event ID → struct{}, used only when ring == nil
}

func newIdempotencyTracker(capacity int) *idempotencyTracker {
	if capacity <= 0 {
		return &idempotencyTracker{}
	}

	return &idempotencyTracker{ring: dedup.NewRing(capacity)}
}

// CheckAndRecord returns true if the eventID was already seen (duplicate).
// If not seen, it records the eventID and returns false.
func (t *idempotencyTracker) CheckAndRecord(eventID string) bool {
	if t.ring != nil {
		t.mu.Lock()
		defer t.mu.Unlock()

		if t.ring.Has(eventID) {
			return true
		}

		t.ring.Add(eventID)

		return false
	}

	_, exists := t.seen.LoadOrStore(eventID, struct{}{})

	return exists
}

// Len reports how many IDs are currently tracked. Test/diagnostic helper.
func (t *idempotencyTracker) Len() int {
	if t.ring != nil {
		t.mu.Lock()
		defer t.mu.Unlock()

		return t.ring.Len()
	}

	count := 0

	t.seen.Range(func(_, _ any) bool {
		count++
		return true
	})

	return count
}

// workloadMeter tracks read/write counts and diagnostic counters for workload
// statistics and operational health.
type workloadMeter struct {
	writeCount atomic.Int64
	// pad separates writeCount from readCount by >=128 bytes so concurrent
	// writers (IncWrite on Save/Append) and readers (IncRead on Execute) never
	// false-share a cache line. 128 covers 128-byte-line ARM cores as well as
	// the 64-byte x86 line.
	_         [120]byte
	readCount atomic.Int64

	reificationFailures atomic.Int64
	startTime           time.Time
}

func newWorkloadMeter() *workloadMeter {
	return &workloadMeter{startTime: time.Now()}
}

func (m *workloadMeter) IncWrite()              { m.writeCount.Add(1) }
func (m *workloadMeter) IncRead()               { m.readCount.Add(1) }
func (m *workloadMeter) IncReificationFailure() { m.reificationFailures.Add(1) }

// ReificationFailures returns the number of watcher values that could not be
// reified to the declared type V. Non-zero values indicate a bug in an engine
// or a mismatch between the planned value type and the engine's stored shape.
func (m *workloadMeter) ReificationFailures() int64 {
	return m.reificationFailures.Load()
}

// Stats returns observed workload rates derived from internal counters.
func (m *workloadMeter) Stats() WorkloadStats {
	uptime := time.Since(m.startTime).Seconds()
	if uptime < 1 {
		uptime = 1
	}

	return WorkloadStats{
		WriteRatePerSec:     float64(m.writeCount.Load()) / uptime,
		ReadRatePerSec:      float64(m.readCount.Load()) / uptime,
		ReificationFailures: m.reificationFailures.Load(),
	}
}
