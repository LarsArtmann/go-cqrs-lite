package watermill

import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"

const watermillComponent = "watermill"

// tracer returns an OpenTelemetry tracer for the watermill module.
func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(watermillComponent)
}
