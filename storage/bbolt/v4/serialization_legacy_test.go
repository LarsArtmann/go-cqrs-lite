package bbolt

import (
	"bytes"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// legacyRow is the pre-rename wire shape: the same envelope with
// aggregate_id/aggregate_type identity keys (v5 sweep §4). Decoding these
// bytes through the current readers must restore the stream identity —
// journals written before the rename stay readable.
type legacyRow struct {
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

// TestDeserializeEvent_ReadsLegacyAggregateKeys pins the decode-only legacy
// fallback: CBOR rows written with aggregate_id/aggregate_type decode with
// their stream identity intact.
func TestDeserializeEvent_ReadsLegacyAggregateKeys(t *testing.T) {
	t.Parallel()

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	row := legacyRow{
		ID:         id.NewEventID(),
		Type:       "user.created",
		StreamID:   streamID,
		StreamType: "User",
		Version:    1,
		Payload:    []byte(`{"n":1}`),
		OccurredAt: time.Now().UTC().UnixNano(),
	}

	data, err := marshalCBOR(row)
	if err != nil {
		t.Fatalf("marshal legacy row: %v", err)
	}

	evt, err := deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserializeEvent(legacy): %v", err)
	}

	if evt.StreamID() != streamID {
		t.Fatalf("stream ID: got %v, want %v", evt.StreamID(), streamID)
	}

	if string(evt.StreamType()) != "User" {
		t.Fatalf("stream type: got %q", evt.StreamType())
	}
}

// TestSerializeEvent_WritesStreamKeys pins the writer side of the rename:
// fresh rows carry ONLY the stream_id/stream_type keys — the aggregate
// spellings must never come back.
func TestSerializeEvent_WritesStreamKeys(t *testing.T) {
	t.Parallel()

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	evt, err := event.New("user.created", streamID, "User", 1, map[string]int{"n": 1})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	data, err := serializeEvent(evt)
	if err != nil {
		t.Fatalf("serializeEvent: %v", err)
	}

	for _, banned := range []string{"aggregate_id", "aggregate_type"} {
		if bytes.Contains(data, []byte(banned)) {
			t.Fatalf("serialized event contains banned key %q", banned)
		}
	}

	if !bytes.Contains(data, []byte("stream_id")) {
		t.Fatal("serialized event lacks stream_id key")
	}
}
