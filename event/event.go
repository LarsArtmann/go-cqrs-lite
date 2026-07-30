package event

import (
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

// NewTypeSet converts a slice of event types into a set for O(1) membership
// checks. Returns a non-nil empty map when types is empty.
func NewTypeSet(types []Type) map[Type]struct{} {
	set := make(map[Type]struct{}, len(types))

	for _, t := range types {
		set[t] = struct{}{}
	}

	return set
}

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
// Value-type accessors (ID, Type, StreamID, etc.) are inherently safe.
// A nil Event is a nil *ImmutableEvent; compare with == nil as usual.
type Event = *ImmutableEvent

// ImmutableEvent is the single concrete event value. Methods are defined on
// the pointer receiver *ImmutableEvent, which is what [Event] aliases.
type ImmutableEvent struct {
	id            id.EventID
	eventType     Type
	streamID      id.StreamID
	streamType    id.StreamType
	version       Version
	schemaVersion SchemaVersion
	encoding      codec.Encoding
	payload       []byte
	metadata      Metadata
	//cqrs-lint:ignore(C013) library code or intentional pattern
	occurredAt    time.Time
	opts          *eventOptions
}

type eventOptions struct {
	clock    Clock
	newCodec codec.Codec
	deadline time.Time
}

// ID returns the event ID.
//cqrs-lint:ignore(A001,E005) library code or intentional pattern
func (e *ImmutableEvent) ID() id.EventID { return e.id }

// Type returns the event type.
func (e *ImmutableEvent) Type() Type { return e.eventType }

// StreamID returns the stream identifier.
func (e *ImmutableEvent) StreamID() id.StreamID { return e.streamID }

// StreamType returns the stream type.
func (e *ImmutableEvent) StreamType() id.StreamType { return e.streamType }

// Deprecated: use StreamID.
func (e *ImmutableEvent) AggregateID() id.StreamID { return e.streamID }

// Deprecated: use StreamType.
func (e *ImmutableEvent) AggregateType() id.StreamType { return e.streamType }

// Version returns the stream position of this event within the stream.
func (e *ImmutableEvent) Version() Version { return e.version }

// SchemaVersion returns the schema version of the event payload.
// Defaults to 1 for events created with NewEvent.
// Used by upcasters to determine if an event needs transformation.
func (e *ImmutableEvent) SchemaVersion() SchemaVersion { return e.schemaVersion }

// Encoding returns the serialization format used for the event payload.
// Defaults to [codec.EncodingJSON] for events created with [NewEvent] (raw
// bytes path). Events created with [New] are auto-stamped from the codec used.
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
		e.eventType, e.id, e.version, e.streamType, e.streamID)
}
