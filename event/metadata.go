package event

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Metadata contains tracing and contextual information for events.
// The common tracing fields (CorrelationID, CausationID, ActorID, RequestID,
// timestamps, SchemaVersion) come from [record.CommonMetadata], which is the
// single structural base shared with commands (ADR-0111 Phase 3).
type Metadata struct {
	record.CommonMetadata
	Source    Source                 `json:"source,omitempty"`
	IPAddress IPAddress              `json:"ipAddress,omitempty"`
	UserAgent UserAgent              `json:"userAgent,omitempty"`
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
	cp := m

	if m.Custom != nil {
		cp.Custom = maps.Clone(m.Custom)
	}

	if m.Causation != nil {
		c := *m.Causation
		cp.Causation = &c
	}

	return cp
}

// WithCustom returns a copy of m with the given key-value pair added to
// Custom. The original Metadata is not modified.
func (m Metadata) WithCustom(key MetadataKey, value string) Metadata {
	cp := m
	cp.Custom = maps.Clone(m.Custom)
	if cp.Custom == nil {
		cp.Custom = make(map[MetadataKey]string)
	}
	cp.Custom[key] = value
	return cp
}

// EnsureCustom lazily initializes the Custom map if nil.
// Call before writing to m.Custom from outside this package.
//
// Deprecated: Use Metadata.WithCustom, which returns a new value without
// mutating the receiver. EnsureCustom mutates in place via a pointer,
// breaking the immutability contract that Clone and Merge establish.
func EnsureCustom(m *Metadata) {
	if m.Custom == nil {
		m.Custom = make(map[MetadataKey]string)
	}
}

// Merge returns a new Metadata with non-zero fields from other overlaid onto m.
func (m Metadata) Merge(other Metadata) Metadata {
	result := m
	result.CommonMetadata = m.CommonMetadata.Merge(other.CommonMetadata)

	if other.Source != "" {
		result.Source = other.Source
	}

	if other.IPAddress != "" {
		result.IPAddress = other.IPAddress
	}

	if other.UserAgent != "" {
		result.UserAgent = other.UserAgent
	}

	if other.Causation != nil {
		c := *other.Causation
		result.Causation = &c
	}

	result.Custom = metadata.MergeCustomMaps(result.Custom, other.Custom)

	return result
}
