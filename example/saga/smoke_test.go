package main

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestOrderSaga_CompletesAllSteps(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := saga.NewMemoryStore()
	dispatcher := &loggingDispatcher{}

	runner := saga.NewRunner(
		store, dispatcher,
		saga.WithRetryPolicy(3, 100*time.Millisecond),
	)

	if err := runner.Register(orderSaga{}); err != nil {
		t.Fatalf("register saga: %v", err)
	}

	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start saga: %v", err)
	}

	if instance.Status != saga.StatusRunning {
		t.Fatalf("expected running, got %s", instance.Status)
	}

	for instance.Status == saga.StatusRunning {
		if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
			t.Fatalf("execute step: %v", err)
		}

		loaded, loadErr := store.Load(ctx, instance.ID)
		if loadErr != nil {
			t.Fatalf("load state: %v", loadErr)
		}

		instance.State = *loaded
	}

	if instance.Status != saga.StatusCompleted {
		t.Errorf("expected completed, got %s", instance.Status)
	}

	expected := []string{"reserve-inventory", "charge-payment", "confirm-order"}
	if len(dispatcher.dispatched) != len(expected) {
		t.Fatalf(
			"expected %d dispatched commands, got %d: %v",
			len(expected),
			len(dispatcher.dispatched),
			dispatcher.dispatched,
		)
	}

	for i, cmd := range expected {
		if dispatcher.dispatched[i] != cmd {
			t.Errorf("dispatched[%d] = %q, want %q", i, dispatcher.dispatched[i], cmd)
		}
	}
}
