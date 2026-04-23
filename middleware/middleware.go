// Package middleware provides cross-cutting concerns for CQRS handlers.
package middleware

import "time"

// Logger provides structured logging for middleware.
type Logger interface {
	Info(msg string, keyvals ...any)
	Error(msg string, keyvals ...any)
}

// MetricsRecorder records handler execution metrics.
type MetricsRecorder interface {
	Observe(name string, duration time.Duration, labels ...string)
}

// RetryConfig configures retry behavior for transient failures.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	IsRetryable  func(error) bool
}

// DefaultRetryConfig returns sensible defaults for retry.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		IsRetryable:  func(error) bool { return false },
	}
}

// Validator checks a message and returns an error if invalid.
type Validator func(any) error
