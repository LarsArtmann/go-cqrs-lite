package middleware

import (
	"context"
	"log/slog"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// CommandTraceLogging returns a command middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func CommandTraceLogging(logger *slog.Logger) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			tLogger := cqrsotel.ContextLogger(logger, ctx)
			tLogger.Info("command dispatching",
				"type", string(cmd.Type()),
				"aggregate_id", cmd.AggregateID().String(),
			)

			err := next(ctx, cmd)
			if err != nil {
				tLogger.Error("command failed",
					"type", string(cmd.Type()),
					"error", err,
				)

				return err
			}

			tLogger.Info("command succeeded", "type", string(cmd.Type()))

			return nil
		}
	}
}

// EventTraceLogging returns an event middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func EventTraceLogging(logger *slog.Logger) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			tLogger := cqrsotel.ContextLogger(logger, ctx)
			tLogger.Info("event handling",
				"type", string(evt.Type()),
				"aggregate_id", evt.AggregateID().String(),
			)

			err := next(ctx, evt)
			if err != nil {
				tLogger.Error("event handler failed",
					"type", string(evt.Type()),
					"error", err,
				)

				return err
			}

			tLogger.Info("event handled", "type", string(evt.Type()))

			return nil
		}
	}
}

// QueryTraceLogging returns a query middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func QueryTraceLogging(logger *slog.Logger) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			tLogger := cqrsotel.ContextLogger(logger, ctx)
			tLogger.Info("query dispatching", "type", string(q.Type()))

			result, err := next(ctx, q)
			if err != nil {
				tLogger.Error("query failed",
					"type", string(q.Type()),
					"error", err,
				)

				return nil, err
			}

			tLogger.Info("query succeeded", "type", string(q.Type()))

			return result, nil
		}
	}
}
