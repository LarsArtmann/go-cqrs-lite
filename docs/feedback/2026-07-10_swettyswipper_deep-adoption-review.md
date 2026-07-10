# Consumer Feedback: go-cqrs-lite Deep Adoption Review

**From:** SwettySwipperWeb deep adoption session (2026-07-10)
**Perspective:** Production event-sourced media voting platform — 6 aggregates, SQLite persistence, 12 of 24 available modules in active use
**Prior feedback:** [2026-07-05_swettyswipper-consumer-feedback.md](./2026-07-05_swettyswipper-consumer-feedback.md) — see appendix for resolution status
**Tone:** Direct, specific, grateful. Every item includes a concrete suggestion.

---

## What Has Improved Since July 5 Feedback

These items from the prior session are now resolved or improved:

| Prior Item            | Status          | What Changed                                                                                                                           |
| --------------------- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| FakeBus naming        | ✅ **Resolved** | `watermill.NewEventBus()` shipped — we adopted it immediately, replacing `eventtest.FakeBus` in production. Clean API, no memory leak. |
| Query middleware docs | ✅ **Resolved** | Query middleware is now discoverable in recipes. We wired `QueryRecovery` + `QueryLogging` + OTel tracing.                             |
| Snapshot guidance     | ✅ **Resolved** | FAQ threshold guidance exists. We now know our aggregates (5-20 events) don't need snapshots yet.                                      |
| Event causality       | ✅ **Resolved** | Functions exist at `event/causality.go`. Our prior claim was wrong.                                                                    |

These improvements directly enabled the 6 concrete changes we shipped this session.

---

## What Works Superbly (New Discoveries)

### 1. OTelBundle — One Call, Full Pipeline Visibility

The `middleware.NewOTelBundle(tracer, meter)` is the best "batteries included" API in the library. We wired it across all three CQRS paths in a single commit:

```go
bundle, _ := cqrsmiddleware.NewOTelBundle(
    cqrsotel.NewTracer("swetty-swipper"), nil,
    cqrsmiddleware.WithMetricsDisabled(),
)
cmdDispatcher.Use(bundle.Command()...)
queryDispatcher.Use(bundle.Query()...)
bus.Use(bundle.Event()...)
bus.UsePublish(bundle.Publish()...)
```

Every `Dispatch`, `Publish`, and projection `Handle` call now emits a span. The `WithMetricsDisabled()` option is a thoughtful touch — we're tracing-only until we set up a MeterProvider. No partial-configuration errors.

### 2. Circuit Breaker — Sensible Defaults

`DefaultCircuitBreakerConfig()` uses `errorfamily.IsRetryable` as its `IsFailure` predicate. This means domain rejections (`errorfamily.Rejection`) don't trip the breaker — only infrastructure and transient errors do. This is the correct default. We added it with zero configuration.

### 3. Stack/SQLite Preset Design

The `stack.Bundle` composition model is exceptional architecture. The ISP-at-field-level design (each field is a segregated interface like `event.EventSink`, `snapshot.SnapshotSource`) is rare in Go. The `Bundle.Close()` pointer-deduplication and partial-construction rollback are production-grade.

We haven't adopted it yet (our custom XDG path guard and integrity check don't map 1:1), but the design informed our understanding of how the modules compose.

### 4. projectionhost DLQ + Checkpoint Design

The `projectionhost.Host` with dead-letter queue, per-projection goroutines, exponential backoff crash recovery, and replay→live dedup via `dedup.Ring` is the most complete projection lifecycle manager I've seen in any CQRS framework. The `ReplayDeadLetters` method for re-feeding poison messages is particularly thoughtful.

### 5. ADR Trail (49 Decisions)

The 49 ADRs are a goldmine for understanding WHY decisions were made. ADR-0001 (Decider over OO aggregate root), ADR-0006 (Sink/Source ISP split), and ADR-0044 (blind store codec envelopes) are all exemplary. Most libraries have zero architectural documentation — this one has enough to reconstruct the entire design philosophy.

---

## What's Still Confusing or Painful

### 1. `eventtest` Module Is STILL Not Published (HIGH PRIORITY)

**Prior feedback (July 5):** Documented but not structurally fixed.
**Current state:** Still requires manual `require` + `replace` in every consuming `go.mod`. The version must be `v0.0.0` (not `v3.0.0`). This is documented in AGENTS.md gotcha #9, but it still trips up every new module that transitively depends on it.

