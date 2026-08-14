package command

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// MetadataKey represents a custom metadata key for commands.
// It is command-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for commands.
// It is a type alias for [metadata.Metadata] keyed by the command-local
// MetadataKey, so Clone, Merge, and WithCustom are inherited from the
// canonical generic (ADR-0031, WAL unification). The JSON shape is
// unchanged: {"correlationId": ..., "custom": {...}}.
//
// Unlike event.Metadata, command.Metadata does NOT carry event-only concerns
// (Tombstone, Causation): commands have no tombstones and no event-causation
// link.
type Metadata = metadata.Metadata[MetadataKey]

// Option configures command creation.
type Option func(*BasicCommand)

// WithCommandID overrides the auto-minted command ID. Use this for idempotency:
// pass the same ID when retrying a logical command so the receiver can dedup.
func WithCommandID(v id.CommandID) Option {
	return func(c *BasicCommand) { c.commandID = v }
}

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(v id.CorrelationID) Option {
	return func(c *BasicCommand) { c.metadata.CorrelationID = v }
}

// WithCausationID sets the causation ID (indicates what triggered this command).
func WithCausationID(v id.CausationID) Option {
	return func(c *BasicCommand) { c.metadata.CausationID = v }
}

// WithUserID sets the user ID who issued the command.
func WithUserID(v id.UserID) Option {
	return func(c *BasicCommand) { c.metadata.UserID = v }
}

// WithActor sets the effective actor (user, bot, system, or service) that
// issued the command. This is the primary audit-trail field for compliance.
func WithActor(v id.ActorID) Option {
	return func(c *BasicCommand) { c.metadata.ActorID = v }
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return func(c *BasicCommand) { c.metadata.RequestID = v }
}

// WithCustomMetadata sets a custom metadata key-value pair on the command.
// Multiple calls accumulate. Used by transport adapters to carry wire-level
// metadata (e.g. gRPC payload, correlation context).
func WithCustomMetadata(key, value string) Option {
	return func(c *BasicCommand) {
		c.metadata = c.metadata.WithCustom(MetadataKey(key), value)
	}
}
