package event

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
)

// Metadata contains tracing and contextual information for events.
// art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
type Metadata struct {
	metadata.Tracing

	Source    Source         `json:"source,omitempty"`
	IPAddress IPAddress      `json:"ipAddress,omitempty"`
	UserAgent UserAgent      `json:"userAgent,omitempty"`
	Tombstone *TombstoneMark `json:"tombstone,omitempty"`
	// Causation carries the command-causation twin.
	//
	// Deprecated: removed with the v4 train (T26). The canonical home is the
	// event's cause field (ImmutableEvent.Cause); this twin is populated in
	// lockstep during the dual-support window.
	Causation *Causation             `json:"causation,omitempty"`
	Custom    map[MetadataKey]string `json:"custom,omitempty"`
}

// NewMetadata creates a Metadata with zero-value fields.
// The Custom map is lazily initialized on first write via WithCustom.
func NewMetadata() Metadata {
	return Metadata{}
}

// Clone returns a deep copy of the metadata.
func (m Metadata) Clone() Metadata {
	clone := m

	if m.Custom != nil {
		clone.Custom = maps.Clone(m.Custom)
	}

	if m.Tombstone != nil {
		mark := *m.Tombstone
		clone.Tombstone = &mark
	}

	if m.Causation != nil {
		c := *m.Causation
		clone.Causation = &c
	}

	return clone
}

// WithCustom returns a copy of m with the given key-value pair added to
// Custom. The original Metadata is not modified.
func (m Metadata) WithCustom(key MetadataKey, value string) Metadata {
	clone := m

	clone.Custom = maps.Clone(m.Custom)
	if clone.Custom == nil {
		clone.Custom = make(map[MetadataKey]string)
	}

	clone.Custom[key] = value

	return clone
}

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom from outside this package.
//
// Deprecated: Use Metadata.WithCustom, which returns a new value without
// mutating the receiver. EnsureCustom mutates in place via a pointer,
// breaking the immutability contract that Clone and Merge establish.
// Removed at v5.
func EnsureCustom(m *Metadata) {
	if m.Custom == nil {
		m.Custom = make(map[MetadataKey]string)
	}
}

// Merge returns a new Metadata with non-zero fields from other overlaid onto m.
func (m Metadata) Merge(other Metadata) Metadata {
	result := m
	result.Tracing = m.Tracing.Merge(other.Tracing)

	if other.Source != "" {
		result.Source = other.Source
	}

	if other.IPAddress != "" {
		result.IPAddress = other.IPAddress
	}

	if other.UserAgent != "" {
		result.UserAgent = other.UserAgent
	}

	if other.Tombstone != nil {
		mark := *other.Tombstone
		result.Tombstone = &mark
	}

	if other.Causation != nil {
		c := *other.Causation
		result.Causation = &c
	}

	result.Custom = metadata.MergeCustomMaps(result.Custom, other.Custom)

	return result
}
