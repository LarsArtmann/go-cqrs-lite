package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
	state     atomic.Int32
	failures  atomic.Int32
	successes atomic.Int32

	mu          sync.Mutex
	lastFailure time.Time
	config      CircuitBreakerConfig
}

func (cb *circuitBreaker) allow() error {
	switch circuitState(cb.state.Load()) {
	case circuitClosed:
		return nil
	case circuitOpen:
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if circuitState(cb.state.Load()) != circuitOpen {
			return nil
		}

		if time.Since(cb.lastFailure) > cb.config.Timeout {
			cb.state.Store(int32(circuitHalfOpen))
			cb.successes.Store(0)

			return nil
		}

		return ErrCircuitBreakerOpen
	case circuitHalfOpen:
		return nil
	}

	return nil
}

func (cb *circuitBreaker) recordSuccess() {
	switch circuitState(cb.state.Load()) {
	case circuitHalfOpen:
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if circuitState(cb.state.Load()) != circuitHalfOpen {
			cb.failures.Store(0)

			return
		}

		newSuccesses := int(cb.successes.Add(1))
		if newSuccesses >= cb.config.SuccessThreshold {
			cb.state.Store(int32(circuitClosed))
			cb.failures.Store(0)
		}
	case circuitClosed, circuitOpen:
		cb.failures.Store(0)
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures.Add(1)
	cb.lastFailure = time.Now()

	if circuitState(cb.state.Load()) == circuitHalfOpen {
		cb.state.Store(int32(circuitOpen))
	} else if int(cb.failures.Load()) >= cb.config.FailureThreshold {
		cb.state.Store(int32(circuitOpen))
	}
}

func newCircuitBreaker(config CircuitBreakerConfig) *circuitBreaker {
	breaker := &circuitBreaker{ //nolint:exhaustruct // atomics are zero-valued
		lastFailure: time.Time{},
		config:      config,
	}
	breaker.state.Store(int32(circuitClosed))

	return breaker
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

	isFailure := cb.config.IsFailure
	if isFailure == nil {
		isFailure = event.IsRetryable
	}

	if isFailure(err) {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return event.Wrap(err, event.Classify(err), opName, err.Error())
}

// NewCircuitBreaker returns a generic middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func NewCircuitBreaker[M any](
	adapter MessageAdapter[M],
	config CircuitBreakerConfig,
	opts ...Option,
) Middleware[M] {
	err := config.Validate()
	if err != nil {
		return failingMiddleware[M](err)
	}

	cfg := applyOptions(opts)
	breaker := newCircuitBreaker(config)

	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			return breaker.execute(ctx, cfg.logger, adapter.ExtractType(msg), func() error {
				return next(ctx, msg)
			})
		}
	}
}

// CommandCircuitBreaker returns a command middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func CommandCircuitBreaker(config CircuitBreakerConfig, opts ...Option) command.Middleware {
	return AsCommand(NewCircuitBreaker(CommandAdapter, config, opts...))
}

// EventCircuitBreaker returns an event subscribe-side middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func EventCircuitBreaker(config CircuitBreakerConfig, opts ...Option) event.Middleware {
	return AsEvent(NewCircuitBreaker(EventAdapter, config, opts...))
}

// QueryCircuitBreaker returns a query middleware that implements the circuit breaker pattern.
// Returns a middleware that always fails if config is invalid.
func QueryCircuitBreaker(config CircuitBreakerConfig, opts ...Option) query.Middleware {
	return AsQuery(NewCircuitBreaker(QueryAdapter, config, opts...))
}

var ErrCircuitBreakerOpen = event.NewInfrastructure(
	"middleware.circuit_breaker_open",
	"circuit breaker open",
)
