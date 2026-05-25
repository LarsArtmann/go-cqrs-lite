package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// logContext holds common logging context for middleware.
type logContext struct {
	prefix      string
	msgType     string
	aggregateID id.AggregateID
}

func logWithContext(logger *slog.Logger, lc logContext, fn func() error) error {
	start := time.Now()

	aggregateIDStr := lc.aggregateID.String()

	logger.Info(
		lc.prefix+" dispatching",
		"type", lc.msgType,
		"aggregateID", aggregateIDStr,
	)

	err := fn()
	duration := time.Since(start)

	if err != nil {
		logger.Error(
			lc.prefix+" failed",
			"type", lc.msgType,
			"aggregateID", aggregateIDStr,
			"duration", duration,
			"error", err,
		)

		return err
	}

	logger.Info(
		lc.prefix+" succeeded",
		"type", lc.msgType,
		"aggregateID", aggregateIDStr,
		"duration", duration,
	)

	return nil
}

// CommandLogging returns a command middleware that logs dispatch details with timing.
func CommandLogging(logger *slog.Logger) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			lc := logContext{
				prefix:      "command",
				msgType:     string(cmd.Type()),
				aggregateID: cmd.AggregateID(),
			}

			return logWithContext(logger, lc, func() error {
				return next(ctx, cmd)
			})
		}
	}
}

// EventLogging returns an event middleware that logs handler details with timing.
func EventLogging(logger *slog.Logger) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			lc := logContext{
				prefix:      "event",
				msgType:     string(evt.Type()),
				aggregateID: evt.AggregateID(),
			}

			return logWithContext(logger, lc, func() error {
				return next(ctx, evt)
			})
		}
	}
}

// QueryLogging returns a query middleware that logs dispatch details with timing.
func QueryLogging(logger *slog.Logger) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			lc := logContext{ //nolint:exhaustruct // queries have no aggregateID
				prefix:  "query",
				msgType: string(q.Type()),
			}

			var result any

			err := logWithContext(logger, lc, func() error {
				var e error

				result, e = next(ctx, q)

				return e
			})

			return result, err
		}
	}
}
