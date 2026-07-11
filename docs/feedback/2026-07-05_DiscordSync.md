# go-cqrs-lite — Consumer Feedback (DiscordSync)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** v3.5.0 (14 modules)
**Usage depth:** Heavy — event capture, projections, projectionhost, CBOR codec, watermill bus, middleware (tracing/recovery/logging/metrics), OTel, Prometheus bridge, storage (SQLite + Turso), catalog docs, scenario tests, testutil, id generation
**Date:** 2026-07-05

---

## What Works Superbly

### 1. Event capture without decider/command — the "capture-only" pattern

DiscordSync captures external Discord Gateway events. We don't issue commands or decide aggregate state. The library's `EventCapture.Emit()` + `EventCapture.Capture()` APIs let us write events to the store + bus WITHOUT needing the decider/command/query stack. This is the library's killer feature for event-sourcing-adjacent use cases.

```go
ec.Emit(ctx, eventType, aggregateRef, version, payload, correlationID)
```

One call. Event stored. Event published. Done.

### 2. `projectionhost.Host` is production-grade

Per-projection checkpoints, exponential-backoff crash recovery with jitter, SQLite-backed dead-letter queue, `WithSubscriber` for replay→live transition with event-ID dedup, `WithBatchSize(500)` for faster backfill replay. This replaced a 297-line custom Runner in our codebase. It just works.

### 3. `DecodePayloadAuto[T]` is the right codec API

Encoding-agnostic payload decoding. Works for mixed JSON+CBOR streams during migration. Type-safe via generics. No type assertions. No runtime panics on wrong types. This is how event payload decoding should work.

```go
payload, err := event.DecodePayloadAuto[MessageCreatedPayload](evt)
```

### 4. CBOR as default codec (19% smaller, 66% faster decode)

The `codec.CBORCodec{}` + `event.DefaultCodec` pattern is clean. We set it once in init, and all new events are CBOR-encoded. Historical JSON events decode transparently via `DecodePayloadAuto`. The migration is invisible.

### 5. Middleware chain: tracing → recovery → logging → metrics

```go
bus.Use(
    cqrsmiddleware.EventTracing(tracer),
    cqrsmiddleware.EventRecovery(),
    cqrsmiddleware.EventLogging(logger),
    cqrsmiddleware.EventOTelMetricsWithCounter(meter),
)
```

Four lines, complete observability stack. Ordering matters (tracing first so the span covers recovery + logging + metrics) and is documented.

### 6. Prometheus bridge: one `/metrics` endpoint for everything

`prometheus.Setup(WithRegistry(prometheus.DefaultRegisterer))` sets the global meter provider. Both hand-rolled Prometheus counters AND OTel-backed instruments serve from one registry. No duplicate metrics, no port conflicts.

### 7. `SeekableJournal.ReadFrom(afterID, limit)` — efficient SSE replay

The SSE reconnection replay uses this for cursor-based replay from the last event ID. No full journal scan. The `JournalSSEStore` in cqrs-htmx wraps this perfectly.

### 8. `scenario/v4` BDD DSL for projection testing

`GivenProjection`/`ThenNoError` reads like a spec document. We use it in `scenario_poc_test.go`.

### 9. `testutil/v4` Rapid property-based test generators

`EventType`, `AggregateType`, `Version`, `MetadataMap` generators for property-based testing with `pgregory.net/rapid`. Our idempotency test runs 100 random events through the projection and asserts byte-exact SQL equality.

---

## What's Painful

### 1. The `event/v4/eventtest` tagless nested module breaks local `go mod tidy`

`go-cqrs-lite/event/v4` test files import `event/v4/eventtest` (`event/eventtest/go.mod`), which has no published version. **Local `go mod tidy` fails on the host.** The authoritative tidy runs inside the Nix sandbox where `mkPreparedSource` auto-discovers all nested go.mod files.

This is documented in our AGENTS.md but it's a real pain point — every developer who clones go-cqrs-lite and runs `go mod tidy` hits this.

**Suggestion:** Either publish `eventtest` as a tagged module, or make it a non-module (move test helpers into `_test.go` files that share the package).

### 2. 28 modules is a lot of cognitive overhead

The dependency layering (Layer 0–6) is well-documented, but discovering which module contains which type/API requires the skill file's decision matrix. Consumers import from 14 separate module paths.

**Impact:** `go.mod` has 16 direct go-cqrs-lite dependencies. Version drift is a constant risk (we have `dispatcher/v4` at v3.3.0 while everything else is v3.5.0).

