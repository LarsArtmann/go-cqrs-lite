package snapshot

import (
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// ReadPressure is a SnapshotStrategy that triggers snapshots based on how
// many times an aggregate has been read since its last snapshot.
//
// EveryNEvents snapshots based on write count: every N persisted events.
// ReadPressure snapshots based on read count: when an aggregate has been
// loaded at least threshold times since its last snapshot, the next write
// triggers a snapshot.
//
// This is ideal for "hot read" aggregates — ones queried frequently but
// written rarely. Without read pressure, these never hit an EveryNEvents
// threshold and pay full replay cost on every load.
//
// Combine with EveryNEvents via the Inner option to snapshot when EITHER
// condition fires:
//
//	rp, _ := snapshot.NewReadPressure(50,
//	    snapshot.WithInnerStrategy(mustEveryN(100)))
//
// The strategy is safe for concurrent use.
type ReadPressure struct {
	threshold int
	inner     SnapshotStrategy
	mu        sync.Mutex
	reads     map[string]int
}

var (
	_ SnapshotStrategy       = (*ReadPressure)(nil)
	_ AggregateAwareStrategy = (*ReadPressure)(nil)
	_ ReadTracker            = (*ReadPressure)(nil)
)

// ReadPressureOption configures a ReadPressure strategy.
type ReadPressureOption func(*ReadPressure)

// WithInnerStrategy wraps an inner SnapshotStrategy (e.g., EveryNEvents).
// The ReadPressure strategy snapshots when EITHER the read threshold is
// exceeded OR the inner strategy triggers.
func WithInnerStrategy(inner SnapshotStrategy) ReadPressureOption {
	return func(rp *ReadPressure) {
		rp.inner = inner
	}
}

// NewReadPressure creates a read-pressure-aware snapshot strategy.
//
// threshold is the minimum number of reads since the last snapshot before
// a snapshot is triggered on the next write. Must be positive.
func NewReadPressure(threshold int, opts ...ReadPressureOption) (*ReadPressure, error) {
	if threshold <= 0 {
		return nil, ErrInvalidThreshold
	}

	strategy := &ReadPressure{ //nolint:exhaustruct // inner/mu are zero-valued
		threshold: threshold,
		reads:     make(map[string]int),
	}

	for _, opt := range opts {
		opt(strategy)
	}

	return strategy, nil
}

// ShouldSnapshot implements SnapshotStrategy.
//
// Without the aggregate identity this method cannot evaluate read pressure.
// It delegates to the inner strategy if one is set, otherwise returns false.
// The Repository calls ShouldSnapshotFor when the strategy implements
// AggregateAwareStrategy.
func (rp *ReadPressure) ShouldSnapshot(
	aggregateType id.AggregateType,
	version event.Version,
) bool {
	if rp.inner != nil {
		return rp.inner.ShouldSnapshot(aggregateType, version)
	}

	return false
}

// ShouldSnapshotFor implements AggregateAwareStrategy.
//
// Returns true when EITHER:
//   - The inner strategy triggers (e.g., EveryNEvents), OR
//   - The aggregate has been read at least threshold times since its last snapshot
//
// On a positive decision the read counter for this aggregate is reset so
// the next snapshot cycle starts fresh.
func (rp *ReadPressure) ShouldSnapshotFor(
	ref id.AggregateRef,
	version event.Version,
) bool {
	if rp.inner != nil && rp.inner.ShouldSnapshot(ref.Type, version) {
		rp.reset(ref)

		return true
	}

	rp.mu.Lock()
	count := rp.reads[ref.String()]
	rp.mu.Unlock()

	if count >= rp.threshold {
		rp.reset(ref)

		return true
	}

	return false
}

// RecordRead implements ReadTracker.
//
// Called by the Repository on every successful Load. Increments the read
// counter for the given aggregate.
func (rp *ReadPressure) RecordRead(ref id.AggregateRef, _ event.Version) {
	key := ref.String()

	rp.mu.Lock()
	rp.reads[key]++
	rp.mu.Unlock()
}

// ReadCount returns the number of reads since the last snapshot for the
// given aggregate. Primarily for testing and observability.
func (rp *ReadPressure) ReadCount(ref id.AggregateRef) int {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	return rp.reads[ref.String()]
}

func (rp *ReadPressure) reset(ref id.AggregateRef) {
	rp.mu.Lock()
	delete(rp.reads, ref.String())
	rp.mu.Unlock()
}
