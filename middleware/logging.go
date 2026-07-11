package middleware

import (
	"context"
	"log/slog"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func logWithContext(
	logger *slog.Logger,
	prefix, msgType string,
	aggregateID id.AggregateID,
	fn func() error,
) error {
	start := time.Now()

	aggregateIDStr := aggregateID.String()

	logger.Info(
		prefix+" dispatching",
		"type", msgType,
		"aggregateID", aggregateIDStr,
	)

	err := fn()
	duration := time.Since(start)

	if err != nil {
		logger.Error(
			prefix+" failed",
			"type", msgType,
			"aggregateID", aggregateIDStr,
			"duration", duration,
			"error", err,
		)

		return errorfamily.Wrapf(
			err,
			errorfamily.Classify(err),
			"middleware.logging",
			"%s %s",
			prefix,
			msgType,
		)
	}

	logger.Info(
		prefix+" succeeded",
		"type", msgType,
		"aggregateID", aggregateIDStr,
		"duration", duration,
	)

	return nil
}

// NewLogging returns a generic middleware that logs dispatch details with timing.
func NewLogging[M any](adapter MessageAdapter[M], logger *slog.Logger) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			var aggID id.AggregateID

			if adapter.ExtractID != nil {
				aggID = adapter.ExtractID(msg)
			}

			return logWithContext(
				logger,
				adapter.Kind,
				adapter.ExtractType(msg),
				aggID,
				func() error {
					return next(ctx, msg)
				},
			)
		}
	}
}

// CommandLogging returns a command middleware that logs dispatch details with timing.
func CommandLogging(logger *slog.Logger) command.Middleware {
	return AsCommand(NewLogging(CommandAdapter, logger))
}

// EventLogging returns an event middleware that logs handler details with timing.
func EventLogging(logger *slog.Logger) event.Middleware {
	return AsEvent(NewLogging(EventAdapter, logger))
}

// QueryLogging returns a query middleware that logs dispatch details with timing.
func QueryLogging(logger *slog.Logger) query.Middleware {
	return AsQuery(NewLogging(QueryAdapter, logger))
}
