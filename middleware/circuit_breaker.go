package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
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
		IsFailure:        errorfamily.IsRetryable,
	}
}

// Validate checks that the circuit breaker configuration is valid.
func (c CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold < 1 {
		return errorfamily.WrapRejection(
			ErrValidationFailed,
			"middleware.cb_invalid_failure_threshold",
			fmt.Sprintf("FailureThreshold must be >= 1, got %d", c.FailureThreshold),
		)
	}

	if c.SuccessThreshold < 1 {
		return errorfamily.WrapRejection(
			ErrValidationFailed,
			"middleware.cb_invalid_success_threshold",
			fmt.Sprintf("SuccessThreshold must be >= 1, got %d", c.SuccessThreshold),
		)
	}

	if c.Timeout <= 0 {
		return errorfamily.WrapRejection(ErrValidationFailed, "middleware.cb_invalid_timeout",
			fmt.Sprintf("Timeout must be positive, got %s", c.Timeout))
	}

	return nil
}

// circuitBreaker wraps a failsafe-go CircuitBreaker for production-grade
// resilience (sliding window thresholds, half-open recovery, state listeners).
// The IsFailure predicate is applied manually (not via HandleIf) so that
// non-failure errors count as successes toward closing the circuit — matching
// the original hand-rolled semantics.
type circuitBreaker struct {
	breaker circuitbreaker.CircuitBreaker[any]
	config  CircuitBreakerConfig
}

func newCircuitBreaker(config CircuitBreakerConfig) *circuitBreaker {
	breaker := circuitbreaker.NewBuilder[any]().
		WithFailureThreshold(uint(config.FailureThreshold)).
		WithSuccessThreshold(uint(config.SuccessThreshold)).
		WithDelay(config.Timeout).
		Build()

	return &circuitBreaker{
		breaker: breaker,
		config:  config,
	}
}

func (cb *circuitBreaker) execute(
	ctx context.Context,
	logger *slog.Logger,
	opName string,
	fn func() error,
) error {
	if !cb.breaker.TryAcquirePermit() {
		if logger != nil {
			logger.WarnContext(ctx, "circuit breaker rejected",
				"operation", opName)
		}

		return errorfamily.WrapTransient(ErrCircuitBreakerOpen, "middleware.circuit_open",
			"circuit breaker rejected "+opName)
	}

	err := fn()
	if err == nil {
		cb.breaker.RecordSuccess()

		return nil
	}

	isFailure := cb.config.IsFailure
	if isFailure == nil {
		isFailure = errorfamily.IsRetryable
	}

	if isFailure(err) {
		cb.breaker.RecordError(err)
	} else {
		cb.breaker.RecordSuccess()
	}

	return errorfamily.Wrap(err, errorfamily.Classify(err), opName, err.Error())
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

var ErrCircuitBreakerOpen = errorfamily.NewInfrastructure(
	"middleware.circuit_breaker_open",
	"circuit breaker open",
)
