package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// serializeEvent converts a CQRS event to JSON.
func (a *CQRSAdapter) serializeEvent(evt event.Event) ([]byte, error) {
	s := serializableEvent{
		ID:            evt.ID().String(),
		Type:          string(evt.Type()),
		AggregateID:   evt.AggregateID().String(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       evt.Payload(),
		OccurredAt:    evt.OccurredAt().UnixNano(),
	}

	if m := evt.Metadata(); m != nil {
		s.Metadata = &serializableMetadata{
			CorrelationID: m.CorrelationID.String(),
			CausationID:   m.CausationID.String(),
			UserID:        m.UserID.String(),
			RequestID:     m.RequestID.String(),
			Source:        string(m.Source),
			IPAddress:     string(m.IPAddress),
			UserAgent:     string(m.UserAgent),
		}
		if len(m.Custom) > 0 {
			s.Metadata.Custom = make(map[string]string, len(m.Custom))
			for k, v := range m.Custom {
				s.Metadata.Custom[string(k)] = v
			}
		}
	}

	return json.Marshal(s)
}

// deserializeEvent converts JSON to a CQRS-compatible event.
func (a *CQRSAdapter) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	err := json.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	aggregateID, err := id.ParseAggregateID(s.AggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse aggregate ID: %w", err)
	}

	var opts []event.Option

	eventID, err := id.ParseEventID(s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event ID: %w", err)
	}

	opts = append(
		opts,
		event.WithEventID(eventID),
		event.WithOccurredAt(time.Unix(0, s.OccurredAt)),
	)

	if s.Metadata != nil {
		m := event.NewMetadata()
		if s.Metadata.CorrelationID != "" {
			m.CorrelationID, _ = id.ParseCorrelationID(s.Metadata.CorrelationID)
		}

		if s.Metadata.CausationID != "" {
			m.CausationID, _ = id.ParseCausationID(s.Metadata.CausationID)
		}

		if s.Metadata.UserID != "" {
			m.UserID, _ = id.ParseUserID(s.Metadata.UserID)
		}

		if s.Metadata.RequestID != "" {
			m.RequestID, _ = id.ParseRequestID(s.Metadata.RequestID)
		}

		m.Source = event.Source(s.Metadata.Source)
		m.IPAddress = event.IPAddress(s.Metadata.IPAddress)

		m.UserAgent = event.UserAgent(s.Metadata.UserAgent)
		if len(s.Metadata.Custom) > 0 {
			m.Custom = make(map[event.MetadataKey]string, len(s.Metadata.Custom))
			for k, v := range s.Metadata.Custom {
				m.Custom[event.MetadataKey(k)] = v
			}
		}

		opts = append(opts, event.WithMetadata(m))
	}

	if s.SchemaVersion > 0 {
		opts = append(opts, event.WithSchemaVersion(event.SchemaVersion(s.SchemaVersion)))
	}

	return event.NewEvent(
		event.Type(s.Type),
		aggregateID,
		event.AggregateType(s.AggregateType),
		event.Version(s.Version),
		s.Payload,
		opts...,
	)
}

// serializableEvent represents the JSON storage format for events.
type serializableEvent struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	AggregateID   string                `json:"aggregate_id"`
	AggregateType string                `json:"aggregate_type"`
	Version       int                   `json:"version"`
	SchemaVersion int                   `json:"schema_version,omitempty"`
	Payload       []byte                `json:"payload"`
	OccurredAt    int64                 `json:"occurred_at"`
	Metadata      *serializableMetadata `json:"metadata,omitempty"`
}

type serializableMetadata struct {
	CorrelationID string            `json:"correlation_id,omitempty"`
	CausationID   string            `json:"causation_id,omitempty"`
	UserID        string            `json:"user_id,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	Source        string            `json:"source,omitempty"`
	IPAddress     string            `json:"ip_address,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	Custom        map[string]string `json:"custom,omitempty"`
}
