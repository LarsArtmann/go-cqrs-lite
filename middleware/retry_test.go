package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func assertRetryCanceled(t *testing.T, err error) {
	t.Helper()

	if !strings.Contains(err.Error(), "retry canceled") {
		t.Errorf("expected retry canceled error, got: %s", err.Error())
	}
}

func TestCommandRetry_Success(t *testing.T) {
	t.Parallel()

	config := retryConfigBasic()

	mw := CommandRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++
		if callCount < 2 {
			return errors.New("transient")
		}

		return nil
	})

	cmd := &testCommand{streamID: id.NewStreamID()}

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

	config := retryConfigFast()

	cmdMw := CommandRetry(config)

	handler := cmdMw(failingCommandHandler("always fail"))

	cmd := &testCommand{streamID: id.NewStreamID()}

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
	handler, cmd := setupCommandNonRetryableTest(t, mw, &callCount, "non-retryable")

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

	config := retryConfigSlow()

	mw := CommandRetry(config)

	callCount := 0
	handler, cmd := setupCommandNonRetryableTest(t, mw, &callCount, "transient")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := handler(ctx, cmd)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}

	assertRetryCanceled(t, err)
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

func TestRetryExhausted_SentinelError(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 2
	config.InitialDelay = time.Millisecond
	config.IsRetryable = func(_ error) bool { return true }

	mw := CommandRetry(config)
	handler := mw(failingCommandHandler("always fail"))

	cmd := &testCommand{streamID: id.NewStreamID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrRetryExhausted) {
		t.Errorf("expected errors.Is(err, ErrRetryExhausted), got: %v", err)
	}
}

func TestRetryCanceled_SentinelError(t *testing.T) {
	t.Parallel()

	config := retryConfigSlow()

	mw := CommandRetry(config)
	handler := mw(failingCommandHandler("transient"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := &testCommand{streamID: id.NewStreamID()}

	err := handler(ctx, cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrRetryCanceled) {
		t.Errorf("expected errors.Is(err, ErrRetryCanceled), got: %v", err)
	}
}

func TestRetryConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		config := DefaultRetryConfig()
		if err := config.Validate(); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	invalidConfigs := []struct {
		name   string
		config RetryConfig
	}{
		{
			name:   "MaxAttempts zero",
			config: RetryConfig{MaxAttempts: 0, InitialDelay: time.Second, Multiplier: 2.0},
		},
		{
			name:   "InitialDelay zero",
			config: RetryConfig{MaxAttempts: 3, InitialDelay: 0, Multiplier: 2.0},
		},
		{
			name:   "Multiplier one",
			config: RetryConfig{MaxAttempts: 3, InitialDelay: time.Second, Multiplier: 1.0},
		},
	}

	for _, tt := range invalidConfigs {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}

			if !errors.Is(err, ErrValidationFailed) {
				t.Errorf("expected ErrValidationFailed, got: %v", err)
			}
		})
	}
}

func TestCommandRetry_InvalidConfig(t *testing.T) {
	t.Parallel()

	mw := CommandRetry(RetryConfig{})
	handler := mw(NoopCommandHandler())

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
}

func TestEventRetry_InvalidConfig(t *testing.T) {
	t.Parallel()

	mw := EventRetry(RetryConfig{})
	handler := mw(eventtest.NoopEventHandler())

	evt, evtErr := eventtest.NewTestEvent()
	if evtErr != nil {
		t.Fatalf("unexpected error: %v", evtErr)
	}

	err := handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
}

func TestQueryRetry_InvalidConfig(t *testing.T) {
	t.Parallel()

	mw := QueryRetry(RetryConfig{})
	handler := mw(noopQueryHandler())

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
}

func TestDefaultRetryConfig_IsRetryable(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()

	if !config.IsRetryable(errors.New("any error")) {
		t.Error("default IsRetryable should return true for unknown errors")
	}

	if !config.IsRetryable(errorfamily.NewTransient("test", "transient")) {
		t.Error("default IsRetryable should return true for Transient errors")
	}

	if config.IsRetryable(errorfamily.NewRejection("test", "rejected")) {
		t.Error("default IsRetryable should return false for Rejection errors")
	}

	if config.IsRetryable(errorfamily.NewConflict("test", "conflict")) {
		t.Error("default IsRetryable should return false for Conflict errors")
	}

	if config.IsRetryable(event.ErrStoreClosed) {
		t.Error("default IsRetryable should return false for Infrastructure errors")
	}
}
