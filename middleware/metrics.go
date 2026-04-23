package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
)

func recordMetrics(
	rec MetricsRecorder,
	operation string,
	err error,
	label string,
	elapsed time.Duration,
) {
	if err != nil {
		rec.Observe(operation+"_error", elapsed, "type", label)
	} else {
		rec.Observe(operation+"_success", elapsed, "type", label)
	}
}

// CommandMetrics returns a middleware that records command handler metrics.
func CommandMetrics(recorder MetricsRecorder) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			start := time.Now()
			err := next(ctx, cmd)
			recordMetrics(recorder, "command", err, string(cmd.Type()), time.Since(start))

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
			recordMetrics(recorder, "event", err, string(evt.Type()), time.Since(start))

			return err
		}
	}
}
