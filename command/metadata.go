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
// It is a type alias for metadata.CustomData[MetadataKey] so that Clone, Merge,
// and EnsureCustom are inherited directly — no per-module wrapper methods are
// needed (the previous wrapper struct existed only to retype Clone/Merge
// returns, which Go's lack of covariant return types forced; the alias removes
// that constraint entirely). See ADR-0031.
//
// Unlike the old alias of event.Metadata, command.Metadata does NOT carry
// event-only concerns (Tombstone, Causation): commands have no tombstones and
// no event-causation link. Each module owns its own Metadata type so a change
// to the event's shape cannot silently reshape commands.
type Metadata = metadata.CustomData[MetadataKey]

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

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return func(c *BasicCommand) { c.metadata.RequestID = v }
}

// WithCustomMetadata sets a custom metadata key-value pair on the command.
// Multiple calls accumulate. Used by transport adapters to carry wire-level
// metadata (e.g. gRPC payload, correlation context).
func WithCustomMetadata(key, value string) Option {
	return func(c *BasicCommand) {
		c.metadata.EnsureCustom()
		c.metadata.Custom[MetadataKey(key)] = value
	}
}
