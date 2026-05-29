package watermill_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	wm "github.com/larsartmann/go-cqrs-lite/watermill"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	publisher := wm.NewPublisherAdapter(bus)
	subscriber := wm.NewSubscriberAdapter(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh, err := subscriber.Subscribe(ctx, "user.created")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Build a fully populated event
	aggID := id.NewAggregateID()
	correlationID := id.NewCorrelationID()
	causationID := id.NewCausationID()
	userID := id.NewUserID()
	requestID := id.NewRequestID()
	fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	original, err := event.NewEvent(
		"user.created",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
		event.WithCorrelationID(correlationID),
		event.WithCausationID(causationID),
		event.WithUserID(userID),
		event.WithRequestID(requestID),
		event.WithSource("web"),
		event.WithIPAddress("192.168.1.1"),
		event.WithUserAgent("Mozilla/5.0"),
		event.WithCustom("tenant", "acme"),
		event.WithOccurredAt(fixedTime),
		event.WithSchemaVersion(2),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	// Publish via adapter (wraps event in Watermill message)
	wmMsg := message.NewMessage(original.ID().String(), original.Payload())
	wmMsg.Metadata.Set("event_id", original.ID().String())
	wmMsg.Metadata.Set("event_type", string(original.Type()))
	wmMsg.Metadata.Set("aggregate_id", original.AggregateID().String())
	wmMsg.Metadata.Set("aggregate_type", string(original.AggregateType()))
	wmMsg.Metadata.Set("version", "1")
	wmMsg.Metadata.Set("schema_version", "2")
	wmMsg.Metadata.Set("occurred_at", fixedTime.Format(time.RFC3339Nano))
	wmMsg.Metadata.Set("correlation_id", correlationID.String())
	wmMsg.Metadata.Set("causation_id", causationID.String())
	wmMsg.Metadata.Set("user_id", userID.String())
	wmMsg.Metadata.Set("request_id", requestID.String())
	wmMsg.Metadata.Set("source", "web")
	wmMsg.Metadata.Set("ip_address", "192.168.1.1")
	wmMsg.Metadata.Set("user_agent", "Mozilla/5.0")
	wmMsg.Metadata.Set("custom.tenant", "acme")

	if err := publisher.Publish("user.created", wmMsg); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Receive via subscriber adapter
	receiveMessageWithTimeout(t, msgCh, "user.created", func(received *message.Message) {
		if string(received.Payload) != `{"name":"Alice"}` {
			t.Errorf("payload = %q, want %q", received.Payload, `{"name":"Alice"}`)
		}
		if received.Metadata.Get("aggregate_id") != aggID.String() {
			t.Errorf("aggregate_id mismatch")
		}
		assertMetadata(t, received.Metadata, "version", "1")
		assertMetadata(t, received.Metadata, "schema_version", "2")
		if received.Metadata.Get("correlation_id") != correlationID.String() {
			t.Errorf("correlation_id mismatch")
		}
		assertMetadata(t, received.Metadata, "custom.tenant", "acme")
	})
}

func TestPublisherAdapter_BadMetadata(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	// Missing aggregate_id, aggregate_type, version

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for missing metadata")
	}
}

func TestPublisherAdapter_InvalidVersion(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	publisher := wm.NewPublisherAdapter(bus)

	msg := message.NewMessage("test-id", []byte(`{}`))
	msg.Metadata.Set("aggregate_id", id.NewAggregateID().String())
	msg.Metadata.Set("aggregate_type", "User")
	msg.Metadata.Set("version", "not-a-number")

	if err := publisher.Publish("user.created", msg); err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestPublisherAdapter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	publisher := wm.NewPublisherAdapter(bus)

	if err := publisher.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestSubscriberAdapter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	subscriber := wm.NewSubscriberAdapter(bus)

	if err := subscriber.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func assertMetadata(t *testing.T, md message.Metadata, key, want string) {
	t.Helper()
	if got := md.Get(key); got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}
