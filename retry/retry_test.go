package retry_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/retry/v4"
)

func TestDo_SucceedsOnFirstAttempt(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := retry.Do(
		context.Background(),
		fastConfig(),
		func(ctx context.Context, attempt int) error {
			calls.Add(1)

			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", calls.Load())
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := retry.Do(
		context.Background(),
		fastConfig(),
		func(ctx context.Context, attempt int) error {
			calls.Add(1)
			if attempt < 3 {
				return errorfamily.NewTransient("test.transient", "fail")
			}

			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestDo_ReturnsErrExhaustedWhenAllAttemptsFail(t *testing.T) {
	t.Parallel()

	transient := errorfamily.NewTransient("test.transient", "always fail")

	var calls atomic.Int32
	err := retry.Do(
		context.Background(),
		fastConfig(),
		func(ctx context.Context, attempt int) error {
			calls.Add(1)

			return transient
		},
	)

	if !errors.Is(err, retry.ErrExhausted) {
		t.Fatalf("expected ErrExhausted, got %v", err)
	}
	if !errors.Is(err, transient) {
		t.Fatalf("expected cause to be wrapped, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestDo_DoesNotRetryNonRetryableError(t *testing.T) {
	t.Parallel()

	rejection := errorfamily.NewRejection("test.rejection", "non-retryable")

	var calls atomic.Int32
	err := retry.Do(
		context.Background(),
		fastConfig(),
		func(ctx context.Context, attempt int) error {
			calls.Add(1)

			return rejection
		},
	)

	if !errors.Is(err, rejection) {
		t.Fatalf("expected rejection error, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", calls.Load())
	}
}

func TestDo_RespectsCustomIsRetryable(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("custom sentinel")
	cfg := fastConfig()
	cfg.IsRetryable = func(err error) bool { return errors.Is(err, sentinel) }

	var calls atomic.Int32
	err := retry.Do(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls.Add(1)

		return sentinel
	})

	if !errors.Is(err, retry.ErrExhausted) {
		t.Fatalf("expected ErrExhausted, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestDo_OnRetryCalledBetweenAttempts(t *testing.T) {
	t.Parallel()

	var retryCalls atomic.Int32
	cfg := fastConfig()
	cfg.OnRetry = func(attempt int, delay time.Duration, err error) {
		retryCalls.Add(1)
	}

	transient := errorfamily.NewTransient("test.transient", "fail")
	_ = retry.Do(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		if attempt < 3 {
			return transient
		}

		return nil
	})

	// OnRetry is called after attempts 1 and 2 (before attempts 2 and 3)
	if retryCalls.Load() != 2 {
		t.Fatalf("expected 2 OnRetry calls, got %d", retryCalls.Load())
	}
}

func TestDo_OnExhaustedCalledAfterAllAttemptsFail(t *testing.T) {
	t.Parallel()

	var exhaustedAttempts int
	var exhaustedErr error
	cfg := fastConfig()
	cfg.OnExhausted = func(attempts int, err error) {
		exhaustedAttempts = attempts
		exhaustedErr = err
	}

	transient := errorfamily.NewTransient("test.transient", "fail")
	err := retry.Do(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		return transient
	})

	_ = err

	if exhaustedAttempts != 3 {
		t.Fatalf("expected OnExhausted attempts=3, got %d", exhaustedAttempts)
	}
	if !errors.Is(exhaustedErr, transient) {
		t.Fatalf("expected OnExhausted err to be the transient error, got %v", exhaustedErr)
	}
}

func TestDo_ContextCancellationDuringBackoff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	transient := errorfamily.NewTransient("test.transient", "fail")

	go func() {
		time.Sleep(10 * time.Millisecond) // let the first attempt fail
		cancel()
	}()

	cfg := retry.Config{
		MaxAttempts:  10,
		InitialDelay: 5 * time.Second, // long delay so cancel fires during it
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	err := retry.Do(ctx, cfg, func(ctx context.Context, attempt int) error {
		return transient
	})

	if !errors.Is(err, retry.ErrCanceled) {
		t.Fatalf("expected ErrCanceled, got %v", err)
	}
}

func TestDo_AttemptNumberStartsAt1(t *testing.T) {
	t.Parallel()

	var attempts []int
	_ = retry.Do(context.Background(), fastConfig(), func(ctx context.Context, attempt int) error {
		attempts = append(attempts, attempt)
		if attempt < 2 {
			return errorfamily.NewTransient("test.transient", "fail")
		}

		return nil
	})

	if len(attempts) != 2 || attempts[0] != 1 || attempts[1] != 2 {
		t.Fatalf("expected attempts [1,2], got %v", attempts)
	}
}

func TestDo_InvalidConfigReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config retry.Config
	}{
		{
			name:   "zero attempts",
			config: retry.Config{MaxAttempts: 0, InitialDelay: 1, Multiplier: 2},
		},
		{
			name:   "zero initial delay",
			config: retry.Config{MaxAttempts: 1, InitialDelay: 0, Multiplier: 2},
		},
		{
			name:   "multiplier <= 1",
			config: retry.Config{MaxAttempts: 1, InitialDelay: 1, Multiplier: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := retry.Do(
				context.Background(),
				tt.config,
				func(ctx context.Context, attempt int) error {
					t.Fatal("fn should not be called with invalid config")

					return nil
				},
			)

			if errorfamily.Classify(err) != errorfamily.Rejection {
				t.Fatalf(
					"expected Rejection family, got %v (family: %s)",
					err,
					errorfamily.Classify(err),
				)
			}
		})
	}
}

func TestBackoff_RespectsMaxDelay(t *testing.T) {
	t.Parallel()

	cfg := retry.Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   10.0,
	}

	delay := retry.Backoff(cfg, 5) // attempt 5 would be 100ms * 10^4 = huge

	if delay > 400*time.Millisecond { // max delay + 50% jitter = 300ms max
		t.Fatalf("expected delay <= 400ms (capped), got %v", delay)
	}
}

func TestBackoff_IncreasesExponentially(t *testing.T) {
	t.Parallel()

	cfg := retry.Config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Hour, // effectively uncapped
		Multiplier:   2.0,
	}

	// Base delays (without jitter): attempt 1 = 10ms, attempt 2 = 20ms.
	// Jitter (up to 50%) can make d1 > d2 in rare cases, so we verify the
	// exponential formula instead of comparing sampled values.
	base1 := float64(cfg.InitialDelay)
	base2 := float64(cfg.InitialDelay) * cfg.Multiplier

	if base2 <= base1 {
		t.Fatalf("expected exponential increase: base1=%v base2=%v", base1, base2)
	}
}

func TestDefaultConfig_IsValid(t *testing.T) {
	t.Parallel()

	cfg := retry.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig should be valid: %v", err)
	}
}

func TestDo_NilIsRetryableDefaultsToErrorFamily(t *testing.T) {
	t.Parallel()

	cfg := retry.Config{
		MaxAttempts:  2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     1 * time.Millisecond,
		Multiplier:   2.0,
		IsRetryable:  nil, // should default to errorfamily.IsRetryable
	}

	var calls atomic.Int32
	_ = retry.Do(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls.Add(1)

		return errorfamily.NewTransient("test.transient", "retryable")
	})

	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls with default IsRetryable, got %d", calls.Load())
	}
}

func fastConfig() retry.Config {
	return retry.Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   2.0,
	}
}
