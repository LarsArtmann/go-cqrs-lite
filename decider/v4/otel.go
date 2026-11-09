package decider

import (
	"sync"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const deciderComponent = "decider"

// tracer returns a cached OpenTelemetry tracer for the decider module.
// The tracer is created once via sync.OnceValue to avoid repeated allocations
// on every Repository.Execute call.
var tracer = sync.OnceValue( //nolint:gochecknoglobals // lazy init pattern
	func() cqrsotel.Tracer {
		return cqrsotel.NewTracer(deciderComponent)
	},
)
