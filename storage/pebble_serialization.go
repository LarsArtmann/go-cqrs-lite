package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// serializeEvent converts a CQRS event to JSON.
func (a *PebbleEventStore) serializeEvent(evt event.Event) ([]byte, error) {
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
func (a *PebbleEventStore) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	err := json.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	var metadataJSON []byte
	if s.Metadata != nil {
		metadataJSON, _ = marshalMetadata(deserializeMetadata(s.Metadata))
	}

	return reconstructEvent(
		s.ID, s.Type, s.AggregateType, s.AggregateID,
		s.Version, s.SchemaVersion,
		s.Payload, metadataJSON,
		time.Unix(0, s.OccurredAt),
	)
}

func deserializeMetadata(s *serializableMetadata) *event.Metadata {
	m := event.NewMetadata()

	if s.CorrelationID != "" {
		parsed, err := id.ParseCorrelationID(s.CorrelationID)
		if err != nil {
			slog.Warn("pebble: corrupt correlation ID", "value", s.CorrelationID, "error", err)
		} else {
			m.CorrelationID = parsed
		}
	}

	if s.CausationID != "" {
		parsed, err := id.ParseCausationID(s.CausationID)
		if err != nil {
			slog.Warn("pebble: corrupt causation ID", "value", s.CausationID, "error", err)
		} else {
			m.CausationID = parsed
		}
	}

	if s.UserID != "" {
		parsed, err := id.ParseUserID(s.UserID)
		if err != nil {
			slog.Warn("pebble: corrupt user ID", "value", s.UserID, "error", err)
		} else {
			m.UserID = parsed
		}
	}

	if s.RequestID != "" {
		parsed, err := id.ParseRequestID(s.RequestID)
		if err != nil {
			slog.Warn("pebble: corrupt request ID", "value", s.RequestID, "error", err)
		} else {
			m.RequestID = parsed
		}
	}

	m.Source = event.Source(s.Source)
	m.IPAddress = event.IPAddress(s.IPAddress)
	m.UserAgent = event.UserAgent(s.UserAgent)

	if len(s.Custom) > 0 {
		m.Custom = make(map[event.MetadataKey]string, len(s.Custom))
		for k, v := range s.Custom {
			m.Custom[event.MetadataKey(k)] = v
		}
	}

	return m
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
