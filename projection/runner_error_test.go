package projection_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

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

	defer startRunner(t, runner, ready)()

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

	defer startRunner(t, runner, ready)()

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	_ = bus.Publish(context.Background(), evt)

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

func TestRunner_ReplayError_ReadAllFails(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	loader := &failingJournal{err: errors.New("read all failed")}

	runner, err := projection.NewRunner(loader, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	registerNoopProjection(t, runner, "test-proj", []event.Type{"UserCreated"})

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when ReadAll fails")
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

	registerNoopProjection(t, runner, "test-proj", []event.Type{"UserCreated"})

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

	checkpoint := memory.NewMemoryCheckpointStore()

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

func TestRunner_CloseStopsRun(t *testing.T) {
	t.Parallel()

	runner, _, ready := newTestRunnerWithReady(t)

	registerNoopProjection(t, runner, "close-proj", []event.Type{"UserCreated"})

	done := make(chan struct{})

	go func() {
		_ = runner.Run(context.Background())
		close(done)
	}()

	<-ready

	if err := runner.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Close")
	}
}

func TestRunner_SubscribeError(t *testing.T) {
	t.Parallel()

	bus := testhelpers.NewFakeBus().SubscribeAllFn(
		func(_ event.Handler) error { return errors.New("subscribe all failed") },
	)

	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	registerNoopProjection(t, runner, "test-proj", []event.Type{"UserCreated"})

	runErr := runner.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected error when SubscribeAll fails")
	}
}
