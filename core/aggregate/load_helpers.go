package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Save persists uncommitted events. If an outbox is configured, events are
// appended to the outbox; otherwise published directly to the bus.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateType := root.Type()
	aggregateID := root.ID()
	expectedVersion := root.Version() - event.Version(len(changes))

	err := r.persistChanges(
		ctx,
		aggregateType,
		aggregateID,
		changes,
		expectedVersion,
	)
	if err != nil {
		return err
	}

	root.MarkChangesAsCommitted()

	return r.trySnapshot(ctx, root)
}

func (r *EventSourcedRepository) persistChanges(
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
	changes []event.Event,
	expectedVersion event.Version,
) error {
	if r.outbox == nil {
		return r.persistDirect(ctx, aggType, aggID, changes, expectedVersion)
	}

	if ts, ok := r.store.(event.TransactionalStore); ok {
		err := ts.SaveWithOutbox(ctx, aggType, aggID, changes, expectedVersion, r.outbox)
		if err != nil {
			return opError("save with outbox", aggType, aggID, err)
		}

		return nil
	}

	err := r.store.Save(ctx, aggType, aggID, changes, expectedVersion)
	if err != nil {
		return opError("save", aggType, aggID, err)
	}

	err = r.outbox.Append(ctx, changes)
	if err != nil {
		return opError("stage events in outbox", aggType, aggID, err)
	}

	return nil
}

func (r *EventSourcedRepository) persistDirect(
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
	changes []event.Event,
	expectedVersion event.Version,
) error {
	err := r.store.Save(ctx, aggType, aggID, changes, expectedVersion)
	if err != nil {
		return opError("save", aggType, aggID, err)
	}

	err = r.publisher.Publish(ctx, changes...)
	if err != nil {
		return opError("publish events", aggType, aggID, err)
	}

	return nil
}

func (r *EventSourcedRepository) trySnapshot(ctx context.Context, root Root) error {
	if !r.shouldSnapshot(root) {
		return nil
	}

	var state []byte

	if r.codec != nil {
		encoded, err := r.codec.Encode(root)
		if err != nil {
			return fmt.Errorf("encode snapshot state: %w", err)
		}

		state = encoded
	}

	err := event.SaveSnapshot(
		ctx,
		r.snapshotStore,
		root.Type(),
		root.ID(),
		root.Version(),
		state,
	)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	return nil
}

func (r *EventSourcedRepository) loadEvents(
	ctx context.Context,
	root Root,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	if r.snapshotStore == nil {
		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	snapshot, err := r.snapshotStore.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		if !errors.Is(err, event.ErrSnapshotNotFound) {
			return nil, opError("load snapshot", aggregateType, aggregateID, err)
		}

		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	if snapshot == nil {
		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	root.SetVersion(snapshot.Version)

	err = root.ApplySnapshot(snapshot.State)
	if err != nil {
		return nil, opError("apply snapshot", aggregateType, aggregateID, err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snapshot.Version)
	if err != nil {
		return nil, opError("load events from snapshot", aggregateType, aggregateID, err)
	}

	return events, nil
}

func (r *EventSourcedRepository) loadFromStore(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	events, err := r.store.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		return nil, opError("load events", aggregateType, aggregateID, err)
	}

	return events, nil
}

func (r *EventSourcedRepository) shouldSnapshot(root Root) bool {
	return event.ShouldSnapshot(
		r.snapshotStrategy,
		r.snapshotStore,
		r.codec,
		root.Type(),
		root.Version(),
	)
}

// opError formats an error for aggregate operations.
func opError(op string, aggType event.AggregateType, aggID id.AggregateID, err error) error {
	return fmt.Errorf("%s for %s %s: %w", op, aggType, aggID, err)
}
