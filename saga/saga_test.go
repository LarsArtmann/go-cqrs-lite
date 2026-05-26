package saga_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

// nopDispatcher is a command dispatcher that always succeeds.
type nopDispatcher struct{}

func (nopDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	return nil
}

// failingDispatcher is a command dispatcher that always fails.
type failingDispatcher struct{}

func (failingDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	return errors.New("dispatch failed")
}

// countingDispatcher tracks how many times Dispatch is called.
type countingDispatcher struct {
	count int
}

func (d *countingDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	d.count++
	return nil
}

// dispatchFunc is a function-based command dispatcher for testing.
type dispatchFunc func(ctx context.Context, cmd command.Command) error

func (f dispatchFunc) Dispatch(ctx context.Context, cmd command.Command) error {
	return f(ctx, cmd)
}

// testDefinition is a saga definition for testing.
type testDefinition struct {
	sagaType string
	steps    []saga.Step
}

func (d testDefinition) SagaType() string { return d.sagaType }
func (d testDefinition) Steps() []saga.Step { return d.steps }

// testCommand is a minimal command implementation for testing.
type testCommand struct {
	command.BasicCommand
}

func newTestCommand(_ context.Context, _ id.AggregateID) command.Command {
	return &testCommand{}
}

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()
	inst := &saga.Instance{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusPending,
	}

	if err := store.Save(ctx, inst); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, inst.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.ID != inst.ID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, inst.ID)
	}
}

func TestMemoryStore_LoadNotFound(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Load(ctx, id.NewAggregateID())
	if !errors.Is(err, saga.ErrSagaNotFound) {
		t.Fatalf("expected ErrSagaNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadAllRunning(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	ctx := context.Background()

	// Create running instance
	running := &saga.Instance{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusRunning,
	}
	if err := store.Save(ctx, running); err != nil {
		t.Fatalf("save running: %v", err)
	}

	// Create completed instance
	completed := &saga.Instance{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusCompleted,
	}
	if err := store.Save(ctx, completed); err != nil {
		t.Fatalf("save completed: %v", err)
	}

	all, err := store.LoadAllRunning(ctx)
	if err != nil {
		t.Fatalf("load all running: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 running, got %d", len(all))
	}

	if all[0].ID != running.ID {
		t.Errorf("wrong instance: got %v, want %v", all[0].ID, running.ID)
	}
}

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

func TestRunner_Start(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "create", Action: newTestCommand},
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

	// Execute first step
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.CurrentStep != 1 {
		t.Errorf("expected step 1, got %d", loaded.CurrentStep)
	}

	// Execute second step
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 2: %v", err)
	}

	loaded, _ = store.Load(ctx, instance.ID)
	if loaded.Status != saga.StatusCompleted {
		t.Errorf("expected status Completed, got %s", loaded.Status)
	}

	if dispatcher.count != 2 {
		t.Errorf("expected 2 dispatches, got %d", dispatcher.count)
	}
}

func TestRunner_ExecuteStep_FailureWithCompensation(t *testing.T) {
	t.Parallel()

	// Use a dispatcher that succeeds on first call, fails on second
	// This simulates: step 1 succeeds, step 2 fails → compensate step 1
	callCount := 0
	var dispatchErr error
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount == 1 {
			return nil // step 1 succeeds
		}
		return dispatchErr // step 2 fails
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
			{Name: "charge", Action: newTestCommand}, // fails, triggers compensation
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

	// Execute step 1 — succeeds
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	loaded, _ := store.Load(ctx, instance.ID)
	if loaded.CurrentStep != 1 {
		t.Fatalf("expected step 1, got %d", loaded.CurrentStep)
	}

	// Execute step 2 — fails, triggers compensation
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

func TestRunner_ExecuteStep_Timeout(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := nopDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	slowAction := func(ctx context.Context, _ id.AggregateID) command.Command {
		// Simulate slow operation
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

	// Execute step — will timeout
	err = runner.ExecuteStep(ctx, instance.ID)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
