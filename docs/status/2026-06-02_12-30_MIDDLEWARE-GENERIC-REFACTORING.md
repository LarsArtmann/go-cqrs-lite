# Middleware Generic Refactoring — Session 8

**Date**: 2026-06-02 12:30
**Branch**: master
**Commits**: `c9e2986d`, `55060978`

---

## Executive Summary

Collapsed 27 duplicated middleware implementations into 9 generic concern functions backed by `Handler[M]`/`Middleware[M]`/`MessageAdapter[M]`. All 27 backward-compatible typed wrappers preserved. **0 breaking changes.** 80+ tests pass. 0 lint issues.

---

## a) FULLY DONE

| Item | Status | Details |
|------|--------|---------|
| Generic `Handler[M]` / `Middleware[M]` types | ✅ | `generic.go` — core abstraction |
| `MessageAdapter[M]` with pre-built adapters | ✅ | `CommandAdapter`, `EventAdapter`, `QueryAdapter` |
| `AsCommand` / `AsEvent` / `AsQuery` adapters | ✅ | Type-safe conversion from generic → domain middleware |
| `NewRecovery[M]` | ✅ | Replaces 3 functions |
| `NewLogging[M]` | ✅ | Replaces 3 functions |
| `NewRetry[M]` | ✅ | Replaces 3 functions |
| `NewValidation[M]` | ✅ | Replaces 3 functions |
| `NewMetrics[M]` | ✅ | Replaces 3 functions |
| `NewOTelMetrics[M]` | ✅ | Replaces 3 functions |
| `NewTracing[M]` | ✅ | Replaces 3 functions |
| `NewTraceLogging[M]` | ✅ | Replaces 3 functions |
| `NewCircuitBreaker[M]` | ✅ | Replaces 3 functions |
| Deleted `common.go` | ✅ | Unused `*ErrMiddleware` functions removed |
| Backward compatibility | ✅ | All 27 typed wrappers preserved as thin delegates |
| Query result propagation tests | ✅ | 6 tests covering all middleware variants + stacked chains |
| Lint: 0 issues | ✅ | `gochecknoglobals`, `exhaustruct`, `goconst`, `varnamelen` all resolved |
| Example builds | ✅ | `example/user/` compiles and works |
| Integration tests pass | ✅ | All 5 integration suites green |

## b) PARTIALLY DONE

| Item | Status | What remains |
|------|--------|-------------|
| Validator type consolidation | Partial | `CommandValidator`, `EventValidator`, `QueryValidator` are 3 identical `func(X) error` types. Could be `func(M) error` with the generic pattern, but the public API uses them as separate types. |

## c) NOT STARTED

See section e) — Top 25 improvements below.

## d) TOTALLY FUCKED UP

| Item | What happened | Resolution |
|------|--------------|------------|
| AsQuery evaluation order scare | Agent analysis claimed `return result, h(ctx, q)` had a bug where `result` was read before `h` set it. Empirically tested: **Go evaluates function calls in return expressions before assigning results**, so the original code was correct. | Refactored to explicit `err := wrapped(ctx, q); return result, err` anyway for clarity. Not a bug, but improvement. |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Type Model

1. **Unify `Type()` across command/event/query** — All three have `type Type string` with `Type() Type`. A shared `Message` interface with `Type() string` (or generic `Type() T`) would eliminate the need for `MessageAdapter.ExtractType` entirely. Middleware could use a `Message` constraint directly.

2. **`MessageAdapter` is a runtime struct, not a compile-time guarantee** — If a user passes the wrong adapter for the wrong message type, it fails at runtime. A `Message` interface constraint would make this compile-time safe.

3. **Validator types are redundant** — `CommandValidator`, `EventValidator`, `QueryValidator` are all `func(X) error`. With generics, `func(M) error` suffices. The backward-compat wrappers already adapt them.

4. **`AsQuery` allocates per call** — The `errOnly` closure is created on every handler invocation, unlike `AsCommand`/`AsEvent` which apply middleware once at setup. This is a minor perf concern but worth noting.

5. **Middleware doc comments** — The generic `New*` functions have good docs, but the package-level doc comment (`middleware.go:1`) is just "provides cross-cutting concerns". Should mention the generic pattern and link to `MessageAdapter`.

### Code Quality

6. **Test coverage gap for query results** — The existing query tests (`retry_query_test.go`, etc.) checked errors but NOT result values. The new `query_result_test.go` fixes this, but older tests should also verify results.

7. **`infertypeargs` hints** — LSP flags unnecessary explicit type args in `NewOTelMetrics[command.Command](...)`, `NewTraceLogging[command.Command](...)`. These aid readability but could be removed.