**Suggestion:** Consider a "bundle" meta-module for the common case (event + storage + projection + projectionhost + watermill + codec + middleware + otel + prometheus). One import, one version.

### 3. `projectionhost.Host` API is slightly opaque

The `New(...)` builder takes variadic options. The `WithSubscriber` option is critical for replay→live transition but its interaction with `WithBatchSize` and checkpoint behavior isn't obvious. We had to read the source to understand the replay flow.

**Suggestion:** Document the projection lifecycle: "1. Host starts → 2. Reads events from journal in batches → 3. For each event, calls projection.Handle → 4. Updates checkpoint → 5. When caught up, transitions to live (subscriber mode) → 6. On error, retries with exponential backoff → 7. Poison messages go to DLQ after exhausting restart budget."

### 4. No built-in idempotency for projection handlers

Our projection handlers are idempotent by design (INSERT OR REPLACE, INSERT OR IGNORE), but the library provides no idempotency primitive at the projection level. If the same event is replayed (e.g., after a crash), the handler runs again.

**Impact:** We wrote our own idempotency tests (`property_test.go`, 100 rapid iterations) to verify replaying yields identical SQL. This works but it's consumer-implemented.

**Suggestion:** Consider adding `idempotency/v4` integration at the projectionhost level — track processed event IDs per projection, skip already-processed events on replay. (The `WithSubscriber` dedup handles the replay→live boundary, but not cross-restart replay idempotency.)

### 5. The `storage` package restructure (v3.5.0) into `view/relational/` subdirs

v3.5.0 restructured the storage package with backward-compatible aliases, but the new directory structure is confusing for consumers who remember the old layout. The aliases work but add cognitive overhead when reading stack traces.

**Suggestion:** Document the old → new path mapping prominently in the migration guide.

---

## What's Missing

### 1. ~~No projection parallelism~~ (RESOLVED — was always the default)

> **Update (2026-07-10):** This was a misunderstanding. `projectionhost.Host` **already runs each projection in its own goroutine** with its own independent checkpoint (`host.go:147`, one `go worker.run()` per registered projection, 10ms staggered launch). A slow message projection does **not** block the reaction projection — they are fully independent readers with separate cursors. No `WithParallelProjections()` option is needed; parallelism is the default and only behavior.

### 2. No projection lag metric built-in

We calculate projection lag ourselves: `eventCapture.LastCaptureAt - projHost.LastProcessedAt()`. The library exposes both timestamps but doesn't provide a pre-built gauge.

**Suggestion:** Add `projectionhost.Host.LagDuration() time.Duration` that returns the delta. Consumers register it as a Prometheus gauge.

### 3. No guidance on "shared database" architecture

DiscordSync uses ONE SQLite database for CQRS events AND relational reads. The `stack/*` presets create separate `*sql.DB` connections, which is incompatible. We have an ADR (`docs/adr/ADR-go-cqrs-lite-v3-leverage.md`) documenting why we don't use stacks.

**Suggestion:** Add a "shared database" recipe to the skill: "If your events and read model share a database, don't use stack presets. Construct EventStore and read-model DB separately, passing the same `*sql.DB`."

### 4. No `event.CaptureFromGateway` convenience for external event sources

DiscordSync captures Discord Gateway events. The pattern is: receive WebSocket event → map to typed payload → `EventCapture.Emit()`. This is 100% consumer code, which is correct, but a `CaptureFromExternal` helper that standardizes the metadata enrichment (correlation ID, causation ID, timestamp) would reduce boilerplate.

---

## What Deliberately Excluded (And Why)

| Module                | Why                                                            |
| --------------------- | -------------------------------------------------------------- |
| `command/v4`          | DiscordSync captures external events, doesn't issue commands   |
| `decider/v4`          | No aggregate decision logic. Events are captured, not decided. |
| `snapshot/v4`         | No aggregate state to snapshot. Projections ARE the state.     |
| `listing/v4`          | SQL read model handles listing with cursors                    |
| `query/v4`            | The 49-method Database interface IS the query layer            |
| `kv/v4`               | No key-value storage need. SQLite handles everything.          |
| `stack/*` presets     | Shared DB architecture — presets create separate `*sql.DB`     |
| `schema/v4` Validator | Go typed structs ARE the schema validation                     |
| `signing/v4`          | No security requirement for a personal backup                  |
| `encryption/v4`       | Same                                                           |
| `transport/grpc`      | No gRPC transport need                                         |

---

## Skill Feedback

