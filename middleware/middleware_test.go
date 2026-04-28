package middleware

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type testCommand struct {
	aggregateID id.AggregateID
}

func (c *testCommand) Type() command.Type          { return "test.cmd" }
func (c *testCommand) AggregateID() id.AggregateID { return c.aggregateID }

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

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := CommandLogging(logger)

	handler := mw(testhelpers.NoopCommandHandler())

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

	handler := cmdMw(testhelpers.FailingCommandHandler("boom"))

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

	handler := mw(testhelpers.NoopEventHandler())

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
	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(testhelpers.PanicCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	if !strings.Contains(err.Error(), "panic recovered in command test.cmd: boom") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestEventRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := EventRecovery()
	handler := mw(testhelpers.PanicEventHandler("event boom"))

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

	handler := cmdMw(testhelpers.FailingCommandHandler("always fail"))

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
	handler := mw(testhelpers.CallbackCommandHandler(&called))

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
		return errors.New("invalid")
	}
	mw := CommandValidation(validate)

	handler := mw(testhelpers.FailingCommandHandler("should not be called"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEventValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ any) error { return nil }
	mw := EventValidation(validate)

	called := false
	handler := mw(func(_ context.Context, _ event.Event) error {
		called = true

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

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ any) error {
		return errors.New("invalid event")
	}
	mw := EventValidation(validate)

	handler := mw(testhelpers.FailingEventHandler("should not be called"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandMetrics(t *testing.T) {
	t.Parallel()

	metrics := &testhelpers.TestMetrics{}
	mw := CommandMetrics(metrics)

	handler := mw(testhelpers.NoopCommandHandler())

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

	metrics := &testhelpers.TestMetrics{}
	mw := EventMetrics(metrics)

	handler := mw(testhelpers.FailingEventHandler("middleware failure"))

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
