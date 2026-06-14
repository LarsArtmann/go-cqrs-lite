package event

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Metadata contains tracing and contextual information for events.
type Metadata struct {
	CorrelationID id.CorrelationID       `json:"correlationId"`
	CausationID   id.CausationID         `json:"causationId"`
	UserID        id.UserID              `json:"userId"`
	RequestID     id.RequestID           `json:"requestId"`
	Source        Source                 `json:"source,omitempty"`
	IPAddress     IPAddress              `json:"ipAddress,omitempty"`
	UserAgent     UserAgent              `json:"userAgent,omitempty"`
	Custom        map[MetadataKey]string `json:"custom,omitempty"`
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
// Call before writing to m.Custom from outside this package.
func EnsureCustom(m *Metadata) {
	if m.Custom == nil {
		m.Custom = make(map[MetadataKey]string)
	}
}

// Merge returns a new Metadata with non-zero fields from other overlaid onto m.
func (m Metadata) Merge(other Metadata) Metadata {
	result := m

	if !other.CorrelationID.IsZero() {
		result.CorrelationID = other.CorrelationID
	}

	if !other.CausationID.IsZero() {
		result.CausationID = other.CausationID
	}

	if !other.UserID.IsZero() {
		result.UserID = other.UserID
	}

	if !other.RequestID.IsZero() {
		result.RequestID = other.RequestID
	}

	if other.Source != "" {
		result.Source = other.Source
	}

	if other.IPAddress != "" {
		result.IPAddress = other.IPAddress
	}

	if other.UserAgent != "" {
		result.UserAgent = other.UserAgent
	}

	if len(other.Custom) > 0 {
		EnsureCustom(&result)

		for k, v := range other.Custom {
			result.Custom[k] = v
		}
	}

	return result
}
