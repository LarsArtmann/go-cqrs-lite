package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
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

// NewMetrics returns a generic middleware that records handler execution metrics.
func NewMetrics[M any](adapter MessageAdapter[M], recorder MetricsRecorder) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			start := time.Now()
			err := next(ctx, msg)
			recordMetrics(
				ctx,
				recorder,
				adapter.Kind,
				err,
				adapter.ExtractType(msg),
				time.Since(start),
			)

			return err
		}
	}
}

// CommandMetrics returns a middleware that records command handler metrics.
func CommandMetrics(recorder MetricsRecorder) command.Middleware {
	return AsCommand(NewMetrics(CommandAdapter, recorder))
}

// EventMetrics returns a middleware that records event handler metrics.
func EventMetrics(recorder MetricsRecorder) event.Middleware {
	return AsEvent(NewMetrics(EventAdapter, recorder))
}

// QueryMetrics returns a middleware that records query handler metrics.
func QueryMetrics(recorder MetricsRecorder) query.Middleware {
	return AsQuery(NewMetrics(QueryAdapter, recorder))
}
