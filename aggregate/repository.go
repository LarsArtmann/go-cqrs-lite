package aggregate

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Repository loads and saves aggregate roots.
type Repository interface {
	// Save persists uncommitted events from the aggregate.
	Save(ctx context.Context, root Root) error

	// Load replays event history into the provided aggregate.
	// The aggregate must have its ID and Type set (via its constructor).
	Load(ctx context.Context, root Root) error
}

// HistoryLoader is an optional interface for aggregates that rebuild state
// from events with proper version tracking.
//
// Aggregates that embed Core should implement this by delegating to Core:
//
//	func (u *User) LoadEvents(events []event.Event) error {
//	    return u.Core.LoadFromHistory(u, events)
//	}
type HistoryLoader interface {
	LoadEvents(events []event.Event) error
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store event.Store
	bus   event.Bus
}

var _ Repository = (*EventSourcedRepository)(nil)

// NewRepository creates a new event-sourced repository.
func NewRepository(store event.Store, bus event.Bus) *EventSourcedRepository {
	return &EventSourcedRepository{store: store, bus: bus}
}

// Save persists uncommitted events and publishes them to the bus.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID, err := id.ParseAggregateID(root.ID())
	if err != nil {
		return fmt.Errorf("parse aggregate ID for save: %w", err)
	}

	expectedVersion := event.Version(root.Version() - len(changes))

	if err := r.store.Save(ctx, root.Type(), aggregateID, changes, expectedVersion); err != nil {
		return fmt.Errorf("save %s %s: %w", root.Type(), root.ID(), err)
	}

	if err := r.bus.Publish(ctx, changes...); err != nil {
		return fmt.Errorf("publish events for %s %s: %w", root.Type(), root.ID(), err)
	}

	root.MarkChangesAsCommitted()

	return nil
}

// Load replays event history into the aggregate.
// If the aggregate implements HistoryLoader, it uses that for proper version tracking.
// Otherwise, it applies each event via Root.Apply.
func (r *EventSourcedRepository) Load(ctx context.Context, root Root) error {
	aggregateID, err := id.ParseAggregateID(root.ID())
	if err != nil {
		return fmt.Errorf("parse aggregate ID for load: %w", err)
	}

	events, err := r.store.Load(ctx, root.Type(), aggregateID)
	if err != nil {
		return fmt.Errorf("load events for %s %s: %w", root.Type(), root.ID(), err)
	}

	if loader, ok := root.(HistoryLoader); ok {
		err := loader.LoadEvents(events)
		if err != nil {
			return fmt.Errorf(
				"replay %d events for %s %s: %w",
				len(events),
				root.Type(),
				root.ID(),
				err,
			)
		}

		return nil
	}

	for i, evt := range events {
		err := root.Apply(evt)
		if err != nil {
			return fmt.Errorf(
				"apply event %d (%s) to %s %s: %w",
				i,
				evt.Type(),
				root.Type(),
				root.ID(),
				err,
			)
		}
	}

	return nil
}
