package event

import (
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

// NewMetadata creates a Metadata with all fields initialized.
func NewMetadata() *Metadata {
	return &Metadata{
		CorrelationID: id.CorrelationID{},
		CausationID:   id.CausationID{},
		UserID:        id.UserID{},
		RequestID:     id.RequestID{},
		Source:        "",
		IPAddress:     "",
		UserAgent:     "",
		Custom:        make(map[MetadataKey]string),
	}
}

func (m *Metadata) mergeFrom(other *Metadata) {
	if !other.CorrelationID.IsZero() {
		m.CorrelationID = other.CorrelationID
	}

	if !other.CausationID.IsZero() {
		m.CausationID = other.CausationID
	}

	if !other.UserID.IsZero() {
		m.UserID = other.UserID
	}

	if !other.RequestID.IsZero() {
		m.RequestID = other.RequestID
	}

	if other.Source != "" {
		m.Source = other.Source
	}

	if other.IPAddress != "" {
		m.IPAddress = other.IPAddress
	}

	if other.UserAgent != "" {
		m.UserAgent = other.UserAgent
	}

	for k, v := range other.Custom {
		if m.Custom == nil {
			m.Custom = make(map[MetadataKey]string)
		}

		m.Custom[k] = v
	}
}
