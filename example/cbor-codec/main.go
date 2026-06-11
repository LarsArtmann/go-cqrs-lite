// Binary event payloads with CBOR encoding.
//
// This example shows how to use CBORCodec for binary event payloads.
// CBOR produces smaller, faster, and deterministic output compared to JSON.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type UserCreated struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()
	aggID := id.NewAggregateID()

	// Create an event with CBOR encoding.
	// The payload is marshaled using CBORCodec instead of the default JSONCodec.
	evt, err := event.New(
		"UserCreated",
		aggID,
		"User",
		1,
		UserCreated{Name: "Alice", Email: "alice@example.com"},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Event encoding: %s\n", evt.Encoding())
	fmt.Printf(
		"Payload bytes:  %d (CBOR is typically 20-40%% smaller than JSON)\n",
		len(evt.Payload()),
	)

	// Decode the payload back using the same codec.
	decoded, err := event.DecodePayload[UserCreated](evt, codec.CBORCodec{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded name:   %s\n", decoded.Name)
	fmt.Printf("Decoded email:  %s\n", decoded.Email)

	// CBOR encoding is deterministic — same payload always produces same bytes.
	// This makes it safe for HMAC/Ed25519 signing.
	_ = ctx
}
