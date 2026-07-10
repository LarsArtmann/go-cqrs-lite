package sql

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

const storageComponent = "storage"

// Tracer returns an OpenTelemetry tracer for the storage module.
func Tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

// StartAggregateSpan creates a span for an aggregate operation with aggregate attributes.
func StartAggregateSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		Tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.AggregateAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

// StartSaveSpan creates a span for a save operation with aggregate attributes.
func StartSaveSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx, Tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, expectedVersion.Int()),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, eventCount),
		)...),
	)
}
