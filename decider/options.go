package decider

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
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
