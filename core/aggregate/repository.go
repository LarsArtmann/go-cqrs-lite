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

	// Delete removes all events for the aggregate from the store.
	Delete(ctx context.Context, root Root) error
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store            event.Store
	publisher        event.Publisher
	snapshotStore    event.SnapshotStore
	outbox           event.Outbox
	codec            event.Codec
	snapshotStrategy event.SnapshotStrategy
}

var _ Repository = (*EventSourcedRepository)(nil)

// NewRepository creates a new event-sourced repository.
// Returns an error if store or publisher is nil.
func NewRepository(
	store event.Store,
	publisher event.Publisher,
	opts ...RepositoryOption,
) (*EventSourcedRepository, error) {
	if store == nil {
		return nil, fmt.Errorf("%w", ErrNilStore)
	}

	if publisher == nil {
		return nil, fmt.Errorf("%w", ErrNilBus)
	}

	r := &EventSourcedRepository{ //nolint:exhaustruct // options fill remaining fields
		store:     store,
		publisher: publisher,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
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
