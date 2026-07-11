package grpc

import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"

const transportComponent = "transport.grpc"

// tracer returns an OpenTelemetry tracer for the transport/grpc module.
func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(transportComponent)
}
