package watermill

import (
	"context"
	"testing"
	"time"

	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestEventPublisher_RoundTrip(t *testing.T) {
	t.Parallel()

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{},
		nil,
	)
	defer pubSub.Close()

	// Subscribe first (GoChannel doesn't retain messages for late subscribers).
	msgs, err := pubSub.Subscribe(context.Background(), "test.roundtrip")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Publish a cqrs event through the EventPublisher.
	eventPub := NewEventPublisher(pubSub, "test.roundtrip")

	aggID := id.NewAggregateID()
	originalEvt, _ := event.NewEvent(
		event.Type("test.roundtrip.event"),
		aggID, "TestAggregate", event.Version(1),
		[]byte(`{"key":"value"}`),
	)

	err = eventPub.Publish(context.Background(), originalEvt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Receive the Watermill message and convert back to a cqrs event.
	select {
	case msg := <-msgs:
		if msg == nil {
			t.Fatal("received nil message")
		}

		decoded, err := MessageToEvent("test.roundtrip", msg)
		if err != nil {
			t.Fatalf("MessageToEvent: %v", err)
		}

		if decoded.Type() != originalEvt.Type() {
			t.Fatalf("type mismatch: %s vs %s", decoded.Type(), originalEvt.Type())
		}

		if decoded.StreamID().String() != originalEvt.StreamID().String() {
			t.Fatalf("aggregate ID mismatch: %s vs %s",
				decoded.StreamID().String(), originalEvt.StreamID().String())
		}

		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestEventPublisher_RoundTripCBOR(t *testing.T) {
	t.Parallel()

	type roundtripPayload struct {
		Key string `json:"key"`
	}

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{},
		nil,
	)
	defer pubSub.Close()

	msgs, err := pubSub.Subscribe(context.Background(), "test.roundtrip.cbor")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	eventPub := NewEventPublisher(pubSub, "test.roundtrip.cbor")

	aggID := id.NewAggregateID()
	originalEvt, err := event.New(
		event.Type("test.roundtrip.cbor.event"),
		aggID,
		"TestAggregate",
		event.Version(1),
		roundtripPayload{Key: "value"},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("create CBOR event: %v", err)
	}

	if originalEvt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("source encoding = %q, want cbor", originalEvt.Encoding())
	}

	err = eventPub.Publish(context.Background(), originalEvt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case msg := <-msgs:
		if msg == nil {
			t.Fatal("received nil message")
		}

		decoded, err := MessageToEvent("test.roundtrip.cbor", msg)
		if err != nil {
			t.Fatalf("MessageToEvent: %v", err)
		}

		if decoded.Encoding() != codec.EncodingCBOR {
			t.Fatalf("encoding = %q, want cbor", decoded.Encoding())
		}

		payload, err := event.DecodePayloadAuto[roundtripPayload](decoded)
		if err != nil {
			t.Fatalf("DecodePayloadAuto: %v", err)
		}

		if payload.Key != "value" {
			t.Fatalf("payload key = %q, want value", payload.Key)
		}

		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}
