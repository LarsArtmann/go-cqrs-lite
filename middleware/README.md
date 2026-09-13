# middleware — Cross-Cutting Concerns for CQRS

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/middleware/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/middleware/v4)

Pre-built middleware for command, event, and query handlers. **38 typed middleware factories** (Command/Event/Query variants) plus generic builders and dead-letter stores — covering logging, trace-logging, recovery, retry, validation, metrics, tracing, circuit breaking, idempotency, flight recording, and actor context.

```bash
go get github.com/larsartmann/go-cqrs-lite/middleware/v4
```

## Available Middleware

### Logging

- `CommandLogging(logger)` — logs type, aggregateID, duration
- `EventLogging(logger)` — logs type, aggregateID, event count
- `QueryLogging(logger)` — logs type, duration

### Recovery

- `CommandRecovery()` — catches panics, returns error
- `EventRecovery()` — catches panics in event handlers
- `QueryRecovery()` — catches panics in query handlers

### Retry

- `CommandRetry(count, delay)` — retries on transient errors
- `EventRetry(count, delay)` — retries event handling
- `QueryRetry(count, delay)` — retries query handling

### Validation

- `CommandValidation(validate)` — validates commands before handling
- `EventValidation(validate)` — validates events before handling
- `QueryValidation(validate)` — validates queries before dispatch

### Metrics

- `CommandOTelMetrics(histogram)` / `EventOTelMetrics(histogram)` / `QueryOTelMetrics(histogram)` — OTel histogram per dispatch/handle
- `CommandOTelMetricsWithCounter(...)` / `Event...` / `Query...` — histogram + counter combo
- `CommandTypedMetrics(recorder)` / `Event...` / `Query...` — custom `TypedMetricsRecorder`
- `NewOTelBundle(...)` — wires tracer + metrics recorder in one call

### Tracing (OpenTelemetry)

- `CommandTracing(tracer)` — creates spans per command dispatch
- `EventTracing(tracer)` — creates spans per event handling
- `EventPublishTracing(tracer)` — creates spans per publish
- `QueryTracing(tracer)` — creates spans per query dispatch

### Circuit Breaker

- `CommandCircuitBreaker(opts)` — prevents cascading failures
- `EventCircuitBreaker(opts)` — circuit breaker for event handlers
- `QueryCircuitBreaker(opts)` — circuit breaker for queries

### Idempotency

- `CommandIdempotency(store, ttl, keyExtractor)` — deduplicates commands by ID (at-least-once delivery)
- `EventIdempotency(store, ttl, keyExtractor)` — deduplicates events by ID (webhooks, cross-system delivery)
- `QueryIdempotency(store, ttl, keyExtractor)` — deduplicates queries by custom key (requires non-nil keyExtractor)

### Flight Recording

- `CommandFlightRecorder(...)` / `EventFlightRecorder(...)` / `QueryFlightRecorder(...)` — ring-buffer capture of message + error for post-incident debugging

### Trace Logging

- `CommandTraceLogging(logger)` / `EventTraceLogging(logger)` / `QueryTraceLogging(logger)` — trace-context-aware logging

### Actor Context

- `CommandActorContext()` — stamps the acting `id.ActorID` from context onto the command

### Dead-Letter Stores

- `NewMemoryDeadLetterStore()` — in-memory DLQ (dev/test)
- `NewSQLDeadLetterStore(db, dialect)` — persistent DLQ

## In Sibling Modules

Signing and encryption middleware live in their own modules (not here):

- `signing.SignMiddleware(signer)` — signs events on publish
- `signing.VerifyMiddleware(verifier)` — verifies signatures on handle
- `signing.RequireSignatureMiddleware(verifier)` — rejects unsigned events
- `encryption.EncryptMiddleware(encrypter)` / `encryption.DecryptMiddleware(decrypter)` — payload crypto on publish/handle

## Usage

```go
cmds := command.NewDispatcher()
cmds.Use(middleware.CommandLogging(logger))
cmds.Use(middleware.CommandRecovery())
cmds.Use(middleware.CommandRetry(3, 100*time.Millisecond))
```

## Related Modules

- [**command**](../command/README.md) — `command.Dispatcher.Use()` applies command middleware
- [**event**](../event/README.md) — `event.Bus.Use()` / `UsePublish()` applies event middleware
- [**query**](../query/README.md) — `query.Dispatcher.Use()` applies query middleware
- [**go-idempotency**](https://github.com/larsartmann/go-idempotency) — `Store`, `MemoryStore`, `KVStore`, `ErrDuplicate` (used by idempotency middleware)
- [**signing**](../signing/README.md) — `SignMiddleware` / `VerifyMiddleware` / `RequireSignatureMiddleware` live there
- [**encryption**](../encryption/README.md) — `EncryptMiddleware` / `DecryptMiddleware` live there
- [**otel**](../otel/README.md) — Tracing middleware uses OTel tracers from this module
