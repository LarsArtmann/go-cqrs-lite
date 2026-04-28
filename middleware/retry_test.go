package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

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

func TestCommandRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 50 * time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return errors.New("transient")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(ctx, cmd)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}

	if !strings.Contains(err.Error(), "retry canceled") {
		t.Errorf("expected retry canceled error, got: %s", err.Error())
	}
}

func TestEventRetry_Success(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return true }

	mw := EventRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ event.Event) error {
		callCount++
		if callCount < 2 {
			return errors.New("transient")
		}

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

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestEventRetry_AllAttemptsFail(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	mw := EventRetry(config)

	handler := mw(testhelpers.FailingEventHandler("always fail"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQueryRetry_Success(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return true }

	mw := QueryRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		callCount++
		if callCount < 2 {
			return nil, errors.New("transient")
		}

		return "ok", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "ok" {
		t.Errorf("expected ok, got %v", result)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestQueryRetry_AllAttemptsFail(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	mw := QueryRetry(config)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("always fail")
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQueryRetry_NonRetryable(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return false }

	mw := QueryRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		callCount++

		return nil, errors.New("non-retryable")
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", config.MaxAttempts)
	}

	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("expected InitialDelay 100ms, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("expected MaxDelay 5s, got %v", config.MaxDelay)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %f", config.Multiplier)
	}
}
