package event_test

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestTracePayloadPrefix(t *testing.T) {
	payload := []byte(`{"name":"Alice"}`)

	fresh := id.NewStreamID()
	parsed, err := id.ParseStreamID(fresh.String())
	if err != nil {
		t.Fatal(err)
	}

	for name, sid := range map[string]id.StreamID{"fresh": fresh, "parsed": parsed} {
		evt, err := event.New("user.created", sid, "User", 1, payload,
			event.WithSchemaVersion(1),
			event.WithEncoding(codec.EncodingJSON),
			event.WithMetadata(event.NewMetadata()),
		)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		fmt.Printf("event.New %s-sid: %q\n", name, string(evt.Payload()))
	}
}