8. **Benchmark coverage** — Only command benchmarks exist (`benchmark_test.go`). Should add event and query benchmarks to verify no perf regression from generics.

9. **`logWithContext` still uses `id.AggregateID` directly** — Logging passes an empty `id.AggregateID` for queries. The zero-value AggregateID's `String()` might produce unexpected output.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **impact × effort** (1 = highest priority):

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Add `Message` interface to `command`/`event`/`query` with `Type() string` — enables compile-time middleware safety | HIGH | MED | Architecture |
| 2 | Write status report to `AGENTS.md` about generic middleware pattern for future sessions | HIGH | LOW | Docs |
| 3 | Add event + query benchmark tests | MED | LOW | Testing |
| 4 | Verify existing query tests check result values, not just errors | MED | LOW | Testing |
| 5 | Remove `CommandValidator`/`EventValidator`/`QueryValidator` — use generic `func(M) error` in public API | MED | LOW | Cleanup |
| 6 | Update package doc comment in `middleware.go` to describe generic pattern | MED | LOW | Docs |
| 7 | Consider `middleware` example in `example/` showing both old API and new generic API | MED | MED | Docs |
| 8 | Investigate `AsQuery` per-call allocation — can we restructure to apply once? | MED | MED | Perf |
| 9 | Remove explicit type args where LSP suggests (infertypeargs) | LOW | LOW | Cleanup |
| 10 | Consider shared `Type` type across command/event/query (DRY the `type Type string` definitions) | HIGH | HIGH | Architecture |
| 11 | Add `NewPublishTracing` generic for event publish middleware (currently special-cased) | LOW | LOW | Feature |
| 12 | Add `NewPublishMetrics` / `NewPublishLogging` for event publish path | LOW | MED | Feature |
| 13 | Rate limiter middleware (generic) | MED | MED | Feature |
| 14 | Timeout middleware (generic) — `context.WithTimeout` per handler | MED | LOW | Feature |
| 15 | Deduplication middleware (idempotency key check) | MED | HIGH | Feature |
| 16 | Bulkhead middleware (concurrency limiter per type) | MED | MED | Feature |
| 17 | Health check middleware | LOW | LOW | Feature |
| 18 | Propagate context through `logWithContext` — currently uses `logger.Info` not `logger.InfoContext` | MED | LOW | Bug |
| 19 | Consider structured logging fields standardization (consistent key names across all middleware) | LOW | MED | Consistency |
| 20 | Add `middleware.Handler[M]` to `AGENTS.md` Key Patterns section | MED | LOW | Docs |
| 21 | Consider `Middleware[M]` composition helper — `Chain[M](...Middleware[M]) Middleware[M]` | MED | LOW | Feature |
| 22 | Explore if Go 1.26+ type inference improvements make `AsCommand`/`AsEvent` unnecessary | LOW | LOW | Research |
| 23 | API stability test for middleware — verify exported symbols haven't changed | MED | LOW | CI |
| 24 | Consider moving `OTelMetricsRecorder` to `otel/` module (separation of concerns) | LOW | MED | Architecture |
| 25 | Add circuit breaker state change hooks/callbacks for observability | LOW | MED | Feature |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we introduce a shared `Message` interface across command/event/query?**

All three packages define `type Type string` and `Type() Type` independently. If we add a shared `Message` interface (e.g., in a new `internal/types` or top-level package), the middleware `MessageAdapter` becomes unnecessary — the generic functions could use `Message` as a type constraint directly. But this requires:

1. A new shared package (or putting it in an existing leaf like `id/`)
2. All three `Type()` methods returning the shared type instead of their own named type
3. OR: A `String() string` method on each Type (already exists on command.Type/event.Type/query.Type)

The alternative is simpler: since all three `Type` types are `string` underneath, we could just have the `MessageAdapter` use an interface constraint like:

```go
type Message interface {
    TypeString() string  // new method, returns string directly
}
```

But adding `TypeString()` to command.Command/event.Event/query.Query is also a change. Is it worth it for the middleware optimization, or is the current `MessageAdapter` pattern good enough?

This is a design decision that affects the entire library's type model and needs your input.

---

## Stats

| Metric | Value |
|--------|-------|
| Files changed | 11 (9 modified, 1 new, 1 deleted) |
| Net lines | -398 (498 removed, 100 added + 91 generic.go + 133 test + 8 fix) |
| Generic functions | 9 |
| Backward-compat wrappers | 27 |
| Tests | 80+ pass, 0 fail |
| Lint issues | 0 |
| Breaking changes | 0 |
