package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrExhausted is returned by [Do] when all retry attempts have failed.
// It is classified as Infrastructure because retry exhaustion typically
// indicates a downstream system problem.
var ErrExhausted = errorfamily.NewInfrastructure(
	"retry.exhausted",
	"all retry attempts failed",
)

// ErrCanceled is returned by [Do] when the context is canceled during
// a retry delay. It wraps context.Canceled as its cause.
var ErrCanceled = errorfamily.NewInfrastructure(
	"retry.canceled",
	"retry canceled during backoff delay",
)

// AttemptFunc is the function retried by [Do]. The attempt argument
// starts at 1 (the first attempt) and increments with each retry.
type AttemptFunc func(ctx context.Context, attempt int) error

// Do executes fn with retry logic according to config. It calls fn at
// most config.MaxAttempts times, sleeping with exponential backoff + jitter
// between attempts. If fn returns nil, Do returns nil immediately.
//
// If config.IsRetryable returns false for an error, that error is returned
// immediately without further attempts.
//
// If the context is canceled during a backoff delay, Do returns an error
// wrapping [ErrCanceled].
//
// If all attempts fail, Do calls config.OnExhausted (if set) and returns
// an error wrapping [ErrExhausted] with the last error as its cause.
func Do(ctx context.Context, config Config, fn AttemptFunc) error {
	if err := config.Validate(); err != nil {
		return err
	}

	isRetryable := config.IsRetryable
	if isRetryable == nil {
		isRetryable = errorfamily.IsRetryable
	}

	var err error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err = fn(ctx, attempt)
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		if attempt == config.MaxAttempts {
			break
		}

		delay := Backoff(config, attempt)

		if config.OnRetry != nil {
			config.OnRetry(attempt, delay, err)
		}

		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()

			return errorfamily.WrapInfrastructure(ErrCanceled, "retry.canceled",
				"retry canceled during backoff").WithCause(err)
		}

		timer.Stop()
	}

	if config.OnExhausted != nil {
		config.OnExhausted(config.MaxAttempts, err)
	}

	return errorfamily.WrapInfrastructure(ErrExhausted, "retry.exhausted",
		"all attempts failed").WithCause(err)
}

// Backoff calculates the delay before the next attempt using exponential
// backoff with optional jitter. The delay for attempt n is:
//
//	InitialDelay * Multiplier^(n-1) + random jitter (up to 50% of the delay)
//
// The result is capped at MaxDelay. Exported so callers can preview or
// log the planned delay without executing the retry loop.
func Backoff(config Config, attempt int) time.Duration {
	return ComputeDelay(config.InitialDelay, config.MaxDelay, config.Multiplier, attempt)
}

// ComputeDelay calculates the exponential backoff delay with jitter from raw
// parameters, without requiring a [Config]. The delay for attempt n is:
//
//	initial * multiplier^(n-1) + random jitter (up to 50% of the delay)
//
// The result is capped at max.
func ComputeDelay(initial, max time.Duration, multiplier float64, attempt int) time.Duration {
	delay := time.Duration(
		float64(initial) * math.Pow(multiplier, float64(attempt-1)),
	)
	delay = min(delay, max)

	delay += time.Duration(
		rand.Int64N(int64(delay) / 2), //nolint:mnd,gosec // jitter divisor; weak rand fine
	)

	return delay
}
