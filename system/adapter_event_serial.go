package system

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// serializedEvent is the JSON envelope for persisting events in SQL-based
// StreamLogBackends. The Memory engine stores pointers directly; SQL engines
// store this envelope as a TEXT value.
type serializedEvent struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	StreamID      string    `json:"stream_id"`
	StreamType    string    `json:"stream_type"`
	Version       int       `json:"version"`
	SchemaVersion int       `json:"schema_version"`
	Payload       []byte    `json:"payload"`
	Encoding      string    `json:"encoding"`
	Metadata      []byte    `json:"metadata"`
	OccurredAt    time.Time `json:"occurred_at"`
}

func (a *EventAdapter) encodeEvent(evt event.Event) string {
	metaJSON, _ := event.MarshalMetadataJSON(evt.Metadata(), "system")

	env := serializedEvent{
		ID:            evt.ID().String(),
		Type:          string(evt.Type()),
		StreamID:      evt.StreamID().String(),
		StreamType:    string(evt.StreamType()),
		Version:       evt.Version().Int(),
		SchemaVersion: evt.SchemaVersion().Int(),
		Payload:       event.PayloadReadOnly(evt),
		Encoding:      string(evt.Encoding()),
		Metadata:      metaJSON,
		OccurredAt:    evt.OccurredAt(),
	}

	data, _ := json.Marshal(env)

	return string(data)
}

func (a *EventAdapter) decodeEvent(s string) (event.Event, error) {
	var env serializedEvent
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, fmt.Errorf("event adapter: decode envelope: %w", err)
	}

	eventID, err := id.ParseEventID(env.ID)
	if err != nil {
		return nil, fmt.Errorf("event adapter: parse event ID: %w", err)
	}

	streamID, err := id.ParseStreamID(env.StreamID)
	if err != nil {
		return nil, fmt.Errorf("event adapter: parse stream ID: %w", err)
	}

	evt, err := event.ReconstructEventFromFields(
		eventID,
		event.Type(env.Type),
		id.StreamType(env.StreamType),
		streamID,
		env.Version, env.SchemaVersion,
		env.Payload, env.Metadata,
		env.OccurredAt,
		codec.Encoding(env.Encoding),
		"system",
	)
	if err != nil {
		return nil, fmt.Errorf("event adapter: reconstruct event: %w", err)
	}

	return evt, nil
}

// ─── encode/decode helpers ───

func (a *EventAdapter) eventsToAny(events []event.Event) []any {
	if !a.serialize {
		result := make([]any, len(events))
		for i, evt := range events {
			result[i] = evt
		}

		return result
	}

	result := make([]any, len(events))
	for i, evt := range events {
		result[i] = a.encodeEvent(evt)
	}

	return result
}

func (a *EventAdapter) anyToEvents(values []any) ([]event.Event, error) {
	result := make([]event.Event, 0, len(values))
	for _, val := range values {
		evt, err := a.decodeValue(val)
		if err != nil {
			return nil, err
		}

		result = append(result, evt)
	}

	return result, nil
}

func (a *EventAdapter) decodeValue(val any) (event.Event, error) {
	// Direct pointer (Memory engine).
	if evt, ok := val.(event.Event); ok {
		return evt, nil
	}

	// Serialized string (SQLite/Pebble engine, raw string passthrough).
	if s, ok := val.(string); ok {
		return a.decodeEvent(s)
	}

	// Decoded JSON map (SQLite engine auto-decodes JSON strings on read).
	// Re-marshal to JSON and decode as a serializedEvent envelope.
	if m, ok := val.(map[string]any); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("event adapter: re-marshal decoded value: %w", err)
		}

		return a.decodeEvent(string(data))
	}

	return nil, fmt.Errorf("%w: %T", ErrUnsupportedValueType, val)
}
