package otel

import (
	"go.opentelemetry.io/otel/sdk"
)

const (
	// Name is the instrumentation scope name used for all go-cqrs-lite tracers and meters.
	Name = "github.com/larsartmann/go-cqrs-lite"
)

// ScopeName is the canonical instrumentation scope name, aligned with the
// OpenTelemetry contrib convention. Instrumentation packages MUST expose
// this constant. It matches Name — separated for discoverability and to
// match upstream naming.
//
// See: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/CONTRIBUTING.md#instrumentation-packages
const ScopeName = Name

// Version returns the version of the go-cqrs-lite OTel instrumentation.
// Follows the otel-contrib convention where every instrumentation package
// exposes a Version() function for telemetry scope attribution.
//
// For consumers, this lets the SDK report which instrumentation version
// produced the telemetry — useful for debugging and upgrade planning.
func Version() string {
	return sdk.Version()
}
