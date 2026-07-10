package pebble

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

const pebbleComponent = "pebble"

// tracer returns an OpenTelemetry tracer scoped to the pebble module.
// Uses the global TracerProvider, which returns a no-op tracer when no
// provider is configured.
func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(pebbleComponent)
}

// startAggregateSpan creates a span for an aggregate-scoped snapshot operation
// with standard aggregate type and ID attributes.
func startAggregateSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.AggregateAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

// startProjectionSpan creates a span for a checkpoint operation scoped to
// a projection name.
func startProjectionSpan(
	ctx context.Context,
	spanName, projectionName string,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrProjectionName, projectionName),
		),
	)
}
