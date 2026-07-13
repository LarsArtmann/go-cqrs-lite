// Package retry provides a dependency-free retry loop with exponential
// backoff and jitter. It is the zero-CQRS, zero-OTel core so that consumers
// who only need retry logic (CLI tools, batch processors, simple services)
// can import it without pulling in CQRS message types or OpenTelemetry SDK.
//
// For the CQRS-wrapped version (MessageAdapter, OTel spans, dead-letter
// entries with AggregateID), use
// [github.com/larsartmann/go-cqrs-lite/middleware/v4].
package retry
