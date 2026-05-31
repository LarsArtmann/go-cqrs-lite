package pebble

import (
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

// serializeEvent converts a CQRS event to JSON.
func (a *PebbleEventStore) serializeEvent(evt event.Event) ([]byte, error) {
	s := serializableEvent{
		ID:            evt.ID(),
		Type:          string(evt.Type()),
		AggregateID:   evt.AggregateID(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       evt.Payload(),
		OccurredAt:    evt.OccurredAt().UnixNano(),
		Metadata:      evt.Metadata(),
		Encoding:      string(evt.Encoding()),
	}

	return json.Marshal(s) //nolint:wrapcheck // storage serialization, not domain error
}

// deserializeEvent converts JSON to a CQRS-compatible event.
func (a *PebbleEventStore) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	err := json.Unmarshal(data, &s)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_event",
			"failed to unmarshal event")
	}

	metadataJSON, _ := marshalMetadata(s.Metadata)

	return event.ReconstructEventFromFields(
		s.ID, s.Type, s.AggregateType, s.AggregateID,
		s.Version, s.SchemaVersion,
		s.Payload, metadataJSON,
		time.Unix(0, s.OccurredAt),
		codec.Encoding(s.Encoding),
		"pebble",
	)
}

// serializableEvent represents the JSON storage format for events.
type serializableEvent struct {
	ID            id.EventID     `json:"id"`
	Type          string         `json:"type"`
	AggregateID   id.AggregateID `json:"aggregate_id"`   //nolint:tagliatelle // on-disk format uses snake_case
	AggregateType string         `json:"aggregate_type"` //nolint:tagliatelle
	Version       int            `json:"version"`
	SchemaVersion int            `json:"schema_version,omitempty"` //nolint:tagliatelle
	Payload       []byte         `json:"payload"`
	OccurredAt    int64          `json:"occurred_at"`        //nolint:tagliatelle
	Metadata      event.Metadata `json:"metadata,omitempty"` //nolint:modernize // intentional: omit zero-value metadata
	Encoding      string         `json:"encoding,omitempty"`
}
