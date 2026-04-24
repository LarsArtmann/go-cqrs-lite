package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// CommandRetry returns a command middleware that retries on retryable errors.
func CommandRetry(config RetryConfig) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return retry(config, string(cmd.Type()), func() error {
				return next(ctx, cmd)
			})
		}
	}
}

// EventRetry returns an event middleware that retries on retryable errors.
func EventRetry(config RetryConfig) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			return retry(config, string(evt.Type()), func() error {
				return next(ctx, evt)
			})
		}
	}
}

func retry(config RetryConfig, opName string, fn func() error) error {
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
		case <-context.Background().Done():
			timer.Stop()

			return fmt.Errorf("retry canceled for %s: %w", opName, err)
		}
	}

	return fmt.Errorf("all %d attempts failed for %s: %w", config.MaxAttempts, opName, err)
}

func backoff(config RetryConfig, attempt int) time.Duration {
	delay := time.Duration(float64(config.InitialDelay) * pow(config.Multiplier, attempt-1))
	if delay > config.MaxDelay {
		return config.MaxDelay
	}

	return delay
}

func pow(base float64, exp int) float64 {
	result := 1.0

	for range exp {
		result *= base
	}

	return result
}
