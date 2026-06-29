package middleware

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// OTelMetricsRecorder implements MetricsRecorder using OpenTelemetry histograms.
type OTelMetricsRecorder struct {
	histogram cqrsotel.Float64Histogram
	counter   cqrsotel.Int64Counter
}

// Histogram returns the underlying OTel histogram instrument.
func (r *OTelMetricsRecorder) Histogram() cqrsotel.Float64Histogram {
	return r.histogram
}

// Counter returns the underlying OTel counter instrument.
// Used by the OTel bundle to wire typed metrics middleware.
func (r *OTelMetricsRecorder) Counter() cqrsotel.Int64Counter {
	return r.counter
}

// NewOTelMetricsRecorder creates a new OTelMetricsRecorder from the given meter.
// The histogram instrument name is "cqrs.operation.duration".
// The counter instrument name is "cqrs.operation.count".
func NewOTelMetricsRecorder(meter cqrsotel.Meter) (*OTelMetricsRecorder, error) {
	h, err := meter.Float64Histogram(
		"cqrs.operation.duration",
		cqrsotel.MetricWithDescription("Duration of CQRS operations"),
		cqrsotel.MetricWithUnit("ms"),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	c, err := meter.Int64Counter(
		"cqrs.operation.count",
		cqrsotel.CounterMetricWithDescription("Total count of CQRS operations"),
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // otel SDK error
	}

	return &OTelMetricsRecorder{histogram: h, counter: c}, nil
}

// Observe records a metric observation with the given name, duration, and labels.
// Labels are passed as alternating key-value string pairs.
//
// Deprecated: Use ObserveTyped with attribute.KeyValue pairs instead. The
// string-label form silently drops malformed (odd-length) label pairs and
// offers no compile-time type safety.
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
	r.counter.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(attrs...))
}

// ObserveTyped records a metric observation with typed attributes, satisfying
// TypedMetricsRecorder. It prepends the operation attribute and records both
// the duration histogram and operation counter.
func (r *OTelMetricsRecorder) ObserveTyped(
	ctx context.Context,
	operation string,
	duration time.Duration,
	attrs ...cqrsotel.KeyValue,
) {
	fullAttrs := make([]cqrsotel.KeyValue, 0, len(attrs)+1)
	fullAttrs = append(fullAttrs, cqrsotel.AttrString("operation", operation))
	fullAttrs = append(fullAttrs, attrs...)

	r.histogram.Record(ctx, float64(duration.Milliseconds()),
		cqrsotel.MetricWithAttributes(fullAttrs...))
	r.counter.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(fullAttrs...))
}

// NewOTelMetricsWithCounter returns a generic middleware that records duration
// using an OTel histogram and increments an OTel counter.
func NewOTelMetricsWithCounter[M any](
	kindAttr, typeAttr string,
	extractType func(M) string,
	histogram cqrsotel.Float64Histogram,
	counter cqrsotel.Int64Counter,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			start := time.Now()
			err := next(ctx, msg)

			status := cqrsotel.StatusSuccess
			if err != nil {
				status = cqrsotel.StatusError
			}

			attrs := []cqrsotel.KeyValue{
				cqrsotel.AttrString(cqrsotel.AttrMessageKind, kindAttr),
				cqrsotel.AttrString(typeAttr, extractType(msg)),
				cqrsotel.AttrString(cqrsotel.AttrStatus, status),
			}

			histogram.Record(
				ctx, float64(time.Since(start).Milliseconds()),
				cqrsotel.MetricWithAttributes(attrs...),
			)
			counter.Add(ctx, 1, cqrsotel.CounterAddWithAttributes(attrs...))

			return err
		}
	}
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
	return AsCommand(NewOTelMetrics(
		cqrsotel.KindCommand, cqrsotel.AttrCommandType,
		func(cmd command.Command) string { return string(cmd.Type()) },
		histogram,
	))
}

// EventOTelMetrics returns an event middleware that records duration using an OTel histogram.
func EventOTelMetrics(histogram cqrsotel.Float64Histogram) event.Middleware {
	return AsEvent(NewOTelMetrics(
		cqrsotel.KindEvent, cqrsotel.AttrEventType,
		func(evt event.Event) string { return string(evt.Type()) },
		histogram,
	))
}

// QueryOTelMetrics returns a query middleware that records duration using an OTel histogram.
func QueryOTelMetrics(histogram cqrsotel.Float64Histogram) query.Middleware {
	return AsQuery(NewOTelMetrics(
		cqrsotel.KindQuery, cqrsotel.AttrQueryType,
		func(q query.Query) string { return string(q.Type()) },
		histogram,
	))
}

// CommandOTelMetricsWithCounter returns a command middleware that records both
// duration (histogram) and count (counter) using OTel instruments.
func CommandOTelMetricsWithCounter(
	histogram cqrsotel.Float64Histogram,
	counter cqrsotel.Int64Counter,
) command.Middleware {
	return AsCommand(NewOTelMetricsWithCounter(
		cqrsotel.KindCommand, cqrsotel.AttrCommandType,
		func(cmd command.Command) string { return string(cmd.Type()) },
		histogram, counter,
	))
}

// EventOTelMetricsWithCounter returns an event middleware that records both
// duration (histogram) and count (counter) using OTel instruments.
func EventOTelMetricsWithCounter(
	histogram cqrsotel.Float64Histogram,
	counter cqrsotel.Int64Counter,
) event.Middleware {
	return AsEvent(NewOTelMetricsWithCounter(
		cqrsotel.KindEvent, cqrsotel.AttrEventType,
		func(evt event.Event) string { return string(evt.Type()) },
		histogram, counter,
	))
}

// QueryOTelMetricsWithCounter returns a query middleware that records both
// duration (histogram) and count (counter) using OTel instruments.
func QueryOTelMetricsWithCounter(
	histogram cqrsotel.Float64Histogram,
	counter cqrsotel.Int64Counter,
) query.Middleware {
	return AsQuery(NewOTelMetricsWithCounter(
		cqrsotel.KindQuery, cqrsotel.AttrQueryType,
		func(q query.Query) string { return string(q.Type()) },
		histogram, counter,
	))
}
