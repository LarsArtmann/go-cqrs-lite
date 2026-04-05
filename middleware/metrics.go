package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// CommandMetrics returns a middleware that records command handler metrics.
func CommandMetrics(recorder MetricsRecorder) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			start := time.Now()
			err := next(ctx, cmd)
			duration := time.Since(start)

			label := string(cmd.Type())
			if err != nil {
				recorder.Observe("command_error", duration, "type", label)
			} else {
				recorder.Observe("command_success", duration, "type", label)
			}

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
			duration := time.Since(start)

			label := string(evt.Type())
			if err != nil {
				recorder.Observe("event_error", duration, "type", label)
			} else {
				recorder.Observe("event_success", duration, "type", label)
			}

			return err
		}
	}
}
