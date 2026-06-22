package middleware

import (
	"context"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// TraceLogMessages holds the log messages for each trace logging phase.
type TraceLogMessages struct {
	Dispatching string
	Failed      string
	Succeeded   string
}

// NewTraceLogging returns a generic middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func NewTraceLogging[M any](
	logger *slog.Logger,
	msgs TraceLogMessages,
	extractType func(M) string,
	extractAggID func(M) string,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			tLogger := cqrsotel.ContextLogger(logger, ctx)

			args := []any{"type", extractType(msg)}
			if extractAggID != nil {
				args = append(args, "aggregate_id", extractAggID(msg))
			}

			tLogger.Info(msgs.Dispatching, args...)

			err := next(ctx, msg)
			if err != nil {
				tLogger.Error(msgs.Failed, "type", extractType(msg), "error", err)

				return err
			}

			tLogger.Info(msgs.Succeeded, "type", extractType(msg))

			return nil
		}
	}
}

// CommandTraceLogging returns a command middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func CommandTraceLogging(logger *slog.Logger) command.Middleware {
	return AsCommand(NewTraceLogging[command.Command](
		logger, TraceLogMessages{
			Dispatching: "command dispatching",
			Failed:      "command failed",
			Succeeded:   "command succeeded",
		}, func(cmd command.Command) string { return string(cmd.Type()) },
		func(cmd command.Command) string { return cmd.AggregateID().String() },
	))
}

// EventTraceLogging returns an event middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func EventTraceLogging(logger *slog.Logger) event.Middleware {
	return AsEvent(NewTraceLogging[event.Event](
		logger, TraceLogMessages{
			Dispatching: "event handling",
			Failed:      "event handler failed",
			Succeeded:   "event handled",
		}, func(evt event.Event) string { return string(evt.Type()) },
		func(evt event.Event) string { return evt.AggregateID().String() },
	))
}

// QueryTraceLogging returns a query middleware that injects trace_id and
// span_id from the OTel context into structured log entries.
func QueryTraceLogging(logger *slog.Logger) query.Middleware {
	return AsQuery(NewTraceLogging[query.Query](logger, TraceLogMessages{
		Dispatching: "query dispatching",
		Failed:      "query failed",
		Succeeded:   "query succeeded",
	}, func(q query.Query) string { return string(q.Type()) }, nil))
}
