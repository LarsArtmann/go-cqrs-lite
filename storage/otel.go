package storage

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

const storageComponent = "storage"

func tracer() trace.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

func startSaveSpan(
	ctx context.Context,
	spanName string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, trace.Span) {
	return cqrsotel.StartSpan(
		ctx, tracer(), spanName,
		trace.SpanKindClient,
		trace.WithAttributes(append(
			cqrsotel.AggregateAttrs(aggregateType, aggregateID),
			attribute.Int(cqrsotel.AttrAggregateVersion, expectedVersion.Int()),
			attribute.Int(cqrsotel.AttrEventCount, eventCount),
		)...),
	)
}
