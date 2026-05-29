package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestRunner_Register(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps:    []saga.Step{{Name: "create"}},
	}

	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}
}

func TestRunner_RegisterDuplicate(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{sagaType: "order", steps: []saga.Step{{Name: "create"}}}

	if err := runner.Register(def); err != nil {
		t.Fatalf("first register: %v", err)
	}

	err := runner.Register(def)
	if !errors.Is(err, saga.ErrSagaAlreadyExists) {
		t.Fatalf("expected ErrSagaAlreadyExists, got: %v", err)
	}
}

func TestRunner_RegisterNil(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	if err := runner.Register(nil); err == nil {
		t.Fatal("expected error for nil definition")
	}
}

func TestRunner_RegisterEmptyType(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{sagaType: ""}
	if err := runner.Register(def); err == nil {
		t.Fatal("expected error for empty saga type")
	}
}

func TestRunner_Start(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	registerSimpleSaga(t, runner)

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if instance.Status != saga.StatusRunning {
		t.Errorf("expected status Running, got %s", instance.Status)
	}

	if len(instance.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(instance.Steps))
	}
}

func TestRunner_StartNotRegistered(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	ctx := context.Background()
	_, err := runner.Start(ctx, "unknown", nil)
	if !errors.Is(err, saga.ErrSagaNotRegistered) {
		t.Fatalf("expected ErrSagaNotRegistered, got: %v", err)
	}
}

func TestRunner_StartInitialCommandFails(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := failingDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	registerSimpleSaga(t, runner)

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", &testCommand{})
	if err == nil {
		t.Fatal("expected error when initial command fails")
	}
	if instance.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", instance.Status)
	}
}
