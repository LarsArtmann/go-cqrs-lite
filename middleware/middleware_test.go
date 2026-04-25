package middleware

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type testCommand struct {
	aggregateID id.AggregateID
}

func (c *testCommand) Type() command.Type  { return "test.cmd" }
func (c *testCommand) AggregateID() string { return c.aggregateID.String() }

type testLogger struct {
	mu     sync.Mutex
	Logs   []string
	Errors []string
}

func (l *testLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Logs = append(l.Logs, msg)
}

func (l *testLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Errors = append(l.Errors, msg)
}

type testMetrics struct {
	mu        sync.Mutex
	Records   []string
	Durations []time.Duration
}

func (m *testMetrics) Observe(name string, duration time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Records = append(m.Records, name)
	m.Durations = append(m.Durations, duration)
}

func noopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

func panicCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		panic(msg)
	}
}

func failingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		//nolint:err113
		return errors.New(msg)
	}
}

func noopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

func failingEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		//nolint:err113
		return errors.New(msg)
	}
}

func panicEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		panic(msg)
	}
}

func callbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := CommandLogging(logger)

	handler := mw(noopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.Logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.Logs))
	}
}

func TestCommandLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	cmdMw := CommandLogging(logger)

	handler := cmdMw(failingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if len(logger.Errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.Errors))
	}
}

func TestEventLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := EventLogging(logger)

	handler := mw(noopEventHandler())

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.Logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.Logs))
	}
}

func TestCommandRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(noopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(panicCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	if err.Error() != "panic recovered in command test.cmd: boom" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestEventRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := EventRecovery()
	handler := mw(panicEventHandler("event boom"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
}

func TestCommandRetry_Success(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return true }

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 2 {
			//nolint:err113
			return errors.New("transient")
		}

		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestCommandRetry_AllAttemptsFail(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	cmdMw := CommandRetry(config)

	handler := cmdMw(failingCommandHandler("always fail"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandRetry_NonRetryable(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return false }

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++
		//nolint:err113
		return errors.New("non-retryable")
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestCommandValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ any) error { return nil }
	mw := CommandValidation(validate)

	called := false
	handler := mw(callbackCommandHandler(&called))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestCommandValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ any) error {
		//nolint:err113
		return errors.New("invalid")
	}
	mw := CommandValidation(validate)

	handler := mw(failingCommandHandler("should not be called"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandMetrics(t *testing.T) {
	t.Parallel()

	metrics := &testMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(noopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metrics.Records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(metrics.Records))
	}

	if metrics.Records[0] != "command_success" {
		t.Errorf("expected command_success, got %s", metrics.Records[0])
	}
}

func TestEventMetrics(t *testing.T) {
	t.Parallel()

	metrics := &testMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(failingEventHandler("middleware failure"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	if len(metrics.Records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(metrics.Records))
	}

	if metrics.Records[0] != "event_error" {
		t.Errorf("expected event_error, got %s", metrics.Records[0])
	}
}
