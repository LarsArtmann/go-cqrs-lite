package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// NewRetry returns a generic middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func NewRetry[M any](adapter MessageAdapter[M], config RetryConfig, opts ...Option) Middleware[M] {
	validateErr := config.Validate()
	if validateErr != nil {
		return failingMiddleware[M](validateErr)
	}

	cfg := applyOptions(opts)

	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			entry := DeadLetterEntry{ //nolint:exhaustruct // fields set incrementally during retry loop
				Kind: adapter.Kind,
				Type: adapter.ExtractType(msg),
			}

			if adapter.ExtractID != nil {
				entry.AggregateID = adapter.ExtractID(msg)
			}

			return retry(ctx, config, cfg.logger, entry, func(attemptCtx context.Context) error {
				return next(attemptCtx, msg)
			})
		}
	}
}

// CommandRetry returns a command middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func CommandRetry(config RetryConfig, opts ...Option) command.Middleware {
	return AsCommand(NewRetry(CommandAdapter, config, opts...))
}

// EventRetry returns an event middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func EventRetry(config RetryConfig, opts ...Option) event.Middleware {
	return AsEvent(NewRetry(EventAdapter, config, opts...))
}

// QueryRetry returns a query middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
func QueryRetry(config RetryConfig, opts ...Option) query.Middleware {
	return AsQuery(NewRetry(QueryAdapter, config, opts...))
}

func retry(
	ctx context.Context,
	config RetryConfig,
	logger *slog.Logger,
	entry DeadLetterEntry,
	fn func(context.Context) error,
) error {
	var err error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		attemptCtx, attemptSpan := cqrsotel.StartSpan(
			ctx, retryTracer(), fmt.Sprintf("retry.attempt.%d", attempt),
			cqrsotel.SpanKindInternal,
			cqrsotel.WithAttributes(
				cqrsotel.AttrInt("cqrs.retry.attempt", attempt),
				cqrsotel.AttrInt("cqrs.retry.max_attempts", config.MaxAttempts),
			),
		)

		err = fn(attemptCtx)
		if err == nil {
			attemptSpan.End()

			return nil
		}

		cqrsotel.RecordError(attemptSpan, err)
		attemptSpan.End()

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
				"operation", entry.Type,
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

			return event.WrapInfrastructure(ErrRetryCanceled, "middleware.retry_canceled",
				entry.Type+": retry canceled").WithCause(err)
		}

		timer.Stop()
	}

	if config.OnDeadLetter != nil {
		entry.Error = err
		entry.Attempts = config.MaxAttempts
		entry.FailedAt = time.Now()
		config.OnDeadLetter(ctx, entry)
	}

	return event.WrapInfrastructure(ErrRetryExhausted, "middleware.retry_exhausted",
		fmt.Sprintf("all %d attempts failed for %s", config.MaxAttempts, entry.Type)).WithCause(err)
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
