package metaengine

import (
	"sync"
	"sync/atomic"
	"time"
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
type idempotencyTracker struct {
	applied sync.Map // event ID → struct{}
}

func newIdempotencyTracker() *idempotencyTracker {
	return &idempotencyTracker{}
}

// CheckAndRecord returns true if the eventID was already seen (duplicate).
// If not seen, it records the eventID and returns false.
func (t *idempotencyTracker) CheckAndRecord(eventID string) bool {
	_, exists := t.applied.LoadOrStore(eventID, struct{}{})
	return exists
}

// workloadMeter tracks read/write counts for workload statistics.
type workloadMeter struct {
	writeCount atomic.Int64
	readCount  atomic.Int64
	startTime  time.Time
}

func newWorkloadMeter() *workloadMeter {
	return &workloadMeter{startTime: time.Now()}
}

func (m *workloadMeter) IncWrite() { m.writeCount.Add(1) }
func (m *workloadMeter) IncRead()  { m.readCount.Add(1) }

// Stats returns observed workload rates derived from internal counters.
func (m *workloadMeter) Stats() WorkloadStats {
	uptime := time.Since(m.startTime).Seconds()
	if uptime < 1 {
		uptime = 1
	}

	return WorkloadStats{
		WriteRatePerSec: float64(m.writeCount.Load()) / uptime,
		ReadRatePerSec:  float64(m.readCount.Load()) / uptime,
	}
}
