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

func TestRunner_ExecuteStep_HappyPath(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := &countingDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "create", Action: newTestCommand},
			{Name: "confirm", Action: newTestCommand},
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

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.CurrentStep != 1 {
		t.Errorf("expected step 1, got %d", loaded.CurrentStep)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 2: %v", err)
	}

	loaded, _ = store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusCompleted {
		t.Errorf("expected status Completed, got %s", loaded.Status)
	}

	if dispatcher.Count() != 2 {
		t.Errorf("expected 2 dispatches, got %d", dispatcher.Count())
	}
}

func TestRunner_ExecuteStep_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	runner, instance, _ := setupTestSaga(t, nopDispatcher{},
		[]saga.Step{{Name: "create", Action: newTestCommand}})

	ctx := context.Background()

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected error when executing completed saga")
	}
}

func TestRunner_ExecuteStep_NilAction(t *testing.T) {
	t.Parallel()

	runner, instance, _ := setupTestSaga(t, nopDispatcher{}, []saga.Step{
		{
			Name:   "nil",
			Action: func(_ context.Context, _ id.AggregateID) command.Command { return nil },
		},
	})

	ctx := context.Background()
	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected error for nil action")
	}
}

func TestRunner_ExecuteStep_FailureWithCompensation(t *testing.T) {
	t.Parallel()

	callCount := 0
	var dispatchErr error
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return dispatchErr
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

	compensateCalled := false
	compensateFn := func(_ context.Context, _ id.AggregateID) command.Command {
		compensateCalled = true
		return &testCommand{}
	}

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "create", Action: newTestCommand, Compensate: compensateFn},
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

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.CurrentStep != 1 {
		t.Fatalf("expected step 1, got %d", loaded.CurrentStep)
	}

	dispatchErr = errors.New("payment failed")
	err = runner.ExecuteStep(ctx, instance.ID)
	if err == nil {
		t.Fatal("expected error on step 2, got nil")
	}

	loaded, _ = store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", loaded.Status)
	}

	if !compensateCalled {
		t.Error("expected compensation to be called")
	}
}

func TestRunner_ExecuteStep_FirstStepFailsNoCompensation(t *testing.T) {
	t.Parallel()

	runner, instance, store := setupTestSaga(t, failingDispatcher{},
		[]saga.Step{{Name: "create", Action: newTestCommand}})

	ctx := context.Background()

	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected error on first step failure")
	}

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", loaded.Status)
	}
}

func TestRunner_ExecuteStep_Timeout(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	slowAction := func(ctx context.Context, _ id.AggregateID) command.Command {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "slow", Action: slowAction, Timeout: 10 * time.Millisecond},
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

	err = runner.ExecuteStep(ctx, instance.ID)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestRunner_ExecuteStep_RetryExhaustion(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		return event.NewTransient("test.transient", "temporary failure")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

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

	if callCount != 4 {
		t.Errorf("expected 4 dispatch calls, got %d", callCount)
	}
}

func TestRunner_ExecuteStep_NonRetryable(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		return event.NewRejection("test.rejection", "permanent failure")
	})

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher)

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
		t.Fatal("expected error")
	}

	if callCount != 1 {
		t.Errorf("expected 1 dispatch call (non-retryable), got %d", callCount)
	}
}
