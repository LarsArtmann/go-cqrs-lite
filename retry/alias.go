package retry

import (
	"context"
	"time"

	goretry "github.com/larsartmann/go-retry"
)

// Config configures retry behavior. Alias of goretry.Config — preserves
// the Validate method.
type Config = goretry.Config

// AttemptFunc is the function signature for retryable operations.
type AttemptFunc = goretry.AttemptFunc

// ErrExhausted is returned when all retry attempts have failed.
var ErrExhausted = goretry.ErrExhausted

// ErrCanceled is returned when the context is canceled during retry.
var ErrCanceled = goretry.ErrCanceled

// Do executes fn with retries according to config.
func Do(ctx context.Context, config Config, fn AttemptFunc) error {
	return goretry.Do(ctx, config, fn) //nolint:wrapcheck // thin alias — caller sees the same errors
}

// Backoff computes the delay before the next attempt.
func Backoff(config Config, attempt int) time.Duration {
	return goretry.Backoff(config, attempt)
}

// ComputeDelay calculates the delay for a given attempt.
func ComputeDelay(initial, maxDelay time.Duration, multiplier float64, attempt int) time.Duration {
	return goretry.ComputeDelay(initial, maxDelay, multiplier, attempt)
}

// DefaultConfig returns sensible defaults for retry.
func DefaultConfig() Config {
	return goretry.DefaultConfig()
}
