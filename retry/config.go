package retry

import (
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	defaultMaxAttempts = 3
	defaultInitDelay   = 100 * time.Millisecond
	defaultMaxDelay    = 5 * time.Second
	defaultMultiplier  = 2.0
)

// Config configures retry behavior for [Do]. All fields have sensible
// defaults via [DefaultConfig].
type Config struct {
	// MaxAttempts is the total number of attempts including the first call
	// (not retries on top). Must be >= 1. Default: 3.
	MaxAttempts int

	// InitialDelay is the delay before the second attempt. Default: 100ms.
	InitialDelay time.Duration

	// MaxDelay caps the backoff delay between attempts. Default: 5s.
	MaxDelay time.Duration

	// Multiplier is the exponential backoff factor. Must be > 1. Default: 2.0.
	Multiplier float64

	// IsRetryable decides whether an error should trigger a retry.
	// If nil, defaults to [errorfamily.IsRetryable].
	IsRetryable func(error) bool

	// OnRetry is called after each failed attempt before sleeping.
	// The attempt argument starts at 1 (the first failed attempt).
	// Use this for structured logging or metrics without OTel.
	OnRetry func(attempt int, delay time.Duration, err error)

	// OnExhausted is called once after all attempts have failed.
	// Unlike middleware.DeadLetterHandler, this receives no CQRS-specific
	// types — just the attempt count and the last error.
	OnExhausted func(attempts int, err error)
}

// DefaultConfig returns sensible defaults for retry.
func DefaultConfig() Config {
	return Config{ //nolint:exhaustruct // OnRetry and OnExhausted are optional
		MaxAttempts:  defaultMaxAttempts,
		InitialDelay: defaultInitDelay,
		MaxDelay:     defaultMaxDelay,
		Multiplier:   defaultMultiplier,
		IsRetryable:  errorfamily.IsRetryable,
	}
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.MaxAttempts < 1 {
		return errorfamily.NewRejection(
			"retry.invalid_max_attempts",
			fmt.Sprintf("MaxAttempts must be >= 1, got %d", c.MaxAttempts),
		)
	}

	if c.InitialDelay <= 0 {
		return errorfamily.NewRejection(
			"retry.invalid_initial_delay",
			fmt.Sprintf("InitialDelay must be positive, got %s", c.InitialDelay),
		)
	}

	if c.Multiplier <= 1 {
		return errorfamily.NewRejection(
			"retry.invalid_multiplier",
			fmt.Sprintf("Multiplier must be > 1, got %f", c.Multiplier),
		)
	}

	return nil
}
