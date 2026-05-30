package watermill_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
	wm "github.com/larsartmann/go-cqrs-lite/watermill"
)

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

func missingAggregateTypeMetadata() map[string]string {
	return map[string]string{
		"aggregate_id": id.NewAggregateID().String(),
		"version":      "1",
	}
}

func missingVersionMetadata() map[string]string {
	return map[string]string{
		"aggregate_id":   id.NewAggregateID().String(),
		"aggregate_type": "User",
	}
}

func invalidMetadataTestCases() []struct {
	name     string
	metadata map[string]string
} {
	base := map[string]string{
		"aggregate_id":   id.NewAggregateID().String(),
		"aggregate_type": "User",
		"version":        "1",
	}

	return []struct {
		name     string
		metadata map[string]string
	}{
		{
			name:     "invalid_schema_version",
			metadata: mergeMetadata(base, "schema_version", "not-a-number"),
		},
		{
			name:     "invalid_event_id",
			metadata: mergeMetadata(base, "event_id", "not-a-valid-event-id"),
		},
		{
			name:     "invalid_occurred_at",
			metadata: mergeMetadata(base, "occurred_at", "not-a-timestamp"),
		},
	}
}

func mergeMetadata(base map[string]string, key, value string) map[string]string {
	out := make(map[string]string, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[key] = value

	return out
}

func TestMessageToEvent_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct { //nolint:prealloc // appended below with additional cases
		name     string
		metadata map[string]string
	}{
		{name: "missing_aggregate_type", metadata: missingAggregateTypeMetadata()},
		{name: "missing_version", metadata: missingVersionMetadata()},
	}
	tests = append(tests, invalidMetadataTestCases()...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bus := memory.NewMemoryBus()
			defer bus.Close()

			publisher := wm.NewPublisherAdapter(bus)

			msg := message.NewMessage("test-id", []byte(`{}`))
			for k, v := range tt.metadata {
				msg.Metadata.Set(k, v)
			}

			if err := publisher.Publish("user.created", msg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
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

	receiveMessageWithTimeout(t, msgCh, "user.created", nil)
}

func receiveMessageWithTimeout(
	t *testing.T,
	msgCh <-chan *message.Message,
	expectedEventType string,
	assertions func(*message.Message),
) {
	select {
	case received := <-msgCh:
		if received.Metadata.Get("event_type") != expectedEventType {
			t.Errorf(
				"event_type = %q, want %q",
				received.Metadata.Get("event_type"),
				expectedEventType,
			)
		}
		if assertions != nil {
			assertions(received)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
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
