package watermill_test

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

func TestPublisherAdapter_Publish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	adapter := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{"type":"user.created"}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "1")

	if err := adapter.Publish("user.created", msg); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestPublisherAdapter_Close(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	adapter := wm.NewPublisherAdapter(bus)

	if err := adapter.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
