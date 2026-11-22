package main

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestScheduler_FiresDueTimer pins the engine-path wiring end-to-end: a
// timer scheduled as already-due is claimed and dispatched by the scheduler
// polling loop, and MarkFired retires it (a second poll dispatches nothing
// new).
func TestScheduler_FiresDueTimer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eng := metaengine.NewMemoryEngine()

	store, err := engine.NewTimerStore[reminder](eng, engine.WithCollection("test_timers"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Schedule(ctx, scheduling.Timer[reminder]{
		ID:      scheduling.MustParseTimerID("engine-path-test"),
		FireAt:  time.Now().Add(-time.Second),
		Payload: reminder{OrderID: "order-1", Note: "test"},
	}); err != nil {
		t.Fatal(err)
	}

	fired := make(chan scheduling.Timer[reminder], 1)

	scheduler := scheduling.New[reminder](
		store,
		func(_ context.Context, timer scheduling.Timer[reminder]) error {
			fired <- timer

			return nil
		},
	)

	go func() {
		_ = scheduler.Start(ctx)
	}()

	select {
	case timer := <-fired:
		if timer.Payload.OrderID != "order-1" {
			t.Fatalf("dispatched payload: got %+v, want order-1", timer.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("due timer was not dispatched within 5s")
	}

	// The claim retires the timer: no double dispatch within a window that
	// spans several poll intervals.
	select {
	case timer := <-fired:
		t.Fatalf("timer dispatched twice: %+v", timer)
	case <-time.After(300 * time.Millisecond):
	}
}
