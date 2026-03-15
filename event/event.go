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

	"github.com/google/uuid"
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
	CorrelationID string
	CausationID   string
	UserID        string
	RequestID     string
	Source        string
	IPAddress     string
	UserAgent     string
	Custom        map[string]string
}

// BaseEvent provides a default implementation of Event interface
type BaseEvent struct {
	id            string
	eventType     EventType
	aggregateID   string
	aggregateType AggregateType
	version       int
	payload       []byte
	metadata      *EventMetadata
	occurredAt    time.Time
}

func (e *BaseEvent) ID() string                   { return e.id }
func (e *BaseEvent) Type() EventType              { return e.eventType }
func (e *BaseEvent) AggregateID() string          { return e.aggregateID }
func (e *BaseEvent) AggregateType() AggregateType { return e.aggregateType }
func (e *BaseEvent) Version() int                 { return e.version }
func (e *BaseEvent) Payload() []byte              { return e.payload }
func (e *BaseEvent) Metadata() *EventMetadata     { return e.metadata }
func (e *BaseEvent) OccurredAt() time.Time        { return e.occurredAt }

// NewEvent creates a new event with validation
func NewEvent(
	eventType EventType,
	aggregateID string,
	aggregateType AggregateType,
	version int,
	payload []byte,
	opts ...EventOption,
) (*BaseEvent, error) {
	if aggregateID == "" {
		return nil, fmt.Errorf("aggregate ID is required for event type %q", eventType)
	}
	if aggregateType == "" {
		return nil, fmt.Errorf("aggregate type is required for aggregate %q (event type %q)", aggregateID, eventType)
	}
	if version < 0 {
		return nil, fmt.Errorf("version must be non-negative but got %d for aggregate %q (event type %q)", version, aggregateID, eventType)
	}

	event := &BaseEvent{
		id:            uuid.New().String(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
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
type EventOption func(*BaseEvent)

// WithMetadata sets custom metadata
func WithMetadata(m *EventMetadata) EventOption {
	return func(e *BaseEvent) { e.metadata = m }
}

// WithCorrelationID sets the correlation ID for distributed tracing
func WithCorrelationID(id string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.CorrelationID = id
	}
}

// WithCausationID sets the causation ID (indicates what triggered this event)
func WithCausationID(id string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.CausationID = id
	}
}

// WithUserID sets the user ID who triggered the event
func WithUserID(id string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.UserID = id
	}
}

// WithRequestID sets the request ID for debugging
func WithRequestID(id string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.RequestID = id
	}
}

// WithSource sets the source of the event
func WithSource(source string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.Source = source
	}
}

// WithIPAddress sets the client IP address
func WithIPAddress(ip string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.IPAddress = ip
	}
}

// WithUserAgent sets the client user agent
func WithUserAgent(ua string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		e.metadata.UserAgent = ua
	}
}

// WithCustom sets a custom metadata field
func WithCustom(key, value string) EventOption {
	return func(e *BaseEvent) {
		if e.metadata == nil {
			e.metadata = &EventMetadata{}
		}
		if e.metadata.Custom == nil {
			e.metadata.Custom = make(map[string]string)
		}
		e.metadata.Custom[key] = value
	}
}
