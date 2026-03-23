package xtypes

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedAggregate provides type-safe aggregate roots with branded IDs.
type TypedAggregate struct {
	base          *aggregate.Base
	aggregateID   id.AggregateID
	aggregateType event.AggregateType
}

// NewTypedAggregate creates a new typed aggregate root.
func NewTypedAggregate(
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) *TypedAggregate {
	return &TypedAggregate{
		base:          aggregate.NewBase(aggregateID, aggregateType),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
	}
}

// ID returns the strongly-typed aggregate ID.
func (a *TypedAggregate) ID() id.AggregateID {
	return a.aggregateID
}

// Type returns the aggregate type.
func (a *TypedAggregate) Type() event.AggregateType {
	return a.aggregateType
}

// Version returns the current version.
func (a *TypedAggregate) Version() int {
	return a.base.Version()
}

// Base returns the underlying aggregate.Base for advanced operations.
func (a *TypedAggregate) Base() *aggregate.Base {
	return a.base
}

// ApplyEvent records a typed event and increments version.
func (a *TypedAggregate) ApplyEvent(ctx context.Context, evt *TypedEvent) {
	a.base.ApplyEvent(ctx, evt.Event())
}

// UncommittedChanges returns pending events.
func (a *TypedAggregate) UncommittedChanges() []event.Event {
	return a.base.UncommittedChanges()
}

// MarkChangesAsCommitted clears pending events.
func (a *TypedAggregate) MarkChangesAsCommitted() {
	a.base.MarkChangesAsCommitted()
}

// LoadFromHistory rebuilds aggregate state from events.
func (a *TypedAggregate) LoadFromHistory(events []event.Event) error {
	return a.base.LoadFromHistory(events)
}
