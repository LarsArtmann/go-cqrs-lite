package event

import (
	"context"
	"log/slog"
	"math"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RejectingPublishMiddleware returns a PublishMiddleware whose publisher
// rejects every publish with a Rejection carrying the given code and message.
//
// It is the shared nil-guard for middleware constructors (e.g.
// signing.SignMiddleware, encryption.EncryptMiddleware): when a required
// dependency is nil, return this instead of panicking so the error surfaces
// at first use, at the call site, with a clear code.
func RejectingPublishMiddleware(code, msg string) PublishMiddleware {
	return func(_ Publisher) Publisher {
		return PublisherFunc(func(_ context.Context, _ ...Event) error {
			return errorfamily.NewRejection(code, msg)
		})
	}
}

// RejectingHandlerMiddleware returns a Middleware whose handler rejects
// every event with a Rejection carrying the given code and message.
//
// It is the handler-side counterpart to [RejectingPublishMiddleware].
func RejectingHandlerMiddleware(code, msg string) Middleware {
	return func(_ Handler) Handler {
		return func(_ context.Context, _ Event) error {
			return errorfamily.NewRejection(code, msg)
		}
	}
}

// PublishRetry returns a PublishMiddleware that retries transient publish
// failures with exponential backoff. Non-retryable errors (Rejection,
// Corruption, Conflict — per errorfamily.IsRetryable) fail immediately on the
// first attempt instead of burning retry cycles.
//
// Backoff: base * exp^(attempt-1), capped at maxDelay. Context cancellation
// aborts between attempts.
func PublishRetry(attempts int, baseDelay, maxDelay time.Duration, logger *slog.Logger) PublishMiddleware {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	if attempts < 1 {
		attempts = 1
	}

	return func(next Publisher) Publisher {
		return PublisherFunc(func(ctx context.Context, events ...Event) error {
			var lastErr error

			for attempt := range attempts {
				if attempt > 0 {
					backoff := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
					backoff = min(backoff, maxDelay)

					if logger != nil {
						logger.Debug("retrying event publish",
							"event_count", len(events),
							"attempt", attempt,
							"backoff", backoff,
						)
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(backoff):
					}
				}

				lastErr = next.Publish(ctx, events...)
				if lastErr == nil {
					return nil
				}

				if !errorfamily.IsRetryable(lastErr) {
					return lastErr
				}
			}

			return lastErr
		})
	}
}
