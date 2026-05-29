package middleware

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/query"
)

const (
	metricNameCommandDuration = "cqrs.command.duration"
	metricNameEventDuration   = "cqrs.event.duration"
	metricNameQueryDuration   = "cqrs.query.duration"
)

// OTelMetricsRecorder implements MetricsRecorder using OpenTelemetry histograms.
type OTelMetricsRecorder struct {
	histogram metric.Float64Histogram
}

// NewOTelMetricsRecorder creates a new OTelMetricsRecorder from the given meter.
// The histogram instrument name is "cqrs.operation.duration".
func NewOTelMetricsRecorder(meter metric.Meter) (*OTelMetricsRecorder, error) {
	h, err := meter.Float64Histogram(
		"cqrs.operation.duration",
		metric.WithDescription("Duration of CQRS operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return &OTelMetricsRecorder{histogram: h}, nil
}

// Observe records a metric observation with the given name, duration, and labels.
// Labels are passed as alternating key-value string pairs.
func (r *OTelMetricsRecorder) Observe(name string, duration time.Duration, labels ...string) {
	opts := make([]metric.RecordOption, 0, 1)
	attrs := make([]attribute.KeyValue, 0, (len(labels)/2)+1)
	attrs = append(attrs, attribute.String("operation", name))

	for i := 0; i+1 < len(labels); i += 2 {
		attrs = append(attrs, attribute.String(labels[i], labels[i+1]))
	}

	opts = append(opts, metric.WithAttributes(attrs...))
	r.histogram.Record(context.Background(), float64(duration.Milliseconds()), opts...)
}

// CommandOTelMetrics returns a command middleware that records duration using an OTel histogram.
func CommandOTelMetrics(histogram metric.Float64Histogram) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			start := time.Now()
			err := next(ctx, cmd)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			histogram.Record(
				ctx, float64(time.Since(start).Milliseconds()),
				metric.WithAttributes(
					attribute.String(cqrsotel.AttrMessageKind, cqrsotel.KindCommand),
					attribute.String(cqrsotel.AttrCommandType, string(cmd.Type())),
					attribute.String(cqrsotel.AttrStatus, status),
				),
			)

			return err
		}
	}
}

// EventOTelMetrics returns an event middleware that records duration using an OTel histogram.
func EventOTelMetrics(histogram metric.Float64Histogram) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			start := time.Now()
			err := next(ctx, evt)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			histogram.Record(
				ctx, float64(time.Since(start).Milliseconds()),
				metric.WithAttributes(
					attribute.String(cqrsotel.AttrMessageKind, cqrsotel.KindEvent),
					attribute.String(cqrsotel.AttrEventType, string(evt.Type())),
					attribute.String(cqrsotel.AttrStatus, status),
				),
			)

			return err
		}
	}
}

// QueryOTelMetrics returns a query middleware that records duration using an OTel histogram.
func QueryOTelMetrics(histogram metric.Float64Histogram) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			start := time.Now()
			result, err := next(ctx, q)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			histogram.Record(
				ctx, float64(time.Since(start).Milliseconds()),
				metric.WithAttributes(
					attribute.String(cqrsotel.AttrMessageKind, cqrsotel.KindQuery),
					attribute.String(cqrsotel.AttrQueryType, string(q.Type())),
					attribute.String(cqrsotel.AttrStatus, status),
				),
			)

			return result, err
		}
	}
}
