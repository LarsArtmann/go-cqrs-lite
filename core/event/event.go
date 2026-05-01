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
//   - No `any` types
//   - Context as first parameter
//   - Sentinels for common error states
//   - No external dependencies (except oklog/ulid, cockroachdb/errors)
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

// AggregateType is a type identifier for aggregate roots.
type AggregateType string

// Event represents a domain event with rich metadata.
type Event interface {
	ID() id.EventID
	Type() Type
	AggregateID() id.AggregateID
	AggregateType() AggregateType
	Version() int
	Payload() []byte
	Metadata() *Metadata
	OccurredAt() time.Time
}

// Metadata contains tracing and contextual information for events.
type Metadata struct {
	CorrelationID id.CorrelationID       `json:"correlationId"`
	CausationID   id.CausationID         `json:"causationId"`
	UserID        id.UserID              `json:"userId"`
	RequestID     id.RequestID           `json:"requestId"`
	Source        Source                 `json:"source,omitempty"`
	IPAddress     IPAddress              `json:"ipAddress,omitempty"`
	UserAgent     UserAgent              `json:"userAgent,omitempty"`
	Custom        map[MetadataKey]string `json:"custom,omitempty"`
}

// Core provides a default implementation of Event interface.
type Core struct {
	id            id.EventID
	eventType     Type
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	payload       []byte
	metadata      *Metadata
	occurredAt    time.Time
}

// NewMetadata creates a Metadata with all fields initialized.
func NewMetadata() *Metadata {
	return &Metadata{
		CorrelationID: id.CorrelationID{},
		CausationID:   id.CausationID{},
		UserID:        id.UserID{},
		RequestID:     id.RequestID{},
		Source:        "",
		IPAddress:     "",
		UserAgent:     "",
		Custom:        make(map[MetadataKey]string),
	}
}

// ID returns the event ID.
func (e *Core) ID() id.EventID { return e.id }

// Type returns the event type.
func (e *Core) Type() Type { return e.eventType }

// AggregateID returns the aggregate ID.
func (e *Core) AggregateID() id.AggregateID { return e.aggregateID }

// AggregateType returns the aggregate type.
func (e *Core) AggregateType() AggregateType { return e.aggregateType }

// Version returns the event version.
func (e *Core) Version() int { return e.version.Int() }

// Payload returns a copy of the event payload.
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
	version int,
	payload []byte,
	opts ...Option,
) (*Core, error) {
	if eventType == "" {
		//nolint:err113 // dynamic error required for validation
		return nil, fmt.Errorf(
			"event type is required (got empty) for aggregate %q of type %q",
			aggregateID,
			aggregateType,
		)
	}

	v, err := ParseVersion(version)
	if err != nil {
		return nil, fmt.Errorf(
			"version %d invalid for aggregate %q of type %q (event type %q, payload size %d): %w",
			version,
			aggregateID,
			aggregateType,
			eventType,
			len(payload),
			err,
		)
	}

	if aggregateID.IsZero() {
		//nolint:err113 // dynamic error required to include event details for debugging
		return nil, fmt.Errorf(
			"aggregate ID is required for event type %q, aggregate type %q, version %d",
			eventType,
			aggregateType,
			version,
		)
	}

	if aggregateType == "" {
		//nolint:err113 // dynamic error required to include event details for debugging
		return nil, fmt.Errorf(
			"aggregate type is required for aggregate %q, event type %q, version %d",
			aggregateID,
			eventType,
			version,
		)
	}

	event := &Core{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       v,
		payload:       payload,
		metadata:      NewMetadata(),
		occurredAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(event)
	}

	return event, nil
}
