// Package middleware provides cross-cutting concerns for CQRS handlers.
// Middleware factories, OTel correlation bridging, SSE broker, and profiling endpoints.
//
// # Available Concerns
//
// Logging, Recovery, Retry, Validation, Metrics, Tracing (OTel),
// Circuit Breaker, Event Signing, OTel Correlation Enricher, and SSE Broker.
//
// Each middleware concern has 3 variants: Command*, Event*, Query*.
//
// # Usage
//
//	cmds.Use(middleware.CommandLogging(logger))
//	cmds.Use(middleware.CommandRecovery())
//	cmds.Use(middleware.CommandRetry(3, 100*time.Millisecond))
package middleware
