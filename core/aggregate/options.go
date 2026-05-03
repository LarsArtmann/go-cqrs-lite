package aggregate

import (
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SnapshotStrategy decides when to create a snapshot after saving events.
//
// Deprecated: Use event.SnapshotStrategy instead. This type alias is provided
// for backward compatibility and will be removed in a future version.
type SnapshotStrategy = event.SnapshotStrategy

// EveryNEvents creates a SnapshotStrategy that snapshots every N events.
// Panics if n <= 0.
//
// Deprecated: Use event.EveryNEvents instead. This function is provided
// for backward compatibility and will be removed in a future version.
var EveryNEvents = event.EveryNEvents //nolint:gochecknoglobals

// RepositoryOption configures an EventSourcedRepository.
type RepositoryOption func(*EventSourcedRepository)

// WithSnapshotStore enables snapshot support for the repository.
func WithSnapshotStore(store event.SnapshotStore) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.snapshotStore = store
	}
}

// WithOutbox enables outbox support for reliable event publishing.
// When configured, Save appends events to the outbox instead of
// publishing directly to the bus. The caller must run an OutboxPublisher
// background process to drain the outbox.
func WithOutbox(outbox event.Outbox) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.outbox = outbox
	}
}

// WithCodec sets the codec for snapshot serialization.
// When set, Save encodes snapshot state via the codec instead of
// relying on the aggregate to serialize itself. Load decodes via
// the codec before calling ApplySnapshot.
func WithCodec(codec event.Codec) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.codec = codec
	}
}

// WithSnapshotStrategy sets the strategy for automatic snapshotting.
// When set, Save checks the strategy after persisting events and
// creates a snapshot if the strategy triggers.
func WithSnapshotStrategy(strategy SnapshotStrategy) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.snapshotStrategy = strategy
	}
}
