package otel

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// NewMeter creates a meter scoped to a specific go-cqrs-lite component.
// The component name is appended to the base instrumentation name:
//
//	NewMeter("middleware") → "github.com/larsartmann/go-cqrs-lite/middleware/v2"
//
// Uses the global MeterProvider, which returns a no-op meter when
// no provider is configured.
func NewMeter(component string) metric.Meter {
	return otel.GetMeterProvider().Meter(
		fmt.Sprintf("%s/%s", Name, component),
	)
}
