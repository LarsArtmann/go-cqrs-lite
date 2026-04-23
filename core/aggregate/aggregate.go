// Package aggregate provides aggregate root functionality for CQRS.
package aggregate

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Root represents an aggregate root in DDD.
type Root interface {
	ID() string
	Type() event.AggregateType
	Version() int
	Apply(event.Event) error
	UncommittedChanges() []event.Event
	MarkChangesAsCommitted()
}

// Core provides common aggregate root functionality.
type Core struct {
	id            id.AggregateID
	aggregateType event.AggregateType
	version       event.Version
	changes       []event.Event
}

// NewCore creates a new aggregate core.
func NewCore(id id.AggregateID, aggregateType event.AggregateType) *Core {
	return &Core{
		id:            id,
		aggregateType: aggregateType,
		version:       0,
		changes:       make([]event.Event, 0),
	}
}

// ID returns the aggregate ID as a string.
func (a *Core) ID() string { return a.id.String() }

// Type returns the aggregate type.
func (a *Core) Type() event.AggregateType { return a.aggregateType }

// Version returns the current version.
func (a *Core) Version() int { return a.version.Int() }

// RecordEvent records an event and increments version.
func (a *Core) RecordEvent(_ context.Context, evt event.Event) {
	a.changes = append(a.changes, evt)
	a.version = a.version.Increment()
}

// UncommittedChanges returns pending events.
func (a *Core) UncommittedChanges() []event.Event {
	return a.changes
}

// MarkChangesAsCommitted clears pending events.
func (a *Core) MarkChangesAsCommitted() {
	a.changes = make([]event.Event, 0)
}

// LoadFromHistory rebuilds aggregate state by applying each event via the
// provided Root's Apply method and incrementing the version.
func (a *Core) LoadFromHistory(root Root, events []event.Event) error {
	for _, evt := range events {
		err := root.Apply(evt)
		if err != nil {
			return fmt.Errorf("apply event %s: %w", evt.Type(), err)
		}

		a.version = a.version.Increment()
	}

	return nil
}
