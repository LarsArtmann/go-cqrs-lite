// Package middleware provides cross-cutting concerns for CQRS handlers.
// 24 middleware factories covering 8 concerns across command, event, and query handlers.
//
// # Available Concerns
//
// Logging, Recovery, Retry, Validation, Metrics, Tracing (OTel),
// Circuit Breaker, and Event Signing.
//
// Each concern has 3 variants: Command*, Event*, Query*.
//
// # Usage
//
//	cmds.Use(middleware.CommandLogging(logger))
//	cmds.Use(middleware.CommandRecovery())
//	cmds.Use(middleware.CommandRetry(3, 100*time.Millisecond))
package middleware
