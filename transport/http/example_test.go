package http_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ExampleCBORToJSONTransform demonstrates the one-liner consumer path for
// serving JSON to browser SSE clients while storing events in compact CBOR.
// The transform is schema-free: it decodes CBOR generically and re-encodes
// as JSON without needing the original Go struct type.
func ExampleCBORToJSONTransform() {
	// Events are created with CBOR encoding (the default codec).
	type UserCreated struct {
		Name string `cbor:"name"`
	}

	streamID := id.NewStreamID()
	evt, err := event.New("user.created", streamID, "User", 1,
		UserCreated{Name: "alice"})
	if err != nil {
		panic(err)
	}

	// CBORToJSONTransform converts the CBOR payload to JSON bytes.
	// Wire it once: NewSSEBroker(bus, WithPayloadTransform(CBORToJSONTransform))
	jsonBytes := cqrshttp.CBORToJSONTransform(evt)

	fmt.Println(evt.Encoding() == codec.EncodingCBOR)
	fmt.Println(string(jsonBytes))

	// Output:
	// true
	// {"name":"alice"}
}
