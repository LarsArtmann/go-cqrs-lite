package watermill_test

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/memory"
	wm "github.com/larsartmann/go-cqrs-lite/watermill"
)

func TestPublisherAdapter_Publish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	adapter := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{"type":"user.created"}`))

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
