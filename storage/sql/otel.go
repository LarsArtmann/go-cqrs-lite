package sql

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

const storageComponent = "storage"

// Tracer returns an OpenTelemetry tracer for the storage module.
func Tracer() trace.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

// StartSaveSpan creates a span for a save operation with aggregate attributes.
func StartSaveSpan(
	ctx context.Context,
	spanName string,
	ref event.AggregateRef,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, trace.Span) {
	return cqrsotel.StartSpan(
		ctx, Tracer(), spanName,
		trace.SpanKindClient,
		trace.WithAttributes(append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			attribute.Int(cqrsotel.AttrAggregateVersion, expectedVersion.Int()),
			attribute.Int(cqrsotel.AttrEventCount, eventCount),
		)...),
	)
}
