package decider

import (
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// RepositoryOption configures a Repository.
type RepositoryOption[State any] func(*Repository[State])

// WithOutbox enables outbox support for reliable event publishing.
// When configured, Execute appends events to the outbox instead of
// publishing directly to the bus.
func WithOutbox[State any](outbox event.Outbox) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.outbox = outbox
	}
}

// WithSnapshotStore enables snapshot support for the repository.
func WithSnapshotStore[State any](store event.SnapshotStore) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStore = store
	}
}

// WithCodec sets the codec for snapshot serialization.
// Required when using WithSnapshotStore — the codec encodes State to bytes
// and decodes bytes back to State.
func WithCodec[State any](codec event.Codec) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.codec = codec
	}
}

// WithSnapshotStrategy sets the strategy for automatic snapshotting.
// When set, Execute checks the strategy after persisting events and
// creates a snapshot if the strategy triggers.
func WithSnapshotStrategy[State any](strategy event.SnapshotStrategy) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStrategy = strategy
	}
}

