package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/query"
)

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

const (
	defaultFailureThreshold = 5
	defaultSuccessThreshold = 3
	defaultTimeout          = 30 * time.Second
)

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	FailureThreshold int           // failures before opening (default: 5)
	SuccessThreshold int           // successes in half-open to close (default: 3)
	Timeout          time.Duration // time before half-open (default: 30s)
	IsFailure        func(error) bool
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: defaultFailureThreshold,
		SuccessThreshold: defaultSuccessThreshold,
		Timeout:          defaultTimeout,
		IsFailure:        event.IsRetryable,
	}
}

// Validate checks that the circuit breaker configuration is valid.
func (c CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold < 1 {
		return event.WrapRejection(ErrValidationFailed, "middleware.cb_invalid_failure_threshold",
			fmt.Sprintf("FailureThreshold must be >= 1, got %d", c.FailureThreshold))
	}

	if c.SuccessThreshold < 1 {
		return event.WrapRejection(ErrValidationFailed, "middleware.cb_invalid_success_threshold",
			fmt.Sprintf("SuccessThreshold must be >= 1, got %d", c.SuccessThreshold))
	}

	if c.Timeout <= 0 {
		return event.WrapRejection(ErrValidationFailed, "middleware.cb_invalid_timeout",
			fmt.Sprintf("Timeout must be positive, got %s", c.Timeout))
	}

	return nil
}

type circuitBreaker struct {
	mu          sync.Mutex
	state       circuitState
	failures    int
	successes   int
	lastFailure time.Time
	config      CircuitBreakerConfig
}

func (cb *circuitBreaker) allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitClosed:
		return nil
	case circuitOpen:
		if time.Since(cb.lastFailure) > cb.config.Timeout {
			cb.state = circuitHalfOpen
			cb.successes = 0

			return nil
		}

		return event.WrapTransient(ErrCircuitBreakerOpen, "middleware.circuit_open",
			"circuit breaker is open")
	case circuitHalfOpen:
		return nil
	}

	return nil
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == circuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = circuitClosed
			cb.failures = 0
		}
	} else {
		cb.failures = 0
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == circuitHalfOpen {
		cb.state = circuitOpen
	} else if cb.failures >= cb.config.FailureThreshold {
		cb.state = circuitOpen
	}
}

func newCircuitBreaker(config CircuitBreakerConfig) *circuitBreaker {
	return &circuitBreaker{
		mu:          sync.Mutex{},
		state:       circuitClosed,
		failures:    0,
		successes:   0,
		lastFailure: time.Time{},
		config:      config,
	}
}

// CommandCircuitBreaker returns a command middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func CommandCircuitBreaker(config CircuitBreakerConfig, opts ...Option) command.Middleware {
	err := config.Validate()
	if err != nil {
		return commandErrMiddleware(err)
	}

	cfg := applyOptions(opts)
	breaker := newCircuitBreaker(config)

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return breaker.execute(ctx, cfg.logger, string(cmd.Type()), func() error {
				return next(ctx, cmd)
			})
		}
	}
}

// EventCircuitBreaker returns an event subscribe-side middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func EventCircuitBreaker(config CircuitBreakerConfig, opts ...Option) event.Middleware {
	err := config.Validate()
	if err != nil {
		return eventErrMiddleware(err)
	}

	cfg := applyOptions(opts)
	breaker := newCircuitBreaker(config)

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			return breaker.execute(ctx, cfg.logger, string(evt.Type()), func() error {
				return next(ctx, evt)
			})
		}
	}
}

// QueryCircuitBreaker returns a query middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func QueryCircuitBreaker(config CircuitBreakerConfig, opts ...Option) query.Middleware {
	err := config.Validate()
	if err != nil {
		return queryErrMiddleware(err)
	}

	cfg := applyOptions(opts)
	breaker := newCircuitBreaker(config)

	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			var result any

			err := breaker.execute(ctx, cfg.logger, string(q.Type()), func() error {
				var execErr error

				result, execErr = next(ctx, q)

				return execErr
			})

			return result, err
		}
	}
}

func (cb *circuitBreaker) execute(
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	fn func() error,
) error {
	err := cb.allow()
	if err != nil {
		if logger != nil {
			logger.WarnContext(ctx, "circuit breaker rejected",
				"operation", opName, "error", err)
		}

		return event.WrapTransient(err, "middleware.circuit_open",
			"circuit breaker rejected "+opName)
	}

	err = fn()
	if err == nil {
		cb.recordSuccess()

		return nil
	}

	if cb.config.IsFailure(err) {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return event.Wrap(err, event.Classify(err), opName, err.Error())
}

// ErrCircuitBreakerOpen is returned when the circuit breaker is open.
var ErrCircuitBreakerOpen = errors.New("circuit breaker open")
