package decider

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

func (r *Repository[State]) shouldSnapshot(
	aggType event.AggregateType,
	version event.Version,
) bool {
	return event.ShouldSnapshot(r.snapshotStrategy, r.snapshotStore, r.codec, aggType, version)
}

func (r *Repository[State]) loadFromSnapshot(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) (State, event.Version, error) {
	snap, err := r.snapshotStore.Load(ctx, aggType, aggID)
	if err != nil {
		if !errors.Is(err, event.ErrSnapshotNotFound) {
			var zero State

			return zero, 0, opError(aggType, aggID, "load snapshot: %w", err)
		}

		return r.loadFromStore(ctx, aggID, aggType)
	}

	if snap == nil {
		return r.loadFromStore(ctx, aggID, aggType)
	}

	var state State

	err = r.codec.Decode(snap.State, &state)
	if err != nil {
		var zero State

		return zero, 0, opError(aggType, aggID, "decode snapshot: %w", err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggType, aggID, snap.Version)
	if err != nil {
		var zero State

		return zero, 0, opError(aggType, aggID, "%w: %w", ErrLoadFailed, err)
	}

	state, err = r.foldEvents(state, events, aggType, aggID)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, snap.Version.Add(len(events)), nil
}
