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
	RecordOption     = metric.RecordOption
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

func MetricWithAttributes(attrs ...KeyValue) RecordOption {
	return metric.WithAttributes(attrs...)
}

func MetricWithDescription(desc string) metric.Float64HistogramOption {
	return metric.WithDescription(desc)
}

func MetricWithUnit(unit string) metric.Float64HistogramOption {
	return metric.WithUnit(unit)
}
