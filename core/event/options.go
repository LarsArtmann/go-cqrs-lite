package event

import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// Option configures event creation.
type Option func(*Core)

// ensureMetadata initializes metadata if nil.
func (e *Core) ensureMetadata() {
	if e.metadata == nil {
		e.metadata = NewMetadata()
	}
}

// setMetadataField calls ensureMetadata and sets a metadata field.
func (e *Core) setMetadataField(field func(*Metadata)) {
	e.ensureMetadata()
	field(e.metadata)
}

// WithMetadata sets custom metadata.
func WithMetadata(m *Metadata) Option {
	return func(e *Core) { e.metadata = m }
}

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(correlationID id.CorrelationID) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.CorrelationID = correlationID })
	}
}

// WithCausationID sets the causation ID (indicates what triggered this event).
func WithCausationID(causationID id.CausationID) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.CausationID = causationID })
	}
}

// WithUserID sets the user ID who triggered the event.
func WithUserID(userID id.UserID) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.UserID = userID })
	}
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(requestID id.RequestID) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.RequestID = requestID })
	}
}

// WithSource sets the source of the event.
func WithSource(source Source) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.Source = source })
	}
}

// WithIPAddress sets the client IP address.
func WithIPAddress(ip IPAddress) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.IPAddress = ip })
	}
}

// WithUserAgent sets the client user agent.
func WithUserAgent(ua UserAgent) Option {
	return func(e *Core) {
		e.setMetadataField(func(m *Metadata) { m.UserAgent = ua })
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
