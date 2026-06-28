package middleware

import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"

const middlewareComponent = "middleware"

// retryTracer returns an OpenTelemetry tracer for retry attempt spans.
func retryTracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(middlewareComponent)
}
