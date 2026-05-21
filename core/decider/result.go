package decider

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Result describes the outcome of a decider execution.
type Result struct {
	Events  []event.Event
	Version event.Version
	Created bool
	Updated bool
	NoOp    bool
}

// ExecuteWithResult is like Execute but returns a Result describing what happened.
//
// Created is true when the aggregate had no prior events (version went from 0 to N).
// Updated is true when events were appended to an existing aggregate.
// NoOp is true when decide returned no events.
func (r *Repository[State]) ExecuteWithResult(
	ctx context.Context,
	aggID id.AggregateID,
	aggType event.AggregateType,
	decide DecideFunc[State],
) (Result, error) {
	state, currentVersion, err := r.loadState(ctx, aggID, aggType)
	if err != nil {
		return Result{}, err
	}

	newEvents, err := decide(state, currentVersion)
	if err != nil {
		return Result{}, err
	}

	if len(newEvents) == 0 {
		return Result{NoOp: true, Version: currentVersion}, nil
	}

	if ts, ok := r.store.(event.TransactionalStore); ok && r.outbox != nil {
		err = ts.SaveWithOutbox(ctx, aggType, aggID, newEvents, currentVersion)
		if err != nil {
			return Result{}, opError(aggType, aggID, "%w: %w", ErrSaveFailed, err)
		}
	} else {
		err = r.store.Save(ctx, aggType, aggID, newEvents, currentVersion)
		if err != nil {
			return Result{}, opError(aggType, aggID, "%w: %w", ErrSaveFailed, err)
		}

		err = event.PublishChanges(ctx, r.publisher, r.outbox, newEvents)
		if err != nil {
			return Result{}, opError(aggType, aggID, "publish events: %w", err)
		}
	}

	newVersion := currentVersion.Add(len(newEvents))

	r.saveSnapshotAfterEvents(ctx, aggType, aggID, newVersion, state, newEvents)

	return Result{
		Events:  newEvents,
		Version: newVersion,
		Created: currentVersion.IsZero(),
		Updated: !currentVersion.IsZero(),
	}, nil
}
