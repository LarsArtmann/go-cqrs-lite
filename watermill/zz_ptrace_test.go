package watermill

import (
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func runNew(t *testing.T, payload []byte) string {
	evt, err := event.New("user.created", id.NewStreamID(), "User", 1, payload,
		event.WithSchemaVersion(1), event.WithEncoding(codec.EncodingJSON), event.WithMetadata(event.NewMetadata()))
	if err != nil {
		t.Fatal(err)
	}
	return string(evt.Payload())
}

func TestTracePayloadPrefix(t *testing.T) {
	m1 := message.NewMessage("id-1", []byte(`{"name":"Alice"}`))
	fmt.Printf("no-set: %q\n", runNew(t, m1.Payload))

	m2 := message.NewMessage("id-1", []byte(`{"name":"Alice"}`))
	m2.Metadata.Set("stream_id", id.NewStreamID().String())
	fmt.Printf("with-set: %q\n", runNew(t, m2.Payload))

	m3 := message.NewMessage("id-1", []byte(`{"name":"Alice"}`))
	m3.Metadata.Set("unrelated", "x")
	fmt.Printf("set-unrelated: %q\n", runNew(t, m3.Payload))
}
