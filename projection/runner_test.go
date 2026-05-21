package projection_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func registerReplayProjection(t *testing.T, runner *projection.Runner, name string, replayDone chan struct{}, replayed *[]id.EventID, replayMu *sync.Mutex) {
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

func TestNewRunner_NilBus(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, nil, memory.NewCheckpointStore())
	if err == nil {
		t.Fatal("expected error for nil bus")
	}
}

func TestNewRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, memory.NewMemoryBus(), nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
	}
}

func TestNewRunner_NilLoaderIsOK(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, memory.NewMemoryBus(), memory.NewCheckpointStore())
	if err != nil {
		t.Fatalf("nil loader should be ok: %v", err)
	}
}

func TestRunner_Register_NilProjection(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil projection")
	}
}

func TestRunner_Register_ValidProjection(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Register(
		event.NewProjection("test", func(_ context.Context, _ event.Event) error {
			return nil
		}, []event.Type{"UserCreated"}),
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestRunner_NoProjections(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when no projections registered")
	}
}

func TestRunner_ProcessesLiveEvents(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 1)

	runner, bus, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"user-proj",
		func(_ context.Context, evt event.Event) error {
			handled <- string(evt.Type())

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case h := <-handled:
		if h != "UserCreated" {
			t.Errorf("handled = %q, want UserCreated", h)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event to be handled")
	}
}

func TestRunner_SavesCheckpoint(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, bus, ready := newTestRunnerWithReadyAndCheckpoint(t, checkpoint)

	err := runner.Register(event.NewProjection(
		"user-proj",
		func(_ context.Context, _ event.Event) error {
			close(done)

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler")
	}

	cpResult, err := runner.CurrentCheckpoint(context.Background(), "user-proj")
	if err != nil {
		t.Fatalf("CurrentCheckpoint: %v", err)
	}

	if cpResult != evt.ID() {
		t.Errorf("checkpoint = %v, want %v", cpResult, evt.ID())
	}
}

func TestRunner_FiltersUnregisteredTypes(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 1)

	runner, bus, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"user-proj",
		func(_ context.Context, evt event.Event) error {
			handled <- string(evt.Type())

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	otherEvt := mustNewEvent(t, "OrderPlaced", id.NewAggregateID())

	_ = bus.Publish(context.Background(), otherEvt)

	select {
	case h := <-handled:
		t.Errorf("should not have handled %q", h)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRunner_WildcardProjection(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 2)

	runner, bus, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"all-proj",
		func(_ context.Context, evt event.Event) error {
			handled <- string(evt.Type())

			return nil
		},
		nil,
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	_ = bus.Publish(context.Background(), mustNewEvent(t, "UserCreated", id.NewAggregateID()))
	_ = bus.Publish(context.Background(), mustNewEvent(t, "OrderPlaced", id.NewAggregateID()))

	for i := range 2 {
		select {
		case h := <-handled:
			_ = h
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}
}

func TestRunner_ReplayFromStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", evt1.AggregateID(), []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("Save evt1: %v", err)
	}

	err = store.Save(ctx, "User", evt2.AggregateID(), []event.Event{evt2}, 0)
	if err != nil {
		t.Fatalf("Save evt2: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []string

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection(
		"replay-proj",
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()

			replayed = append(replayed, string(evt.Type()))

			if len(replayed) == 2 {
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

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)

		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replay to complete")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 2 {
		t.Errorf("replayed %d events, want 2", len(replayed))
	}

	savedCP, err := checkpoint.Load(ctx, "replay-proj")
	if err != nil {
		t.Fatalf("checkpoint load: %v", err)
	}

	if savedCP.IsZero() {
		t.Error("checkpoint should be saved after replay")
	}
}

func TestRunner_MultipleProjections(t *testing.T) {
	t.Parallel()

	userHandled := make(chan string, 1)
	orderHandled := make(chan string, 1)

	runner, bus, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"user-proj",
		func(_ context.Context, evt event.Event) error {
			userHandled <- string(evt.Type())

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	err = runner.Register(event.NewProjection(
		"order-proj",
		func(_ context.Context, evt event.Event) error {
			orderHandled <- string(evt.Type())

			return nil
		},
		[]event.Type{"OrderPlaced"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	_ = bus.Publish(context.Background(), mustNewEvent(t, "UserCreated", id.NewAggregateID()))
	_ = bus.Publish(context.Background(), mustNewEvent(t, "OrderPlaced", id.NewAggregateID()))

	select {
	case h := <-userHandled:
		if h != "UserCreated" {
			t.Errorf("user-proj got %q, want UserCreated", h)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user-proj")
	}

	select {
	case h := <-orderHandled:
		if h != "OrderPlaced" {
			t.Errorf("order-proj got %q, want OrderPlaced", h)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for order-proj")
	}
}

func TestRunner_ReplayWithCheckpoint(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", evt1.AggregateID(), []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("Save evt1: %v", err)
	}

	err = store.Save(ctx, "User", evt2.AggregateID(), []event.Event{evt2}, 0)
	if err != nil {
		t.Fatalf("Save evt2: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	err = checkpoint.Save(ctx, "replay-proj", evt1.ID())
	if err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []id.EventID

	var replayMu sync.Mutex

	registerReplayProjection(t, runner, "replay-proj", replayDone, &replayed, &replayMu)

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)

		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replay to complete")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (skipped past checkpoint)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != evt2.ID() {
		t.Errorf("replayed event = %v, want %v (evt after checkpoint)", replayed[0], evt2.ID())
	}
}

func TestRunner_ReplayFiltersUnmatchedTypes(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	userEvt := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	orderEvt := mustNewEvent(t, "OrderPlaced", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", userEvt.AggregateID(), []event.Event{userEvt}, 0)
	if err != nil {
		t.Fatalf("Save userEvt: %v", err)
	}

	err = store.Save(ctx, "Order", orderEvt.AggregateID(), []event.Event{orderEvt}, 0)
	if err != nil {
		t.Fatalf("Save orderEvt: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	var replayed []string

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection(
		"user-only-proj",
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()
			replayed = append(replayed, string(evt.Type()))
			replayMu.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (OrderPlaced should be filtered)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != "UserCreated" {
		t.Errorf("replayed event = %q, want UserCreated", replayed[0])
	}
}

func TestRunner_RetryOnTransientError(t *testing.T) {
	t.Parallel()

	attempts := make(chan int, 3)
	callCount := 0

	runner, bus, ready := newTestRunnerWithReadyAndOpts(
		t,
		projection.WithRetry(3, time.Millisecond),
	)

	err := runner.Register(event.NewProjection(
		"retry-proj",
		func(_ context.Context, _ event.Event) error {
			callCount++
			attempts <- callCount

			if callCount < 3 {
				return event.NewTransient("db.timeout", "connection timed out")
			}

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		select {
		case a := <-attempts:
			if a != i+1 {
				t.Errorf("attempt %d, got call count %d", i+1, a)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for attempt %d", i+1)
		}
	}
}

func TestRunner_NoRetryOnNonRetryableError(t *testing.T) {
	t.Parallel()

	attempts := make(chan int, 2)
	callCount := 0

	runner, bus, ready := newTestRunnerWithReadyAndOpts(
		t,
		projection.WithRetry(3, time.Millisecond),
	)

	err := runner.Register(event.NewProjection(
		"no-retry-proj",
		func(_ context.Context, _ event.Event) error {
			callCount++
			attempts <- callCount

			return event.NewConflict("already.exists", "duplicate entity")
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case a := <-attempts:
		if a != 1 {
			t.Errorf("expected exactly 1 attempt, got call count %d", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler")
	}

	select {
	case a := <-attempts:
		t.Errorf("should not retry non-retryable error, got attempt %d", a)
	case <-time.After(50 * time.Millisecond):
	}
}

// subscribeSignalBus wraps an event.Subscriber and signals when SubscribeAll is called.
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
	checkpoint := memory.NewCheckpointStore()

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

func TestHandlerRegistry_OnAll_NilHandler(t *testing.T) {
	t.Parallel()

	registry := projection.NewHandlerRegistry()

	err := registry.OnAll(nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestRunner_ReplayError_LoadAllFails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	loader := &failingGlobalLoader{err: errors.New("load all failed")}

	runner, err := projection.NewRunner(loader, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Register(event.NewProjection(
		"test-proj",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when LoadAll fails")
	}
}

func TestRunner_ReplayError_CheckpointLoadFails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	saveErr := store.Save(context.Background(), "User", evt.AggregateID(), []event.Event{evt}, 0)
	if saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	checkpoint := &failingCheckpointStore{loadErr: errors.New("checkpoint load failed")}

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Register(event.NewProjection(
		"test-proj",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when checkpoint load fails")
	}
}

func TestRunner_ReplayError_HandlerFails(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	saveErr := store.Save(context.Background(), "User", evt.AggregateID(), []event.Event{evt}, 0)
	if saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	handlerErr := errors.New("handler failed")

	err = runner.Register(event.NewProjection(
		"fail-proj",
		func(_ context.Context, _ event.Event) error { return handlerErr },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when handler fails during replay")
	}
}

type failingGlobalLoader struct {
	err error
}

func (f *failingGlobalLoader) LoadAll(_ context.Context) ([]event.Event, error) {
	return nil, f.err
}

type failingCheckpointStore struct {
	loadErr error
}

func (f *failingCheckpointStore) Load(_ context.Context, _ string) (id.EventID, error) {
	return id.EventID{}, f.loadErr
}

func (f *failingCheckpointStore) Save(_ context.Context, _ string, _ id.EventID) error {
	return nil
}

func (f *failingCheckpointStore) Close() error { return nil }

func TestRunner_ReplayWithPositionalLoader(t *testing.T) {
	t.Parallel()

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt3 := mustNewEvent(t, "OrderPlaced", id.NewAggregateID())

	store := &positionalLoaderStore{events: []event.Event{evt1, evt2, evt3}}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	ctx := context.Background()

	err := checkpoint.Save(ctx, "user-proj", evt1.ID())
	if err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []id.EventID

	var replayMu sync.Mutex

	registerReplayProjection(t, runner, "user-proj", replayDone, &replayed, &replayMu)

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for positional replay")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (evt2 via PositionalLoader)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != evt2.ID() {
		t.Errorf("replayed event = %v, want %v", replayed[0], evt2.ID())
	}
}

func TestRunner_ReplayEmptyStore(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	loader := &emptyGlobalLoader{}

	runner, err := projection.NewRunner(loader, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Register(event.NewProjection(
		"test-proj",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
}

func TestRunner_SubscribeError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus().SubscribeAllFn(
		func(_ event.Handler) error { return errors.New("subscribe all failed") },
	)

	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Register(event.NewProjection(
		"test-proj",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when SubscribeAll fails")
	}
}

type emptyGlobalLoader struct{}

func (e *emptyGlobalLoader) LoadAll(_ context.Context) ([]event.Event, error) {
	return nil, nil
}

type positionalLoaderStore struct {
	events []event.Event
}

func (p *positionalLoaderStore) LoadAll(_ context.Context) ([]event.Event, error) {
	return p.events, nil
}

func (p *positionalLoaderStore) LoadAllFromPosition(
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
