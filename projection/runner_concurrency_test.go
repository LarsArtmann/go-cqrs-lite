package projection_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestRunner_Concurrency_RegisterThenRunAndHandle(t *testing.T) {
	t.Parallel()

	runner, bus, ready := newTestRunnerWithReady(t)
	defer func() { _ = runner.Close() }()

	var handled atomic.Int32

	err := runner.Register(event.NewProjection(
		"test-proj",
		func(_ context.Context, _ event.Event) error {
			handled.Add(1)
			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() { _ = runner.Run(ctx) }()

	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	if err := bus.Publish(t.Context(), evt); err != nil {
		t.Fatal(err)
	}

	if handled.Load() != 1 {
		t.Fatalf("expected 1 handled event, got %d", handled.Load())
	}
}

func TestRunner_Concurrency_RunThenClose(t *testing.T) {
	t.Parallel()

	runner, _, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"test-proj",
		eventtest.NoopEventHandler(),
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() { done <- runner.Run(ctx) }()

	<-ready

	err = runner.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if runErr := <-done; runErr != nil {
		t.Fatalf("Run returned error after Close: %v", runErr)
	}
}

func TestRunner_Concurrency_MultipleCloses(t *testing.T) {
	t.Parallel()

	runner, _, _ := newTestRunnerWithReady(t)

	for range 3 {
		err := runner.Close()
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func TestRunner_Concurrency_DoubleRun(t *testing.T) {
	t.Parallel()

	runner, _, ready := newTestRunnerWithReady(t)
	defer func() { _ = runner.Close() }()

	err := runner.Register(event.NewProjection(
		"test-proj",
		eventtest.NoopEventHandler(),
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() { _ = runner.Run(ctx) }()
	<-ready

	err = runner.Run(ctx)
	if err == nil {
		t.Fatal("expected error on second Run call")
	}
}

func TestRunner_Concurrency_HandlerStopsAfterClose(t *testing.T) {
	t.Parallel()

	runner, bus, ready := newTestRunnerWithReady(t)

	err := runner.Register(event.NewProjection(
		"test-proj",
		eventtest.NoopEventHandler(),
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() { _ = runner.Run(ctx) }()
	<-ready

	evt := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	if err := bus.Publish(t.Context(), evt); err != nil {
		t.Fatal(err)
	}

	_ = runner.Close()

	_ = mustNewEvent(t, "UserCreated", id.NewAggregateID())
	_ = mustNewEvent(t, "UserCreated", id.NewAggregateID())
}
