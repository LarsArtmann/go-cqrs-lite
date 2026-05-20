package decider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func (r *Repository[State]) loadState(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) (State, event.Version, error) {
	if r.snapshotStore != nil && r.codec != nil {
		return r.loadFromSnapshot(ctx, aggID, aggType)
	}

	return r.loadFromStore(ctx, aggID, aggType)
}

func (r *Repository[State]) loadFromStore(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
) (State, event.Version, error) {
	events, err := r.store.Load(ctx, aggType, aggID)
	if err != nil {
		if errors.Is(err, event.ErrAggregateNotFound) {
			return r.decider.Initial, 0, nil
		}

		var zero State

		return zero, 0, opError(aggType, aggID, "%w: %w", ErrLoadFailed, err)
	}

	state, err := r.foldEvents(r.decider.Initial, events, aggType, aggID)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, event.Version(len(events)), nil
}

func (r *Repository[State]) foldEvents(
	state State,
	events []event.Event,
	aggType event.AggregateType,
	aggID id.AggregateID,
) (State, error) {
	var err error

	for _, evt := range events {
		state, err = r.decider.Fold(state, evt)
		if err != nil {
			var zero State

			return zero, opError(
				aggType,
				aggID,
				"%w (event %s): %w",
				ErrFoldFailed,
				evt.Type(),
				err,
			)
		}
	}

	return state, nil
}

func opError(aggType event.AggregateType, aggID id.AggregateID, msg string, args ...any) error {
	prefix := aggType.String() + " " + aggID.String() + ": "

	return fmt.Errorf(prefix+msg, args...) //nolint:err113
}

// LoadAtVersion reconstructs state from events up to and including maxVersion.
// Useful for time-travel queries: "what was the state at version N?".
func (r *Repository[State]) LoadAtVersion(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
	maxVersion event.Version,
) (State, event.Version, error) {
	return r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToVersion(ctx, aggType, aggID, maxVersion)
		},
		aggType, aggID,
	)
}

// LoadAtTime reconstructs state from events up to and including maxTime.
// Useful for temporal queries: "what was the state at this point in time?".
func (r *Repository[State]) LoadAtTime(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
	maxTime time.Time,
) (State, event.Version, error) {
	return r.loadByEvents(
		func() ([]event.Event, error) {
			return r.store.LoadToTimestamp(ctx, aggType, aggID, maxTime)
		},
		aggType, aggID,
	)
}

func (r *Repository[State]) loadByEvents(
	loadFn func() ([]event.Event, error),
	aggType event.AggregateType,
	aggID id.AggregateID,
) (State, event.Version, error) {
	events, err := loadFn()
	if err != nil {
		if errors.Is(err, event.ErrAggregateNotFound) {
			return r.decider.Initial, 0, nil
		}

		var zero State

		return zero, 0, opError(aggType, aggID, "%w: %w", ErrLoadFailed, err)
	}

	state, err := r.foldEvents(r.decider.Initial, events, aggType, aggID)
	if err != nil {
		var zero State

		return zero, 0, err
	}

	return state, event.Version(len(events)), nil
}
