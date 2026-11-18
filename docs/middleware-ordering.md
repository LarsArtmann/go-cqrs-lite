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
cmds.Use(middleware.CommandRecovery())                       // 1. catch panics from everything below
cmds.Use(middleware.CommandIdempotency(store, ttl, nil))     // 2. dedup ONCE, outside retries
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
| Outermost | Recovery               | Catch panics from ALL middleware below, including a broken idempotency store. A panic in a retry loop or handler must never crash the process. |
| Second    | Idempotency            | Reject duplicates before anything does work, and — critically — OUTSIDE retry. Retry outside idempotency re-enters the dedup check on every attempt, so attempt 2 fails with `ErrDuplicate` and the retry is defeated by the dedup it wraps. |
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

## Pinned by a test

`command/v4/memory_bus_test.go::TestMemoryBus_MiddlewareOrder_FirstAddedIsOutermost` registers
two middlewares and asserts the chain direction this guide relies on.

That test exists because the claim was previously documented backwards: `command/v4/doc.go` and its
v5 mirror `core/v5/command/doc.go` said "last added runs first". Both were corrected 2026-11-18,
after a fleet survey found consumer wiring built on the inverted belief — `github-local-sync` used
`Use(Recovery, Retry, Idempotency)`, putting retry outside dedup, so every retried command failed
`ErrDuplicate` and the retry never ran.
