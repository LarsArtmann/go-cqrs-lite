package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// logContext holds common logging context for middleware.
type logContext struct {
	prefix      string
	msgType     string
	aggregateID string
}

func logWithContext(logger Logger, lc logContext, fn func() error) error {
	start := time.Now()

	logger.Info(lc.prefix+" dispatching",
		"type", lc.msgType,
		"aggregateID", lc.aggregateID,
	)

	err := fn()
	duration := time.Since(start)

	if err != nil {
		logger.Error(lc.prefix+" failed",
			"type", lc.msgType,
			"aggregateID", lc.aggregateID,
			"duration", duration,
			"error", err,
		)

		return err
	}

	logger.Info(lc.prefix+" succeeded",
		"type", lc.msgType,
		"aggregateID", lc.aggregateID,
		"duration", duration,
	)

	return nil
}

// CommandLogging returns a command middleware that logs dispatch details with timing.
func CommandLogging(logger Logger) command.Middleware {
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
func EventLogging(logger Logger) event.Middleware {
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
