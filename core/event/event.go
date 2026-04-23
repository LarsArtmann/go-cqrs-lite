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

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Type is a type identifier for domain events.
type Type string

// AggregateType is a type identifier for aggregate roots.
type AggregateType string

// Event represents a domain event with rich metadata.
type Event interface {
	ID() string
	Type() Type
	AggregateID() string
	AggregateType() AggregateType
	Version() int
	Payload() []byte
	Metadata() *Metadata
	OccurredAt() time.Time
}

// Metadata contains tracing and contextual information for events.
type Metadata struct {
	CorrelationID id.CorrelationID
	CausationID   id.CausationID
	UserID        id.UserID
	RequestID     id.RequestID
	Source        Source
	IPAddress     IPAddress
	UserAgent     UserAgent
	Custom        map[MetadataKey]string
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
		CorrelationID: "",
		CausationID:   "",
		UserID:        "",
		RequestID:     "",
		Source:        "",
		IPAddress:     "",
		UserAgent:     "",
		Custom:        make(map[MetadataKey]string),
	}
}

// ID returns the event ID.
func (e *Core) ID() string { return e.id.String() }

// Type returns the event type.
func (e *Core) Type() Type { return e.eventType }

// AggregateID returns the aggregate ID.
func (e *Core) AggregateID() string { return e.aggregateID.String() }

// AggregateType returns the aggregate type.
func (e *Core) AggregateType() AggregateType { return e.aggregateType }

// Version returns the event version.
func (e *Core) Version() int { return e.version.Int() }

// Payload returns the event payload.
func (e *Core) Payload() []byte { return e.payload }

// Metadata returns the event metadata.
func (e *Core) Metadata() *Metadata { return e.metadata }

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
		//nolint:err113 // dynamic error required to include event details for debugging
		return nil, fmt.Errorf(
			"aggregate ID is required (got empty) for event type %q, aggregate type %q, version %d, payload size %d, opts count: %d",
			eventType,
			aggregateType,
			version,
			len(payload),
			len(opts),
		)
	}

	if aggregateType == "" {
		//nolint:err113 // dynamic error required to include event details for debugging
		return nil, fmt.Errorf(
			"aggregate type is required (got empty) for aggregate %q, event type %q, version %d, payload size %d, opts count: %d",
			aggregateID,
			eventType,
			version,
			len(payload),
			len(opts),
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

// Option configures event creation.
type Option func(*Core)

// ensureMetadata initializes metadata if nil.
func (e *Core) ensureMetadata() {
	if e.metadata == nil {
		e.metadata = NewMetadata()
	}
}

// WithMetadata sets custom metadata.
func WithMetadata(m *Metadata) Option {
	return func(e *Core) { e.metadata = m }
}

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(correlationID id.CorrelationID) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.CorrelationID = correlationID
	}
}

// WithCausationID sets the causation ID (indicates what triggered this event).
func WithCausationID(causationID id.CausationID) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.CausationID = causationID
	}
}

// WithUserID sets the user ID who triggered the event.
func WithUserID(userID id.UserID) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.UserID = userID
	}
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(requestID id.RequestID) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.RequestID = requestID
	}
}

// WithSource sets the source of the event.
func WithSource(source Source) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.Source = source
	}
}

// WithIPAddress sets the client IP address.
func WithIPAddress(ip IPAddress) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.IPAddress = ip
	}
}

// WithUserAgent sets the client user agent.
func WithUserAgent(ua UserAgent) Option {
	return func(e *Core) {
		e.ensureMetadata()
		e.metadata.UserAgent = ua
	}
}

// MetadataKey represents a custom metadata key.
type MetadataKey string

// WithCustom sets a custom metadata field.
func WithCustom(key MetadataKey, value string) Option {
	return func(e *Core) {
		e.ensureMetadata()

		if e.metadata.Custom == nil {
			e.metadata.Custom = make(map[MetadataKey]string)
		}

		e.metadata.Custom[key] = value
	}
}
