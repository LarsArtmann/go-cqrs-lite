package watermill

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestCatchUpSubscriber_RestartRecoversSkewSuppressedEvent pins the
// self-healing claim from the CatchUpSubscriber doc (watermark ordering
// assumption): a live event whose ID sorts at or below the replay watermark
// is suppressed as "already covered", and the very same event is re-delivered
// by the NEXT subscription's replay — because the journal re-read is in
// append order, the event sits after the checkpoint position even though its
// ULID sorts below it.
func TestCatchUpSubscriber_RestartRecoversSkewSuppressedEvent(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()
	const total = 3

	events := make([]event.Event, 0, total)
	for i := range total {
		evt, _ := event.NewEvent(
			"test.skewrecover", streamID, "TestStream", event.Version(i+1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	if err := store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	catchUp, err := NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}

	ch, err := catchUp.Subscribe(context.Background(), "test.skewrecover")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Consume and ack the replayed events; the checkpoint ends at the last
	// one, which is also the replay watermark.
	for i := range total {
		select {
		case msg := <-ch:
			msg.Ack()
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for replayed event %d", i+1)
		}
	}

	// The clock-skewed event: a zero-timestamp ULID sorts below every real
	// event ID, but it is appended to the journal and published live AFTER
	// replay drained the journal — the skew scenario from the doc comment.
	skewID, err := id.ParseEventID("0000000000000000000000000A")
	if err != nil {
		t.Fatalf("parse skew ULID: %v", err)
	}

	skewEvt, err := event.NewEvent(
		"test.skewrecover", streamID, "TestStream", event.Version(total+1),
		[]byte(`{"skew":true}`), event.WithEventID(skewID),
	)
	if err != nil {
		t.Fatalf("NewEvent(skew): %v", err)
	}

	if err := store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), []event.Event{skewEvt}); err != nil {
		t.Fatalf("AppendBatch(skew): %v", err)
	}

	_ = bus.Publish(context.Background(), skewEvt)

	// The live event must be suppressed: its ID sorts at or below the
	// watermark. Suppression is an immediate string compare in drainLive, so
	// a (buggy) delivery would arrive within microseconds of the publish.
	select {
	case msg := <-ch:
		t.Fatalf("skewed live event was not suppressed: %s", msg.UUID)
	case <-time.After(250 * time.Millisecond):
	}

	closePromptly(t, "Close before restart", catchUp.Close)

	// Restart with the SAME checkpoint store: replay resumes from the
	// checkpoint and re-delivers the suppressed event (append-order read).
	catchUp2, err := NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber(restart): %v", err)
	}

	ch2, err := catchUp2.Subscribe(context.Background(), "test.skewrecover")
	if err != nil {
		t.Fatalf("Subscribe(restart): %v", err)
	}

	select {
	case msg := <-ch2:
		if got := msg.Metadata.Get(metaEventID); got != skewID.String() {
			t.Fatalf("expected the skew-suppressed event to be re-delivered, got event_id=%s", got)
		}
		msg.Ack()
	case <-time.After(5 * time.Second):
		t.Fatal("restart did not re-deliver the skew-suppressed event")
	}

	closePromptly(t, "Close after recovery", catchUp2.Close)
}

// gatingJournal parks the FIRST ReadFrom call inside the journal read until
// released (or ctx is cancelled — as real stores honor ctx). It makes
// "replay parked mid-journal-read" an observable, deterministic state.
type gatingJournal struct {
	event.SeekableJournal

	entered chan struct{}
	once    sync.Once
}

func (g *gatingJournal) ReadFrom(
	ctx context.Context, after id.EventID, limit int,
) ([]event.Event, error) {
	g.once.Do(func() { close(g.entered) })

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return g.SeekableJournal.ReadFrom(ctx, after, limit)
}

// TestCatchUpSubscriber_CloseWhileReplayParkedInJournal pins that Close()
// terminates a subscription whose replay goroutine is parked INSIDE a journal
// read: no deadlock, the subscription channel closes, Close returns promptly.
//
// This replaces the retired CloseWhileBlockedOnFullBuffer test: replay
// forwards are serialized with awaitAck (send → await ack → next send), so
// the 256-slot output buffer can never actually fill and that park state was
// unreachable — the parkable states are awaitAck (pinned by
// CloseWhileBlockedOnAck) and the journal read (pinned here).
func TestCatchUpSubscriber_CloseWhileReplayParkedInJournal(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.closepark", streamID, "TestStream", event.Version(1),
		[]byte(`{}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), []event.Event{evt})

	journal := &gatingJournal{
		SeekableJournal: store,
		entered:         make(chan struct{}),
	}

	catchUp, err := NewCatchUpSubscriber(journal, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}

	ch, err := catchUp.Subscribe(context.Background(), "test.closepark")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Deterministic park point: replay is blocked inside ReadFrom.
	select {
	case <-journal.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("replay never entered the journal read")
	}

	closePromptly(t, "Close while replay parked in journal read", catchUp.Close)

	// The first batch may or may not have been forwarded before Close's ctx
	// cancellation reached the journal read — 0 or 1 deliveries are both
	// correct; more would mean the replay kept running past Close.
	if got := drainUntilClosed(t, ch); got > 1 {
		t.Errorf("replay delivered %d messages after Close; only the first batch is admissible", got)
	}

	if _, err := catchUp.Subscribe(context.Background(), "test.closepark"); err == nil {
		t.Error("Subscribe after Close must fail")
	}
}
