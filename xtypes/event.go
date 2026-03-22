package xtypes

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedEvent wraps event.Event with a strongly-typed aggregate ID.
type TypedEvent[A any] struct {
	base        *event.BaseEvent
	aggregateID id.Of[A]
}

// Base returns the underlying event.BaseEvent.
func (e *TypedEvent[A]) Base() *event.BaseEvent {
	return e.base
}

// Event returns the underlying event.Event interface.
func (e *TypedEvent[A]) Event() event.Event {
	return e.base
}

// AggregateID returns the strongly-typed aggregate ID.
func (e *TypedEvent[A]) AggregateID() id.Of[A] {
	return e.aggregateID
}

// EventBuilder constructs events with compile-time type safety.
type EventBuilder[A any] struct {
	eventType     event.EventType
	aggregateID   id.Of[A]
	aggregateType event.AggregateType
	version       int
	payload       []byte
	opts          []event.EventOption
}

// NewEventBuilder creates a new event builder with type-safe aggregate ID.
func NewEventBuilder[A any](
	eventType event.EventType,
	aggregateID id.Of[A],
	aggregateType event.AggregateType,
	version int,
) *EventBuilder[A] {
	return &EventBuilder[A]{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
	}
}

// WithPayload sets the event payload.
func (b *EventBuilder[A]) WithPayload(payload []byte) *EventBuilder[A] {
	b.payload = payload
	return b
}

// WithMetadata adds event options (correlation ID, user ID, etc.).
func (b *EventBuilder[A]) WithMetadata(opts ...event.EventOption) *EventBuilder[A] {
	b.opts = append(b.opts, opts...)
	return b
}

// WithCorrelationID adds a correlation ID for distributed tracing.
func (b *EventBuilder[A]) WithCorrelationID(correlationID string) *EventBuilder[A] {
	b.opts = append(b.opts, event.WithCorrelationID(correlationID))
	return b
}

// WithCausationID adds a causation ID (what triggered this event).
func (b *EventBuilder[A]) WithCausationID(causationID string) *EventBuilder[A] {
	b.opts = append(b.opts, event.WithCausationID(causationID))
	return b
}

// WithUserID adds the user ID who triggered the event.
func (b *EventBuilder[A]) WithUserID(userID string) *EventBuilder[A] {
	b.opts = append(b.opts, event.WithUserID(userID))
	return b
}

// Build creates the typed event.
func (b *EventBuilder[A]) Build() (*TypedEvent[A], error) {
	if b.aggregateID.IsEmpty() {
		return nil, fmt.Errorf("aggregate ID is required for event type %q", b.eventType)
	}
	if b.aggregateType == "" {
		return nil, fmt.Errorf("aggregate type is required for event type %q", b.eventType)
	}
	if b.version < 0 {
		return nil, fmt.Errorf("version must be non-negative but got %d", b.version)
	}

	base, err := event.NewEvent(
		b.eventType,
		b.aggregateID.String(),
		b.aggregateType,
		b.version,
		b.payload,
		b.opts...,
	)
	if err != nil {
		return nil, err
	}

	return &TypedEvent[A]{
		base:        base,
		aggregateID: b.aggregateID,
	}, nil
}

// MustBuild creates the typed event or panics on error.
func (b *EventBuilder[A]) MustBuild() *TypedEvent[A] {
	evt, err := b.Build()
	if err != nil {
		panic(err)
	}
	return evt
}
