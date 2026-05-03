package middleware

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandRetry returns a command middleware that retries on retryable errors.
func CommandRetry(config RetryConfig) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return retry(ctx, config, string(cmd.Type()), func() error {
				return next(ctx, cmd)
			})
		}
	}
}

// EventRetry returns an event middleware that retries on retryable errors.
func EventRetry(config RetryConfig) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			return retry(ctx, config, string(evt.Type()), func() error {
				return next(ctx, evt)
			})
		}
	}
}

// QueryRetry returns a query middleware that retries on retryable errors.
func QueryRetry(config RetryConfig) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			var result any

			err := retry(ctx, config, string(q.Type()), func() error {
				var err error

				result, err = next(ctx, q)

				return err
			})

			return result, err
		}
	}
}

func retry(ctx context.Context, config RetryConfig, opName string, fn func() error) error {
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
		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()

			return fmt.Errorf("retry canceled for %s: %w", opName, err)
		}
	}

	return fmt.Errorf("all %d attempts failed for %s: %w", config.MaxAttempts, opName, err)
}

func backoff(config RetryConfig, attempt int) time.Duration {
	delay := time.Duration(
		float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt-1)),
	)
	delay = min(delay, config.MaxDelay)

	delay += time.Duration(rand.Int64N(int64(delay) / 2)) //nolint:mnd // jitter divisor

	return delay
}
