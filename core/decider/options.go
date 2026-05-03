package decider

import (
	"context"
	"errors"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// SnapshotStrategy decides when to create a snapshot after saving events.
type SnapshotStrategy interface {
	ShouldSnapshot(aggregateType event.AggregateType, version event.Version) bool
}

// EveryNEvents creates a SnapshotStrategy that snapshots every N events.
// Panics if n <= 0.
func EveryNEvents(n int) SnapshotStrategy {
	if n <= 0 {
		panic("EveryNEvents: interval must be positive")
	}

	return &everyN{interval: n}
}

type everyN struct{ interval int }

func (s *everyN) ShouldSnapshot(_ event.AggregateType, version event.Version) bool {
	return version.Int() > 0 && version.Int()%s.interval == 0
}

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
func WithSnapshotStrategy[State any](strategy SnapshotStrategy) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.snapshotStrategy = strategy
	}
}

func (r *Repository[State]) shouldSnapshot(
	aggType event.AggregateType,
	version event.Version,
) bool {
	return r.snapshotStrategy != nil &&
		r.snapshotStore != nil &&
		r.codec != nil &&
		r.snapshotStrategy.ShouldSnapshot(aggType, version)
}

func (r *Repository[State]) saveSnapshot(
	ctx context.Context,
	state State,
	aggType event.AggregateType,
	aggID id.AggregateID,
	version event.Version,
) error {
	encoded, err := r.codec.Encode(state)
	if err != nil {
		return opError(aggType, aggID, "encode snapshot: %w", err)
	}

	err = r.snapshotStore.Save(ctx, event.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         encoded,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return opError(aggType, aggID, "save snapshot: %w", err)
	}

	return nil
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
	if err = r.codec.Decode(snap.State, &state); err != nil {
		var zero State
		return zero, 0, opError(aggType, aggID, "decode snapshot: %w", err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggType, aggID, snap.Version)
	if err != nil {
		var zero State
		return zero, 0, opError(aggType, aggID, "%w: %w", ErrLoadFailed, err)
	}

	for _, evt := range events {
		state, err = r.decider.Fold(state, evt)
		if err != nil {
			var zero State
			return zero, 0, opError(
				aggType,
				aggID,
				"%w (event %s): %w",
				ErrFoldFailed,
				evt.Type(),
				err,
			)
		}
	}

	return state, snap.Version + event.Version(len(events)), nil
}
