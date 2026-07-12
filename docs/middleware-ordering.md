# Middleware Ordering Guide

> Recommended application order for the 30+ middlewares shipped across
> command, query, and event dispatch paths.

## Why order matters

Middleware is applied as a chain: the first `Use()` call is outermost (runs
first on the way in, last on the way out). The order determines behaviour in
subtle ways:

- **Recovery** must be outermost so it catches panics from everything below.
- **Idempotency** must run before any work is done (why retry a duplicate?).
- **Tracing** should wrap the actual handler so the span covers real work,
  not retry sleep loops.
- **Logging** should be innermost so it logs the final outcome.

## Recommended order

### Command dispatch

```go
cmds.Use(middleware.CommandIdempotency(store, ttl, nil))   // 1. dedup before work
cmds.Use(middleware.CommandRecovery())                       // 2. catch panics early
cmds.Use(middleware.CommandRetry(retryConfig))               // 3. retry transient failures
cmds.Use(middleware.CommandCircuitBreaker(brkConfig))        // 4. stop cascading failures
cmds.Use(bundle.Command()...)                                // 5. OTel tracing + metrics
cmds.Use(middleware.CommandLogging(logger))                  // 6. log final outcome
```

### Event dispatch (publish side)

```go
bus.UsePublish(middleware.EventRecovery())                   // 1. catch panics
bus.UsePublish(bundle.Publish()...)                          // 2. OTel tracing
bus.UsePublish(signing.SignMiddleware(signer))               // 3. sign after tracing
```

### Event dispatch (consume side)

```go
bus.Use(middleware.EventRecovery())                          // 1. catch panics
bus.Use(middleware.EventIdempotency(store, ttl, nil))        // 2. dedup
bus.Use(bundle.Event()...)                                   // 3. OTel tracing + metrics
bus.Use(middleware.EventLogging(logger))                     // 4. log final outcome
bus.Use(signing.VerifyMiddleware(signer))                    // 5. verify before handler
```

### Query dispatch

```go
qry.Use(middleware.QueryRecovery())                          // 1. catch panics
qry.Use(middleware.QueryRetry(retryConfig))                  // 2. retry transient failures
qry.Use(bundle.Query()...)                                   // 3. OTel tracing + metrics
qry.Use(middleware.QueryLogging(logger))                     // 4. log final outcome
```

## Rationale

| Position  | Middleware             | Why here                                                                                                         |
| --------- | ---------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Outermost | Idempotency            | Reject duplicates before any other middleware runs. No point retrying or logging a duplicate.                    |
| Early     | Recovery               | Catch panics from all middleware below. A panic in a retry loop or handler should never crash the process.       |
| Middle    | Retry / CircuitBreaker | Retry wraps the actual handler. CircuitBreaker opens after repeated failures, preventing retry storms.           |
| Inner     | OTel Tracing           | The span should cover the handler execution, not the retry sleep loop. This gives accurate latency measurements. |
| Innermost | Logging                | Log the final outcome (success or error) after all retries and circuit-breaker decisions.                        |

## Signing and encryption

Event signing and encryption are **publish-side transforms**, not behavioural
middleware. Apply them after tracing so the span covers the transform cost,
but before the event reaches the wire:

```
Publish: Recovery → OTel → Sign/Encrypt → Wire
Consume: Recovery → Dedup → OTel → Verify/Decrypt → Handler
```

## OTelBundle

`middleware.NewOTelBundle(tracer, meter)` returns a bundle that applies both
tracing and metrics in the correct order. Pass it as a single unit:

```go
bundle, _ := middleware.NewOTelBundle(tracer, nil, middleware.WithMetricsDisabled())
cmds.Use(bundle.Command()...)
bus.Use(bundle.Event()...)
bus.UsePublish(bundle.Publish()...)
qry.Use(bundle.Query()...)
```

## Anti-patterns

- **Tracing outermost** — spans will include retry sleep time, giving misleading latency data.
- **Logging before retry** — logs every attempt, including retries that succeed. Noise.
- **Recovery innermost** — misses panics from other middleware (e.g., a tracing bug).
- **Verify before dedup** — wastes CPU verifying signatures on duplicate events that will be discarded.
