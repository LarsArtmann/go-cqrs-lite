package watermill

import (
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestTracePayloadPrefix(t *testing.T) {
	m := message.NewMessage("id-1", []byte(`{"name":"Alice"}`))
	m.Metadata.Set("stream_id", id.NewStreamID().String())
	m.Metadata.Set("event_type", "user.created")
	m.Metadata.Set("stream_type", "User")
	m.Metadata.Set("version", "1")
	evt, err := MessageToEvent("user.created", m)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("MessageToEvent: %q enc=%v\n", string(evt.Payload()), evt.Encoding())
}
