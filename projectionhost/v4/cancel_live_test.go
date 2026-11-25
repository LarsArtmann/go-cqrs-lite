package projectionhost_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func awaitSignal(t *testing.T, what string, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not happen within 5s", what)
	}
}

func deadLetters(t *testing.T, dlq *projectionhost.MemoryDeadLetterStore) int {
	t.Helper()

	entries, err := dlq.List(context.Background(), "")
	if err != nil {
		t.Fatalf("dlq list: %v", err)
	}

	return len(entries)
}

// TestHost_CancelDuringDrainRetryDoesNotDeadLetter: a shutdown that lands
// between a failed attempt and its retry must not park the in-flight event
// as poison. The wrapped context.Canceled classifies as a Rejection, which
// used to route the event to the DLQ and mark it seen.
func TestHost_CancelDuringDrainRetryDoesNotDeadLetter(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	proj := &cancelThenFailProjection{name: "cancel-drain", entered: entered}
	journal := &memoryJournal{}
	journal.append(makeEvent("task.created"))

	dlq := projectionhost.NewMemoryDeadLetterStore()
	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 2),
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = host.Start(ctx) }()

	awaitSignal(t, "first Handle attempt", entered)
	cancel()

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got := deadLetters(t, dlq); got != 0 {
		t.Fatalf("%d event(s) dead-lettered on a cancelled worker, want 0", got)
	}

	if cp, err := cpStore.Load(context.Background(), proj.name); err != nil || !cp.IsZero() {
		t.Fatalf("checkpoint advanced past an unapplied event: (%v, %v)", cp, err)
	}
}

// TestHost_CancelDuringLiveHandleDoesNotDeadLetter is the live-phase twin:
// a handler error that only exists because the worker context ended must not
// park the live event as poison.
func TestHost_CancelDuringLiveHandleDoesNotDeadLetter(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{})
	proj := &interruptibleProjection{name: "cancel-live", entered: entered}
	sub := &recordingSubscriber{}
	dlq := projectionhost.NewMemoryDeadLetterStore()
	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(
		&memoryJournal{}, cpStore,
		projectionhost.WithSubscriber(sub),
		projectionhost.WithDeadLetterStore(dlq, 1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = host.Start(ctx) }()

	requireEventually(t, 5*time.Second, func() bool {
		return sub.count() == 1 && workerStatus(t, host, proj.name) == projectionhost.WorkerStopped
	})

	go func() {
		<-entered
		cancel()
	}()

	_ = sub.registered()[0](context.Background(), makeEvent("task.created"))

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got := deadLetters(t, dlq); got != 0 {
		t.Fatalf("%d live event(s) dead-lettered on a cancelled worker, want 0", got)
	}

	if cp, err := cpStore.Load(context.Background(), proj.name); err != nil || !cp.IsZero() {
		t.Fatalf("checkpoint advanced past an unapplied event: (%v, %v)", cp, err)
	}
}

// interruptibleProjection fails every Handle the way an interrupted store
// write does: wrapping the ended context's error (a Rejection by family).
type interruptibleProjection struct {
	name    string
	entered chan struct{}
}

func (p *interruptibleProjection) Name() string             { return p.name }
func (p *interruptibleProjection) EventTypes() []event.Type { return nil }

func (p *interruptibleProjection) Handle(ctx context.Context, _ event.Event) error {
	close(p.entered)
	<-ctx.Done()

	return fmt.Errorf("write interrupted: %w", ctx.Err())
}

// cancelThenFailProjection fails its first attempt with a retryable error
// once the worker context ends, so the retry is what gets cancelled.
type cancelThenFailProjection struct {
	name    string
	entered chan struct{}
	calls   atomic.Int64
}

func (p *cancelThenFailProjection) Name() string             { return p.name }
func (p *cancelThenFailProjection) EventTypes() []event.Type { return nil }

func (p *cancelThenFailProjection) Handle(ctx context.Context, _ event.Event) error {
	if p.calls.Add(1) > 1 {
		return nil
	}

	close(p.entered)
	<-ctx.Done()

	return errorfamily.NewTransient("test.interrupted", "store busy")
}

// recordingSubscriber is a non-blocking subscriber that keeps every handler
// ever registered, the way an in-process bus without unsubscribe does.
type recordingSubscriber struct {
	mu         sync.Mutex
	handlers   []event.Handler
	onRegister func()
}

