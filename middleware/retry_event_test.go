package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

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

	evt, err := testhelpers.NewTestEvent()
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

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventRetry_NonRetryable(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return false }

	mw := EventRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ event.Event) error {
		callCount++

		return errors.New("non-retryable")
	})

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestEventRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 50 * time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	mw := EventRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ event.Event) error {
		callCount++

		return errors.New("transient")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(ctx, evt)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}

	if !strings.Contains(err.Error(), "retry canceled") {
		t.Errorf("expected retry canceled error, got: %s", err.Error())
	}
}
