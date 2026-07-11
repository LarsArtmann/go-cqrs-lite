package http

import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"

const transportComponent = "transport.http"

// tracer returns an OpenTelemetry tracer for the transport/http module.
func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(transportComponent)
}
