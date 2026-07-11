package decider

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

// RepositoryOption configures a Repository.
type RepositoryOption[State any] func(*Repository[State])

// WithSnapshotStore enables snapshot support for the repository.
func WithSnapshotStore[State any](store snapshot.SnapshotStore) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStore = store
	}
}

// WithCodec sets the codec for snapshot serialization.
// Required when using WithSnapshotStore — the codec encodes State to bytes
// and decodes bytes back to State.
func WithCodec[State any](codec codec.Codec) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.codec = codec
	}
}

// WithSnapshotStrategy sets the strategy for automatic snapshotting.
// When set, Execute checks the strategy after persisting events and
// creates a snapshot if the strategy triggers.
func WithSnapshotStrategy[State any](strategy snapshot.SnapshotStrategy) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStrategy = strategy
	}
}

// WithEnricher sets a context enricher that automatically enriches events
// with metadata derived from context (correlation IDs, user IDs, etc.).
func WithEnricher[State any](enricher event.ContextEnricher) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.enricher = enricher
	}
}

// WithLoadCoalescing enables or disables singleflight load coalescing.
// Enabled by default — concurrent loads of the same aggregate share a
// single store.Load call. Pass false to disable when the store already
// provides its own caching or deduplication layer.
func WithLoadCoalescing[State any](enabled bool) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.loadCoalescing = enabled
	}
}

// WithStateCache enables an in-memory hot-state cache for folded aggregate
// state. On a cache hit, Load fetches only events since the cached version
// instead of replaying the full history — O(new events) instead of O(total).
//
// The cache is process-local, LRU-bounded, and best-effort: misses and
// staleness fall back to the normal load path. Execute updates the cache
// after every successful write, so it stays fresh for the current process.
//
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithStateCache[MyState](decider.NewStateCache[MyState](256)))
//
// Profile before enabling: for small aggregates the fold cost may be
// negligible, and the cache adds map+mutex overhead on every Load/Execute.
func WithStateCache[State any](cache StateCache[State]) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.stateCache = cache
	}
}
