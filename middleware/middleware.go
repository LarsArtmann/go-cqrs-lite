package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// MetricsRecorder records handler execution metrics.
//
// Deprecated: Use TypedMetricsRecorder with attribute.KeyValue pairs instead.
// The string-label Observe accepts alternating key-value pairs that silently
// drop malformed (odd-length) input. Prefer TypedMetricsRecorder and the
// CommandTypedMetrics/EventTypedMetrics/QueryTypedMetrics constructors for
// compile-time safety.
type MetricsRecorder interface {
	Observe(ctx context.Context, name string, duration time.Duration, labels ...string)
}

// RetryConfig configures retry behavior for transient failures.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	IsRetryable  func(error) bool  // defaults to event.IsRetryable (classifies via error taxonomy)
	OnDeadLetter DeadLetterHandler // optional; called when retries are exhausted
}

const (
	defaultMaxRetryAttempts = 3
	defaultRetryInitDelay   = 100 * time.Millisecond
	defaultRetryMaxDelay    = 5 * time.Second
	defaultRetryMultiplier  = 2.0
)

// DefaultRetryConfig returns sensible defaults for retry.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{ //nolint:exhaustruct // OnDeadLetter is optional, defaults to nil
		MaxAttempts:  defaultMaxRetryAttempts,
		InitialDelay: defaultRetryInitDelay,
		MaxDelay:     defaultRetryMaxDelay,
		Multiplier:   defaultRetryMultiplier,
		IsRetryable:  event.IsRetryable,
	}
}

// Validate checks that the retry configuration is valid.
func (c RetryConfig) Validate() error {
	if c.MaxAttempts < 1 {
		return event.WrapRejection(ErrValidationFailed, "middleware.invalid_max_attempts",
			fmt.Sprintf("MaxAttempts must be >= 1, got %d", c.MaxAttempts))
	}

	if c.InitialDelay <= 0 {
		return event.WrapRejection(ErrValidationFailed, "middleware.invalid_initial_delay",
			fmt.Sprintf("InitialDelay must be positive, got %s", c.InitialDelay))
	}

	if c.Multiplier <= 1 {
		return event.WrapRejection(ErrValidationFailed, "middleware.invalid_multiplier",
			fmt.Sprintf("Multiplier must be > 1, got %f", c.Multiplier))
	}

	return nil
}

// CommandValidator checks a command and returns an error if invalid.
type CommandValidator func(command.Command) error

// EventValidator checks an event and returns an error if invalid.
type EventValidator func(event.Event) error

// QueryValidator checks a query and returns an error if invalid.
type QueryValidator func(query.Query) error
