# Consumer Feedback: go-cqrs-lite

**From:** SwettySwipperWeb integration session (2026-07-05)
**Perspective:** AI agent building a production event-sourced CQRS app
**Tone:** Honest, direct, grateful but critical where warranted

---

## What Works Superbly

### 1. The Decider Pattern

The `Decider[State]` + `decider.Repository` pattern is the cleanest event sourcing abstraction I've used:

```go
d := decider.Decider[BattleState]{Initial: initBattleState, Apply: foldBattle}
repo, _ := decider.NewRepository(store, bus, d)
repo.Execute(ctx, aggID, "Battle", decideFunc)
```

Load → fold → decide → save → publish in one call. Pure functions for decide and apply. No side effects in domain logic. This is textbook DDD done right.

### 2. Middleware Module — Complete and Symmetric

The `middleware/v3` module provides symmetric middleware for all three message types:

|         | Recovery | Logging | Retry | CircuitBreaker | Metrics | Validation |
| ------- | -------- | ------- | ----- | -------------- | ------- | ---------- |
| Command | ✅       | ✅      | ✅    | ✅             | ✅      | ✅         |
| Event   | ✅       | ✅      | ✅    | ✅             | ✅      | ✅         |
| Query   | ✅       | ✅      | ✅    | ✅             | ✅      | —          |

Each is a one-liner: `CommandRecovery()`, `QueryLogging(logger)`, etc. The `RetryConfig` with `IsRetryable` that delegates to `event.IsRetryable` → `errorfamily.IsRetryable` is the correct integration point.

**This session:** We added `QueryRecovery()` + `QueryLogging()` to our query dispatcher (which had zero middleware) and `CommandRetry()` to our command dispatcher. Both were 2-line changes. Excellent ergonomics.

### 3. Sink/Source Split (ISP)

The `Store` interface split into `EventSink` (write) and `EventSource` (read) is the correct ISP application. Consumers that only read events take `EventSource`, not the full `Store`. This prevents accidental writes.

### 4. Codec System

The `codec.JSONCodec{}` / `codec.CBORCodec{}` system is clean. `event.NewEvents` accepting `[]any` and encoding internally prevents the #1 pitfall (passing structs where `[]byte` expected). `DecodePayloadAuto[T]` handling mixed streams is a nice touch.

### 5. Branded IDs

`id.Of[Marker]` prevents mixing ID types at compile time. `id.NewAggregateID()` / `id.NewEventID()` / `id.NewCommandID()` are convenient. The deterministic ID pattern (`DeriveAggregateID`) works perfectly for dedup — our vote dedup uses it with a 20-goroutine concurrency test.

---

## What's Confusing or Hard to Discover

### 1. The `eventtest` Module Path Split-Brain

**Problem:** `event/v3/eventtest` is a standalone Go module with its own `go.mod` that was never published to the Go proxy. The path changed at some point from `event/eventtest` to `event/v3/eventtest`. This broke ALL Go tests in our project.

**Impact:** Hours of debugging. The `go.work` file and 5 consuming `go.mod` files needed `require` + `replace` directives with version `v0.0.0` (not `v3.0.0` because the path ends in `/eventtest`, not `/v3`).

**Ask:** Either publish `eventtest` to the Go proxy, or document the exact `require` + `replace` needed in every consuming module. The current state is a recurring footgun.

### 2. `eventtest.NewFakeBus()` Used in Production

**Problem:** We use `eventtest.NewFakeBus()` as our production event bus (wrapped in a factory function). The name suggests it's test-only.

**Impact:** This is a naming problem, not a functional one. The implementation is production-safe for single-process apps — it's a synchronous in-memory bus. But the name makes every code reviewer ask "why is test code in production?"

**Ask:** Either (a) rename to `SyncBus` / `InMemoryBus` in a non-test module, or (b) document clearly that `FakeBus` is production-safe for single-process apps and the name is a historical artifact.

### 3. `RegisterTyped` Type Safety

**Problem:** Every typed query handler needs a manual type assertion:

```go
func(_ context.Context, q cqrsquery.Query) (*BattleState, error) {
    getQ, ok := q.(*battle.GetBattleQuery)
    if !ok {
        return nil, errorfamily.Newf(Corruption, "api.error", "expected GetBattleQuery, got %T", q)
    }
    // ...
}
```

**Impact:** 12 query handlers × 5 lines of boilerplate = 60 lines of defensive code. If `RegisterTyped` is truly typed, the dispatcher should only dispatch the correct type.

**Ask:** Either (a) make `RegisterTyped` generic enough that the handler receives the concrete type directly (no assertion needed), or (b) document whether the assertion is truly necessary (defensive) or just boilerplate.

### 4. `event.WithCommandCausality` — Does Not Exist

**Problem:** The SKILL.md references `event.WithCommandCausality(ctx, cmdType, cmdID)` and `event.CommandCausalityFromContext`. These functions **do not exist** in the codebase.

**Impact:** Consumers following the skill will hit dead ends. The actual mechanism is `WithCorrelationID` / `CorrelationIDFromContext`, which is a different concept (correlation ≠ causation).

**Ask:** Remove the causality reference from the SKILL.md, or implement it. Command-to-event causation linking is valuable for audit trails.

### 5. Projection `On` Function — Free Function, Not Method

**Problem:** `projection.On[T](b, "user.created", codec, handler)` is a free function, not a method on `*Builder`. This breaks the builder chain expectation.

**Impact:** Minor confusion. The SKILL.md documents this correctly (which is great!), but it's a surprising API shape.

**Ask:** Consider adding `b.On[T]("user.created", codec, handler)` as a method for consistency with the builder pattern.

---

