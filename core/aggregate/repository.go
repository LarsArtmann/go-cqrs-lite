package aggregate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Repository loads and saves aggregate roots.
type Repository interface {
	// Save persists uncommitted events from the aggregate.
	Save(ctx context.Context, root Root) error

	// Load replays event history into the provided aggregate.
	// The aggregate must have its ID and Type set (via its constructor).
	Load(ctx context.Context, root Root) error

	// Delete removes all events for the aggregate from the store.
	Delete(ctx context.Context, root Root) error
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store            event.Store
	bus              event.Bus
	snapshotStore    event.SnapshotStore
	outbox           event.Outbox
	codec            event.Codec
	snapshotStrategy SnapshotStrategy
}

var _ Repository = (*EventSourcedRepository)(nil)

// NewRepository creates a new event-sourced repository.
// Returns an error if store or bus is nil.
func NewRepository(
	store event.Store,
	bus event.Bus,
	opts ...RepositoryOption,
) (*EventSourcedRepository, error) {
	if store == nil {
		return nil, fmt.Errorf("%w", ErrNilStore)
	}

	if bus == nil {
		return nil, fmt.Errorf("%w", ErrNilBus)
	}

	r := &EventSourcedRepository{ //nolint:exhaustruct // options fill remaining fields
		store: store,
		bus:   bus,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// opError formats an error for aggregate operations.
func opError(op string, aggType event.AggregateType, aggID id.AggregateID, err error) error {
	return fmt.Errorf("%s for %s %s: %w", op, aggType, aggID, err)
}

// Save persists uncommitted events. If an outbox is configured, events are
// appended to the outbox for reliable eventual publishing. Otherwise, they
// are published directly to the bus.
//
// Partial-failure contract: if store.Save succeeds but bus.Publish or
// outbox.Append fails, events are durably stored but not published. The
// aggregate's uncommitted changes are NOT marked as committed, so the caller
// can retry. On retry, store.Save will fail with ErrVersionConflict because
// the events already exist. To recover, either use the outbox pattern (which
// decouples persistence from publishing) or handle the conflict by loading
// the current state and comparing.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID := root.ID()
	aggregateType := root.Type()
	expectedVersion := root.Version() - event.Version(len(changes))

	err := r.store.Save(ctx, aggregateType, aggregateID, changes, expectedVersion)
	if err != nil {
		return opError("save", aggregateType, aggregateID, err)
	}

	err = r.publishChanges(ctx, changes, aggregateType, aggregateID)
	if err != nil {
		return err
	}

	root.MarkChangesAsCommitted()

	if r.shouldSnapshot(root) {
		err := r.saveSnapshot(ctx, root)
		if err != nil {
			return opError("save snapshot", aggregateType, aggregateID, err)
		}
	}

	return nil
}

func (r *EventSourcedRepository) publishChanges(
	ctx context.Context,
	changes []event.Event,
	aggType event.AggregateType,
	aggID id.AggregateID,
) error {
	if r.outbox != nil {
		err := r.outbox.Append(ctx, changes)
		if err != nil {
			return opError("stage events in outbox", aggType, aggID, err)
		}
	} else {
		err := r.bus.Publish(ctx, changes...)
		if err != nil {
			return opError("publish events", aggType, aggID, err)
		}
	}

	return nil
}

// Load replays event history into the aggregate.
// If a snapshot store is configured, it loads the latest snapshot first,
// sets the aggregate version, then replays events from the snapshot version onward.
func (r *EventSourcedRepository) Load(ctx context.Context, root Root) error {
	aggregateID := root.ID()
	aggregateType := root.Type()

	events, err := r.loadEvents(ctx, root, aggregateType, aggregateID)
	if err != nil {
		return err
	}

	err = root.LoadEvents(events)
	if err != nil {
		return fmt.Errorf(
			"replay %d events for %s %s: %w",
			len(events),
			aggregateType,
			aggregateID,
			err,
		)
	}

	return nil
}

// Delete removes all events for the aggregate from the store.
func (r *EventSourcedRepository) Delete(ctx context.Context, root Root) error {
	aggregateType := root.Type()
	aggregateID := root.ID()

	err := r.store.Delete(ctx, aggregateType, aggregateID)
	if err != nil {
		return opError("delete", aggregateType, aggregateID, err)
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

	snapshot, snapErr := r.snapshotStore.Load(ctx, aggregateType, aggregateID)
	if snapErr != nil {
		if !errors.Is(snapErr, event.ErrSnapshotNotFound) {
			return nil, opError("load snapshot", aggregateType, aggregateID, snapErr)
		}

		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	if snapshot == nil {
		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	root.SetVersion(snapshot.Version)

	err := root.ApplySnapshot(snapshot.State)
	if err != nil {
		return nil, opError("apply snapshot", aggregateType, aggregateID, err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snapshot.Version)
	if err != nil {
		return nil, fmt.Errorf(
			"load events from version %d for %s %s: %w",
			snapshot.Version, aggregateType, aggregateID, err,
		)
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
	return r.snapshotStrategy != nil &&
		r.snapshotStore != nil &&
		r.codec != nil &&
		r.snapshotStrategy.ShouldSnapshot(root.Type(), root.Version())
}

func (r *EventSourcedRepository) saveSnapshot(ctx context.Context, root Root) error {
	var state []byte

	if r.codec != nil {
		encoded, err := r.codec.Encode(root)
		if err != nil {
			return fmt.Errorf("encode snapshot state: %w", err)
		}

		state = encoded
	}

	err := r.snapshotStore.Save(ctx, event.Snapshot{
		AggregateID:   root.ID(),
		AggregateType: root.Type(),
		Version:       root.Version(),
		State:         state,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("snapshot store save: %w", err)
	}

	return nil
}
