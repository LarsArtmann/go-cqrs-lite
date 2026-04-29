package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Builder constructs events with a fluent API.
type Builder struct {
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	payload       []byte
	opts          []Option
}

// NewBuilder creates a new event builder with the given required fields.
func NewBuilder(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
) *Builder {
	return &Builder{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
	}
}

// WithPayload sets the event payload.
func (b *Builder) WithPayload(payload []byte) *Builder {
	b.payload = payload

	return b
}

// WithOptions adds event options (correlation ID, user ID, etc.).
func (b *Builder) WithOptions(opts ...Option) *Builder {
	b.opts = append(b.opts, opts...)

	return b
}

// WithCorrelationID adds a correlation ID for distributed tracing.
func (b *Builder) WithCorrelationID(correlationID id.CorrelationID) *Builder {
	b.opts = append(b.opts, WithCorrelationID(correlationID))

	return b
}

// WithCausationID adds a causation ID (what triggered this event).
func (b *Builder) WithCausationID(causationID id.CausationID) *Builder {
	b.opts = append(b.opts, WithCausationID(causationID))

	return b
}

// WithUserID adds the user ID who triggered the event.
func (b *Builder) WithUserID(userID id.UserID) *Builder {
	b.opts = append(b.opts, WithUserID(userID))

	return b
}

// Build creates the event.
func (b *Builder) Build() (*Core, error) {
	evt, err := NewEvent(
		b.eventType,
		b.aggregateID,
		b.aggregateType,
		b.version.Int(),
		b.payload,
		b.opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("build event %s: %w", b.eventType, err)
	}

	return evt, nil
}

// MustBuild creates the event or panics on error.
//
// WARNING: This method panics if the event cannot be built. Use only in:
//   - Test code where inputs are guaranteed valid
//   - When you explicitly want a panic on invalid input
//
// For production code, prefer Build() which returns an error.
func (b *Builder) MustBuild() *Core {
	evt, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("event.Builder.MustBuild: %v", err))
	}

	return evt
}
