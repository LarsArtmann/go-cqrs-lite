package event

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Option configures event creation.
type Option func(*ImmutableEvent)

// metadataOption sets a single field on Metadata.
type metadataOption[T any] func(*Metadata, T)

// apply applies a metadataOption to a Metadata pointer.
func apply[T any](field metadataOption[T], value T) Option {
	return func(e *ImmutableEvent) {
		if e.metadata == nil {
			e.metadata = NewMetadata()
		}

		field(e.metadata, value)
	}
}

// WithEventID overrides the auto-generated event ID.
// Use for reconstructing events from storage where the original ID must be preserved.
func WithEventID(v id.EventID) Option {
	return func(e *ImmutableEvent) { e.id = v }
}

// WithOccurredAt overrides the event timestamp.
// Use for reconstructing events from storage where the original timestamp must be preserved.
func WithOccurredAt(v time.Time) Option {
	return func(e *ImmutableEvent) { e.occurredAt = v }
}

// WithMetadata merges the given metadata into the event's existing metadata.
// Existing fields are overwritten by the provided metadata.
func WithMetadata(m *Metadata) Option {
	return func(e *ImmutableEvent) {
		if e.metadata == nil {
			e.metadata = m

			return
		}

		e.metadata.mergeFrom(m)
	}
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

const (
	MetadataKeyClientID         MetadataKey = "client.id"
	MetadataKeyClientOccurredAt MetadataKey = "client.occurred_at"
)

// WithCustom sets a custom metadata field.
func WithCustom(key MetadataKey, value string) Option {
	return func(e *ImmutableEvent) {
		if e.metadata == nil {
			e.metadata = NewMetadata()
		}

		if e.metadata.Custom == nil {
			e.metadata.Custom = make(map[MetadataKey]string)
		}

		e.metadata.Custom[key] = value
	}
}

// WithSchemaVersion sets the schema version of the event payload.
// Defaults to 1. Use when reconstructing events from storage or
// when creating events with a known schema version.
func WithSchemaVersion(v SchemaVersion) Option {
	return func(e *ImmutableEvent) { e.schemaVersion = v }
}

// WithEncoding sets the encoding of the event payload.
// Defaults to [codec.EncodingJSON]. Use when reconstructing events from storage
// or when creating events with a non-JSON codec.
func WithEncoding(v codec.Encoding) Option {
	return func(e *ImmutableEvent) { e.encoding = v }
}

// WithNewCodec sets the codec used by [New] to marshal typed payloads.
// The codec's encoding is auto-stamped on the resulting event.
// Has no effect when used with [NewEvent] (which accepts pre-marshaled []byte).
func WithNewCodec(c codec.Codec) Option {
	return func(e *ImmutableEvent) { e.newCodec = c }
}

// WithClock sets the clock function used to determine OccurredAt.
// Override for deterministic testing. Without this option, events use time.Now.
func WithClock(clock Clock) Option {
	return func(e *ImmutableEvent) { e.clock = clock }
}

// WithClientID sets the client device ID in event metadata.
// Used for offline-first attribution and conflict detection.
func WithClientID(v id.ClientID) Option {
	return WithCustom(MetadataKeyClientID, v.String())
}

// WithClientOccurredAt sets the timestamp when the event occurred on the client device.
// Used for offline-first timing analysis.
func WithClientOccurredAt(t time.Time) Option {
	return WithCustom(MetadataKeyClientOccurredAt, t.Format(time.RFC3339Nano))
}

// WithDeadline sets the event's deadline for cancellation propagation.
// Handlers can use Event.Context() to check if the deadline has passed.
func WithDeadline(t time.Time) Option {
	return func(e *ImmutableEvent) { e.deadline = t }
}

// FromContext extracts the deadline from the given context and sets it on the event.
// If the context has no deadline, this is a no-op.
// This allows propagating cancellation/deadline from the caller's context to event handlers.
func FromContext(ctx context.Context) Option {
	deadline, ok := ctx.Deadline()
	if !ok {
		return func(*ImmutableEvent) {} // no-op
	}

	return WithDeadline(deadline)
}
