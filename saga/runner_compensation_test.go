package saga_test

import (
	"context"
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
	dispatcher := newFailingDispatcher(&callCount)

	runner, instance, _ := setupTestSaga(
		t, dispatcher,
		[]saga.Step{{Name: "create", Action: newTestCommand}},
		saga.WithRetryPolicy(1, 10*time.Millisecond),
		saga.WithRetryMultiplier(3.0),
	)

	ctx := context.Background()

	requireExecuteStepError(t, ctx, runner, instance.ID, "expected retry exhaustion error")

	if callCount != 2 {
		t.Errorf("expected 2 dispatch calls with maxRetries=1, got %d", callCount)
	}
}

func TestRunner_CompensateFailure(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := newThresholdDispatcher(&callCount, 1, "always fails")

	compensateFails := func(_ context.Context, _ id.AggregateID) command.Command {
		return &testCommand{}
	}

	runner, instance, store := setupTestSaga(t, dispatcher, []saga.Step{
		{Name: "create", Action: newTestCommand, Compensate: compensateFails},
		{Name: "charge", Action: newTestCommand},
	})

	ctx := context.Background()

	requireExecuteStep(t, ctx, runner, instance.ID, "step 1")

	requireExecuteStepError(t, ctx, runner, instance.ID, "expected error")

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", loaded.Status)
	}
}

func TestRunner_CompensateNilCompensateSkipped(t *testing.T) {
	t.Parallel()

	callCount := 0
	compensateCalled := false
	dispatcher := newThresholdDispatcher(&callCount, 2, "step 3 fails")

	runner, instance, _ := setupTestSaga(t, dispatcher, []saga.Step{
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
	})

	ctx := context.Background()

	requireExecuteStep(t, ctx, runner, instance.ID, "step 1")
	requireExecuteStep(t, ctx, runner, instance.ID, "step 2")

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

	runner, instance, _ := setupTestSaga(t, dispatcher, []saga.Step{
		{
			Name:   "create",
			Action: newTestCommand,
			Compensate: func(_ context.Context, _ id.AggregateID) command.Command {
				compensateCalled = true
				return nil
			},
		},
		{Name: "charge", Action: newTestCommand},
	})

	ctx := context.Background()

	requireExecuteStep(t, ctx, runner, instance.ID, "step 1")

	_ = runner.ExecuteStep(ctx, instance.ID)

	if !compensateCalled {
		t.Error("expected compensation to be called (even though it returns nil)")
	}

	if callCount != 2 {
		t.Errorf("expected 2 dispatches, got %d", callCount)
	}
}
