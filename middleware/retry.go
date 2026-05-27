package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandRetry returns a command middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func CommandRetry(config RetryConfig, opts ...Option) command.Middleware {
	err := config.Validate()
	if err != nil {
		return func(_ command.Handler) command.Handler {
			return func(_ context.Context, _ command.Command) error {
				return err
			}
		}
	}

	cfg := applyOptions(opts)

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return retry(ctx, config, cfg.logger, string(cmd.Type()), func() error {
				return next(ctx, cmd)
			})
		}
	}
}

// EventRetry returns an event middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func EventRetry(config RetryConfig, opts ...Option) event.Middleware {
	err := config.Validate()
	if err != nil {
		return func(_ event.Handler) event.Handler {
			return func(_ context.Context, _ event.Event) error {
				return err
			}
		}
	}

	cfg := applyOptions(opts)

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			return retry(ctx, config, cfg.logger, string(evt.Type()), func() error {
				return next(ctx, evt)
			})
		}
	}
}

// QueryRetry returns a query middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func QueryRetry(config RetryConfig, opts ...Option) query.Middleware {
	err := config.Validate()
	if err != nil {
		return func(_ query.Handler) query.Handler {
			return func(_ context.Context, _ query.Query) (any, error) {
				return nil, err
			}
		}
	}

	cfg := applyOptions(opts)

	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			var result any

			err := retry(ctx, config, cfg.logger, string(q.Type()), func() error {
				var err error

				result, err = next(ctx, q)

				return err
			})

			return result, err
		}
	}
}

func retry(
	ctx context.Context,
	config RetryConfig,
	logger *slog.Logger,
	opName string,
	fn func() error,
) error {
	var err error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !config.IsRetryable(err) {
			return err
		}

		if attempt == config.MaxAttempts {
			break
		}

		delay := backoff(config, attempt)

		if logger != nil {
			logger.Warn(
				"retry attempt",
				"operation", opName,
				"attempt", attempt,
				"maxAttempts", config.MaxAttempts,
				"delay", delay,
				"error", err,
			)
		}

		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()

			return fmt.Errorf("%w: %s: %w", ErrRetryCanceled, opName, err)
		}

		timer.Stop()
	}

	return fmt.Errorf(
		"%w: all %d attempts failed for %s: %w",
		ErrRetryExhausted,
		config.MaxAttempts,
		opName,
		err,
	)
}

func backoff(config RetryConfig, attempt int) time.Duration {
	delay := time.Duration(
		float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt-1)),
	)
	delay = min(delay, config.MaxDelay)

	delay += time.Duration(
		rand.Int64N(int64(delay) / 2), //nolint:mnd,gosec // jitter divisor; weak rand fine
	)

	return delay
}
