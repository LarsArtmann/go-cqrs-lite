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
//   - No external dependencies (except google/uuid, cockroachdb/errors)
//   - Files under 250 lines
//   - All exported types have Go doc comments
//   - Use errors.Is for error comparison (not assertions)
//   - Use package names that match domain (e.g., "event", not "events")
package event

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// EventType is a type identifier for domain events
type EventType string

// AggregateType is a type identifier for aggregate roots
type AggregateType string

// Event represents a domain event with rich metadata
type Event interface {
	ID() string
	Type() EventType
	AggregateID() string
	AggregateType() AggregateType
	Version() int
	Payload() []byte
	Metadata() *EventMetadata
	OccurredAt() time.Time
}

// EventMetadata contains tracing and contextual information for events
type EventMetadata struct {
	CorrelationID id.CorrelationID
	CausationID   id.CausationID
	UserID        id.UserID
	RequestID     id.RequestID
	Source        Source
	IPAddress     IPAddress
	UserAgent     UserAgent
	Custom        map[MetadataKey]string
}

// Core provides a default implementation of Event interface
type Core struct {
	id            id.EventID
	eventType     EventType
	aggregateID   id.AggregateID
	aggregateType AggregateType
	version       Version
	payload       []byte
	metadata      *EventMetadata
	occurredAt    time.Time
}

func (e *Core) ID() string                   { return e.id.String() }
func (e *Core) Type() EventType              { return e.eventType }
func (e *Core) AggregateID() string          { return e.aggregateID.String() }
func (e *Core) AggregateType() AggregateType { return e.aggregateType }
func (e *Core) Version() int                 { return e.version.Int() }
func (e *Core) Payload() []byte              { return e.payload }
func (e *Core) Metadata() *EventMetadata     { return e.metadata }
func (e *Core) OccurredAt() time.Time        { return e.occurredAt }

// NewEvent creates a new event with validation
func NewEvent(
	eventType EventType,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version int,
	payload []byte,
	opts ...EventOption,
) (*Core, error) {
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
	if aggregateID.IsEmpty() {
		return nil, fmt.Errorf(
			"aggregate ID is required (got empty) for event type %q, aggregate type %q, version %d, payload size %d",
			eventType,
			aggregateType,
			version,
			len(payload),
		)
	}
	if aggregateType == "" {
		return nil, fmt.Errorf(
			"aggregate type is required (got empty) for aggregate %q, event type %q, version %d, payload size %d",
			aggregateID,
			eventType,
			version,
			len(payload),
		)
	}

	event := &Core{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       v,
		payload:       payload,
		metadata:      &EventMetadata{},
		occurredAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(event)
	}

	return event, nil
}

// EventOption configures event creation
type EventOption func(*Core)

// WithMetadata sets custom metadata
func WithMetadata(m *EventMetadata) EventOption {
	return func(e *Core) { e.metadata = m }
}

// WithCorrelationID sets the correlation ID for distributed tracing
func WithCorrelationID(correlationID id.CorrelationID) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.CorrelationID = correlationID
	}
}

// WithCausationID sets the causation ID (indicates what triggered this event)
func WithCausationID(causationID id.CausationID) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.CausationID = causationID
	}
}

// WithUserID sets the user ID who triggered the event
func WithUserID(userID id.UserID) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.UserID = userID
	}
}

// WithRequestID sets the request ID for debugging
func WithRequestID(requestID id.RequestID) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.RequestID = requestID
	}
}

// WithSource sets the source of the event
func WithSource(source Source) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.Source = source
	}
}

// WithIPAddress sets the client IP address
func WithIPAddress(ip IPAddress) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.IPAddress = ip
	}
}

// WithUserAgent sets the client user agent
func WithUserAgent(ua UserAgent) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.UserAgent = ua
	}
}

// WithCustom sets a custom metadata field
type MetadataKey string

func WithCustom(key MetadataKey, value string) EventOption {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		if e.metadata.Custom == nil {
			e.metadata.Custom = make(map[MetadataKey]string)
		}
		e.metadata.Custom[key] = value
	}
}
