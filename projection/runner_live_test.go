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

func registerChanProjection(
	t *testing.T,
	runner *projection.Runner,
	name string,
	types []event.Type,
	handled chan<- string,
) {
	t.Helper()
	err := runner.Register(event.NewProjection(name, func(_ context.Context, evt event.Event) error {
		handled <- string(evt.Type())
		return nil
	}, types))
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunner_ProcessesLiveEvents(t *testing.T) {
	t.Parallel()

	handled := make(chan string, 1)

	runner, bus, ready := newTestRunnerWithReady(t)

	registerChanProjection(t, runner, "user-proj", []event.Type{"UserCreated"}, handled)

	defer startRunner(t, runner, ready)()

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	var err error
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

	checkpoint := memory.NewMemoryCheckpointStore()

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

	defer startRunner(t, runner, ready)()

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

	registerChanProjection(t, runner, "user-proj", []event.Type{"UserCreated"}, handled)

	defer startRunner(t, runner, ready)()

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

	defer startRunner(t, runner, ready)()

	_ = bus.Publish(context.Background(), mustNewEvent(t, "UserCreated", id.NewAggregateID()))
	_ = bus.Publish(context.Background(), mustNewEvent(t, "OrderPlaced", id.NewAggregateID()))

	drainChan(t, handled, 2, "event")
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

	defer startRunner(t, runner, ready)()

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
