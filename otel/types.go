package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type (
	Tracer           = trace.Tracer
	Span             = trace.Span
	SpanKind         = trace.SpanKind
	SpanStartOption  = trace.SpanStartOption
	SpanEndOption    = trace.SpanEndOption
	EventOption      = trace.EventOption
	KeyValue         = attribute.KeyValue
	Meter            = metric.Meter
	Float64Histogram = metric.Float64Histogram
	Int64Counter     = metric.Int64Counter
	RecordOption     = metric.RecordOption
	AddOption        = metric.AddOption
)

const (
	SpanKindInternal = trace.SpanKindInternal
	SpanKindServer   = trace.SpanKindServer
	SpanKindClient   = trace.SpanKindClient
	SpanKindProducer = trace.SpanKindProducer
	SpanKindConsumer = trace.SpanKindConsumer
)

func WithAttributes(attrs ...KeyValue) SpanStartOption {
	return trace.WithAttributes(attrs...)
}

func WithSpanKind(kind SpanKind) SpanStartOption {
	return trace.WithSpanKind(kind)
}

func SpanFromContext(ctx context.Context) Span {
	return trace.SpanFromContext(ctx)
}

func AttrString(key, value string) KeyValue {
	return attribute.String(key, value)
}

func AttrInt(key string, value int) KeyValue {
	return attribute.Int(key, value)
}

func AttrInt64(key string, value int64) KeyValue {
	return attribute.Int64(key, value)
}

// Semantic convention attribute keys for OTel resource identification.
// Use these with resource.NewWithAttributes when configuring TracerProvider/MeterProvider.
const (
	AttrServiceName     = "service.name"
	AttrServiceVersion  = "service.version"
	AttrServiceInstance = "service.instance.id"
)

// ServiceResourceAttributes returns key-value pairs for service identification.
// Use these when creating an OTel Resource for the TracerProvider or MeterProvider.
//
//	resource.NewWithAttributes(
//	    cqrsotel.ServiceResourceAttributes("my-app", "1.0.0", "instance-1")...,
//	)
func ServiceResourceAttributes(name, version, instanceID string) []KeyValue {
	return []KeyValue{
		attribute.String(AttrServiceName, name),
		attribute.String(AttrServiceVersion, version),
		attribute.String(AttrServiceInstance, instanceID),
	}
}

func MetricWithAttributes(attrs ...KeyValue) RecordOption {
	return metric.WithAttributes(attrs...)
}

func MetricWithDescription(desc string) metric.Float64HistogramOption {
	return metric.WithDescription(desc)
}

func MetricWithUnit(unit string) metric.Float64HistogramOption {
	return metric.WithUnit(unit)
}

// CounterMetricWithDescription sets the description for an Int64Counter instrument.
func CounterMetricWithDescription(desc string) metric.Int64CounterOption {
	return metric.WithDescription(desc)
}

// CounterMetricWithUnit sets the unit for an Int64Counter instrument.
func CounterMetricWithUnit(unit string) metric.Int64CounterOption {
	return metric.WithUnit(unit)
}

// CounterAddWithAttributes creates add options for Int64Counter.Add with attributes.
func CounterAddWithAttributes(attrs ...KeyValue) AddOption {
	return metric.WithAttributes(attrs...)
}

// CQRSHistogramBoundaries provides optimized histogram bucket boundaries for
// CQRS operation durations. Covers 0.05ms to 10s with emphasis on the
// 0.1ms–50ms range where most event/command/query operations fall.
//
// Consumers can use these boundaries when configuring SDK Views:
//
//	view.NewView(
//	    view.WithInstrumentName("cqrs.operation.duration"),
//	    view.WithAggregation(metric.AggregationExplicitBucketHistogram{
//	        Boundaries: cqrsotel.CQRSHistogramBoundaries,
//	    }),
//	)
var CQRSHistogramBoundaries = []float64{
	0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
}