func (s *recordingSubscriber) Subscribe(_ event.Type, _ event.Handler) error {
	return errors.New("recordingSubscriber only supports SubscribeAll")
}

func (s *recordingSubscriber) SubscribeAll(handler event.Handler) error {
	s.mu.Lock()
	s.handlers = append(s.handlers, handler)
	s.mu.Unlock()

	if s.onRegister != nil {
		s.onRegister()
	}

	return nil
}

func (s *recordingSubscriber) registered() []event.Handler {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]event.Handler(nil), s.handlers...)
}

func (s *recordingSubscriber) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.handlers)
}

// TestHost_LiveHandlerRetiredWhenWorkerFails: event.Subscriber has no
// unsubscribe, so a worker that exhausted its restart budget used to keep
// applying live events (and saving checkpoints) while reporting
// WorkerFailed.
func TestHost_LiveHandlerRetiredWhenWorkerFails(t *testing.T) {
	t.Parallel()

	journal := &failNextReadJournal{memoryJournal: &memoryJournal{}}
	sub := &recordingSubscriber{}
	proj := &countingProjection{name: "failed-live"}

	// Fail the catch-up drain that follows the subscription: the worker
	// fails for good (no restart budget) right after registering.
	sub.onRegister = func() { journal.failNext() }

	host, err := projectionhost.New(
		journal, newMemoryCheckpointStore(),
		projectionhost.WithSubscriber(sub),
		projectionhost.WithMaxRestarts(0),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 5*time.Second, func() bool {
		return workerStatus(t, host, proj.name) == projectionhost.WorkerFailed &&
			len(sub.registered()) == 1
	})

	if err := sub.registered()[0](context.Background(), makeEvent("task.created")); err != nil {
		t.Fatalf("retired handler returned an error: %v", err)
	}

	if got := proj.count.Load(); got != 0 {
		t.Fatalf("a WorkerFailed worker applied %d live event(s), want 0", got)
	}
}

// TestHost_LiveHandlerRetiredOnRestart: a live phase that ends in an error
// leads to a restart, which registers a fresh handler; every earlier one
// used to stay active too. Afterwards the worker goroutine exits (WorkerStopped)
// while the current handler keeps serving.
func TestHost_LiveHandlerRetiredOnRestart(t *testing.T) {
	t.Parallel()

	journal := &failNextReadJournal{memoryJournal: &memoryJournal{}}
	sub := &recordingSubscriber{}
	proj := &countingProjection{name: "restarted-live"}

	// The first live phase's catch-up drain fails once; the restart drains
	// cleanly and subscribes a second time.
	sub.onRegister = func() {
		if sub.count() == 1 {
			journal.failNext()
		}
	}

	host, err := projectionhost.New(
		journal, newMemoryCheckpointStore(),
		projectionhost.WithSubscriber(sub),
		projectionhost.WithMaxRestarts(-1),
		projectionhost.WithBackoff(time.Millisecond, time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 5*time.Second, func() bool {
		return sub.count() == 2 &&
			workerStatus(t, host, proj.name) == projectionhost.WorkerStopped
	})

	handlers := sub.registered()

	if err := handlers[0](context.Background(), makeEvent("task.created")); err != nil {
		t.Fatalf("retired handler returned an error: %v", err)
	}

	if got := proj.count.Load(); got != 0 {
		t.Fatalf("the handler of an ended live phase applied %d event(s), want 0", got)
	}

	if err := handlers[1](context.Background(), makeEvent("task.created")); err != nil {
		t.Fatalf("current handler returned an error: %v", err)
	}

	if got := proj.count.Load(); got != 1 {
		t.Fatalf("current handler applied %d event(s), want 1", got)
	}
}

func workerStatus(t *testing.T, host *projectionhost.Host, name string) projectionhost.WorkerStatus {
	t.Helper()

	for _, state := range host.Status() {
		if state.Name == name {
			return state.Status
		}
	}

	return projectionhost.WorkerIdle
}

// failNextReadJournal fails the ReadFrom that follows failNext.
type failNextReadJournal struct {
	*memoryJournal

	failNextRead atomic.Bool
}

func (j *failNextReadJournal) failNext() { j.failNextRead.Store(true) }

func (j *failNextReadJournal) ReadFrom(
	ctx context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	if j.failNextRead.Swap(false) {
		return nil, errors.New("journal: read interrupted")
	}

	return j.memoryJournal.ReadFrom(ctx, after, limit)
}
