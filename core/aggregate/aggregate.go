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
	ID() id.AggregateID
	Type() event.AggregateType
	Version() event.Version
	SetVersion(v event.Version)
	Apply(evt event.Event) error
	ApplySnapshot(state []byte) error
	UncommittedChanges() []event.Event
	MarkChangesAsCommitted()
	LoadEvents(events []event.Event) error
}

// Core provides common aggregate root functionality.
type Core struct {
	id            id.AggregateID
	aggregateType event.AggregateType
	version       event.Version
	changes       []event.Event
}

// NewCore creates a new aggregate core.
// Returns an error if the ID is zero or the aggregate type is empty.
func NewCore(id id.AggregateID, aggregateType event.AggregateType) (*Core, error) {
	if id.IsZero() {
		return nil, ErrNilAggregateID
	}

	if aggregateType == "" {
		return nil, ErrEmptyAggregateType
	}

	return &Core{
		id:            id,
		aggregateType: aggregateType,
		version:       0,
		changes:       make([]event.Event, 0),
	}, nil
}

// MustNewCore is like NewCore but panics on error.
// For use in tests and examples where inputs are known valid.
func MustNewCore(id id.AggregateID, aggregateType event.AggregateType) *Core {
	c, err := NewCore(id, aggregateType)
	if err != nil {
		panic(fmt.Sprintf("aggregate.MustNewCore: %v", err))
	}

	return c
}

// ID returns the aggregate ID.
func (a *Core) ID() id.AggregateID { return a.id }

// Type returns the aggregate type.
func (a *Core) Type() event.AggregateType { return a.aggregateType }

// Version returns the current version.
func (a *Core) Version() event.Version {
	return a.version
}

// SetVersion sets the aggregate version directly.
// Used by repositories when loading aggregates from snapshots.
func (a *Core) SetVersion(v event.Version) {
	a.version = v
}

// RecordEvent records an event and increments version.
func (a *Core) RecordEvent(_ context.Context, evt event.Event) {
	a.changes = append(a.changes, evt)
	a.version = a.version.Increment()
}

// UncommittedChanges returns a copy of pending events.
func (a *Core) UncommittedChanges() []event.Event {
	result := make([]event.Event, len(a.changes))
	copy(result, a.changes)

	return result
}

// MarkChangesAsCommitted clears pending events while reusing the backing array.
func (a *Core) MarkChangesAsCommitted() {
	a.changes = a.changes[:0]
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
