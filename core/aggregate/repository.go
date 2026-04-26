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
// The aggregate MUST implement HistoryLoader for proper version tracking.
// If it does not, Load returns an error — aggregates embedding Core should
// delegate to Core.LoadFromHistory via the HistoryLoader interface.
func (r *EventSourcedRepository) Load(ctx context.Context, root Root) error {
	aggregateID := root.ID()

	events, err := r.store.Load(ctx, root.Type(), aggregateID)
	if err != nil {
		return fmt.Errorf("load events for %s %s: %w", root.Type(), root.ID().String(), err)
	}

	loader, ok := root.(HistoryLoader)
	if !ok {
		//nolint:err113 // dynamic error required to include aggregate type details
		return fmt.Errorf(
			"aggregate %s %s must implement HistoryLoader for proper version tracking; "+
				"embed Core and delegate: func (a *%s) LoadEvents(events []event.Event) error { return a.Core.LoadFromHistory(a, events) }",
			root.Type(),
			root.ID().String(),
			root.Type(),
		)
	}

	err = loader.LoadEvents(events)
	if err != nil {
		return fmt.Errorf(
			"replay %d events for %s %s: %w",
			len(events),
			root.Type(),
			root.ID().String(),
			err,
		)
	}

	return nil
}
