package saga_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestRunner_WithLogger_StartedAndCompleted(t *testing.T) {
	t.Parallel()

	log := &mockLogger{}
	store := saga.NewMemoryStore()
	dispatcher := &countingDispatcher{}
	runner := saga.NewRunner(store, dispatcher, saga.WithLogger(log))

	registerSimpleSaga(t, runner)

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	infos := log.getInfos()
	if len(infos) < 1 || infos[0] != "saga started" {
		t.Errorf("expected 'saga started' log, got %v", infos)
	}

	requireExecuteStep(t, ctx, runner, instance.ID, "step")

	infos = log.getInfos()
	found := false
	for _, msg := range infos {
		if msg == "saga completed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'saga completed' log, got %v", infos)
	}
}

func TestRunner_WithLogger_StepFailed(t *testing.T) {
	t.Parallel()

	log := &mockLogger{}
	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, failingDispatcher{}, saga.WithLogger(log))

	registerSimpleSaga(t, runner)

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	_ = runner.ExecuteStep(ctx, instance.ID)

	errs := log.getErrors()
	if len(errs) == 0 {
		t.Fatal("expected error log for step failure")
	}
	found := false
	for _, msg := range errs {
		if msg == "step failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'step failed' log, got %v", errs)
	}
}
