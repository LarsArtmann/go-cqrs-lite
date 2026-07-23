# ADR 0049: Dispatch-Time Middleware Application

> **Status:** ACCEPTED
> **Date:** 2026-07-10
> **Related:** `docs/status/2026-07-09_08-46` items e2, P1 #12-13

## Context

The `dispatcher.Dispatcher[H, M]` previously built its middleware chain at
`Register()` time. Middleware added via `Use()` AFTER `Register()` was
silently bypassed — the handler was already wrapped in the existing chain.

This caused subtle bugs:

1. Middleware ordering was fixed at registration time, not dispatch time.
2. Adding logging or tracing middleware after handlers were registered
   had no effect.
3. The "free ordering" promise of the middleware API was violated.

## Decision

The middleware chain is now rebuilt on every `Dispatch()` call (lazy
construction). This means:

1. Middleware can be added in any order relative to `Register()`.
2. All middleware applies to all handlers, regardless of registration order.
3. The first-added middleware is the outermost wrapper (first to execute).

The chain construction is:

```
For each dispatch:
  chain = handler
  for mw in reverse(middleware):
    chain = mw(chain)
  execute(chain)
```

## Alternatives Considered

- **Rebuild chain on Use()** — Rejected: requires tracking which handlers
  are wrapped, adds complexity, and still has a race window between Use()
  and Dispatch().

- **Immutable middleware list** — Rejected: prevents adding middleware
  dynamically (e.g., adding tracing after initial setup).

## Consequences

- Slight performance cost: the middleware chain is rebuilt on every dispatch.
  This is negligible compared to the handler execution cost.
- Middleware ordering is now predictable and documented.
- The `dispatcher/doc.go` correctly documents dispatch-time application.

> **Cross-reference:** [ADR-0020](0020-performance-optimization-patterns.md)
> Pattern 2 documents the opposite approach (pre-compute chains, rebuild
> only on `Use()`) for `event.MemoryBus`. The tradeoff is documented in
> both ADRs: `MemoryBus` pre-computes because `Publish()` is a per-event
> hot path; `dispatcher.Dispatcher` rebuilds per-dispatch because the cost
> is negligible versus handler execution and it allows free middleware
> ordering.
