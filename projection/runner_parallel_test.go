package projection_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func TestRunner_Parallelism_DispatchesConcurrently(t *testing.T) {
	t.Parallel()

	var maxConcurrent atomic.Int32
	var current atomic.Int32
	var totalHandled atomic.Int32

	handler := func(_ context.Context, _ event.Event) error {
		cur := current.Add(1)

		defer current.Add(-1)

		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		totalHandled.Add(1)

		return nil
	}

	runner, bus, ready := newTestRunnerWithReadyAndOpts(t, projection.WithParallelism(4))

	for i := range 4 {
		err := runner.Register(event.NewProjection(
			"proj-"+string(rune('A'+i)),
			handler,
			[]event.Type{"UserCreated"},
		))
		if err != nil {
			t.Fatal(err)
		}
	}

	defer startRunner(t, runner, ready)()

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err := bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)

	for totalHandled.Load() < 4 {
		select {
		case <-deadline:
			t.Fatalf("timed out: only %d/4 projections handled", totalHandled.Load())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrent execution (maxConcurrent >= 2), got %d", maxConcurrent.Load())
	}
}

func TestRunner_Parallelism_CheckpointsSavedCorrectly(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup

	wg.Add(3)

	runner, bus, ready := newTestRunnerWithReadyAndOpts(t, projection.WithParallelism(3))

	for i := range 3 {
		err := runner.Register(event.NewProjection(
			"proj-"+string(rune('A'+i)),
			func(_ context.Context, _ event.Event) error {
				defer wg.Done()

				return nil
			},
			[]event.Type{"UserCreated"},
		))
		if err != nil {
			t.Fatal(err)
		}
	}

	defer startRunner(t, runner, ready)()

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	err := bus.Publish(context.Background(), evt)
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	for i := range 3 {
		name := "proj-" + string(rune('A'+i))

		cp, cpErr := runner.CurrentCheckpoint(context.Background(), name)
		if cpErr != nil {
			t.Fatalf("checkpoint for %s: %v", name, cpErr)
		}

		if cp.EventID != evt.ID() {
			t.Errorf("checkpoint for %s = %v, want %v", name, cp.EventID, evt.ID())
		}
	}
}
