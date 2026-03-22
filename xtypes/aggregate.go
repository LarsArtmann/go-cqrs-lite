package xtypes

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedAggregate provides type-safe aggregate roots with branded IDs.
type TypedAggregate[A any] struct {
	base          *aggregate.Base
	aggregateID   id.Of[A]
	aggregateType event.AggregateType
}

// NewTypedAggregate creates a new typed aggregate root.
func NewTypedAggregate[A any](
	aggregateID id.Of[A],
	aggregateType event.AggregateType,
) *TypedAggregate[A] {
	return &TypedAggregate[A]{
		base:          aggregate.NewBase(aggregateID.String(), aggregateType),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
	}
}

// ID returns the strongly-typed aggregate ID.
func (a *TypedAggregate[A]) ID() id.Of[A] {
	return a.aggregateID
}

// Type returns the aggregate type.
func (a *TypedAggregate[A]) Type() event.AggregateType {
	return a.aggregateType
}

// Version returns the current version.
func (a *TypedAggregate[A]) Version() int {
	return a.base.Version()
}

// Base returns the underlying aggregate.Base for advanced operations.
func (a *TypedAggregate[A]) Base() *aggregate.Base {
	return a.base
}

// ApplyEvent records a typed event and increments version.
func (a *TypedAggregate[A]) ApplyEvent(ctx context.Context, evt *TypedEvent[A]) {
	a.base.ApplyEvent(ctx, evt.Event())
}

// UncommittedChanges returns pending events.
func (a *TypedAggregate[A]) UncommittedChanges() []event.Event {
	return a.base.UncommittedChanges()
}

// MarkChangesAsCommitted clears pending events.
func (a *TypedAggregate[A]) MarkChangesAsCommitted() {
	a.base.MarkChangesAsCommitted()
}

// LoadFromHistory rebuilds aggregate state from events.
func (a *TypedAggregate[A]) LoadFromHistory(events []event.Event) error {
	return a.base.LoadFromHistory(events)
}
