package storage

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

const storageComponent = "storage"

// tracer returns a tracer for the storage module.
// Uses the global TracerProvider; returns a no-op tracer when no provider is configured.
func tracer() trace.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

func aggregateAttrs(aggType, aggID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(cqrsotel.AttrAggregateType, aggType),
		attribute.String(cqrsotel.AttrAggregateID, aggID),
	}
}

func aggregateAttrsWithVersion(aggType, aggID string, version int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(cqrsotel.AttrAggregateType, aggType),
		attribute.String(cqrsotel.AttrAggregateID, aggID),
		attribute.Int(cqrsotel.AttrAggregateVersion, version),
	}
}
