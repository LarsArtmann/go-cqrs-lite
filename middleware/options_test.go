package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestWithLogger_RetryLogsAttempts(t *testing.T) {
	t.Parallel()

	logger, logHandler := newTestLogger()

	retryErr := event.NewTransient("test.transient", "retry me")

	mw := CommandRetry(RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2,
		IsRetryable:  func(err error) bool { return true },
	}, WithLogger(logger))

	callCount := 0

	cmdHandler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		if callCount == 1 {
			return retryErr
		}

		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := cmdHandler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}

	if logHandler.InfoCount() == 0 {
		t.Error("expected warn log for retry attempt, got none")
	}
}

func TestWithLogger_RecoveryLogsPanic(t *testing.T) {
	t.Parallel()

	logger, logHandler := newTestLogger()

	mw := CommandRecovery(WithLogger(logger))

	cmdHandler := mw(func(_ context.Context, _ command.Command) error {
		panic("test panic")
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := cmdHandler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	if logHandler.ErrorCount() == 0 {
		t.Error("expected error log for recovered panic, got none")
	}
}

func TestWithLogger_ValidationLogsFailure(t *testing.T) {
	t.Parallel()

	logger, logHandler := newTestLogger()

	validationErr := errors.New("invalid command")

	mw := CommandValidation(func(_ command.Command) error {
		return validationErr
	}, WithLogger(logger))

	cmdHandler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := cmdHandler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}

	if logHandler.InfoCount() == 0 {
		t.Error("expected warn log for validation failure, got none")
	}
}

func TestWithLogger_NoLogger_NoPanic(t *testing.T) {
	t.Parallel()

	mw := CommandRetry(RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2,
		IsRetryable:  func(err error) bool { return true },
	})

	callCount := 0

	cmdHandler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return event.NewTransient("test.transient", "always fail")
	})

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := cmdHandler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}
