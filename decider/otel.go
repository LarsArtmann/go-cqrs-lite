package decider

import (
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

const deciderComponent = "decider"

func tracer() trace.Tracer {
	return cqrsotel.NewTracer(deciderComponent)
}
