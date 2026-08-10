package command

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// MetadataKey represents a custom metadata key for commands.
// It is command-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for commands.
// The common tracing fields come from [record.CommonMetadata], which is the
// single structural base shared with events (ADR-0111 Phase 3).
//
// Unlike event.Metadata, command.Metadata does NOT carry event-only concerns
// (Causation): commands have no event-causation link.
type Metadata struct {
	record.CommonMetadata

	Custom map[MetadataKey]string `json:"custom,omitempty"`
}

// Clone returns a copy of m with a cloned Custom map.
func (m Metadata) Clone() Metadata {
	return Metadata{
		CommonMetadata: m.CommonMetadata,
		Custom:         maps.Clone(m.Custom),
	}
}

// Merge returns a new Metadata with tracing and custom entries from other
// overlaid onto m.
func (m Metadata) Merge(other Metadata) Metadata {
	return Metadata{
		CommonMetadata: m.CommonMetadata.Merge(other.CommonMetadata),
		Custom:         metadata.MergeCustomMaps(m.Custom, other.Custom),
	}
}

// WithCustom returns a copy of m with the given key-value pair added to
// Custom. The original Metadata is not modified.
func (m Metadata) WithCustom(key MetadataKey, value string) Metadata {
	custom := maps.Clone(m.Custom)
	if custom == nil {
		custom = make(map[MetadataKey]string)
	}

	custom[key] = value

	return Metadata{
		CommonMetadata: m.CommonMetadata,
		Custom:         custom,
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

// WithUserID sets the user who issued the command as the ActorID.
func WithUserID(v id.UserID) Option {
	return func(c *BasicCommand) { c.metadata.ActorID = id.NewUserActor(v) }
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
