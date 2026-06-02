package otel

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// NewTracer creates a tracer scoped to a specific go-cqrs-lite component.
// The component name is appended to the base instrumentation name:
//
//	NewTracer("storage") → "github.com/larsartmann/go-cqrs-lite/storage/v2"
//
// Uses the global TracerProvider, which returns a no-op tracer when
// no provider is configured.
func NewTracer(component string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(
		fmt.Sprintf("%s/%s", Name, component),
	)
}
