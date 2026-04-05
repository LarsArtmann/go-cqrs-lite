package xtypes

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedAggregate provides type-safe aggregate roots with branded IDs.
type TypedAggregate struct {
	core          *aggregate.Core
	aggregateID   id.AggregateID
	aggregateType event.AggregateType
}

// NewTypedAggregate creates a new typed aggregate root.
func NewTypedAggregate(
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
) *TypedAggregate {
	return &TypedAggregate{
		core:          aggregate.NewCore(aggregateID, aggregateType),
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
	return a.core.Version()
}

// Core returns the underlying aggregate.Core for advanced operations.
func (a *TypedAggregate) Core() *aggregate.Core {
	return a.core
}

// RecordEvent records a typed event and increments version.
func (a *TypedAggregate) RecordEvent(ctx context.Context, evt *TypedEvent) {
	a.core.RecordEvent(ctx, evt.Event())
}

// UncommittedChanges returns pending events.
func (a *TypedAggregate) UncommittedChanges() []event.Event {
	return a.core.UncommittedChanges()
}

// MarkChangesAsCommitted clears pending events.
func (a *TypedAggregate) MarkChangesAsCommitted() {
	a.core.MarkChangesAsCommitted()
}

// LoadFromHistory rebuilds aggregate state by applying each event via root.Apply.
func (a *TypedAggregate) LoadFromHistory(root aggregate.Root, events []event.Event) error {
	err := a.core.LoadFromHistory(root, events)
	if err != nil {
		return fmt.Errorf("load typed aggregate %s from history: %w", a.Type(), err)
	}

	return nil
}
