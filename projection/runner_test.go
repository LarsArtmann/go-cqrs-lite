package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

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

	runner, bus := newTestRunnerWithBus(t)

	err := runner.Register(event.NewProjection("user-proj",
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

	time.Sleep(20 * time.Millisecond)

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

	runner, bus := newTestRunnerWithBusAndCheckpoint(t, checkpoint)

	err := runner.Register(event.NewProjection("user-proj",
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

	time.Sleep(20 * time.Millisecond)

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

	runner, bus := newTestRunnerWithBus(t)

	err := runner.Register(event.NewProjection("user-proj",
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

	time.Sleep(20 * time.Millisecond)

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

	runner, bus := newTestRunnerWithBus(t)

	err := runner.Register(event.NewProjection("all-proj",
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

	time.Sleep(20 * time.Millisecond)

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

	var replayed []string

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection("replay-proj",
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

	runner, bus := newTestRunnerWithBus(t)

	err := runner.Register(event.NewProjection("user-proj",
		func(_ context.Context, evt event.Event) error {
			userHandled <- string(evt.Type())

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	err = runner.Register(event.NewProjection("order-proj",
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

	time.Sleep(20 * time.Millisecond)

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

	var replayed []id.EventID

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection("replay-proj",
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()

			replayed = append(replayed, evt.ID())

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
		t.Errorf("replayed %d events, want 1 (skipped past checkpoint)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != evt2.ID() {
		t.Errorf("replayed event = %v, want %v (evt after checkpoint)", replayed[0], evt2.ID())
	}
}

func newTestRunner(t *testing.T) *projection.Runner {
	t.Helper()

	r, _ := newTestRunnerWithBus(t)

	return r
}

func newTestRunnerWithBus(t *testing.T) (*projection.Runner, *memory.MemoryBus) {
	t.Helper()

	bus := memory.NewMemoryBus()
	checkpoint := memory.NewCheckpointStore()

	t.Cleanup(func() {
		_ = bus.Close()
		_ = checkpoint.Close()
	})

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, bus
}

func newTestRunnerWithBusAndCheckpoint(
	t *testing.T,
	checkpoint *memory.MemoryCheckpointStore,
) (*projection.Runner, *memory.MemoryBus) {
	t.Helper()

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	return runner, bus
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
