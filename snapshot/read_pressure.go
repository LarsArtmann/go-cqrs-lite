package snapshot

import (
	"container/list"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ReadPressure is a SnapshotStrategy that triggers snapshots based on how
// many times a stream has been read since its last snapshot.
//
// EveryNEvents snapshots based on write count: every N persisted events.
// ReadPressure snapshots based on read count: when a stream has been
// loaded at least threshold times since its last snapshot, the next write
// triggers a snapshot.
//
// This is ideal for "hot read" streams — ones queried frequently but
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
//
// Memory model: the read counter map holds one entry per stream that has
// been read since its last snapshot, and entries are removed only when a
// snapshot fires. Read-heavy streams that are never written again leave
// their (small) counter entries behind. Use [WithReadTrackingLimit] to bound
// the map LRU-style: when the limit is reached, the least-recently-read
// stream's counter is dropped (it re-accumulates from zero, which can only
// delay a snapshot trigger, never corrupt one).
type ReadPressure struct {
	threshold int
	inner     SnapshotStrategy
	mu        sync.Mutex
	reads     map[string]int

	// capacity <= 0 means unbounded (default). lru front = least recently read.
	capacity int
	lru      *list.List
	lruElems map[string]*list.Element
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

// WithReadTrackingLimit bounds the read-counter map: when the limit is
// reached, the least-recently-read stream's counter is evicted (it
// re-accumulates from zero). Values <= 0 mean unbounded (default).
func WithReadTrackingLimit(limit int) ReadPressureOption {
	return func(rp *ReadPressure) {
		rp.capacity = limit
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

	strategy := &ReadPressure{ //nolint:exhaustruct_v5 // inner is zero-valued
		threshold: threshold,
		reads:     make(map[string]int),
		lru:       list.New(),
		lruElems:  make(map[string]*list.Element),
	}

	for _, opt := range opts {
		opt(strategy)
	}

	return strategy, nil
}

// ShouldSnapshot implements SnapshotStrategy.
//
// Without the stream identity this method cannot evaluate read pressure.
// It delegates to the inner strategy if one is set, otherwise returns false.
// The Repository calls ShouldSnapshotFor when the strategy implements
// AggregateAwareStrategy.
func (rp *ReadPressure) ShouldSnapshot(
	streamType id.StreamType,
	version event.Version,
) bool {
	if rp.inner != nil {
		return rp.inner.ShouldSnapshot(streamType, version)
	}

	return false
}

// ShouldSnapshotFor implements AggregateAwareStrategy.
//
// Returns true when EITHER:
//   - The inner strategy triggers (e.g., EveryNEvents), OR
//   - The stream has been read at least threshold times since its last snapshot
//
// On a positive decision the read counter for this stream is reset so
// the next snapshot cycle starts fresh.
func (rp *ReadPressure) ShouldSnapshotFor(
	ref id.StreamRef,
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
// counter for the given stream, refreshing its LRU recency.
func (rp *ReadPressure) RecordRead(ref id.StreamRef, _ event.Version) {
	key := ref.String()

	rp.mu.Lock()
	defer rp.mu.Unlock()

	if el, ok := rp.lruElems[key]; ok {
		rp.lru.MoveToBack(el)
	} else {
		if rp.capacity > 0 && len(rp.reads) >= rp.capacity {
			rp.evictOldest()
		}

		rp.lruElems[key] = rp.lru.PushBack(key)
	}

	rp.reads[key]++
}

// ReadCount returns the number of reads since the last snapshot for the
// given stream. Primarily for testing and observability.
func (rp *ReadPressure) ReadCount(ref id.StreamRef) int {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	return rp.reads[ref.String()]
}

func (rp *ReadPressure) reset(ref id.StreamRef) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	key := ref.String()
	delete(rp.reads, key)

	if el, ok := rp.lruElems[key]; ok {
		rp.lru.Remove(el)
		delete(rp.lruElems, key)
	}
}

func (rp *ReadPressure) evictOldest() {
	front := rp.lru.Front()
	if front == nil {
		return
	}

	key, ok := rp.lru.Remove(front).(string)
	if !ok {
		return
	}

	delete(rp.reads, key)
	delete(rp.lruElems, key)
}
