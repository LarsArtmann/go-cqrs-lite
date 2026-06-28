package decider

import (
	"sync"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

const deciderComponent = "decider"

var (
	tracerOnce   sync.Once
	cachedTracer cqrsotel.Tracer
)

// tracer returns a cached OpenTelemetry tracer for the decider module.
// The tracer is created once via sync.Once to avoid repeated allocations
// on every Repository.Execute call.
func tracer() cqrsotel.Tracer {
	tracerOnce.Do(func() {
		cachedTracer = cqrsotel.NewTracer(deciderComponent)
	})

	return cachedTracer
}
