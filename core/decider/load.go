package decider

import (
	"context"
	"errors"
	"fmt"

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
