package event

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id"
)

// Metadata contains tracing and contextual information for events.
type Metadata struct {
	CorrelationID id.CorrelationID       `json:"correlationId,omitempty"`
	CausationID   id.CausationID         `json:"causationId,omitempty"`
	UserID        id.UserID              `json:"userId,omitempty"`
	RequestID     id.RequestID           `json:"requestId,omitempty"`
	Source        Source                 `json:"source,omitempty"`
	IPAddress     IPAddress              `json:"ipAddress,omitempty"`
	UserAgent     UserAgent              `json:"userAgent,omitempty"`
	Custom        map[MetadataKey]string `json:"custom,omitempty"`
}

// NewMetadata creates a Metadata with all fields initialized.
func NewMetadata() Metadata {
	return Metadata{
		Custom: make(map[MetadataKey]string),
	}
}

// Clone returns a deep copy of the metadata.
func (m Metadata) Clone() Metadata {
	cp := m

	if m.Custom != nil {
		cp.Custom = make(map[MetadataKey]string, len(m.Custom))
		maps.Copy(cp.Custom, m.Custom)
	}

	return cp
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

	for k, v := range other.Custom {
		if result.Custom == nil {
			result.Custom = make(map[MetadataKey]string)
		}

		result.Custom[k] = v
	}

	return result
}
