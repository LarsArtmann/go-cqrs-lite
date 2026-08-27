package watermill

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestCatchUpSubscriber_Replay verifies that historical events are replayed
// from the journal when a subscription starts.
func TestCatchUpSubscriber_Replay(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	// Seed the store with a historical event.
	streamID := id.NewStreamID()
	historicalEvt, _ := event.NewEvent(
		event.Type("test.event"),
		streamID, "TestStream", event.Version(1),
		[]byte(`{"msg":"hello"}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID),
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
	// multiple streams so ReadFrom returns them in stable order.
	const total = 1500
	streamID := id.NewStreamID()

	events := make([]event.Event, 0, total)
	for range total {
		evt, _ := event.NewEvent(
			event.Type("bulk.event"),
			streamID, "BulkAggregate", event.Version(1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("BulkAggregate", streamID), events)

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

// publishingJournal wraps a SeekableJournal and publishes an event to the
// live bus on the first ReadFrom call. This deterministically triggers the
// TOCTOU race: the event appears during the replay window, between live
// subscribe and journal drain.
type publishingJournal struct {
	event.SeekableJournal

	bus      event.Publisher
	once     sync.Once
	streamID id.StreamID
}

func (j *publishingJournal) ReadFrom(
	ctx context.Context, after id.EventID, limit int,
) ([]event.Event, error) {
	j.once.Do(func() {
		liveEvt, _ := event.NewEvent(
			"test.race", j.streamID, "TestStream", event.Version(2),
			[]byte(`{"msg":"live-during-replay"}`),
		)
		_ = j.bus.Publish(ctx, liveEvt)
	})

	return j.SeekableJournal.ReadFrom(ctx, after, limit)
}

// TestCatchUpSubscriber_PicksUpEventsPublishedDuringReplay verifies that
// events published to the live bus during the replay window are NOT lost.
// Before the TOCTOU fix, these events were dropped because the live
// subscriber was not registered until after replay completed.
func TestCatchUpSubscriber_PicksUpEventsPublishedDuringReplay(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	historicalEvt, _ := event.NewEvent(
		"test.race", streamID, "TestStream", event.Version(1),
		[]byte(`{"msg":"historical"}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID),
		[]event.Event{historicalEvt})

	journal := &publishingJournal{
		SeekableJournal: store,
		bus:             bus,
		streamID:        streamID,
	}

	liveSub := NewSubscriberAdapter(bus)

	catchUp, err := NewCatchUpSubscriber(journal, liveSub, cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "test.race")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	received := 0
	timeout := time.After(5 * time.Second)

	for received < 2 {
		select {
		case msg := <-ch:
			if msg == nil {
				t.Fatalf("nil message after %d events", received)
			}

			msg.Ack()
			received++
		case <-timeout:
			t.Fatalf("timed out: received %d of 2 events (TOCTOU race)", received)
		}
	}
}

// TestCatchUpSubscriber_CheckpointOnlyAfterAck verifies the at-least-once
// contract: a message that was handed to the consumer but NOT acked (crash
// between delivery and processing) must not advance the checkpoint — the
// old behavior saved the checkpoint at handoff (at-most-once loss).
func TestCatchUpSubscriber_CheckpointOnlyAfterAck(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	historicalEvt, _ := event.NewEvent(
		"test.noack", streamID, "TestStream", event.Version(1),
		[]byte(`{}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID),
		[]event.Event{historicalEvt})

	catchUp, err := NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "test.noack")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case msg := <-ch:
		// Deliberately do NOT ack: simulate a consumer crash after handoff.
		_ = msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed event")
	}

	// Give the subscriber a moment to (wrongly) persist a checkpoint, then
	// verify none was saved.
	time.Sleep(100 * time.Millisecond)

	cp, err := cpStore.Load(context.Background(), "test.noack")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if !cp.IsZero() {
		t.Fatalf("checkpoint advanced without Ack (at-most-once bug): %+v", cp)
	}
}

// TestCatchUpSubscriber_NackStopsDelivery verifies that a Nack stops the
// subscription without advancing the checkpoint past the nacked event, so a
// restart re-delivers it.
func TestCatchUpSubscriber_NackStopsDelivery(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	var events []event.Event

	for i := range 2 {
		evt, _ := event.NewEvent(
			"test.nack", streamID, "TestStream", event.Version(event.Version(i) + 1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), events)

	catchUp, err := NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "test.nack")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case msg := <-ch:
		msg.Nack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first replayed event")
	}

	// The stream must stop: no second event may arrive after the Nack.
	// (A closed channel yields nil — the subscription shutting down is the
	// expected outcome, not a duplicate delivery.)
	select {
	case msg := <-ch:
		if msg != nil {
			t.Fatalf("delivery continued after Nack: %v", msg.UUID)
		}
	case <-time.After(250 * time.Millisecond):
	}

	cp, err := cpStore.Load(context.Background(), "test.nack")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if !cp.IsZero() {
		t.Fatalf("checkpoint advanced past a nacked event: %+v", cp)
	}
}

// replayingPublisher wraps the journal so that the FIRST ReadFrom call also
// publishes the journal's own first event to the live bus — the exact
// replay/live overlap the watermark must suppress.
type replayingPublisher struct {
	event.SeekableJournal

	bus   event.Publisher
	once  sync.Once
	evt   event.Event
	ready chan struct{}
}

func (j *replayingPublisher) ReadFrom(
	ctx context.Context, after id.EventID, limit int,
) ([]event.Event, error) {
	events, err := j.SeekableJournal.ReadFrom(ctx, after, limit)

	j.once.Do(func() {
		if len(events) > 0 {
			j.evt = events[0]
			close(j.ready)
		}
	})

	return events, err
}

// TestCatchUpSubscriber_WatermarkSuppressesLiveOverlap verifies that a live
// delivery of an event that replay already covered is suppressed via the
// last-replayed-ID watermark (the replacement for the bounded dedup ring).
func TestCatchUpSubscriber_WatermarkSuppressesLiveOverlap(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	historicalEvt, _ := event.NewEvent(
		"test.overlap", streamID, "TestStream", event.Version(1),
		[]byte(`{}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID),
		[]event.Event{historicalEvt})

	journal := &replayingPublisher{
		SeekableJournal: store,
		bus:             bus,
		ready:           make(chan struct{}),
	}

	catchUp, err := NewCatchUpSubscriber(journal, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch, err := catchUp.Subscribe(context.Background(), "test.overlap")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Consume and ack the replayed event.
	select {
	case msg := <-ch:
		msg.Ack()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed event")
	}

	// Once replay read the event, mirror it onto the live bus (simulating
	// the overlap window).
	select {
	case <-journal.ready:
		_ = bus.Publish(context.Background(), journal.evt)
	case <-time.After(2 * time.Second):
		t.Fatal("replay never read the journal event")
	}

	// The duplicate live delivery must be suppressed: nothing arrives
	// (nil = closed channel, also acceptable — no duplicate was delivered).
	select {
	case msg := <-ch:
		if msg != nil {
			t.Fatalf("live duplicate of replayed event was not suppressed: %v", msg.UUID)
		}
	case <-time.After(250 * time.Millisecond):
	}
}
