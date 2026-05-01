package projection_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func TestNewRunner_NilStore(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, memory.NewMemoryBus(), memory.NewCheckpointStore())
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestNewRunner_NilBus(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(memory.NewMemoryStore(), nil, memory.NewCheckpointStore())
	if err == nil {
		t.Fatal("expected error for nil bus")
	}
}

func TestNewRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(memory.NewMemoryStore(), memory.NewMemoryBus(), nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
	}
}

func TestRunner_On_RegistersHandler(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.On("UserCreated", func(_ context.Context, _ event.Event) error { return nil })
	if err != nil {
		t.Fatalf("On: %v", err)
	}
}

func TestRunner_On_NilHandler(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.On("UserCreated", nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestRunner_ProcessesLiveEvents(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 1)

	runner, bus := newTestRunnerWithBus(t)

	err := runner.On("UserCreated", func(_ context.Context, evt event.Event) error {
		handled <- string(evt.Type())

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID(), "User", 1)

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

	runner, bus := newTestRunnerWithBus(t)

	err := runner.On("UserCreated", func(_ context.Context, _ event.Event) error {
		close(done)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID(), "User", 1)

	err = bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler")
	}

	cp, err := runner.CurrentCheckpoint(context.Background())
	if err != nil {
		t.Fatalf("CurrentCheckpoint: %v", err)
	}

	if cp != evt.ID() {
		t.Errorf("checkpoint = %v, want %v", cp, evt.ID())
	}
}

func TestRunner_FiltersUnregisteredTypes(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 1)

	runner, bus := newTestRunnerWithBus(t)

	err := runner.On("UserCreated", func(_ context.Context, evt event.Event) error {
		handled <- string(evt.Type())

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	otherEvt := mustNewEvent(t, "OrderPlaced", id.NewAggregateID(), "Order", 1)

	_ = bus.Publish(context.Background(), otherEvt)

	select {
	case h := <-handled:
		t.Errorf("should not have handled %q", h)
	case <-time.After(50 * time.Millisecond):
	}
}

func newTestRunner(t *testing.T) *projection.Runner {
	t.Helper()

	r, _ := newTestRunnerWithBus(t)

	return r
}

func newTestRunnerWithBus(t *testing.T) (*projection.Runner, *memory.MemoryBus) {
	t.Helper()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() {
		_ = store.Close()
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, bus
}

func mustNewEvent(
	t *testing.T,
	eventType string,
	aggID id.AggregateID,
	aggType string,
	version int,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		event.AggregateType(aggType),
		version,
		nil,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}
