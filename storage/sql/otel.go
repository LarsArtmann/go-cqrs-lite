package sql

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const storageComponent = "storage"

// Tracer returns an OpenTelemetry tracer for the storage module.
func Tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

// StartStreamSpan creates a span for a stream operation with stream attributes.
func StartStreamSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		Tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.StreamAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

// Deprecated: use StartStreamSpan.
func StartAggregateSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return StartStreamSpan(ctx, spanName, ref, extraAttrs...)
}

// StartSaveSpan creates a span for a save operation with stream attributes.
func StartSaveSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx, Tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(
			cqrsotel.StreamAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, expectedVersion.Int()),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, eventCount),
		)...),
	)
}
