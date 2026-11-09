package decider

import (
	"github.com/maypok86/otter/v2"

	"github.com/larsartmann/go-codec"

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
// States returned by the default cache's Get are codec-cloned per caller:
// mutating a returned state never aliases the cached instance (owner ruling
// 2026-11-08, brutal-episode7 B1 — Get used to hand out the live cached
// instance while concurrent loads folded onto it, a data race). Put clones
// on intake for the same reason: the caller keeps mutating the instance it
// handed over (the full-load path returns its own state object), so the
// cache must own a private copy (audit finding 2026-11-09 — the B1 fix
// covered only egress).
type StateCache[State any] interface {
	// Get retrieves the cached state and version for the given stream.
	// Returns ok=false if the stream is not in the cache. Implementations
	// MUST NOT return an instance that aliases internal cache state for
	// pointer/slice/map-shaped State values.
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
		&otter.Options[string, *cacheEntry[State]]{ //nolint:exhaustruct_v5 // only MaximumSize needed
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

	// Clone BEFORE returning: callers (and concurrent incremental loads
	// folding new events onto the returned state) must never mutate the
	// shared cached instance. A state that cannot round-trip the clone codec
	// is treated as a miss — the full-load path stays correct, and the
	// uncloneable entry is dropped so it cannot mask the fallback forever.
	cloned, err := cloneState(entry.state)
	if err != nil {
		c.cache.Invalidate(ref.String())

		var zero State

		return zero, 0, false
	}

	return cloned, entry.version, true
}

// cloneState deep-clones a cached state via a codec round-trip (the kv.Cache
// isolation pattern). CBOR matches the repository-wide default codec.
func cloneState[State any](state State) (State, error) {
	data, err := (codec.CBORCodec{}).Encode(state)
	if err != nil {
		var zero State

		return zero, err
	}

	var out State

	if err := (codec.CBORCodec{}).Decode(data, &out); err != nil {
		var zero State

		return zero, err
	}

	return out, nil
}

func (c *otterCache[State]) Put(ref id.StreamRef, state State, version event.Version) {
	// Put is the intake mirror of Get's clone: callers keep a reference to
	// the state they hand over (LoadRef returns the very instance it Put),
	// so storing it by value would alias caller mutations into the cache.
	// A state that cannot round-trip the clone codec is NOT cached (the
	// full-load path stays the source of truth; Get's uncloneable-miss
	// handling stays symmetric).
	private, err := cloneState(state)
	if err != nil {
		return
	}

	c.cache.Set(ref.String(), &cacheEntry[State]{state: private, version: version})
}

func (c *otterCache[State]) Invalidate(ref id.StreamRef) {
	c.cache.Invalidate(ref.String())
}
