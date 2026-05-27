package saga_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
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
	mu    sync.Mutex
	count int
}

func (d *countingDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	d.mu.Lock()
	d.count++
	d.mu.Unlock()
	return nil
}

func (d *countingDispatcher) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.count
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

func (d testDefinition) SagaType() string   { return d.sagaType }
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
	inst := &saga.State{
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
	running := &saga.State{
		ID:       id.NewAggregateID(),
		SagaType: "test",
		Status:   saga.StatusRunning,
	}
	if err := store.Save(ctx, running); err != nil {
		t.Fatalf("save running: %v", err)
	}

	// Create completed instance
	completed := &saga.State{
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

func TestRunner_StartInitialCommandFails(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	dispatcher := failingDispatcher{}
	runner := saga.NewRunner(store, dispatcher)

	def := testDefinition{
		sagaType: "order",
		steps:    []saga.Step{{Name: "create", Action: newTestCommand}},
	}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", &testCommand{})
	if err == nil {
		t.Fatal("expected error when initial command fails")
	}
	if instance.Status != saga.StatusFailed {
		t.Errorf("expected status Failed, got %s", instance.Status)
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

	if dispatcher.Count() != 2 {
		t.Errorf("expected 2 dispatches, got %d", dispatcher.Count())
	}
}

func TestRunner_ExecuteStep_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

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

	// Complete the saga
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step: %v", err)
	}

	// Try to execute again — should return error since saga is completed
	if err := runner.ExecuteStep(ctx, instance.ID); err == nil {
		t.Fatal("expected error when executing completed saga")
	}
}

func TestRunner_ExecuteStep_NilAction(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	def := testDefinition{
		sagaType: "order",
		steps: []saga.Step{
			{Name: "nil", Action: func(_ context.Context, _ id.AggregateID) command.Command { return nil }},
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
		t.Fatal("expected error for nil action")
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

func TestRunner_ExecuteStep_FirstStepFailsNoCompensation(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, failingDispatcher{})

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

	// First step fails — no compensation possible (no completed steps)
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

func TestRunner_ExecuteStep_RetryExhaustion(t *testing.T) {
	t.Parallel()

	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		// Return a retryable error
		return event.NewTransient("test.transient", "temporary failure")
	})

	store := saga.NewMemoryStore()
	// Default config: maxRetries=3, retryDelay=10ms
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

	// 1 initial + 3 retries = 4 calls
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

// errorStore is a saga.Store that always returns an error.
type errorStore struct {
	err error
}

func (s *errorStore) Save(_ context.Context, _ *saga.State) error { return s.err }
func (s *errorStore) Load(_ context.Context, _ id.AggregateID) (*saga.State, error) {
	return nil, s.err
}

func (s *errorStore) LoadAllRunning(_ context.Context) ([]*saga.State, error) { return nil, s.err }

// mockLogger captures log calls for assertions.
type mockLogger struct {
	mu    sync.Mutex
	infos []string
	errs  []string
}

func (l *mockLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}

func (l *mockLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, msg)
}

func (l *mockLogger) getInfos() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.infos))
	copy(result, l.infos)
	return result
}

func (l *mockLogger) getErrors() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.errs))
	copy(result, l.errs)
	return result
}

func TestRunner_WithLogger_StartedAndCompleted(t *testing.T) {
	t.Parallel()

	log := &mockLogger{}
	store := saga.NewMemoryStore()
	dispatcher := &countingDispatcher{}
	runner := saga.NewRunner(store, dispatcher, saga.WithLogger(log))

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

	infos := log.getInfos()
	if len(infos) < 1 || infos[0] != "saga started" {
		t.Errorf("expected 'saga started' log, got %v", infos)
	}

	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step: %v", err)
	}

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

	// 1 initial + 1 retry = 2 calls
	if callCount != 2 {
		t.Errorf("expected 2 dispatch calls with maxRetries=1, got %d", callCount)
	}
}

func TestRunner_CompensateFailure(t *testing.T) {
	t.Parallel()

	// Dispatcher succeeds on first call (step 1), fails on second (step 2),
	// and also fails on compensation dispatch
	callCount := 0
	dispatcher := dispatchFunc(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount == 1 {
			return nil // step 1 succeeds
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

	// Step 1 succeeds
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	// Step 2 fails → compensation fails too
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
			return nil // steps 1 and 2 succeed
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

	// Execute steps 1 and 2
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 2: %v", err)
	}

	// Step 3 fails → compensate step 2 (step 1 has no Compensate)
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
			return nil // step 1 succeeds
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
					return nil // returns nil — dispatch should be skipped
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

	// Step 1 succeeds
	if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
		t.Fatalf("execute step 1: %v", err)
	}

	// Step 2 fails → compensation returns nil, should be skipped
	_ = runner.ExecuteStep(ctx, instance.ID)

	if !compensateCalled {
		t.Error("expected compensation to be called (even though it returns nil)")
	}

	// Only 2 dispatches: step 1 + step 2 (compensate returned nil, no dispatch)
	if callCount != 2 {
		t.Errorf("expected 2 dispatches, got %d", callCount)
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

	// Should timeout
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
	// ExecuteStep with a random ID — store.Load will fail.
	err := runner.ExecuteStep(ctx, id.NewAggregateID())
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestRunner_ExecuteStep_AlreadyAtEnd(t *testing.T) {
	t.Parallel()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, nopDispatcher{})

	// Define saga with no steps
	def := testDefinition{sagaType: "empty", steps: []saga.Step{}}
	if err := runner.Register(def); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "empty", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// ExecuteStep should mark as completed since there are no steps
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

	// Do NOT register any saga definition.

	// Manually insert a state with an unregistered saga type.
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

	// ExecuteStep should fail during hydration because the saga type is not registered.
	err := runner.ExecuteStep(ctx, state.ID)
	if err == nil {
		t.Fatal("expected error for unregistered saga type during hydration")
	}

	if !errors.Is(err, saga.ErrSagaNotRegistered) {
		t.Fatalf("expected ErrSagaNotRegistered, got: %v", err)
	}
}
