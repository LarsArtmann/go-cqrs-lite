// Package otel provides shared OpenTelemetry instrumentation utilities for go-cqrs-lite.
//
// This module centralizes tracer/meter creation, semantic attribute constants,
// and span helpers so that every go-cqrs-lite module produces consistent,
// convention-compliant telemetry without duplicating boilerplate.
//
// All instrumentation is opt-in: if no TracerProvider or MeterProvider is
// configured, the global defaults return no-op implementations with zero overhead.
package otel