The skill file (`/.agents/skills/go-cqrs-lite/SKILL.md`) is **the best skill file I've read**. Specific feedback:

### Good

- The module decision matrix ("I want to…") is the single best entry point
- Critical conventions (tombstone, sink/source split, decode with codec) catch real bugs
- The anti-patterns table is excellent
- Dependency layering (Layer 0–6) helps understand module relationships
- The FAQ/pitfalls section saved us hours

### Could Improve

- **No "shared database" guidance** — the skill assumes stacks/separate DBs. Add a recipe for events + reads in one DB.
- **The `eventtest` gotcha isn't mentioned** — consumers will hit `go mod tidy` failures and not know why
- **`projectionhost` lifecycle isn't documented** — the replay → live transition, checkpoint behavior, and DLQ flow need a sequence diagram
- **No guidance on projection idempotency** — consumers must implement and test this themselves
- **The `scenario/v4` BDD DSL needs more examples** — the skill mentions it but doesn't show a complete Given/When/Then

---

## Summary Scorecard

| Dimension             | Score      | Notes                                                                                  |
| --------------------- | ---------- | -------------------------------------------------------------------------------------- |
| API design            | 9/10       | Clean interfaces, right abstractions, ISP-respecting splits                            |
| Documentation (skill) | 9/10       | Best skill file; minor gaps in shared-DB + projectionhost lifecycle                    |
| Ease of adoption      | 7/10       | 28 modules is high cognitive overhead; eventtest breaks local tidy                     |
| Projection system     | 9/10       | projectionhost is production-grade; parallel by default (one goroutine per projection) |
| Codec support         | 10/10      | CBOR + JSON with auto-detection is exemplary                                           |
| Observability         | 9/10       | OTel + Prometheus bridge is excellent; needs built-in lag metric                       |
| Overall               | **8.5/10** | Best Go CQRS/ES library; modular architecture is a double-edged sword                  |

---

## Appendix: Session Response (2026-07-05)

> Tracking which feedback items were addressed. See `docs/status/2026-07-05_05-14_consumer-feedback-execution.md`.

### What's Painful

| #   | Feedback Item                                        | Status            | What changed                                                                                                                                                                     |
| --- | ---------------------------------------------------- | ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | eventtest tagless nested module breaks `go mod tidy` | ✅ **Documented** | Added to skill FAQ with exact `replace` directive fix. Structural fix (tag/restructure) is a maintainer decision — see open question in status report.                           |
| 2   | 28 modules cognitive overhead                        | ✅ **Documented** | No structural change (inherent to design). Bundle meta-module idea noted as P3.                                                                                                  |
| 3   | projectionhost API opaque                            | ✅ **Documented** | Added full lifecycle sequence (replay → checkpoint → live → DLQ) to `references/advanced.md`.                                                                                    |
| 4   | No built-in idempotency for projection handlers      | ✅ **Documented** | Added projection idempotency patterns (INSERT OR REPLACE / IGNORE / content-hash) to `references/advanced.md`. Cross-restart replay idempotency at framework level: not started. |
| 5   | Storage restructure (v3.5.0) path confusion          | ✅ **Documented** | Added old → new path migration table to skill FAQ.                                                                                                                               |

### What's Missing

| #   | Feedback Item                                 | Status                     | What changed                                                                                                                                        |
| --- | --------------------------------------------- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | ~~No projection parallelism~~                 | ✅ **Already the default** | Projections always ran in parallel — one goroutine each with independent checkpoints. Feedback was based on a misunderstanding of the architecture. |
| 2   | No projection lag metric built-in             | ✅ **SHIPPED**             | `projectionhost.Host.LagDuration() time.Duration` added. Register as Prometheus gauge directly.                                                     |
| 3   | No guidance on "shared database" architecture | ✅ **SHIPPED**             | Added shared-DB recipe to `references/recipes.md` §2.0 — manual wiring with one `*sql.DB`.                                                          |
| 4   | No `event.CaptureFromGateway` convenience     | ❌ **Not started**         | Noted as P2 feature request.                                                                                                                        |

### Skill Feedback

| Item                                    | Status                                  |
| --------------------------------------- | --------------------------------------- |
| No "shared database" guidance           | ✅ Added recipe                         |
| eventtest gotcha not mentioned          | ✅ Added to FAQ                         |
| projectionhost lifecycle not documented | ✅ Added lifecycle + integration recipe |
| No guidance on projection idempotency   | ✅ Added patterns                       |
| scenario/v4 needs more examples         | ✅ Added `GivenState` example           |
