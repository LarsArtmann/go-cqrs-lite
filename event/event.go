package event

import (
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Type is a type identifier for domain events.
type Type string

// String returns the event type as a string.
func (t Type) String() string { return string(t) }

// IsZero reports whether the event type is empty.
func (t Type) IsZero() bool { return t == "" }

// ParseType validates and returns a Type. Returns an error if empty.
func ParseType(s string) (Type, error) {
	if s == "" {
		return "", ErrEmptyEventType
	}

	return Type(s), nil
}

// AggregateType is a type identifier for aggregate roots.
type AggregateType string

// String returns the aggregate type as a string.
func (a AggregateType) String() string { return string(a) }

// IsZero reports whether the aggregate type is empty.
func (a AggregateType) IsZero() bool { return a == "" }

// ParseAggregateType validates and returns an AggregateType.
// Returns an error if empty.
func ParseAggregateType(s string) (AggregateType, error) {
	if s == "" {
		return "", ErrEmptyAggregateType
	}

	return AggregateType(s), nil
}

// MustParseAggregateType is like ParseAggregateType but panics on invalid input.
func MustParseAggregateType(s string) AggregateType {
	t, err := ParseAggregateType(s)
	if err != nil {
		panic("event.MustParseAggregateType: " + err.Error())
	}

	return t
}

// Event represents an immutable domain event with rich metadata.
//
// Implementations must ensure that mutable return values ([]byte, maps, slices)
// are safe copies — callers must be unable to mutate the event's internal state
// through the returned reference. Specifically:
//
//   - Payload() must return a copy of the internal byte slice.
//   - Metadata() must return a deep copy of the internal map.
//
// Value-type accessors (ID, Type, AggregateID, etc.) are inherently safe since
// they return by value.
type Event interface {
	ID() id.EventID
	Type() Type
	AggregateID() id.AggregateID
	AggregateType() AggregateType
	Version() Version
	SchemaVersion() SchemaVersion
	Encoding() codec.Encoding
	Payload() []byte
	Metadata() Metadata
	OccurredAt() time.Time
	Deadline() (time.Time, bool)
}

// ImmutableEvent provides a default implementation of Event interface.
type ImmutableEvent struct {
	id            id.EventID
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	schemaVersion SchemaVersion
	encoding      codec.Encoding
	payload       []byte
	metadata      Metadata
	occurredAt    time.Time
	opts          *eventOptions
}

type eventOptions struct {
	clock    Clock
	newCodec codec.Codec
	deadline time.Time
}

var _ Event = (*ImmutableEvent)(nil)

// ID returns the event ID.
func (e *ImmutableEvent) ID() id.EventID { return e.id }

// Type returns the event type.
func (e *ImmutableEvent) Type() Type { return e.eventType }

// AggregateID returns the aggregate ID.
func (e *ImmutableEvent) AggregateID() id.AggregateID { return e.aggregateID }

// AggregateType returns the aggregate type.
func (e *ImmutableEvent) AggregateType() AggregateType { return e.aggregateType }

// Version returns the stream position of this event within the aggregate.
func (e *ImmutableEvent) Version() Version { return e.version }

// SchemaVersion returns the schema version of the event payload.
// Defaults to 1 for events created with NewEvent.
// Used by upcasters to determine if an event needs transformation.
func (e *ImmutableEvent) SchemaVersion() SchemaVersion { return e.schemaVersion }

// Encoding returns the serialization format used for the event payload.
// Defaults to [codec.EncodingJSON] for events created with [NewEvent].
func (e *ImmutableEvent) Encoding() codec.Encoding {
	if e.encoding == "" {
		return codec.EncodingJSON
	}

	return e.encoding
}

// Payload returns a copy of the event payload. The returned slice is safe to
// modify; mutations will not affect the event's internal state.
func (e *ImmutableEvent) Payload() []byte {
	return slices.Clone(e.payload)
}

// Metadata returns a copy of the event's metadata. The returned value is
// safe to modify; mutations will not affect the event's internal state.
func (e *ImmutableEvent) Metadata() Metadata {
	return e.metadata.Clone()
}

// OccurredAt returns when the event occurred.
func (e *ImmutableEvent) OccurredAt() time.Time { return e.occurredAt }

// Deadline returns the event's deadline (if any).
func (e *ImmutableEvent) Deadline() (time.Time, bool) {
	if e.opts == nil {
		return time.Time{}, false
	}

	return e.opts.deadline, !e.opts.deadline.IsZero()
}

// String returns a human-readable representation of the event for logging and debugging.
func (e *ImmutableEvent) String() string {
	return fmt.Sprintf("%s(%s) v%d %s@%s",
		e.eventType, e.id, e.version, e.aggregateType, e.aggregateID)
}
