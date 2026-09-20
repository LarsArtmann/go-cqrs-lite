# ADR-0145: Retry and Backoff Idioms Are Per-Concern, One Per Class

**Date:** 2026-09-20
**Status:** Accepted
**Related:** ADR-0069 (error-wrapping helpers), docs/agents/gotchas-language-footguns.md (Dgraph contention), dogfooding self-review follow-ups 2026-09-19 (items 35-38)

## Context

The dogfooding self-review flagged four retry/backoff implementations —
`middleware/retry`, `projectionhost` worker restarts, `metaengine`'s
replicator, and `metaengine/dgraphengine.retryOnContention` — for a
reconciliation pass: align them onto one idiom, or document why they differ.

The audit found four DIFFERENT concerns, each with tuned semantics:

| Site | Concern | Policy |
| --- | --- | --- |
| `middleware/retry.go` | transport-facing op retry | `go-retry`-powered, caller-configured (`MaxAttempts`, backoff, `IsRetryable`), DLQ on exhaust, OTel span per attempt |
| `projectionhost/worker.go` | crash-loop damping | exponential + FULL jitter on WORKER RESTART (not per-op), 1s→30s defaults, exponent capped at `1<<30` |
| `metaengine/replicator.go` `applyWithRetry` | shadow freshness budget | fixed attempts, LINEAR 50ms×n, per-op timeout, retries all errors; exhaust converts to stale/demotion downstream |
| `dgraphengine/transaction.go` `retryOnContention` | DB transient-contention retry | classified (contention string classes only), 6 attempts exp 15ms→2s + jitter, per-retry metric, in-tx aborts surface to caller |

These differ on every axis that matters: what gets retried (classified vs
unclassified vs restart), the delay shape (configurable vs linear vs
exponential+jitter), the exhaust outcome (DLQ vs stale vs last-error), and
the observability contract (OTel vs metrics vs log lines).

## Decision

**No unification. Each concern keeps its idiom; new code picks by concern
class:**

1. Transport/pipeline op retry → `middleware.NewRetry` (the only `go-retry`
   consumer; adding go-retry to Tier-3+ modules for internal loops would
   spend dep budget on a middleware-shaped API that internal loops do not
   fit).
2. Worker/crash-loop restart damping → the projectionhost exponential +
   full-jitter restart pattern (full jitter is deliberate: restarts are
   correlated, so fixed delays re-synchronize crash loops).
3. Internal freshness/reconciliation loops (replicator-style) → small fixed
   linear budget under a deadline, with exhaust converting to an explicit
   degraded state (stale), never silent infinite retry.
4. Backend transient-contention retry → the dgraph pattern: classify the
   transient class, bound attempts, exp+jitter, surface non-members
   immediately, metric every retry.

**Alignment rules that DO apply** (the actual findings): every loop must be
ctx-aware (`select` on `ctx.Done()`); every exhaust must have an explicit
downstream state; retry classification must never swallow
Rejection/Conflict-class errors (errorfamily); and any new loop that matches
none of the four classes is a design smell to raise, not a fifth idiom.

## Consequences

- The four sites stay as-is; no code change. This ADR is the "document why
  they differ" branch of the reconciliation.
- `go-retry` remains a middleware-only dependency; budgets in
  `metaengine`/`dgraphengine` unchanged.
- Future audits compare against the four-class table instead of re-litigating
  each site from scratch.
