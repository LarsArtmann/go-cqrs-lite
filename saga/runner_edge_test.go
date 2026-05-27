package saga_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func TestRunner_ConcurrentInstances(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := &countingDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps:    []saga.Step{{Name: "create", Action: newTestCommand}},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance, err := runner.Start(ctx, "order", nil)
			if err != nil {
				t.Errorf("start: %v", err)
				return
			}
			if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
				t.Errorf("execute: %v", err)
			}
		}()
	}

	wg.Wait()

	if dispatcher.Count() != 10 {
		t.Errorf("expected 10 dispatches, got %d", dispatcher.Count())
	}
}

func TestRunner_StoreErrorOnSave(t *testing.T) {
	t.Parallel()

	store := &errorStore{err: errors.New("store unavailable")}
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{sagaType: "order", steps: []saga.Step{{Name: "create"}}}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	_, err := runner.Start(ctx, "order", nil)
	if err == nil {
		t.Fatal("expected store error")
	}
}

func TestRunner_TimeoutCancellation(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "slow", Action: func(ctx context.Context, _ id.AggregateID) command.Command {
				<-ctx.Done()
				return nil
			}, Timeout: 1 * time.Millisecond},
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

	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRunner_ExecuteStep_LoadError(t *testing.T) {
	t.Parallel()

	store := &errorStore{err: errors.New("load error")}
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{sagaType: "order", steps: []saga.Step{{Name: "create"}}}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	err := runner.ExecuteStep(ctx, id.NewAggregateID())
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestRunner_ExecuteStep_AlreadyAtEnd(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{sagaType: "empty", steps: []saga.Step{}}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "empty", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step: %v", err)
	}

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusCompleted {
		t.Errorf("expected Completed for empty-step saga, got %s", loaded.Status)
	}
}

func TestRunner_ExecuteStep_HydrateUnregisteredSagaType(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	ctx := context.Background()
	state := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "unknown-saga",
		Status:      saga.StatusRunning,
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	err := runner.ExecuteStep(ctx, state.ID)
	if err == nil {
		t.Fatal("expected error for unregistered saga type during hydration")
	}

	if !errors.Is(err, saga.ErrSagaNotRegistered) {
		t.Fatalf("expected ErrSagaNotRegistered, got: %v", err)
	}
}
