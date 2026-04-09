package middleware

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type testCommand struct {
	aggregateID id.AggregateID
}

func (c *testCommand) Type() command.Type  { return "test.cmd" }
func (c *testCommand) AggregateID() string { return c.aggregateID.String() }

type testLogger struct {
	mu     sync.Mutex
	logs   []string
	errors []string
}

func (l *testLogger) Info(msg string, keyvals ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, msg)
}

func (l *testLogger) Error(msg string, keyvals ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.errors = append(l.errors, msg)
}

type testMetrics struct {
	mu        sync.Mutex
	records   []string
	durations []time.Duration
}

func (m *testMetrics) Observe(name string, duration time.Duration, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.records = append(m.records, name)
	m.durations = append(m.durations, duration)
}

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := CommandLogging(logger)

	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.logs))
	}
}

func failingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg)
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

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

func TestEventLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := EventLogging(logger)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return nil
	})

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.logs))
	}
}

func TestCommandRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(func(_ context.Context, _ command.Command) error {
		panic("boom")
	})

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
	handler := mw(func(_ context.Context, _ event.Event) error {
		panic("event boom")
	})

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
	config.IsRetryable = func(err error) bool { return true }

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 2 {
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
	config.IsRetryable = func(err error) bool { return true }

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
	config.IsRetryable = func(err error) bool { return false }

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

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

	validate := func(msg any) error { return nil }
	mw := CommandValidation(validate)

	called := false
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

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

	validate := func(_ any) error { return errors.New("invalid") }
	mw := CommandValidation(validate)

	handler := mw(func(_ context.Context, _ command.Command) error {
		t.Fatal("handler should not be called")

		return nil
	})

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

	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metrics.records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(metrics.records))
	}

	if metrics.records[0] != "command_success" {
		t.Errorf("expected command_success, got %s", metrics.records[0])
	}
}

func TestEventMetrics(t *testing.T) {
	t.Parallel()

	metrics := &testMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(func(_ context.Context, _ event.Event) error {
		return errors.New("fail")
	})

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	if len(metrics.records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(metrics.records))
	}

	if metrics.records[0] != "event_error" {
		t.Errorf("expected event_error, got %s", metrics.records[0])
	}
}
