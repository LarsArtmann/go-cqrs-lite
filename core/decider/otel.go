package decider

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

const deciderComponent = "decider"

func tracer() trace.Tracer {
	return cqrsotel.NewTracer(deciderComponent)
}

func aggregateAttrs(aggType string, aggregateID id.AggregateID) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(cqrsotel.AttrAggregateType, aggType),
		attribute.String(cqrsotel.AttrAggregateID, aggregateID.String()),
	}
}
