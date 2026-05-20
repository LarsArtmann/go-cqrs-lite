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
	"fmt"
	"maps"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Type is a type identifier for domain events.
type Type string

// String returns the event type as a string.
func (t Type) String() string { return string(t) }

// AggregateType is a type identifier for aggregate roots.
type AggregateType string

// String returns the aggregate type as a string.
func (a AggregateType) String() string { return string(a) }

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

// Core provides a default implementation of Event interface.
type Core struct {
	id            id.EventID
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	schemaVersion SchemaVersion
	payload       []byte
	metadata      *Metadata
	occurredAt    time.Time
}

var _ Event = (*Core)(nil)

// ID returns the event ID.
func (e *Core) ID() id.EventID { return e.id }

// Type returns the event type.
func (e *Core) Type() Type { return e.eventType }

// AggregateID returns the aggregate ID.
func (e *Core) AggregateID() id.AggregateID { return e.aggregateID }

// AggregateType returns the aggregate type.
func (e *Core) AggregateType() AggregateType { return e.aggregateType }

// Version returns the stream position of this event within the aggregate.
func (e *Core) Version() Version { return e.version }

// SchemaVersion returns the schema version of the event payload.
// Defaults to 1 for events created with NewEvent.
// Used by upcasters to determine if an event needs transformation.
func (e *Core) SchemaVersion() SchemaVersion { return e.schemaVersion }

// Payload returns the event payload. The returned slice is safe to mutate;
// the event stores its own copy at construction time.
func (e *Core) Payload() []byte {
	if e.payload == nil {
		return nil
	}

	cp := make([]byte, len(e.payload))
	copy(cp, e.payload)

	return cp
}

// Metadata returns a copy of the event metadata.
func (e *Core) Metadata() *Metadata {
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
func (e *Core) OccurredAt() time.Time { return e.occurredAt }

// NewEvent creates a new event with validation.
func NewEvent(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
	opts ...Option,
) (*Core, error) {
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

	evt := &Core{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		schemaVersion: schemaV,
		payload:       safePayload,
		metadata:      NewMetadata(),
		occurredAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(evt)
	}

	return evt, nil
}

func validateEventParams(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
) error {
	if eventType == "" {
		return fmt.Errorf(
			"%w: got empty for aggregate %q of type %q",
			ErrEmptyEventType,
			aggregateID,
			aggregateType,
		)
	}

	if aggregateID.IsZero() {
		return fmt.Errorf(
			"%w: for event type %q, aggregate type %q, version %d",
			ErrNilAggregateID,
			eventType,
			aggregateType,
			version,
		)
	}

	if aggregateType == "" {
		return fmt.Errorf(
			"%w: for aggregate %q, event type %q, version %d",
			ErrEmptyAggregateType,
			aggregateID,
			eventType,
			version,
		)
	}

	if version.IsZero() {
		return fmt.Errorf(
			"%w: for aggregate %q of type %q (event type %q, payload size %d)",
			ErrVersionNotPositive,
			aggregateID,
			aggregateType,
			eventType,
			len(payload),
		)
	}

	return nil
}
