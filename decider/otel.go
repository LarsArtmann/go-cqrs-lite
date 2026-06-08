package decider

import (
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

const deciderComponent = "decider"

func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(deciderComponent)
}
