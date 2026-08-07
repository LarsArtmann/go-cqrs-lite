package retry

import (
	"context"
	"time"

	goretry "github.com/larsartmann/go-retry"
)

// Config configures retry behavior. Deprecated: use github.com/larsartmann/go-retry.Config.
type Config = goretry.Config

// AttemptFunc is the function signature for retryable operations.
// Deprecated: use github.com/larsartmann/go-retry.AttemptFunc.
type AttemptFunc = goretry.AttemptFunc

// ErrExhausted is returned when all retry attempts have failed.
// Deprecated: use github.com/larsartmann/go-retry.ErrExhausted.
var ErrExhausted = goretry.ErrExhausted

// ErrCanceled is returned when the context is canceled during retry.
// Deprecated: use github.com/larsartmann/go-retry.ErrCanceled.
var ErrCanceled = goretry.ErrCanceled

// Do executes fn with retries according to config.
// Deprecated: use github.com/larsartmann/go-retry.Do.
//
//nolint:wrapcheck // pure re-export alias; error is the retry package's own
func Do(ctx context.Context, config Config, fn AttemptFunc) error {
	return goretry.Do(
		ctx,
		config,
		fn,
	)
}

// Backoff computes the delay before the next attempt.
// Deprecated: use github.com/larsartmann/go-retry.Backoff.
func Backoff(config Config, attempt int) (time.Duration, error) {
	return goretry.Backoff(config, attempt)
}

// ComputeDelay calculates the delay for a given attempt.
// Deprecated: use github.com/larsartmann/go-retry.ComputeDelay.
func ComputeDelay(initial, maxDelay time.Duration, multiplier float64, attempt int) (time.Duration, error) {
	return goretry.ComputeDelay(initial, maxDelay, multiplier, attempt)
}

// DefaultConfig returns sensible defaults for retry.
// Deprecated: use github.com/larsartmann/go-retry.DefaultConfig.
func DefaultConfig() Config {
	return goretry.DefaultConfig()
}
