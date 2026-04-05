package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// CommandLogging returns a command middleware that logs dispatch details with timing.
func CommandLogging(logger Logger) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			start := time.Now()
			cmdType := cmd.Type()

			logger.Info("command dispatching",
				"type", cmdType,
				"aggregateID", cmd.AggregateID(),
			)

			err := next(ctx, cmd)
			duration := time.Since(start)

			if err != nil {
				logger.Error("command failed",
					"type", cmdType,
					"aggregateID", cmd.AggregateID(),
					"duration", duration,
					"error", err,
				)

				return err
			}

			logger.Info("command succeeded",
				"type", cmdType,
				"aggregateID", cmd.AggregateID(),
				"duration", duration,
			)

			return nil
		}
	}
}

// EventLogging returns an event middleware that logs handler details with timing.
func EventLogging(logger Logger) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			start := time.Now()
			evtType := evt.Type()

			logger.Info("event handling",
				"type", evtType,
				"aggregateID", evt.AggregateID(),
				"version", evt.Version(),
			)

			err := next(ctx, evt)
			duration := time.Since(start)

			if err != nil {
				logger.Error("event handler failed",
					"type", evtType,
					"aggregateID", evt.AggregateID(),
					"duration", duration,
					"error", err,
				)

				return err
			}

			logger.Info("event handled",
				"type", evtType,
				"aggregateID", evt.AggregateID(),
				"duration", duration,
			)

			return nil
		}
	}
}
