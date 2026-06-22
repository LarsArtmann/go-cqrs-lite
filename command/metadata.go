package command

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// MetadataKey represents a custom metadata key for commands.
// It is command-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for commands.
// It embeds event.Tracing for the cross-cutting tracing identifiers and adds
// a Custom map for arbitrary key-value metadata.
//
// Unlike the old alias of event.Metadata, command.Metadata does NOT carry
// event-only concerns (Tombstone, Causation): commands have no tombstones and
// no event-causation link. Each module owns its own Metadata so a change to
// the event's shape cannot silently reshape commands. See ADR-0031.
type Metadata struct {
	event.Tracing

	Custom map[MetadataKey]string `json:"custom,omitempty"`
}

// NewMetadata creates a Metadata with zero-value fields.
// The Custom map is lazily initialized on first write via EnsureCustom.
func NewMetadata() Metadata {
	return Metadata{}
}

// Clone returns a deep copy of the metadata.
func (m Metadata) Clone() Metadata {
	cp := m
	if m.Custom != nil {
		cp.Custom = maps.Clone(m.Custom)
	}

	return cp
}

// Merge returns a new Metadata with non-zero tracing fields and all Custom
// entries from other overlaid onto m. Useful for middleware that enriches
// command metadata (e.g. correlation ID from context).
func (m Metadata) Merge(other Metadata) Metadata {
	result := m
	result.Tracing = m.Tracing.Merge(other.Tracing)

	if len(other.Custom) > 0 {
		merged := make(map[MetadataKey]string, len(result.Custom)+len(other.Custom))
		maps.Copy(merged, result.Custom)
		maps.Copy(merged, other.Custom)
		result.Custom = merged
	}

	return result
}

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom.
func EnsureCustom(m *Metadata) {
	if m.Custom == nil {
		m.Custom = make(map[MetadataKey]string)
	}
}

// Option configures command creation.
type Option func(*BasicCommand)

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
