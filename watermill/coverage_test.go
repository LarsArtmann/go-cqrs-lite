package watermill_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
	wm "github.com/larsartmann/go-cqrs-lite/watermill"
)

func TestMessageToEvent_MissingAggregateType(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("version", "1")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for missing aggregate_type")
	}
}

func TestMessageToEvent_EmptyVersion(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for missing version")
	}
}

func TestMessageToEvent_InvalidSchemaVersion(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("schema_version", "not-a-number")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for invalid schema_version")
	}
}

func TestMessageToEvent_InvalidEventID(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("event_id", "not-a-valid-event-id")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for invalid event_id")
	}
}

func TestMessageToEvent_InvalidOccurredAt(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("occurred_at", "not-a-timestamp")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for invalid occurred_at")
	}
}

func TestMessageToEvent_NoCustomMetadata(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close()

	subscriber := wm.NewSubscriberAdapter(bus)
	msgCh, err := subscriber.Subscribe(context.Background(), "user.created")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	evt, err := event.NewEvent(
		"user.created",
		id.NewAggregateID(),
		"User",
		1,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case received := <-msgCh:
		if received.Metadata.Get("event_type") != "user.created" {
			t.Errorf("event_type = %q", received.Metadata.Get("event_type"))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestPublisherAdapter_PublishFails(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	bus.PublishErr = errors.New("publish failed")
	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "1")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for failing publisher")
	}
}

func TestPublisherAdapter_CloseNonClosable(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus()
	publisher := wm.NewPublisherAdapter(bus)

	if err := publisher.Close(); err != nil {
		t.Fatalf("close non-closable: %v", err)
	}
}
