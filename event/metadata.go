package event

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// Metadata contains tracing and contextual information for events.
type Metadata struct {
	metadata.Tracing
	Source    Source                 `json:"source,omitempty"`
	IPAddress IPAddress              `json:"ipAddress,omitempty"`
	UserAgent UserAgent              `json:"userAgent,omitempty"`
	Tombstone *TombstoneMark         `json:"tombstone,omitempty"`
	Causation *Causation             `json:"causation,omitempty"`
	Custom    map[MetadataKey]string `json:"custom,omitempty"`
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

	if m.Tombstone != nil {
		mark := *m.Tombstone
		cp.Tombstone = &mark
	}

	if m.Causation != nil {
		c := *m.Causation
		cp.Causation = &c
	}

	return cp
}

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom from outside this package.
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
