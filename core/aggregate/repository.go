package aggregate

import (
	"context"
	"fmt"

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
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store         event.Store
	bus           event.Bus
	snapshotStore event.SnapshotStore
	outbox        event.Outbox
}

var _ Repository = (*EventSourcedRepository)(nil)

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

// NewRepository creates a new event-sourced repository.
func NewRepository(
	store event.Store,
	bus event.Bus,
	opts ...RepositoryOption,
) *EventSourcedRepository {
	r := &EventSourcedRepository{
		store:         store,
		bus:           bus,
		snapshotStore: nil,
		outbox:        nil,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// opError formats an error for aggregate operations.
func opError(op string, aggregateType event.AggregateType, aggregateID id.AggregateID, err error) error {
	return fmt.Errorf("%s for %s %s: %w", op, aggregateType, aggregateID.String(), err)
}

// Save persists uncommitted events. If an outbox is configured, events are
// appended to the outbox for reliable eventual publishing. Otherwise, they
// are published directly to the bus.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID := root.ID()
	aggregateType := root.Type()

	expectedVersion := event.Version(root.Version() - len(changes))

	err := r.store.Save(ctx, aggregateType, aggregateID, changes, expectedVersion)
	if err != nil {
		return opError("save", aggregateType, aggregateID, err)
	}

	if r.outbox != nil {
		err = r.outbox.Append(ctx, changes)
		if err != nil {
			return opError("stage events in outbox", aggregateType, aggregateID, err)
		}
	} else {
		err = r.bus.Publish(ctx, changes...)
		if err != nil {
			return opError("publish events", aggregateType, aggregateID, err)
		}
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
			aggregateID.String(),
			err,
		)
	}

	return nil
}

// loadEvents returns events for the aggregate, using a snapshot if available.
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
	if snapErr != nil || snapshot == nil {
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
			snapshot.Version,
			aggregateType,
			aggregateID.String(),
			err,
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
