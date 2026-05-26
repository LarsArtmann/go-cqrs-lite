package saga

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// RunnerOption configures a Runner.
type RunnerOption func(*runnerConfig)

type runnerConfig struct {
	logger          Logger
	maxRetries      int
	retryDelay      time.Duration
	retryMultiplier float64
}

// Logger is the minimal logging interface used by the saga runner.
type Logger interface {
	Info(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}

// WithLogger sets the logger for the saga runner.
func WithLogger(l Logger) RunnerOption {
	return func(c *runnerConfig) {
		c.logger = l
	}
}

// WithRetryPolicy sets the retry policy for failed saga steps.
func WithRetryPolicy(maxRetries int, initialDelay time.Duration) RunnerOption {
	return func(c *runnerConfig) {
		c.maxRetries = maxRetries
		c.retryDelay = initialDelay
	}
}

// WithRetryMultiplier sets the exponential backoff multiplier.
func WithRetryMultiplier(m float64) RunnerOption {
	return func(c *runnerConfig) {
		c.retryMultiplier = m
	}
}

func defaultConfig() runnerConfig {
	return runnerConfig{
		maxRetries:      3,
		retryDelay:      100 * time.Millisecond,
		retryMultiplier: 2.0,
	}
}

// CommandDispatcher is the interface the saga runner uses to dispatch commands.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, cmd command.Command) error
}

// EventSubscriber is the interface the saga runner uses to listen for completion events.
type EventSubscriber interface {
	Subscribe(eventType event.Type, handler event.Handler) error
}
