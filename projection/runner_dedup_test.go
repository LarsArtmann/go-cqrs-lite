package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

// replaySignalBus wraps a MemoryBus and replays buffered events to the handler
// when SubscribeAll is called, simulating a bus that retains events published
// before subscription (e.g. a ReplaySubject or a message queue with seek-back).
type replaySignalBus struct {
	event.Subscriber

	buffered []event.Event
	ready    chan struct{}
	once     sync.Once
}

func (b *replaySignalBus) SubscribeAll(handler event.Handler) error {
	b.once.Do(func() { close(b.ready) })

	err := b.Subscriber.SubscribeAll(handler)
	if err != nil {
		return err
	}

	for _, evt := range b.buffered {
		_ = handler(context.Background(), evt)
	}

	return nil
}

func TestRunner_ReplayLiveDedup(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	evt1 := mustNewEvent(t, "UserCreated", aggID)
	evt2 := mustNewEvent(t, "UserCreated", aggID)
	evt3 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	journal := &seekableJournalStore{events: []event.Event{evt1, evt2}}

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	// The bus will replay evt1 and evt2 when the live handler subscribes,
	// simulating the overlap window where events exist in both the journal
	// and the live stream.
	signalBus := &replaySignalBus{
		Subscriber: bus,
		buffered:   []event.Event{evt1, evt2},
		ready:      make(chan struct{}),
	}

	runner, err := projection.NewRunner(journal, signalBus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	var mutex sync.Mutex
	var processed []id.EventID

	err = runner.Register(event.NewProjection(
		"dedup-proj",
		func(_ context.Context, evt event.Event) error {
			mutex.Lock()
			processed = append(processed, evt.ID())
			mutex.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		_ = runner.Run(ctx)
	}()

	<-signalBus.ready

	err = bus.Publish(context.Background(), evt3)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	requireEventCount(t, &mutex, &processed, 3, 2*time.Second)

	cancel()
	_ = runner.Close()

	mutex.Lock()
	defer mutex.Unlock()

	counts := make(map[id.EventID]int)
	for _, eid := range processed {
		counts[eid]++
	}

	for eid, count := range counts {
		if count > 1 {
			t.Errorf("event %s processed %d times (expected exactly 1)", eid, count)
		}
	}

	if len(processed) != 3 {
		t.Errorf("expected exactly 3 events processed, got %d: %v", len(processed), processed)
	}
}

func TestRunner_LiveStreamDedup(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	signalBus := &replaySignalBus{
		Subscriber: bus,
		ready:      make(chan struct{}),
	}

	runner, err := projection.NewRunner(nil, signalBus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	var mutex sync.Mutex
	var processed []id.EventID

	err = runner.Register(event.NewProjection(
		"live-dedup-proj",
		func(_ context.Context, evt event.Event) error {
			mutex.Lock()
			processed = append(processed, evt.ID())
			mutex.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		_ = runner.Run(ctx)
	}()

	<-signalBus.ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err = bus.Publish(context.Background(), evt, evt)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	requireEventCount(t, &mutex, &processed, 1, 2*time.Second)

	cancel()
	_ = runner.Close()

	mutex.Lock()
	defer mutex.Unlock()

	if len(processed) != 1 {
		t.Errorf("expected 1 event (intra-stream dedup), got %d: %v", len(processed), processed)
	}
}

func requireEventCount(
	t *testing.T,
	mutex *sync.Mutex,
	processed *[]id.EventID,
	want int,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		mutex.Lock()
		count := len(*processed)
		mutex.Unlock()

		if count >= want {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	mutex.Lock()
	defer mutex.Unlock()

	t.Fatalf("timed out waiting for %d events, got %d: %v", want, len(*processed), *processed)
}
