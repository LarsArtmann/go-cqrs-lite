# middleware — Cross-Cutting Concerns for CQRS

Pre-built middleware for command, event, and query handlers. **24 middleware factories** covering 8 concerns across all 3 message types.

```bash
go get github.com/larsartmann/go-cqrs-lite/middleware/v2
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

- `CommandValidation()` — validates commands before handling
- `QueryValidation()` — validates queries before dispatch

### Metrics

- `CommandMetrics(recorder)` — records dispatch count, duration, errors
- `EventMetrics(recorder)` — records publish/handle metrics
- `QueryMetrics(recorder)` — records query metrics

### Tracing (OpenTelemetry)

- `CommandTracing(tracer)` — creates spans per command dispatch
- `EventTracing(tracer)` — creates spans per event handling
- `EventPublishTracing(tracer)` — creates spans per publish
- `QueryTracing(tracer)` — creates spans per query dispatch

### Circuit Breaker

- `CommandCircuitBreaker(opts)` — prevents cascading failures
- `EventCircuitBreaker(opts)` — circuit breaker for event handlers
- `QueryCircuitBreaker(opts)` — circuit breaker for queries

### Event Signing

- `EventSignMiddleware(signer)` — signs events on publish
- `EventVerifyMiddleware(verifier)` — verifies signatures on handle

## Usage

```go
cmds := command.NewDispatcher()
cmds.Use(middleware.CommandLogging(logger))
cmds.Use(middleware.CommandRecovery())
cmds.Use(middleware.CommandRetry(3, 100*time.Millisecond))
```
