package decider

import (
	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultStateCacheCapacity is used when capacity <= 0.
const DefaultStateCacheCapacity = 128

// StateCache caches folded stream state in memory to avoid replaying the
// full event history on every load.
//
// The cache is best-effort: a miss falls back to the normal load path
// (snapshot store or full replay). The cache is process-local and does not
// participate in distributed consistency — each process maintains its own
// cache.
//
// When a cached entry exists at version V, the Repository loads only events
// after V (via store.LoadFromVersion) and folds them onto the cached state,
// producing the current state in O(new events) instead of O(total events).
//
// States stored in the cache must be treated as immutable by the consumer.
type StateCache[State any] interface {
	// Get retrieves the cached state and version for the given stream.
	// Returns ok=false if the stream is not in the cache.
	Get(ref id.StreamRef) (state State, version event.Version, ok bool)

	// Put stores the state and version for the given stream.
	Put(ref id.StreamRef, state State, version event.Version)

	// Invalidate removes the stream from the cache.
	Invalidate(ref id.StreamRef)
}

type cacheEntry[State any] struct {
	state   State
	version event.Version
}

// otterCache implements StateCache using an Otter TinyLFU cache (ADR-0032
// pattern, matching kv.Cache). Otter is lock-free for reads and provides
// better hit rates than plain LRU via frequency-based admission.
type otterCache[State any] struct {
	cache *otter.Cache[string, *cacheEntry[State]]
}

// NewStateCache creates an Otter-backed StateCache with the given capacity.
// If capacity <= 0, DefaultStateCacheCapacity is used.
//
// The cache uses TinyLFU admission policy: entries are admitted based on
// access frequency, providing better hit rates than simple LRU eviction.
func NewStateCache[State any](capacity int) StateCache[State] {
	if capacity <= 0 {
		capacity = DefaultStateCacheCapacity
	}

	cache := otter.Must(
		&otter.Options[string, *cacheEntry[State]]{ //nolint:exhaustruct // only MaximumSize needed
			MaximumSize: capacity,
		},
	)

	return &otterCache[State]{cache: cache}
}

func (c *otterCache[State]) Get(ref id.StreamRef) (State, event.Version, bool) {
	entry, ok := c.cache.GetIfPresent(ref.String())
	if !ok {
		var zero State

		return zero, 0, false
	}

	return entry.state, entry.version, true
}

func (c *otterCache[State]) Put(ref id.StreamRef, state State, version event.Version) {
	c.cache.Set(ref.String(), &cacheEntry[State]{state: state, version: version})
}

func (c *otterCache[State]) Invalidate(ref id.StreamRef) {
	c.cache.Invalidate(ref.String())
}
