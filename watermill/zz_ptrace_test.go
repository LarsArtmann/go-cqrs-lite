package watermill

import (
	"encoding/hex"
	"fmt"
	"slices"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestTracePayloadPrefix(t *testing.T) {
	sid := id.NewStreamID().String()
	parsed, _ := id.ParseStreamID(sid)

	mk := func() *message.Message {
		m := message.NewMessage("id-1", []byte(`{"name":"Alice"}`))
		m.Metadata.Set("stream_id", sid)
		return m
	}

	m := mk()
	fmt.Printf("m.Payload before: %s\n", hex.Dump(m.Payload))

	// F: cloned watermill slice
	evtF, err := event.New("user.created", parsed, "User", 1, slices.Clone(m.Payload),
		event.WithSchemaVersion(1), event.WithEncoding(codec.EncodingJSON), event.WithMetadata(event.NewMetadata()))
	if err == nil {
		fmt.Printf("F cloned-slice: %q\n", string(evtF.Payload()))
	}

	// G: direct watermill slice
	evtG, err := event.New("user.created", parsed, "User", 1, m.Payload,
		event.WithSchemaVersion(1), event.WithEncoding(codec.EncodingJSON), event.WithMetadata(event.NewMetadata()))
	if err == nil {
		fmt.Printf("G direct-slice: %q hex: %s\n", string(evtG.Payload()), hex.Dump(evtG.Payload()))
	}
}
