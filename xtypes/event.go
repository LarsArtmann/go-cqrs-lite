package xtypes

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// TypedEvent wraps event.Event with a strongly-typed aggregate ID.
type TypedEvent struct {
	base        *event.Core
	aggregateID id.AggregateID
}

// Core returns the underlying event.Core.
func (e *TypedEvent) Core() *event.Core {
	return e.base
}

// Event returns the underlying event.Event interface.
func (e *TypedEvent) Event() event.Event {
	return e.base
}

// AggregateID returns the strongly-typed aggregate ID.
func (e *TypedEvent) AggregateID() id.AggregateID {
	return e.aggregateID
}

// EventBuilder constructs events with compile-time type safety.
type EventBuilder struct {
	eventType     event.Type
	aggregateID   id.AggregateID
	aggregateType event.AggregateType
	version       event.Version
	payload       []byte
	opts          []event.Option
}

// NewEventBuilder creates a new event builder with type-safe aggregate ID.
func NewEventBuilder(
	eventType event.Type,
	aggregateID id.AggregateID,
	aggregateType event.AggregateType,
	version event.Version,
) *EventBuilder {
	return &EventBuilder{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		payload:       nil,
		opts:          nil,
	}
}

// WithPayload sets the event payload.
func (b *EventBuilder) WithPayload(payload []byte) *EventBuilder {
	b.payload = payload

	return b
}

// WithMetadata adds event options (correlation ID, user ID, etc.).
func (b *EventBuilder) WithMetadata(opts ...event.Option) *EventBuilder {
	b.opts = append(b.opts, opts...)

	return b
}

// WithCorrelationID adds a correlation ID for distributed tracing.
func (b *EventBuilder) WithCorrelationID(correlationID id.CorrelationID) *EventBuilder {
	b.opts = append(b.opts, event.WithCorrelationID(correlationID))

	return b
}

// WithCausationID adds a causation ID (what triggered this event).
func (b *EventBuilder) WithCausationID(causationID id.CausationID) *EventBuilder {
	b.opts = append(b.opts, event.WithCausationID(causationID))

	return b
}

// WithUserID adds the user ID who triggered the event.
func (b *EventBuilder) WithUserID(userID id.UserID) *EventBuilder {
	b.opts = append(b.opts, event.WithUserID(userID))

	return b
}

// Build creates the typed event.
func (b *EventBuilder) Build() (*TypedEvent, error) {
	if b.aggregateID.IsEmpty() {
		return nil, fmt.Errorf(
			"aggregate ID is required for event type %q (aggregate type %q)",
			b.eventType,
			b.aggregateType,
		)
	}

	if b.aggregateType == "" {
		return nil, fmt.Errorf(
			"aggregate type is required for event type %q (aggregate ID %q)",
			b.eventType,
			b.aggregateID.String(),
		)
	}

	if b.version.Int() < 0 {
		return nil, fmt.Errorf(
			"version must be non-negative but got %d for event type %q (aggregate %q of type %q)",
			b.version.Int(),
			b.eventType,
			b.aggregateID.String(),
			b.aggregateType,
		)
	}

	base, err := event.NewEvent(
		b.eventType,
		b.aggregateID,
		b.aggregateType,
		b.version.Int(),
		b.payload,
		b.opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("build typed event %s: %w", b.eventType, err)
	}

	return &TypedEvent{
		base:        base,
		aggregateID: b.aggregateID,
	}, nil
}

// MustBuild creates the typed event or panics on error.
//
// WARNING: This method panics if the event cannot be built. Use only in:
//   - Test code where inputs are guaranteed valid
//   - When you explicitly want a panic on invalid input
//
// For production code, prefer Build() which returns an error.
func (b *EventBuilder) MustBuild() *TypedEvent {
	evt, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("EventBuilder.MustBuild: %v", err))
	}

	return evt
}
