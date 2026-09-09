package pebble

import (
	"bytes"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// legacyCommandRow is the pre-rename wire shape: the same envelope with
// aggregate_id/aggregate_type identity keys (v5 sweep §4). Decoding these
// bytes through the current reader must restore the stream identity.
type legacyCommandRow struct {
	ID         id.CommandID     `json:"id"`
	Type       string           `json:"type"`
	StreamID   id.StreamID      `json:"aggregate_id"`
	StreamType string           `json:"aggregate_type"`
	ReceivedAt int64            `json:"received_at"`
	Payload    []byte           `json:"payload"`
	Metadata   command.Metadata `json:"metadata"`
}

// TestDeserializeCommand_ReadsLegacyAggregateKeys pins the decode-only
// legacy fallback for CBOR rows written before the stream-key rename.
func TestDeserializeCommand_ReadsLegacyAggregateKeys(t *testing.T) {
	t.Parallel()

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	store := &CommandStore{}

	row := legacyCommandRow{
		ID:         id.NewCommandID(),
		Type:       "task.create",
		StreamID:   streamID,
		StreamType: "Task",
		ReceivedAt: time.Now().UTC().UnixNano(),
		Payload:    []byte(`{"title":"x"}`),
	}

	data, err := marshalCBOROrErr(row, "pebble.test.legacy_marshal", "marshal legacy row")
	if err != nil {
		t.Fatalf("marshal legacy row: %v", err)
	}

	cmd, err := store.deserializeCommand(data)
	if err != nil {
		t.Fatalf("deserializeCommand(legacy): %v", err)
	}

	if cmd.StreamID() != streamID {
		t.Fatalf("stream ID: got %v, want %v", cmd.StreamID(), streamID)
	}

	if string(cmd.StreamType()) != "Task" {
		t.Fatalf("stream type: got %q", cmd.StreamType())
	}

	for _, banned := range []string{"aggregate_id", "aggregate_type"} {
		fresh, err := store.serializeCommand(cmd)
		if err != nil {
			t.Fatalf("serializeCommand: %v", err)
		}

		if bytes.Contains(fresh, []byte(banned)) {
			t.Fatalf("re-serialized command contains banned key %q", banned)
		}
	}
}
