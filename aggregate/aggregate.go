package aggregate

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Root represents an aggregate root in DDD
type Root interface {
	ID() string
	Type() event.AggregateType
	Version() int
	Apply(event.Event) error
	UncommittedChanges() []event.Event
	MarkChangesAsCommitted()
}

// Base provides common aggregate root functionality
type Base struct {
	id            id.AggregateID
	aggregateType event.AggregateType
	version       event.Version
	changes       []event.Event
}

// NewBase creates a new aggregate base
func NewBase(id id.AggregateID, aggregateType event.AggregateType) *Base {
	return &Base{
		id:            id,
		aggregateType: aggregateType,
		version:       0,
		changes:       make([]event.Event, 0),
	}
}

func (a *Base) ID() string                { return a.id.String() }
func (a *Base) Type() event.AggregateType { return a.aggregateType }
func (a *Base) Version() int              { return a.version.Int() }

// ApplyEvent records an event and increments version
func (a *Base) ApplyEvent(_ context.Context, evt event.Event) {
	a.changes = append(a.changes, evt)
	a.version = a.version.Increment()
}

// UncommittedChanges returns pending events
func (a *Base) UncommittedChanges() []event.Event {
	return a.changes
}

// MarkChangesAsCommitted clears pending events
func (a *Base) MarkChangesAsCommitted() {
	a.changes = make([]event.Event, 0)
}

// LoadFromHistory rebuilds aggregate state from events
func (a *Base) LoadFromHistory(events []event.Event) error {
	for range events {
		a.version = a.version.Increment()
	}
	return nil
}
