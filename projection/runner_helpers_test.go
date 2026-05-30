package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/event/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func startRunner(
	t *testing.T,
	runner *projection.Runner,
	ready <-chan struct{},
) context.CancelFunc {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	return cancel
}

func registerNoopProjection(
	t *testing.T,
	runner *projection.Runner,
	name string,
	eventTypes []event.Type,
) {
	t.Helper()

	err := runner.Register(event.NewProjection(
		name,
		eventtest.NoopEventHandler(),
		eventTypes,
	))
	if err != nil {
		t.Fatal(err)
	}
}

func registerReplayProjection(
	t *testing.T,
	runner *projection.Runner,
	name string,
	replayDone chan struct{},
	replayed *[]id.EventID,
	replayMu *sync.Mutex,
) {
	t.Helper()

	err := runner.Register(event.NewProjection(
		name,
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()

			*replayed = append(*replayed, evt.ID())

			if len(*replayed) == 1 {
				close(replayDone)
			}

			replayMu.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}
}

type subscribeSignalBus struct {
	event.Subscriber

	ready chan struct{}
	once  sync.Once
}

func (b *subscribeSignalBus) SubscribeAll(handler event.Handler) error {
	b.once.Do(func() { close(b.ready) })

	return b.Subscriber.SubscribeAll(handler)
}

func newTestRunnerWithReadyAndOpts(
	t *testing.T,
	opts ...projection.RunnerOption,
) (*projection.Runner, *memory.MemoryBus, <-chan struct{}) {
	t.Helper()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	ready := make(chan struct{})
	signalBus := &subscribeSignalBus{Subscriber: bus, ready: ready}

	runner, err := projection.NewRunner(nil, signalBus, checkpoint, opts...)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, bus, ready
}

func newTestRunnerWithReady(t *testing.T) (*projection.Runner, *memory.MemoryBus, <-chan struct{}) {
	t.Helper()

	return newTestRunnerWithReadyAndOpts(t)
}

func newTestRunnerWithReadyAndCheckpoint(
	t *testing.T,
	checkpoint *memory.MemoryCheckpointStore,
) (*projection.Runner, *memory.MemoryBus, <-chan struct{}) {
	t.Helper()

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	ready := make(chan struct{})
	signalBus := &subscribeSignalBus{Subscriber: bus, ready: ready}

	runner, err := projection.NewRunner(nil, signalBus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, bus, ready
}

func newTestRunner(t *testing.T) *projection.Runner {
	t.Helper()

	r, _, _ := newTestRunnerWithReady(t)

	return r
}

func drainChan[T any](t *testing.T, ch <-chan T, count int, label string) {
	t.Helper()

	for i := range count {
		select {
		case v := <-ch:
			_ = v
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s %d", label, i)
		}
	}
}

func newTestBusAndCheckpoint(t *testing.T) (*memory.MemoryBus, *memory.MemoryCheckpointStore) {
	t.Helper()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	return bus, checkpoint
}

type failingJournal struct {
	err error
}

func (f *failingJournal) ReadAll(_ context.Context) ([]event.Event, error) {
	return nil, f.err
}

type failingCheckpointStore struct {
	loadErr error
}

func (f *failingCheckpointStore) Load(_ context.Context, _ string) (event.Checkpoint, error) {
	return event.Checkpoint{}, f.loadErr
}

func (f *failingCheckpointStore) Save(_ context.Context, _ string, _ event.Checkpoint) error {
	return nil
}

func (f *failingCheckpointStore) Close() error { return nil }

type emptyJournal struct{}

func (e *emptyJournal) ReadAll(_ context.Context) ([]event.Event, error) {
	return nil, nil
}

type seekableJournalStore struct {
	events []event.Event
}

func (p *seekableJournalStore) ReadAll(_ context.Context) ([]event.Event, error) {
	return p.events, nil
}

func (p *seekableJournalStore) ReadFrom(
	_ context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	result := make([]event.Event, 0)

	for _, evt := range p.events {
		if !afterEventID.IsZero() {
			if evt.ID() == afterEventID {
				afterEventID = id.EventID{}
			}

			continue
		}

		result = append(result, evt)

		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

func mustNewEvent(
	t *testing.T,
	eventType string,
	aggID id.AggregateID,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		"User",
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}
