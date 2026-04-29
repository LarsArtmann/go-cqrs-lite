package aggregate

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// Repository loads and saves aggregate roots.
type Repository interface {
	// Save persists uncommitted events from the aggregate.
	Save(ctx context.Context, root Root) error

	// Load replays event history into the provided aggregate.
	// The aggregate must have its ID and Type set (via its constructor).
	Load(ctx context.Context, root Root) error
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store         event.Store
	bus           event.Bus
	snapshotStore event.SnapshotStore
}

var _ Repository = (*EventSourcedRepository)(nil)

// NewRepository creates a new event-sourced repository.
func NewRepository(store event.Store, bus event.Bus) *EventSourcedRepository {
	return &EventSourcedRepository{
		store:         store,
		bus:           bus,
		snapshotStore: nil,
	}
}

// NewRepositoryWithSnapshot creates a new event-sourced repository with snapshot support.
func NewRepositoryWithSnapshot(
	store event.Store,
	bus event.Bus,
	snapshotStore event.SnapshotStore,
) *EventSourcedRepository {
	return &EventSourcedRepository{
		store:         store,
		bus:           bus,
		snapshotStore: snapshotStore,
	}
}

// Save persists uncommitted events and publishes them to the bus.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID := root.ID()

	expectedVersion := event.Version(root.Version() - len(changes))

	err := r.store.Save(ctx, root.Type(), aggregateID, changes, expectedVersion)
	if err != nil {
		return fmt.Errorf("save %s %s: %w", root.Type(), root.ID().String(), err)
	}

	err = r.bus.Publish(ctx, changes...)
	if err != nil {
		return fmt.Errorf("publish events for %s %s: %w", root.Type(), root.ID().String(), err)
	}

	root.MarkChangesAsCommitted()

	return nil
}

// Load replays event history into the aggregate.
// If a snapshot store is configured, it loads the latest snapshot first,
// sets the aggregate version, then replays events from the snapshot version onward.
func (r *EventSourcedRepository) Load(ctx context.Context, root Root) error {
	aggregateID := root.ID()
	aggregateType := root.Type()

	var events []event.Event

	var err error

	if r.snapshotStore != nil {
		snapshot, snapErr := r.snapshotStore.Load(ctx, aggregateType, aggregateID)
		if snapErr == nil && snapshot != nil {
			root.SetVersion(snapshot.Version)

			events, err = r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snapshot.Version)
			if err != nil {
				return fmt.Errorf(
					"load events from version %d for %s %s: %w",
					snapshot.Version,
					aggregateType,
					aggregateID.String(),
					err,
				)
			}
		}
	}

	if events == nil {
		events, err = r.store.Load(ctx, aggregateType, aggregateID)
		if err != nil {
			return fmt.Errorf("load events for %s %s: %w", aggregateType, aggregateID.String(), err)
		}
	}

	err = root.LoadEvents(events)
	if err != nil {
		return fmt.Errorf(
			"replay %d events for %s %s: %w",
			len(events),
			aggregateType,
			aggregateID.String(),
			err,
		)
	}

	return nil
}
