// Package retry provides a dependency-free retry loop with exponential
// backoff and jitter. It is the zero-CQRS, zero-OTel core so that consumers
// who only need retry logic (CLI tools, batch processors, simple services)
// can import it without pulling in CQRS message types or OpenTelemetry SDK.
//
// The canonical implementation lives in [github.com/larsartmann/go-retry];
// this package re-exports it for backward compatibility (ADR-0064).
//
// For the CQRS-wrapped version (MessageAdapter, OTel spans, dead-letter
// entries with StreamID), use
// [github.com/larsartmann/go-cqrs-lite/middleware/v4].
package retry