## What's Missing

### 1. Query-Side Middleware Documentation

The SKILL.md and recipes focus heavily on command-side middleware. Query middleware (`QueryRecovery`, `QueryLogging`, `QueryMetrics`) exists but isn't prominently documented. We didn't know it existed until we read the source code.

**Ask:** Add query middleware to the recipes section with examples.

### 2. Snapshot Store — Undocumented for Small Apps

The `snapshot/v3` module exists but there's no guidance on WHEN to use it. For our app (aggregates with 5-20 events), snapshots are unnecessary overhead. For larger apps (100+ events per aggregate), they're essential.

**Ask:** Add a decision guide: "Use snapshots when your largest aggregate exceeds N events" with benchmark numbers.

### 3. Prometheus Integration — Heavyweight for Simple Apps

The `prometheus/v3` module requires OTel infrastructure (`MeterProvider`, `otel/v3` imports). For apps that just want Prometheus metrics without OTel, there's no lightweight path.

**Impact:** We use raw `promhttp.Handler()` because the library's Prometheus module requires OTel we don't have.

**Ask:** Provide a lightweight Prometheus metrics middleware that doesn't require OTel. Something like `middleware.CommandPrometheusMetrics(promRegistry)` that directly registers Prometheus collectors.

### 4. Dead-Letter Queue — In-Memory Only

The `DeadLetterHandler` with `MemoryDeadLetterStore` is great for development but doesn't survive restarts. For production, a SQL-backed dead-letter store would be valuable.

**Ask:** The `deadletter_sql.go` file exists but isn't documented in the SKILL.md. Add a recipe for SQL-backed dead-letter storage.

---

## What's Over-Engineered

### 1. Graph Projections

The `graph/v3` module provides graph/traversal read models. This is a powerful abstraction but extremely niche. Most apps need simple flat read models (list, get by ID). The graph module adds conceptual overhead for the 99% case.

### 2. Deriver / Saga Module

The `deriver/v3` module provides reactive command dispatch from events (essentially a saga/process manager). The SKILL.md correctly says "use projection + command dispatch" instead, which is simpler. The deriver module may be over-engineering for most use cases.

---

## Summary Scorecard

| Area                   | Rating | Notes                               |
| ---------------------- | ------ | ----------------------------------- |
| Decider pattern        | ★★★★★  | Best CQRS abstraction I've used     |
| Middleware module      | ★★★★★  | Complete, symmetric, ergonomic      |
| Event store / bus      | ★★★★☆  | Solid, FakeBus naming is misleading |
| Codec system           | ★★★★★  | Clean, prevents common pitfalls     |
| ID system              | ★★★★★  | Branded types prevent mixing        |
| Module path hygiene    | ★★☆☆☆  | eventtest split-brain is painful    |
| Documentation          | ★★★★☆  | Good skill file, some dead refs     |
| Projection system      | ★★★★☆  | Solid, `On` API shape is odd        |
| Prometheus integration | ★★☆☆☆  | Requires OTel, no lightweight path  |

---

_This feedback is given with gratitude for an excellent CQRS library. The critique is offered to make it even better._

---

## Appendix: Session Response (2026-07-05)

> Tracking which feedback items were addressed. See `docs/status/2026-07-05_05-14_consumer-feedback-execution.md`.

### Confusing or Hard to Discover

| #   | Feedback Item                                                   | Status                | What changed                                                                                                                                                                                                                                                    |
| --- | --------------------------------------------------------------- | --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | eventtest module path split-brain                               | ✅ **Documented**     | Added to skill FAQ with consumer-side `replace` directive. Structural fix is a maintainer decision.                                                                                                                                                             |
| 2   | `eventtest.NewFakeBus()` used in production                     | ✅ **Documented**     | Added to skill FAQ: FakeBus IS production-safe for single-process apps. Name is historical. For multi-process: use `watermill.EventBus`.                                                                                                                        |
| 3   | `RegisterTyped` type safety — manual type assertion boilerplate | ❌ **Not started**    | Noted as P2 feature request. Would require API redesign of the dispatcher internals.                                                                                                                                                                            |
| 4   | `event.WithCommandCausality` — does not exist                   | ⚠️ **CLAIM IS FALSE** | All three functions exist at `event/causality.go`: `WithCommandCausality` (line 33), `CommandCausalityFromContext` (line 46), `CommandCausalityEnricher` (line 64). Full test coverage in `event/causality_test.go`. The skill's §3.8 documentation is correct. |
| 5   | Projection `On` function — free function, not method            | ❌ **Not started**    | Noted as P3. Adding `b.On[T]()` as a method for builder consistency is a 2-hour change but needs API review.                                                                                                                                                    |

### What's Missing

| #   | Feedback Item                                        | Status             | What changed                                                                                                                                             |
| --- | ---------------------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Query-side middleware documentation                  | ✅ **SHIPPED**     | Added full query middleware section to `references/recipes.md` §2.8 with symmetric matrix table (Recovery/Logging/Retry/CircuitBreaker/Metrics/Tracing). |
| 2   | Snapshot store — undocumented for small apps         | ✅ **SHIPPED**     | Added to skill FAQ: "Use snapshots when your largest aggregate exceeds ~100 events. Below that, full replay is faster."                                  |
| 3   | Prometheus integration — heavyweight for simple apps | ❌ **Not started** | Noted as P2. A lightweight `middleware.CommandPrometheusMetrics(registry)` without OTel is a valid request.                                              |
| 4   | Dead-letter queue — in-memory only                   | ✅ **Documented**  | Documented the `DeadLetterStore` interface pattern in `references/advanced.md`. SQL implementation is a P2 feature.                                      |
