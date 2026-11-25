package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	retrypkg "github.com/larsartmann/go-retry"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
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
			entry := DeadLetterEntry{ //nolint:exhaustruct_v5 // fields set incrementally during retry loop
				Kind: adapter.Kind,
				Type: adapter.ExtractType(msg),
			}

			if adapter.ExtractID != nil {
				entry.StreamID = adapter.ExtractID(msg)
			}

			return retry(ctx, config, cfg.logger, entry, func(attemptCtx context.Context) error {
				return next(attemptCtx, msg)
			})
		}
	}
}

// CommandRetry returns a command middleware that retries on retryable errors.
// Returns a middleware that always fails if config is invalid.
//
// Version conflicts (event.ErrVersionConflict, Conflict family) are NOT
// retried by default: the default IsRetryable (errorfamily.IsRetryable)
// classifies only Transient errors as retryable, and a Conflict verdict is a
// domain answer, not an infrastructure failure. If your dispatch pipeline
// reloads journal state per attempt, retrying conflicts is sound via an
// explicit override:
//
//	IsRetryable: func(err error) bool {
//		return errorfamily.IsRetryable(err) || errors.Is(err, event.ErrVersionConflict)
//	}
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
	// The loop mechanics (backoff + jitter, retryable classification,
	// cancel-vs-deadline distinction, exhaustion cause chains) are
	// go-retry's retry.Do; this wrapper contributes the CQRS concerns —
	// per-attempt OTel spans, the dead-letter entry, and the middleware's
	// public sentinels on top of go-retry's.
	attemptFn := func(attemptCtx context.Context, attempt int) error {
		attemptCtx, attemptSpan := cqrsotel.StartSpan(
			attemptCtx, retryTracer(), fmt.Sprintf("retry.attempt.%d", attempt),
			cqrsotel.SpanKindInternal,
			cqrsotel.WithAttributes(
				cqrsotel.AttrInt("cqrs.retry.attempt", attempt),
				cqrsotel.AttrInt("cqrs.retry.max_attempts", config.MaxAttempts),
			),
		)

		err := fn(attemptCtx)
		if err != nil {
			cqrsotel.RecordError(attemptSpan, err)
		}

		attemptSpan.End()

		return err
	}

	isRetryable := config.IsRetryable
	if isRetryable == nil {
		isRetryable = errorfamily.IsRetryable
	}

	rcfg := retrypkg.Config{
		MaxAttempts:  config.MaxAttempts,
		InitialDelay: config.InitialDelay,
		MaxDelay:     config.MaxDelay,
		Multiplier:   config.Multiplier,
		IsRetryable:  isRetryable,
		OnRetry: func(attempt int, delay time.Duration, err error) {
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
		},
		OnExhausted: func(attempts int, err error) {
			if config.OnDeadLetter != nil {
				dlEntry := entry
				dlEntry.Error = err
				dlEntry.Attempts = attempts
				dlEntry.FailedAt = time.Now()

				config.OnDeadLetter(ctx, dlEntry)
			}
		},
	}

	err := retrypkg.Do(ctx, rcfg, attemptFn)
	if err == nil {
		return nil
	}

	// Re-surface the middleware's public sentinels (frozen v4 contract)
	// over go-retry's; the cause chain keeps go-retry's distinctions
	// (ErrCanceled vs ErrDeadlineExceeded, the final attempt error)
	// reachable via errors.Is/errors.AsType.
	switch {
	case errors.Is(err, retrypkg.ErrExhausted):
		return errorfamily.WrapInfrastructure(ErrRetryExhausted, "middleware.retry_exhausted",
			fmt.Sprintf("all %d attempts failed for %s", config.MaxAttempts, entry.Type)).WithCause(err)
	case errors.Is(err, retrypkg.ErrCanceled):
		return errorfamily.WrapInfrastructure(ErrRetryCanceled, "middleware.retry_canceled",
			entry.Type+": retry canceled").WithCause(err)
	default:
		return err
	}
}