**Concrete impact this session:** The `pipeline_test.go` in our API service has a pre-existing build failure because `event.AggregateRef` was removed (it's now `id.AggregateRef` per the v4 migration guide), and we can't easily fix it because the test imports `eventtest` which pulls in the version drift.

**Ask:** Publish `eventtest` to the Go proxy. Even a `v0.0.1` would eliminate the `replace` directive friction across 5+ consuming modules. Alternatively, move `FakeBus` and `FakeSnapshotStore` into the main `event/v3` package behind a `_test.go`-safe interface, or into `testutil/v3` which IS published.

### 2. 49 Modules Is Too Many to Navigate (MEDIUM PRIORITY)

**Problem:** The library has 49 separate `go.mod` files. Discovering which module provides a specific capability requires grep-reading source code. The README lists modules in tables, but there's no visual architecture map showing dependencies between modules.

**Concrete example:** I spent 15 minutes figuring out that `projectionhost` is what I need for managed projection lifecycle, while `projection` just defines the `Projection` interface (58 lines). The naming doesn't communicate the relationship — `projectionhost` sounds like a deployment artifact, not a lifecycle manager.

**Ask:**

- Add a "module relationship diagram" (D2 or mermaid) to the README showing which modules depend on which
- Consider a naming convention that communicates the relationship: `projection` (contract) vs `projectionrunner` (implementation) instead of `projectionhost`
- Group modules in the README by capability tier (as the Four-Tier Model already defines)

### 3. Stale `docs/getting-started.md` (MEDIUM PRIORITY)

**Problem:** `docs/getting-started.md` uses `/v2` import paths, references `memory/v3` (which doesn't exist — it's `storage/memory/v3`), and points to dead example directories (`example/todo/`, `example/user/`). A new user following this guide will hit immediate errors.

**The actual getting-started experience is in `SKILL.md`** which is excellent. But `docs/getting-started.md` is the file that shows up in GitHub's file browser first.

**Ask:** Either rewrite `docs/getting-started.md` to match `example/getting-started/` (which works), or delete it and point to `SKILL.md` + `example/`.

### 4. Module Count Is Inconsistent Everywhere (LOW PRIORITY)

**Problem:** Every document reports a different module count:

- README: "42+ modules"
- CONTRIBUTING.md: "48 modules"
- docs/README.md: "28 modules"
- SKILL.md: "49 modules"
- v4-WISHLIST.md: "49 modules"

**Ask:** Pick a source of truth (the `go.work` file or a generated count) and reference it. Or add a CI check that verifies doc module counts.

### 5. `scheduling` and `listing` Ship Memory-Only Stores (LOW PRIORITY)

**Problem:** `scheduling.TimerStore` only ships `MemoryTimerStore` — no SQL implementation. Same for `listing.AggregateReader` (only `InMemoryAggregateReader`). For a library that ships SQLite/Turso/Pebble event stores, the absence of SQL-backed timer and listing stores is a gap.

**Impact:** We can't adopt `scheduling` for voting-session deadlines without implementing our own SQL `TimerStore`. We can't adopt `listing` for paginated media lists without implementing our own SQL `AggregateReader`.

**Ask:** Either ship SQL implementations (even thin wrappers over `*sql.DB`), or document the expected implementation pattern with a reference example.

### 6. ADR Index Is Stale (LOW PRIORITY)

**Problem:** `docs/adr/README.md` only indexes through ADR-0032. `docs/README.md` goes to 0046. Actual ADRs go to 0049. The most important recent decisions (0033-0049, including the codec envelope ADR-0044 and the metadata extraction ADR-0046) are not in the index.

**Ask:** Regenerate the ADR index. A simple script that scans `docs/adr/` for `0001-*.md` through `NNNN-*.md` and builds the index table would prevent drift.

---

## What Would Make Adoption Easier

### 1. A "Migration Playbook" for Common Upgrades

We needed to go from "FakeBus, no middleware, no tracing" to "watermill bus, full middleware, OTel tracing, circuit breaker" in one session. Each step required reading source to understand the API. A migration playbook with copy-paste recipes would have saved 30+ minutes:

```
Step 1: Replace FakeBus → watermill.NewEventBus()
Step 2: Add bus.Use(EventRecovery(), EventLogging(...))
Step 3: Create OTelBundle and wire on all three paths
Step 4: Add CommandCircuitBreaker
```

The SKILL.md has recipes, but they're per-module, not sequenced as an adoption path.

### 2. Middleware Ordering Guide

The library ships ~30 middlewares across command/query/event. There's no guide on what ORDER to apply them. Through trial and error we arrived at:

```
Command: Idempotency → Recovery → Retry → CircuitBreaker → [OTel] → Logging
Event:   Recovery → Logging → [OTel]
Query:   Recovery → Logging → [OTel]
```

But we're not sure this is optimal. Should tracing wrap everything (outermost)? Should logging be innermost (closest to handler)?

**Ask:** Document recommended middleware ordering with rationale. The ADR on dispatch-time middleware (ADR-0049) explains the mechanism but not the recommended order.

### 3. "When to Use What" Decision Guide for Modules

49 modules is a lot. A decision flowchart would help:

```
Do you need projections?
  → Yes: Use projection (interface) + projectionhost (lifecycle) + readmodel (your impl)
  → No: Skip all three

Do you need snapshots?
  → Your largest aggregate has >100 events? → Yes: snapshot.TypedStore + EveryNEvents(100)
  → No: Skip

Do you need pagination?
  → listing.ListBuilder + AggregateReader
  → But no SQL reader shipped, so implement your own
```

### 4. Document the `stack/sqlite` Preset's Extension Points

The preset is almost exactly what we need, but we have custom requirements:

- XDG path resolution (never put DB inside git repo — we learned this the hard way)
- PRAGMA integrity_check on startup
- Custom app schema initialization (non-CQRS tables)

The preset's `New(dsn, opts...)` doesn't expose hooks for these. We'd need to use the `stack.Bundle` directly and wire individual pieces ourselves.

**Ask:** Either document how to extend the preset with custom DB initialization hooks, or provide the preset as a "reference implementation" that consumers copy-paste-modify rather than call as a black box.

---

## What's Over-Engineered or Confusing

### 1. `graph` Module — Premature for Most Consumers

The `graph/v3` module provides graph/traversal read models. In 6 months of building a production CQRS app, we've never needed graph queries. Flat projections (list by type, get by ID, filter by predicate) cover 100% of our read-model needs. The graph module adds conceptual tax without clear payoff.

**Suggestion:** Keep it (someone might need it), but don't feature it prominently in documentation. Most consumers should start with flat projections.

### 2. Two Competing Dependency Layer Models

`CONTRIBUTING.md` has a 7-layer model. `docs/architecture-understanding/FOUR-TIER-MODEL.md` has a 4-tier model. They use different layer names and different groupings. A new contributor has to choose which model to follow.

**Suggestion:** Pick one. The Four-Tier Model is more recent and more honest (it acknowledges the old model was "fake"). Update CONTRIBUTING.md to reference it.

### 3. `docs/planning/` and `docs/status/` Are Overwhelming

There are ~500+ timestamped files across `docs/planning/` and `docs/status/`. These are session artifacts, not user documentation. They make the docs directory nearly unnavigable — a new contributor opening `docs/` sees hundreds of timestamped files with no indication of which are current.

**Suggestion:** Move planning/status/session artifacts to `docs/archive/` or a separate `sessions/` directory at the repo root. Keep `docs/` for enduring documentation only.

---

## Summary Scorecard (Updated from July 5)

| Area                   | July 5 | July 10 | Delta | Notes                                       |
| ---------------------- | ------ | ------- | ----- | ------------------------------------------- |
| Decider pattern        | ★★★★★  | ★★★★★   | —     | Still the best CQRS abstraction             |
| Middleware module      | ★★★★★  | ★★★★★   | —     | OTelBundle + CircuitBreaker are excellent   |
| Event store / bus      | ★★★★☆  | ★★★★★   | +1    | watermill.NewEventBus() resolved FakeBus    |
| Codec system           | ★★★★★  | ★★★★★   | —     | Clean, prevents pitfalls                    |
| ID system              | ★★★★★  | ★★★★★   | —     | Branded types prevent mixing                |
| OTel integration       | —      | ★★★★★   | NEW   | OTelBundle is exemplary                     |
| projectionhost         | —      | ★★★★★   | NEW   | Best projection lifecycle manager available |
| Module path hygiene    | ★★☆☆☆  | ★★½☆☆   | +0.5  | eventtest still not published               |
| Documentation          | ★★★★☆  | ★★★½☆   | -0.5  | getting-started.md still stale              |
| Module discoverability | —      | ★★★☆☆   | NEW   | 49 modules, no relationship map             |
| Stack presets          | —      | ★★★★☆   | NEW   | Great design, needs extension hooks         |

---

## Appendix: Prior Feedback Resolution Tracking

| #   | July 5 Item                       | July 10 Status     | Notes                                                             |
| --- | --------------------------------- | ------------------ | ----------------------------------------------------------------- |
| C1  | eventtest module path split-brain | ⚠️ **Still open**  | Documented but structurally unfixed. Biggest consumer pain point. |
| C2  | FakeBus naming                    | ✅ **Resolved**    | watermill.NewEventBus() shipped and adopted.                      |
| C3  | RegisterTyped type safety         | ❌ **Not started** | Still 60 lines of boilerplate across 12 handlers.                 |
| C4  | event.WithCommandCausality        | ✅ **Resolved**    | Our claim was false — functions exist.                            |
| C5  | Projection On as free function    | ❌ **Not started** | Low priority, minor ergonomics.                                   |
| M1  | Query middleware docs             | ✅ **Resolved**    | Now in recipes.                                                   |
| M2  | Snapshot guidance                 | ✅ **Resolved**    | FAQ threshold exists.                                             |
| M3  | Prometheus without OTel           | ❌ **Not started** | Still requires OTel infrastructure.                               |
| M4  | Dead-letter SQL store docs        | ✅ **Documented**  | Pattern documented, SQL impl still P2.                            |

---

_This feedback reflects 6 hours of deep adoption work — replacing the event bus, wiring middleware across the entire CQRS pipeline, fixing type-safety bugs in rating algorithms, and evaluating 8 unadopted modules. It is offered with genuine appreciation for the best Go CQRS library available, and the conviction that these improvements would make it even better._

_— SwettySwipperWeb team, July 2026_
