package middleware

import (
	"context"
	"time"


	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// OTelMetricsRecorder implements MetricsRecorder using OpenTelemetry histograms.
type OTelMetricsRecorder struct {
	histogram cqrsotel.Float64Histogram
}

// NewOTelMetricsRecorder creates a new OTelMetricsRecorder from the given meter.
// The histogram instrument name is "cqrs.operation.duration".
func NewOTelMetricsRecorder(meter cqrsotel.Meter) (*OTelMetricsRecorder, error) {
	h, err := meter.Float64Histogram(
		"cqrs.operation.duration",
		cqrsotel.MetricWithDescription("Duration of CQRS operations"),
		cqrsotel.MetricWithUnit("ms"),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	return &OTelMetricsRecorder{histogram: h}, nil
}

// Observe records a metric observation with the given name, duration, and labels.
// Labels are passed as alternating key-value string pairs.
func (r *OTelMetricsRecorder) Observe(
	ctx context.Context,
	name string,
	duration time.Duration,
	labels ...string,
) {
	const keyValuePairs = 2 // labels come in alternating key-value pairs

	opts := make([]cqrsotel.RecordOption, 0, 1)
	attrs := make(
		[]cqrsotel.KeyValue,
		0,
		(len(labels)/keyValuePairs)+1,
	)
	attrs = append(attrs, cqrsotel.AttrString("operation", name))

	for i := 0; i+1 < len(labels); i += 2 {
		attrs = append(attrs, cqrsotel.AttrString(labels[i], labels[i+1]))
	}

	opts = append(opts, cqrsotel.MetricWithAttributes(attrs...))
	r.histogram.Record(ctx, float64(duration.Milliseconds()), opts...)
}

// NewOTelMetrics returns a generic middleware that records duration using an OTel histogram.
func NewOTelMetrics[M any](
	kindAttr, typeAttr string,
	extractType func(M) string,
	histogram cqrsotel.Float64Histogram,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			start := time.Now()
			err := next(ctx, msg)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			histogram.Record(
				ctx, float64(time.Since(start).Milliseconds()),
				cqrsotel.MetricWithAttributes(
					cqrsotel.AttrString(cqrsotel.AttrMessageKind, kindAttr),
					cqrsotel.AttrString(typeAttr, extractType(msg)),
					cqrsotel.AttrString(cqrsotel.AttrStatus, status),
				),
			)

			return err
		}
	}
}

// CommandOTelMetrics returns a command middleware that records duration using an OTel histogram.
func CommandOTelMetrics(histogram cqrsotel.Float64Histogram) command.Middleware {
	return AsCommand(NewOTelMetrics[command.Command](
		cqrsotel.KindCommand, cqrsotel.AttrCommandType,
		func(cmd command.Command) string { return string(cmd.Type()) },
		histogram,
	))
}

// EventOTelMetrics returns an event middleware that records duration using an OTel histogram.
func EventOTelMetrics(histogram cqrsotel.Float64Histogram) event.Middleware {
	return AsEvent(NewOTelMetrics[event.Event](
		cqrsotel.KindEvent, cqrsotel.AttrEventType,
		func(evt event.Event) string { return string(evt.Type()) },
		histogram,
	))
}

// QueryOTelMetrics returns a query middleware that records duration using an OTel histogram.
func QueryOTelMetrics(histogram cqrsotel.Float64Histogram) query.Middleware {
	return AsQuery(NewOTelMetrics[query.Query](
		cqrsotel.KindQuery, cqrsotel.AttrQueryType,
		func(q query.Query) string { return string(q.Type()) },
		histogram,
	))
}
