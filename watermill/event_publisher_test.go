package watermill

import (
	"context"
	"testing"
	"time"

	gochannel "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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

		if decoded.AggregateID().String() != originalEvt.AggregateID().String() {
			t.Fatalf("aggregate ID mismatch: %s vs %s",
				decoded.AggregateID().String(), originalEvt.AggregateID().String())
		}

		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}
