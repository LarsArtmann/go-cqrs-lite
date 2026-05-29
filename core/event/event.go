// Package event provides the core domain event types for building blocks for
// CQRS and Event Sourcing patterns.
//
// Key design principles:
//   - Immutable event data with rich metadata
//   - Type-safe event identification
//   - Context-aware operations
//   - No panics, explicit error handling
//
// Reference: ChastityAPI event Store, Cyberdom EventBus
// HOW_TO_GOLANG.md coding standards
//   - Max 250 lines per file
//   - Max 30 lines per function
//   - No unnecessary `any` types (Codec/Query use `any` where dynamically required)
//   - Context as first parameter
//   - Sentinels for common error states
//   - No external dependencies (except oklog/ulid)
//   - Files under 250 lines
//   - All exported types have Go doc comments
//   - Use errors.Is for error comparison (not assertions)
//   - Use package names that match domain (e.g., "event", not "events")
package event

import (
	"maps"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

// MustParseType is like ParseType but panics on invalid input.
func MustParseType(s string) Type {
	t, err := ParseType(s)
	if err != nil {
		panic("event.MustParseType: " + err.Error())
	}

	return t
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

// Event represents a domain event with rich metadata.
type Event interface {
	ID() id.EventID
	Type() Type
	AggregateID() id.AggregateID
	AggregateType() AggregateType
	Version() Version
	SchemaVersion() SchemaVersion
	Payload() []byte
	Metadata() *Metadata
	OccurredAt() time.Time
}

// ImmutableEvent provides a default implementation of Event interface.
type ImmutableEvent struct {
	id            id.EventID
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	schemaVersion SchemaVersion
	payload       []byte
	metadata      *Metadata
	occurredAt    time.Time
	clock         Clock
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

// Payload returns the event payload. The returned slice is safe to mutate;
// the event stores its own copy at construction time.
func (e *ImmutableEvent) Payload() []byte {
	if e.payload == nil {
		return nil
	}

	cp := slices.Clone(e.payload)

	return cp
}

// Metadata returns a copy of the event metadata.
func (e *ImmutableEvent) Metadata() *Metadata {
	if e.metadata == nil {
		return nil
	}

	metadataCopy := *e.metadata

	if e.metadata.Custom != nil {
		metadataCopy.Custom = make(map[MetadataKey]string, len(e.metadata.Custom))

		maps.Copy(metadataCopy.Custom, e.metadata.Custom)
	}

	return &metadataCopy
}

// OccurredAt returns when the event occurred.
func (e *ImmutableEvent) OccurredAt() time.Time { return e.occurredAt }

// Clone returns a deep copy of the event. The returned event is fully independent —
// mutations to its payload or metadata will not affect the original.
func (e *ImmutableEvent) Clone() *ImmutableEvent {
	var payloadCopy []byte
	if e.payload != nil {
		payloadCopy = make([]byte, len(e.payload))
		copy(payloadCopy, e.payload)
	}

	return &ImmutableEvent{
		id:            e.id,
		eventType:     e.eventType,
		aggregateID:   e.aggregateID,
		aggregateType: e.aggregateType,
		version:       e.version,
		schemaVersion: e.schemaVersion,
		payload:       payloadCopy,
		metadata:      e.Metadata(),
		occurredAt:    e.occurredAt,
		clock:         e.clock,
	}
}

// NewEvent creates a new event with validation.
func NewEvent(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
	opts ...Option,
) (*ImmutableEvent, error) {
	err := validateEventParams(
		eventType,
		aggregateID,
		aggregateType,
		version,
		payload,
	)
	if err != nil {
		return nil, err
	}

	var safePayload []byte
	if payload != nil {
		safePayload = make([]byte, len(payload))
		copy(safePayload, payload)
	}

	schemaV, _ := ParseSchemaVersion(1)

	clk := defaultClock

	evt := &ImmutableEvent{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		schemaVersion: schemaV,
		payload:       safePayload,
		metadata:      NewMetadata(),
		occurredAt:    time.Time{},
		clock:         clk,
	}

	for _, opt := range opts {
		opt(evt)
	}

	if evt.occurredAt.IsZero() {
		evt.occurredAt = evt.clock()
	}

	return evt, nil
}
