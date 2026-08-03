package command

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// MetadataKey represents a custom metadata key for commands.
// It is command-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for commands.
// It is a standalone struct (not a type alias) so that Clone and Merge return
// command.Metadata directly. The JSON shape is identical to the previous
// metadata.CustomData[MetadataKey] alias (ADR-0031).
//
// Unlike event.Metadata, command.Metadata does NOT carry event-only concerns
// (Tombstone, Causation): commands have no tombstones and no event-causation
// link.
type Metadata struct { //nolint:recvcheck // RO methods value, mutator pointer (math/big.Int pattern)
	metadata.Tracing

	Custom map[MetadataKey]string `json:"custom,omitempty"`
}

// Clone returns a copy of m with a cloned Custom map.
func (m Metadata) Clone() Metadata {
	return Metadata{
		Tracing: m.Tracing,
		Custom:  maps.Clone(m.Custom),
	}
}

// Merge returns a new Metadata with tracing and custom entries from other
// overlaid onto m.
func (m Metadata) Merge(other Metadata) Metadata {
	return Metadata{
		Tracing: m.Tracing.Merge(other.Tracing),
		Custom:  metadata.MergeCustomMaps(m.Custom, other.Custom),
	}
}

// EnsureCustom lazily initializes the Custom map if nil.
func (m *Metadata) EnsureCustom() {
	if m.Custom == nil {
		m.Custom = make(map[MetadataKey]string)
	}
}

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
