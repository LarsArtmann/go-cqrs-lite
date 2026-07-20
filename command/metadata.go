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
// It embeds metadata.Tracing for the cross-cutting tracing identifiers and adds
// a Custom map for arbitrary key-value metadata.
//
// Unlike the old alias of event.Metadata, command.Metadata does NOT carry
// event-only concerns (Tombstone, Causation): commands have no tombstones and
// no event-causation link. Each module owns its own Metadata so a change to
// the event's shape cannot silently reshape commands. See ADR-0031.
type Metadata struct {
	metadata.CustomData[MetadataKey]
}

// NewMetadata creates a Metadata with zero-value fields.
// The Custom map is lazily initialized on first write via EnsureCustom.
func NewMetadata() Metadata { return Metadata{} }

// Clone returns a deep copy of the metadata.
//
// This wrapper exists because Go has no covariant return types: the embedded
// CustomData.Clone returns CustomData[MetadataKey], not command.Metadata.
// The same boilerplate appears in query/query.go — see ADR-0031. Extracting
// to a generic helper would require more callback parameters than these
// one-line wrappers save, so the duplication is intentional.
func (m Metadata) Clone() Metadata {
	return Metadata{CustomData: m.CustomData.Clone()}
}

// Merge returns a new Metadata with non-zero tracing fields and all Custom
// entries from other overlaid onto m. Useful for middleware that enriches
// command metadata (e.g. correlation ID from context).
//
// Same Go no-covariant-returns constraint as Clone — see comment above.
func (m Metadata) Merge(other Metadata) Metadata {
	return Metadata{CustomData: m.CustomData.Merge(other.CustomData)}
}

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom.
func EnsureCustom(m *Metadata) {
	m.EnsureCustom()
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
		EnsureCustom(&c.metadata)
		c.metadata.Custom[MetadataKey(key)] = value
	}
}
