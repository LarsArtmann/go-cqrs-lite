package pebble

import (
	"bytes"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// legacyEventRow is the pre-rename wire shape: the same envelope with
// aggregate_id/aggregate_type identity keys (v5 sweep §4). Decoding these
// bytes through the current reader must restore the stream identity.
type legacyEventRow struct {
	ID            id.EventID     `json:"id"`
	Type          string         `json:"type"`
	StreamID      id.StreamID    `json:"aggregate_id"`
	StreamType    string         `json:"aggregate_type"`
	Version       int            `json:"version"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Payload       []byte         `json:"payload"`
	OccurredAt    int64          `json:"occurred_at"`
	Metadata      event.Metadata `json:"metadata"`
	Encoding      string         `json:"encoding,omitempty"`
}

// TestDeserializeEvent_ReadsLegacyAggregateKeys pins the decode-only
// legacy fallback for CBOR event rows written before the stream-key
// rename (the last wire surface still carrying aggregate_* keys).
func TestDeserializeEvent_ReadsLegacyAggregateKeys(t *testing.T) {
	t.Parallel()

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	store := &EventStore{}

	row := legacyEventRow{
		ID:         id.NewEventID(),
		Type:       "task.created",
		StreamID:   streamID,
		StreamType: "Task",
		Version:    1,
		OccurredAt: time.Now().UTC().UnixNano(),
		Payload:    []byte(`{"title":"x"}`),
		Encoding:   "json",
	}

	data, err := marshalCBOROrErr(
		row,
		"pebble.test.legacy_event_marshal",
		"marshal legacy event row",
	)
	if err != nil {
		t.Fatalf("marshal legacy row: %v", err)
	}

	evt, err := store.deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserializeEvent(legacy): %v", err)
	}

	if evt.StreamID() != streamID {
		t.Fatalf("stream ID: got %v, want %v", evt.StreamID(), streamID)
	}

	if string(evt.StreamType()) != "Task" {
		t.Fatalf("stream type: got %q", evt.StreamType())
	}

	if evt.Encoding() != codec.Encoding("json") {
		t.Fatalf("encoding: got %q", evt.Encoding())
	}

	for _, banned := range []string{"aggregate_id", "aggregate_type"} {
		fresh, err := store.serializeEvent(evt)
		if err != nil {
			t.Fatalf("serializeEvent: %v", err)
		}

		if bytes.Contains(fresh, []byte(banned)) {
			t.Fatalf("re-serialized event contains banned key %q", banned)
		}
	}
}

// TestEventWireEncodingOpenNamespace pins E1's design point: the event
// wire struct stamps codec.Encoding (an OPEN namespace — custom codecs
// survive the round trip), unlike the snapshot wire's closed
// record.Encoding enum.
func TestEventWireEncodingOpenNamespace(t *testing.T) {
	t.Parallel()

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	evt, err := event.New("widget.made", streamID, "Widget", 1, map[string]any{"n": 1},
		event.WithEncoding("protobuf"))
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	store := &EventStore{}

	data, err := store.serializeEvent(evt)
	if err != nil {
		t.Fatalf("serializeEvent: %v", err)
	}

	back, err := store.deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserializeEvent: %v", err)
	}

	if back.Encoding() != "protobuf" {
		t.Fatalf("custom codec stamp lost on round trip: got %q", back.Encoding())
	}
}
