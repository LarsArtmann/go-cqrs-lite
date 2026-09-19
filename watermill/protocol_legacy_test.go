package watermill

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestMessageToEvent_ReadsLegacyAggregateMetadata pins the dual-read window
// (v5 sweep §4): messages published before the stream-key rename carry only
// the aggregate_id/aggregate_type spellings and must keep decoding.
func TestMessageToEvent_ReadsLegacyAggregateMetadata(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()

	msg := message.NewMessage(id.NewEventID().String(), []byte(`{}`))
	msg.Metadata.Set(metaEventID, id.NewEventID().String())
	msg.Metadata.Set(metaEventType, "order.placed")
	msg.Metadata.Set(metaLegacyAggregateID, streamID.String())
	msg.Metadata.Set(metaLegacyAggregateType, "Order")
	msg.Metadata.Set(metaVersion, "1")

	evt, err := MessageToEvent("order.placed", msg)
	if err != nil {
		t.Fatalf("messageToEvent(legacy metadata): %v", err)
	}

	if evt.StreamID() != streamID {
		t.Fatalf("stream ID: got %v, want %v", evt.StreamID(), streamID)
	}

	if string(evt.StreamType()) != "Order" {
		t.Fatalf("stream type: got %q", evt.StreamType())
	}
}

// TestEventToMessage_DualWritesStreamKeys pins the dual-write window: fresh
// messages carry BOTH the stream_* keys and the legacy aggregate spellings
// so pre-rename readers in rolling deployments keep working.
func TestEventToMessage_DualWritesStreamKeys(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()

	evt, err := event.New("order.placed", streamID, "Order", 1, map[string]int{"n": 1})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	msg := eventToMessage(evt)

	for key, want := range map[string]string{
		metaStreamID:            streamID.String(),
		metaStreamType:          "Order",
		metaLegacyAggregateID:   streamID.String(),
		metaLegacyAggregateType: "Order",
	} {
		if got := msg.Metadata.Get(key); got != want {
			t.Fatalf("metadata %s: got %q, want %q", key, got, want)
		}
	}
}
