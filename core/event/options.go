package event

import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// Option configures event creation.
type Option func(*Core)

// metadataOption sets a single field on Metadata.
type metadataOption[T any] func(*Metadata, T)

// apply applies a metadataOption to a Metadata pointer.
func apply[T any](field metadataOption[T], value T) Option {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = NewMetadata()
		}

		field(e.metadata, value)
	}
}

// WithMetadata sets custom metadata.
func WithMetadata(m *Metadata) Option {
	return func(e *Core) { e.metadata = m }
}

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(v id.CorrelationID) Option {
	return apply(func(m *Metadata, v id.CorrelationID) { m.CorrelationID = v }, v)
}

// WithCausationID sets the causation ID (indicates what triggered this event).
func WithCausationID(v id.CausationID) Option {
	return apply(func(m *Metadata, v id.CausationID) { m.CausationID = v }, v)
}

// WithUserID sets the user ID who triggered the event.
func WithUserID(v id.UserID) Option {
	return apply(func(m *Metadata, v id.UserID) { m.UserID = v }, v)
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return apply(func(m *Metadata, v id.RequestID) { m.RequestID = v }, v)
}

// WithSource sets the source of the event.
func WithSource(v Source) Option {
	return apply(func(m *Metadata, v Source) { m.Source = v }, v)
}

// WithIPAddress sets the client IP address.
func WithIPAddress(v IPAddress) Option {
	return apply(func(m *Metadata, v IPAddress) { m.IPAddress = v }, v)
}

// WithUserAgent sets the client user agent.
func WithUserAgent(v UserAgent) Option {
	return apply(func(m *Metadata, v UserAgent) { m.UserAgent = v }, v)
}

// MetadataKey represents a custom metadata key.
type MetadataKey string

// WithCustom sets a custom metadata field.
func WithCustom(key MetadataKey, value string) Option {
	return func(e *Core) {
		if e.metadata == nil {
			e.metadata = NewMetadata()
		}

		if e.metadata.Custom == nil {
			e.metadata.Custom = make(map[MetadataKey]string)
		}

		e.metadata.Custom[key] = value
	}
}
