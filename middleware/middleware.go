// Package middleware provides cross-cutting concerns for CQRS handlers.
package middleware

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

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

const (
	defaultMaxRetryAttempts = 3
	defaultRetryInitDelay   = 100 * time.Millisecond
	defaultRetryMaxDelay    = 5 * time.Second
	defaultRetryMultiplier  = 2.0
)

// DefaultRetryConfig returns sensible defaults for retry.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  defaultMaxRetryAttempts,
		InitialDelay: defaultRetryInitDelay,
		MaxDelay:     defaultRetryMaxDelay,
		Multiplier:   defaultRetryMultiplier,
		IsRetryable:  func(error) bool { return false },
	}
}

// CommandValidator checks a command and returns an error if invalid.
type CommandValidator func(command.Command) error

// EventValidator checks an event and returns an error if invalid.
type EventValidator func(event.Event) error

// QueryValidator checks a query and returns an error if invalid.
type QueryValidator func(query.Query) error
