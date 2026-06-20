package command

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

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

	Custom map[event.MetadataKey]string `json:"custom,omitempty"`
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

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom.
func EnsureCustom(m *Metadata) {
	if m.Custom == nil {
		m.Custom = make(map[event.MetadataKey]string)
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
