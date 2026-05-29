package saga

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

func tracer() trace.Tracer {
	return cqrsotel.NewTracer("saga")
}

func sagaAttrs(sagaType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(cqrsotel.AttrSagaType, sagaType),
	}
}
