package watermill

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
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

// TestCatchUpSubscriber_BatchedReplay verifies that the batched streaming
// replay path delivers all events when the journal contains more events than
// a single batch (replayBatchSize = 500). Memory stays bounded regardless of
// journal size because events are loaded and forwarded one batch at a time.
func TestCatchUpSubscriber_BatchedReplay(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	// Seed the store with 3× the batch size of historical events across
	// multiple aggregates so ReadFrom returns them in stable order.
	const total = 1500
	aggID := id.NewAggregateID()

	events := make([]event.Event, 0, total)
	for range total {
		evt, _ := event.NewEvent(
			event.Type("bulk.event"),
			aggID, "BulkAggregate", event.Version(1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	_ = store.AppendBatch(context.Background(),
		event.NewAggregateRef("BulkAggregate", aggID), events)

	liveSub := NewSubscriberAdapter(bus)

	catchUp, err := NewCatchUpSubscriber(store, liveSub, cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "bulk.event")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	received := 0
	timeout := time.After(5 * time.Second)

	for received < total {
		select {
		case msg := <-ch:
			if msg == nil {
				t.Fatalf("nil message after %d events", received)
			}
			msg.Ack()
			received++
		case <-timeout:
			t.Fatalf("timed out: received %d of %d events", received, total)
		}
	}

	if received != total {
		t.Fatalf("expected %d events, got %d", total, received)
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
