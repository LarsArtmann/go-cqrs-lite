package event

import (
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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

// Deprecated: Use id.AggregateType directly. This alias exists for backward
// compatibility and will be removed in v4.
type AggregateType = id.AggregateType

// Deprecated: Use id.ParseAggregateType directly. This alias exists for
// backward compatibility and will be removed in v4.
var ParseAggregateType = id.ParseAggregateType

// Event is the concrete domain-event type: a pointer to [ImmutableEvent].
//
// It is a type alias (not an interface) because ImmutableEvent is the single
// implementation — an interface here bought multiple dispatch at the cost of
// type assertions on every internal hot path. Making Event a concrete type
// removes those assertions and lets the compiler inline accessors.
//
// As before, mutable return values are safe copies:
//   - Payload() returns a clone of the internal byte slice.
//   - Metadata() returns a deep copy of the internal map.
//
// Value-type accessors (ID, Type, AggregateID, etc.) are inherently safe.
// A nil Event is a nil *ImmutableEvent; compare with == nil as usual.
type Event = *ImmutableEvent

// ImmutableEvent is the single concrete event value. Methods are defined on
// the pointer receiver *ImmutableEvent, which is what [Event] aliases.
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
