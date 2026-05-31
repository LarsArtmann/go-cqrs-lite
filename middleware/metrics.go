package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func recordMetrics(
	ctx context.Context,
	rec MetricsRecorder,
	operation string,
	err error,
	label string,
	elapsed time.Duration,
) {
	if err != nil {
		rec.Observe(ctx, operation+"_error", elapsed, "type", label)
	} else {
		rec.Observe(ctx, operation+"_success", elapsed, "type", label)
	}
}

// CommandMetrics returns a middleware that records command handler metrics.
func CommandMetrics(recorder MetricsRecorder) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			start := time.Now()
			err := next(ctx, cmd)
			recordMetrics(ctx, recorder, "command", err, string(cmd.Type()), time.Since(start))

			return err
		}
	}
}

// EventMetrics returns a middleware that records event handler metrics.
func EventMetrics(recorder MetricsRecorder) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			start := time.Now()
			err := next(ctx, evt)
			recordMetrics(ctx, recorder, "event", err, string(evt.Type()), time.Since(start))

			return err
		}
	}
}

// QueryMetrics returns a middleware that records query handler metrics.
func QueryMetrics(recorder MetricsRecorder) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			start := time.Now()
			result, err := next(ctx, q)
			recordMetrics(ctx, recorder, "query", err, string(q.Type()), time.Since(start))

			return result, err
		}
	}
}
