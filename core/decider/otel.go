package decider

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

const deciderComponent = "decider"

func tracer() trace.Tracer {
	return cqrsotel.NewTracer(deciderComponent)
}

func aggregateAttrs(aggregateType event.AggregateType, aggregateID id.AggregateID) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(cqrsotel.AttrAggregateType, aggregateType.String()),
		attribute.String(cqrsotel.AttrAggregateID, aggregateID.String()),
	}
}
