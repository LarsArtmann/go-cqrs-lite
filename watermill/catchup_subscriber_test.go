package watermill

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

// TestCatchUpSubscriber_Replay verifies that historical events are replayed
// from the journal when a subscription starts.
func TestCatchUpSubscriber_Replay(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	// Seed the store with a historical event.
	aggID := id.NewAggregateID()
	historicalEvt, _ := event.NewEvent(
		event.Type("test.event"),
		aggID, "TestAggregate", event.Version(1),
		[]byte(`{"msg":"hello"}`),
	)
	_ = store.AppendBatch(context.Background(),
		event.NewAggregateRef("TestAggregate", aggID),
		[]event.Event{historicalEvt})

	liveSub := NewSubscriberAdapter(bus)

	catchUp, err := NewCatchUpSubscriber(store, liveSub, cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "test.event")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Should receive the replayed historical event.
	select {
	case msg := <-ch:
		if msg == nil {
			t.Fatal("received nil message")
		}

		if msg.Metadata.Get(metaEventType) != "test.event" {
			t.Fatalf("expected event type test.event, got %s", msg.Metadata.Get(metaEventType))
		}

		// Replay messages should be marked with ModeReplay.
		if msg.Metadata.Get(metaProcessingMode) != string(event.ModeReplay) {
			t.Fatalf(
				"expected processing_mode=replay, got %s",
				msg.Metadata.Get(metaProcessingMode),
			)
		}

		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed event")
	}
}

// TestCatchUpSubscriber_NilChecks verifies constructor validation.
func TestCatchUpSubscriber_NilChecks(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	_, err := NewCatchUpSubscriber(nil, NewSubscriberAdapter(bus), cpStore, nil)
	if err == nil {
		t.Error("expected error for nil journal")
	}

	_, err = NewCatchUpSubscriber(store, nil, cpStore, nil)
	if err == nil {
		t.Error("expected error for nil live subscriber")
	}

	_, err = NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), nil, nil)
	if err == nil {
		t.Error("expected error for nil checkpoint store")
	}
}
