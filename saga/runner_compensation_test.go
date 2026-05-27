package saga_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestRunner_WithRetryPolicy(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		return event.NewTransient("test.transient", "temporary failure")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(
		store, dispatcher,
		saga.WithRetryPolicy(1, 10*time.Millisecond),
		saga.WithRetryMultiplier(3.0),
	)

	def := testDefinition{
		sagaType: "order",
		steps:    []saga.Step{{Name: "create", Action: newTestCommand}},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected retry exhaustion error")
	}

	if callCount != 2 {
		t.Errorf("expected 2 dispatch calls with maxRetries=1, got %d", callCount)
	}
}

func TestRunner_CompensateFailure(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return errors.New("always fails")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

	compensateFails := func(_ context.Context, _ id.AggregateID) command.Command {
		return &testCommand{}
	}

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "create", Action: newTestCommand, Compensate: compensateFails},
			{Name: "charge", Action: newTestCommand},
		},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	err = runner.ExecuteStep(ctx, instance.ID)
	if err == nil {
		t.Fatal("expected error")
	}

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", loaded.Status)
	}
}

func TestRunner_CompensateNilCompensateSkipped(t *testing.T) {
	t.Parallel()

	callCount := 0
	compensateCalled := false
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount <= 2 {
			return nil
		}
		return errors.New("step 3 fails")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "step1", Action: newTestCommand},
			{
				Name:   "step2",
				Action: newTestCommand,
				Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
					compensateCalled = true
					return &testCommand{}
				},
			},
			{Name: "step3", Action: newTestCommand},
		},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 2: %v", err)
	}

	_ = runner.ExecuteStep(ctx, instance.ID)

	if !compensateCalled {
		t.Error("expected compensation for step 2 to be called")
	}
}

func TestRunner_CompensateReturnsNilSkipped(t *testing.T) {
	t.Parallel()

	callCount := 0
	compensateCalled := false
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return event.NewRejection("test.reject", "non-retryable failure")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{
				Name:   "create",
				Action: newTestCommand,
				Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
					compensateCalled = true
					return nil
				},
			},
			{Name: "charge", Action: newTestCommand},
		},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	_ = runner.ExecuteStep(ctx, instance.ID)

	if !compensateCalled {
		t.Error("expected compensation to be called (even though it returns nil)")
	}

	if callCount != 2 {
		t.Errorf("expected 2 dispatches, got %d", callCount)
	}
}
