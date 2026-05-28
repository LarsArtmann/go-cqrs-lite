package saga_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

type nopDispatcher struct{}

func (nopDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	return nil
}

type failingDispatcher struct{}

func (failingDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	return errors.New("dispatch failed")
}

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

type dispatchFunc func(ctx context.Context, cmd command.Command) error

func (f dispatchFunc) Dispatch(ctx context.Context, cmd command.Command) error {
	return f(ctx, cmd)
}

type testDefinition struct {
	sagaType string
	steps    []saga.Step
}

func (d testDefinition) SagaType() string   { return d.sagaType }
func (d testDefinition) Steps() []saga.Step { return d.steps }

type testCommand struct {
	command.BasicCommand
}

func newTestCommand(_ context.Context, _ id.AggregateID) command.Command {
	return &testCommand{}
}

type errorStore struct {
	err error
}

func (s *errorStore) Save(_ context.Context, _ *saga.State) error { return s.err }
func (s *errorStore) Load(_ context.Context, _ id.AggregateID) (*saga.State, error) {
	return nil, s.err
}

func (s *errorStore) LoadAllRunning(_ context.Context) ([]*saga.State, error) { return nil, s.err }

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

// setupTestSaga creates a runner, registers a test definition, starts a saga instance,
// and returns the runner, instance, and store for test assertions.
func setupTestSaga(
	tb testing.TB,
	dispatcher saga.CommandDispatcher,
	steps []saga.Step,
	opts ...saga.RunnerOption,
) (*saga.Runner, *saga.Instance, saga.Store) {
	tb.Helper()

	store := saga.NewMemoryStore()
	runner := saga.NewRunner(store, dispatcher, opts...)
	def := testDefinition{sagaType: "order", steps: steps}
	if err := runner.Register(def); err != nil {
		tb.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		tb.Fatalf("start: %v", err)
	}

	return runner, instance, store
}
