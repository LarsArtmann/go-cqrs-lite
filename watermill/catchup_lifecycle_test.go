package watermill

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// closePromptly runs closeFn under a watchdog: if it does not return within
// the deadline the test fails (deadlock regression pin) instead of hanging
// the whole suite.
func closePromptly(t *testing.T, name string, closeFn func() error) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		_ = closeFn()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("%s deadlocked (did not return within 5s)", name)
	}
}

// drainUntilClosed reads messages from ch (acking each) until the channel is
// closed, and returns how many live messages were delivered. Fails the test
// if the channel does not close within the timeout.
func drainUntilClosed(t *testing.T, ch <-chan *message.Message) int {
	t.Helper()

	received := 0
	timeout := time.After(5 * time.Second)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return received
			}

			if msg == nil {
				t.Fatalf("nil message (closed channel yields nil only via ok=false) after %d", received)
			}

			msg.Ack()
			received++
		case <-timeout:
			t.Fatalf("channel did not close; received %d messages", received)

			return received
		}
	}
}

// TestCatchUpSubscriber_CloseWhileBlockedOnFullBuffer pins the property that
// Close() terminates a subscription whose replay is blocked because the
// consumer stopped reading (output buffer full) — Close must return promptly
// and the subscription channel must close instead of deadlocking.
func TestCatchUpSubscriber_CloseWhileBlockedOnFullBuffer(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	// More events than the 256-slot output buffer: with no consumer reading,
	// the replay goroutine must end up blocked on the forwarding select.
	const total = 600
	streamID := id.NewStreamID()

	events := make([]event.Event, 0, total)
	for range total {
		evt, _ := event.NewEvent(
			"test.closefull", streamID, "TestStream", event.Version(1),
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

	ch, err := catchUp.Subscribe(context.Background(), "test.closefull")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Let the replay fill the buffer and block on forwarding.
	time.Sleep(100 * time.Millisecond)

	closePromptly(t, "Close while replay blocked on full output buffer", catchUp.Close)

	delivered := drainUntilClosed(t, ch)
	if delivered > total {
		t.Fatalf("delivered %d messages, journal only had %d", delivered, total)
	}

	if _, err := catchUp.Subscribe(context.Background(), "test.closefull"); err == nil {
		t.Error("Subscribe after Close must fail")
	}
}

// TestCatchUpSubscriber_CloseWhileBlockedOnAck pins the property that Close()
// terminates a subscription blocked in the awaitAck wait (consumer received a
// message but never acked it). Close must return promptly, the channel must
// close, and the checkpoint must stay untouched (the message was never
// acked — at-least-once).
func TestCatchUpSubscriber_CloseWhileBlockedOnAck(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	evt, _ := event.NewEvent(
		"test.closeack", streamID, "TestStream", event.Version(1),
		[]byte(`{}`),
	)
	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), []event.Event{evt})

	catchUp, err := NewCatchUpSubscriber(store, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}

	ch, err := catchUp.Subscribe(context.Background(), "test.closeack")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case msg := <-ch:
		// Deliberately do NOT ack: the subscriber is now parked in awaitAck.
		_ = msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed event")
	}

	closePromptly(t, "Close while subscription blocked in awaitAck", catchUp.Close)

	if got := drainUntilClosed(t, ch); got != 0 {
		t.Errorf("expected no further deliveries after the unacked message, got %d", got)
	}

	cp, err := cpStore.Load(context.Background(), "test.closeack")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}

	if !cp.IsZero() {
		t.Errorf("checkpoint advanced without Ack: %+v", cp)
	}
}

// readBarrierJournal counts ReadFrom calls so the test can wait until BOTH
// subscriptions have loaded their replay batch before any ack advances the
// shared checkpoint (keeps the double-subscribe replay assertion
// deterministic).
type readBarrierJournal struct {
	event.SeekableJournal

	reads chan struct{}
}

func (j *readBarrierJournal) ReadFrom(
	ctx context.Context, after id.EventID, limit int,
) ([]event.Event, error) {
	events, err := j.SeekableJournal.ReadFrom(ctx, after, limit)
	j.reads <- struct{}{}

	return events, err
}

// TestCatchUpSubscriber_DoubleSubscribeSameTopic pins the double-subscribe
// property: two Subscribe calls on the same topic are independent
// subscriptions — each replays the full journal into its own channel, both
// channels close on Close, and a post-Close Subscribe is rejected.
func TestCatchUpSubscriber_DoubleSubscribeSameTopic(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	const total = 5
	streamID := id.NewStreamID()

	events := make([]event.Event, 0, total)
	for range total {
		evt, _ := event.NewEvent(
			"test.dupsub", streamID, "TestStream", event.Version(1),
			[]byte(`{}`),
		)
		events = append(events, evt)
	}

	_ = store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), events)

	journal := &readBarrierJournal{
		SeekableJournal: store,
		reads:           make(chan struct{}, 2),
	}

	catchUp, err := NewCatchUpSubscriber(journal, NewSubscriberAdapter(bus), cpStore, nil)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	defer catchUp.Close()

	ch1, err := catchUp.Subscribe(context.Background(), "test.dupsub")
	if err != nil {
		t.Fatalf("Subscribe #1: %v", err)
	}

	ch2, err := catchUp.Subscribe(context.Background(), "test.dupsub")
	if err != nil {
		t.Fatalf("Subscribe #2: %v", err)
	}

	// Wait until both subscriptions have read the journal (their replay
	// started from a zero checkpoint) before acking anything.
	for range 2 {
		select {
		case <-journal.reads:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for both subscriptions to read the journal")
		}
	}

	for name, ch := range map[string]<-chan *message.Message{"first": ch1, "second": ch2} {
		got := 0
		timeout := time.After(5 * time.Second)

		for got < total {
			select {
			case msg := <-ch:
				if msg == nil {
					t.Fatalf("%s subscription closed after %d of %d replay events", name, got, total)
				}

				msg.Ack()
				got++
			case <-timeout:
				t.Fatalf("%s subscription timed out: %d of %d replay events", name, got, total)
			}
		}

		if got != total {
			t.Fatalf("%s subscription delivered %d events, want %d", name, got, total)
		}
	}

	closePromptly(t, "Close with two active subscriptions", catchUp.Close)

	for name, ch := range map[string]<-chan *message.Message{"first": ch1, "second": ch2} {
		if got := drainUntilClosed(t, ch); got != 0 {
			t.Errorf("%s subscription delivered %d unexpected messages after Close", name, got)
		}
	}

	if _, err := catchUp.Subscribe(context.Background(), "test.dupsub"); err == nil {
		t.Error("Subscribe after Close must fail")
	}
}
