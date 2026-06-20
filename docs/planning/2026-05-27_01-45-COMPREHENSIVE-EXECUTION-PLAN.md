# Comprehensive Execution Plan: All Remaining TODOs

**Date:** 2026-05-27 01:45  
**Source:** TODO_LIST.md + Session 108 status report gaps  
**Total tasks:** 142 (all ≤ 12 minutes)  
**Philosophy:** Pareto-sorted. Do 1% → 51% first, then 4% → 64%, then 20% → 80%.

---

## Pareto Summary

| Tier       | Impact      | Task Count | Cumulative | Theme                                                   |
| ---------- | ----------- | ---------- | ---------- | ------------------------------------------------------- |
| 1% → 51%   | 🔴 Critical | 18         | 18         | Unblock consumers, fix correctness, close coverage gaps |
| 4% → 64%   | 🟡 High     | 32         | 50         | Quality gates, CI, integration tests, examples          |
| 20% → 80%  | 🟢 Medium   | 48         | 98         | Docs, refactoring, test splitting, small features       |
| 80% → 100% | ⚪ Low      | 44         | 142        | Research, strategic, nice-to-haves                      |

---

## Tier 1: 1% → 51% Impact (🔴 Critical — Do First)

These 18 tasks deliver 51% of total consumer value. Each is small (≤ 12 min) and high-impact.

| #   | Task                                                          | Module      | Est. | Impact      | Why                                                                |
| --- | ------------------------------------------------------------- | ----------- | ---- | ----------- | ------------------------------------------------------------------ |
| 1   | Add `SagaStore()` accessor to `SQLBackend`                    | storage     | 5m   | 🔴 Critical | Completes the unified backend — one constructor for all SQL stores |
| 2   | Test `Runner.hydrate` error path (unregistered saga type)     | saga        | 8m   | 🔴 Critical | Production restart scenario; 77.8% → 100% coverage                 |
| 3   | Test `NewSQLSagaStoreWithDialect` (0% coverage)               | storage     | 5m   | 🔴 Critical | Public API surface untested                                        |
| 4   | Test `NewSQLBackendWithDialect` (0% coverage)                 | storage     | 5m   | 🔴 Critical | Public API surface untested                                        |
| 5   | Test `scanStates` time parse error paths                      | storage     | 10m  | 🔴 Critical | Defensive code undertested (76.2%)                                 |
| 6   | Test `newSQLBackendWithDialect` middle error paths            | storage     | 10m  | 🔴 Critical | Error paths at 80%, should be 100%                                 |
| 7   | Push release tags to remote (8 tags local only)               | CI/git      | 10m  | 🔴 Critical | #1 blocker for external `go get` adoption                          |
| 8   | Remove replace directives from go.mod files                   | all         | 12m  | 🔴 Critical | Blocks external consumers; chicken-and-egg with tags               |
| 9   | Add `GOWORK=off` CI matrix job                                | CI          | 10m  | 🔴 Critical | Version drift goes undetected; per-module isolation broken         |
| 10  | Fix `query.Handler` returns `any` → generic `TypedHandler[T]` | core/query  | 12m  | 🔴 Critical | Breaking change; most-requested type safety improvement            |
| 11  | Add minimum coverage gate (80%) to CI                         | CI          | 8m   | 🔴 Critical | Prevents coverage regression                                       |
| 12  | Fix `core→memory` circular dependency                         | core        | 12m  | 🔴 Critical | Blocks publishing `core` independently                             |
| 13  | Add PostgreSQL integration tests (testcontainers)             | storage     | 12m  | 🔴 Critical | Only SQLite tested; PG is the primary target                       |
| 14  | Add full outbox cycle integration test                        | storage     | 12m  | 🔴 Critical | Append → PollPending → Publish → Ack never tested end-to-end       |
| 15  | Add `EventRetry` middleware tests (currently 0% coverage)     | middleware  | 10m  | 🔴 Critical | Public middleware with zero tests                                  |
| 16  | Fix `FuzzParse` case-sensitivity roundtrip mismatch           | core/pkg/id | 10m  | 🔴 Critical | ULID parsing correctness issue                                     |
| 17  | Add context cancellation to `SQLOutbox`                       | storage     | 10m  | 🔴 Critical | Currently ignores `ctx.Err()`                                      |
| 18  | Bump `testhelpers` to v1.2.0 (published v1.1.0 incompatible)  | testhelpers | 8m   | 🔴 Critical | Published version broken for consumers                             |

**Tier 1 cumulative time:** ~2.5 hours  
**Tier 1 consumer value:** Unblocks external adoption, closes all coverage gaps in new code, fixes correctness issues.

---

## Tier 2: 4% → 64% Impact (🟡 High)

These 32 tasks add the next 13% of value. Quality gates, CI hardening, examples, and integration.

| #   | Task                                                                           | Module       | Est. | Impact  | Why                                              |
| --- | ------------------------------------------------------------------------------ | ------------ | ---- | ------- | ------------------------------------------------ |
| 19  | Add saga example to `example/saga/`                                            | example      | 20m  | 🟡 High | Zero runnable reference code for consumers       |
| 20  | Split `saga/saga_test.go` (1132 lines) into per-concern files                  | saga         | 15m  | 🟡 High | Pre-commit hook warning; file size limit         |
| 21  | Add `SagaStore` to `NewTursoBackend` / `NewTursoSagaStore`                     | storage      | 8m   | 🟡 High | Turso parity with other stores                   |
| 22  | Storage coverage recovery: error path tests to reach 90%+                      | storage      | 12m  | 🟡 High | Current 88.9%, target 90%+                       |
| 23  | Add `slog.Warn` for corrupt IDs in Pebble deserialization                      | storage      | 8m   | 🟡 High | Silent corruption detection                      |
| 24  | Fix `storage/dialect.go` using `any` — violates "no any" rule                  | storage      | 12m  | 🟡 High | Project rule violation; 3 methods affected       |
| 25  | Optimize `PebbleEventStore.LoadToTimestamp` — avoid full scan                  | storage      | 12m  | 🟡 High | Performance cliff at scale                       |
| 26  | Fix `filterEvents` O(n) scan in `projection/runner.go`                         | projection   | 12m  | 🟡 High | Performance cliff at scale                       |
| 27  | Extend lint to all 9 production modules                                        | CI           | 12m  | 🟡 High | Only core/ linted currently                      |
| 28  | Add `-race` to CI / test commands                                              | CI           | 5m   | 🟡 High | Race conditions go undetected                    |
| 29  | Add coverage tracking to CI workflow                                           | CI           | 8m   | 🟡 High | Visibility into per-PR coverage delta            |
| 30  | Normalize go.mod version references across workspace                           | all          | 10m  | 🟡 High | v0.0.0 vs v1.1.0 vs pseudo-versions inconsistent |
| 31  | Standardize integration/go.mod + catalog/go.mod + example/user/go.mod versions | all          | 10m  | 🟡 High | Version drift across modules                     |
| 32  | Add `go.work sync` CI check                                                    | CI           | 8m   | 🟡 High | Catches replace directive rot                    |
| 33  | Add `WithLogger` to all middleware constructors for consistency                | middleware   | 10m  | 🟡 High | Some middleware lacks logger option              |
| 34  | Extract deduplication: 3 retry + 3 tracing functions with identical structure  | middleware   | 12m  | 🟡 High | ~150 lines of duplication                        |
| 35  | Remove `cockroachdb/errors` from go-localsync — migrate to stdlib              | all          | 10m  | 🟡 High | Banned dependency removal                        |
| 36  | Add `Publish-side event middleware` — events through middleware on Publish     | core/event   | 12m  | 🟡 High | Currently only subscribe path has middleware     |
| 37  | Implement `Store.ReadBackwards` — interface + MemoryStore + SQLEventStore      | core/event   | 12m  | 🟡 High | Time-travel query capability                     |
| 38  | Add `PublishedAt` to `OutboxEntry`                                             | core/event   | 10m  | 🟡 High | No way to measure outbox lag                     |
| 39  | Add `ProcessedAt` to `CheckpointStore`                                         | core/event   | 10m  | 🟡 High | Store (EventID, time.Time) not just EventID      |
| 40  | Make `time.Now()` injectable across all modules                                | core         | 12m  | 🟡 High | Causes non-deterministic tests                   |
| 41  | Increase decider coverage to 95%+ (loadFromSnapshot at 18.2%)                  | core/decider | 12m  | 🟡 High | Major coverage gap in recommended pattern        |
| 42  | Add `EventRetry` middleware tests                                              | middleware   | 10m  | 🟡 High | Already listed in #15, but broader scope         |
| 43  | Add `SQLSnapshotStore` + `SQLCheckpointStore` sqlmock tests                    | storage      | 12m  | 🟡 High | Persistent stores with mock coverage             |
| 44  | Add Turso integration test (save→load→delete)                                  | storage      | 10m  | 🟡 High | Turso-specific paths untested                    |
| 45  | Add `OutboxSchema` to `storage.Schema()` — currently only events DDL           | storage      | 5m   | 🟡 High | Incomplete schema helper                         |
| 46  | Add storage metadata roundtrip test (save→load→verify all fields)              | storage      | 10m  | 🟡 High | Data integrity verification                      |
| 47  | Add `ServerReceivedAt` and `ServerStoredAt` server-side timestamps             | core/event   | 10m  | 🟡 High | Offline-first metadata gap                       |
| 48  | Add `event.Event.Clone()` method for defensive copy safety                     | core/event   | 8m   | 🟡 High | Defensive copying currently manual               |
| 49  | Add `event.Context` propagation — thread ctx through NewEvent                  | core/event   | 10m  | 🟡 High | Context loss in event creation                   |
| 50  | Increase projection coverage to 95%+ (replay at 73.3%)                         | projection   | 12m  | 🟡 High | Major coverage gap                               |

**Tier 2 cumulative time:** ~5.5 hours  
**Tier 2 + Tier 1:** ~8 hours for 64% of total value

---

## Tier 3: 20% → 80% Impact (🟢 Medium)

These 48 tasks deliver the next 16% of value. Docs, refactoring, test splitting, small features.

| #   | Task                                                                    | Module       | Est. | Impact    | Why                                                    |
| --- | ----------------------------------------------------------------------- | ------------ | ---- | --------- | ------------------------------------------------------ |
| 51  | Fix `catalog/asyncapi/exporter.go` missing CommandMessage case          | catalog      | 10m  | 🟢 Medium | Reported in SESSION_58                                 |
| 52  | Fix `catalog/registry.go` Build() shared backing array corruption       | catalog      | 12m  | 🟢 Medium | Memory safety bug                                      |
| 53  | Move `example/todo` to own repository                                   | example      | 12m  | 🟢 Medium | External dep fragility                                 |
| 54  | Consider renaming `sync` package — shadows stdlib                       | sync         | 10m  | 🟢 Medium | Confusing import                                       |
| 55  | Document time-travel API in README/AGENTS.md                            | docs         | 8m   | 🟢 Medium | LoadToVersion/LoadToTimestamp/PositionalLoader         |
| 56  | Document "state is disposable" as canonical pattern                     | docs         | 8m   | 🟢 Medium | Core decider principle                                 |
| 57  | Document determinism rule — no time.Now() inside projections            | docs         | 8m   | 🟢 Medium | Common pitfall                                         |
| 58  | Document versioned event names convention (v1.EventName)                | docs         | 8m   | 🟢 Medium | Schema versioning convention                           |
| 59  | Document soft deletes over hard deletes                                 | docs         | 8m   | 🟢 Medium | Event sourcing best practice                           |
| 60  | Document offline-first metadata conventions                             | docs         | 8m   | 🟢 Medium | CorrelationID, CausationID usage                       |
| 61  | Add `ContextEnricher` wiring to repositories                            | core         | 10m  | 🟢 Medium | Metadata extraction from context                       |
| 62  | Convert `DispatchTyped` to method on `*query.Dispatcher`                | core/query   | 10m  | 🟢 Medium | API discoverability                                    |
| 63  | Add `query/pagination.go` helpers                                       | core/query   | 8m   | 🟢 Medium | Missing pagination utilities                           |
| 64  | Add `catalog.Exporter` interface + `WalkMessages` helper                | catalog      | 10m  | 🟢 Medium | Extensibility                                          |
| 65  | Delete `catalog/internal/cattest/` package (454 lines, 0% coverage)     | catalog      | 8m   | 🟢 Medium | Dead code                                              |
| 66  | Wire `example/user/aggregate.go` to catalog-aware event constructors    | example      | 10m  | 🟢 Medium | Example quality                                        |
| 67  | Add enum + default struct tag support to Schema/Property                | catalog      | 12m  | 🟢 Medium | Schema generation gap                                  |
| 68  | Make AsyncAPI servers configurable instead of hardcoded kafka:9092      | catalog      | 8m   | 🟢 Medium | Configurability                                        |
| 69  | Simplify `cattest/catalog.go` to use zero-cost API                      | catalog      | 10m  | 🟢 Medium | Deprecated type usage                                  |
| 70  | Remove deprecated `CatalogBuilder` from `catalog/adapters`              | catalog      | 8m   | 🟢 Medium | Dead code                                              |
| 71  | Add `NewVectorClockFromMap` test — negative counter rejection           | sync         | 8m   | 🟢 Medium | Missing test case                                      |
| 72  | Build catch-up projection runner (checkpoint → replay → live)           | projection   | 12m  | 🟢 Medium | Feature gap                                            |
| 73  | Make transactional projection contract explicit in Projection interface | projection   | 10m  | 🟢 Medium | API clarity                                            |
| 74  | Add dead letter queue to projection runner                              | projection   | 12m  | 🟢 Medium | Reliability feature                                    |
| 75  | Add retry and dead-letter for InMemoryRunner projections                | projection   | 12m  | 🟢 Medium | Reliability feature                                    |
| 76  | Add background polling for InMemoryRunner                               | projection   | 12m  | 🟢 Medium | Currently push-only                                    |
| 77  | Implement `projection.Runner.Close()` — currently no-op                 | projection   | 8m   | 🟢 Medium | Resource leak                                          |
| 78  | Add `LifecycleMixin` to `memory/checkpoint` + `memory/outbox`           | memory       | 10m  | 🟢 Medium | Consistency                                            |
| 79  | Consolidate `MemoryBus` handler storage — single map with sentinel      | memory       | 10m  | 🟢 Medium | Simplification                                         |
| 80  | Add concurrent access tests for MemoryBus, MemoryStore, MemoryOutbox    | memory       | 12m  | 🟢 Medium | Thread safety                                          |
| 81  | Extract storage table name constants — replace 30+ inline strings       | storage      | 12m  | 🟢 Medium | Type safety                                            |
| 82  | Move schema DDL onto Dialect interface fully                            | storage      | 10m  | 🟢 Medium | Architecture consistency                               |
| 83  | Rename `CQRSAdapter` → `PebbleEventStore`                               | storage      | 8m   | 🟢 Medium | Naming honesty                                         |
| 84  | Add `SQLEventStoreOption` usage or remove dead type                     | storage      | 5m   | 🟢 Medium | Dead code or missing feature                           |
| 85  | Add command metadata (CorrelationID, CausationID, UserID, etc.)         | core/command | 10m  | 🟢 Medium | Offline-first support                                  |
| 86  | Extract shared `opError` helper — duplicated in aggregate + decider     | core         | 8m   | 🟢 Medium | Deduplication                                          |
| 87  | Wire Codec into snapshot serialization                                  | core/event   | 10m  | 🟢 Medium | ApplySnapshot receives raw []byte                      |
| 88  | Add `Filter`, `Predicate` types to `core/event/`                        | core/event   | 10m  | 🟢 Medium | Query building blocks                                  |
| 89  | Add `ContextQuerier`, `ContextAppender`, `QueryResult` interfaces       | core/event   | 10m  | 🟢 Medium | Hybrid architecture                                    |
| 90  | Extract error classification to standalone package                      | core         | 12m  | 🟢 Medium | 5 modules import event just for RegisterClassification |
| 91  | Standardize storage error wrapping patterns                             | storage      | 10m  | 🟢 Medium | Inconsistent error messages                            |
| 92  | Replace `init()` error registration with explicit setup                 | all          | 12m  | 🟢 Medium | Hidden global side effects                             |
| 93  | Split `testhelpers/fakes.go` (342 lines → per-fake files)               | testhelpers  | 10m  | 🟢 Medium | File size limit                                        |
| 94  | Extract fake test doubles from `repository_test.go` to `testhelpers/`   | testhelpers  | 10m  | 🟢 Medium | Reusability                                            |
| 95  | Consolidate testhelpers fake boilerplate via `fakeBase` struct          | testhelpers  | 10m  | 🟢 Medium | ~80 lines saved                                        |
| 96  | Split large test files: `decider_test.go`, `runner_test.go`             | all          | 12m  | 🟢 Medium | File size limit                                        |
| 97  | Add BDD tests for Version, SchemaVersion, OutboxStatus types            | all          | 10m  | 🟢 Medium | Coverage gaps                                          |
| 98  | Add fuzz tests for event creation, ID parsing, schema reflection        | all          | 12m  | 🟢 Medium | Robustness                                             |

**Tier 3 cumulative time:** ~8 hours  
**Tier 1+2+3:** ~16 hours for 80% of total value

---

## Tier 4: 80% → 100% Impact (⚪ Low — Strategic / Research)

These 44 tasks deliver the remaining 20% of value. Long-term features, research, and nice-to-haves.

| #   | Task                                                                         | Module      | Est. | Impact | Why                                |
| --- | ---------------------------------------------------------------------------- | ----------- | ---- | ------ | ---------------------------------- |
| 99  | Add global `TransactionID` branded type for cross-aggregate consistency      | core        | 12m  | ⚪ Low | Breaking v2 change; deferred       |
| 100 | `io.Closer` removal from core interfaces                                     | core        | 12m  | ⚪ Low | Breaking change; deferred          |
| 101 | Add catalog diff/breaking-change detection tool                              | catalog     | 12m  | ⚪ Low | Nice-to-have for CI                |
| 102 | Modularize ActaFlow — split into sub-modules                                 | external    | 12m  | ⚪ Low | External project                   |
| 103 | Add high-level test utilities — AggregateTester, ProjectionTester fluent API | testhelpers | 12m  | ⚪ Low | DX improvement                     |
| 104 | Add bi-temporal support: `ValidAt` in Metadata, `LoadToValidTime`            | core/event  | 12m  | ⚪ Low | Advanced feature                   |
| 105 | Add `WithAsyncWrites()` option for PebbleEventStore                          | storage     | 10m  | ⚪ Low | Throughput optimization            |
| 106 | SQLite WAL mode configuration — PRAGMA journal_mode=WAL                      | storage     | 8m   | ⚪ Low | Performance                        |
| 107 | Split `event.Store` into Writer/Reader/Deleter — compose back                | core/event  | 12m  | ⚪ Low | ISP refinement                     |
| 108 | Add `DecodePayload[T]` batch decode helper                                   | core/event  | 8m   | ⚪ Low | Convenience                        |
| 109 | Make event `Core` truly immutable — build in NewEvent, return interface      | core/event  | 12m  | ⚪ Low | Architecture purity                |
| 110 | Add projection parallel processing — goroutine pool                          | projection  | 12m  | ⚪ Low | Performance                        |
| 111 | Add projection rebuild/reset API                                             | projection  | 10m  | ⚪ Low | Operational feature                |
| 112 | Add `HandleBatch(ctx, []Event)` to projections for bulk upserts              | projection  | 10m  | ⚪ Low | Performance                        |
| 113 | Absorb `projection/` module into `core/event`                                | projection  | 12m  | ⚪ Low | Reduces module count               |
| 114 | Add OpenAPI/Swagger exporter parallel to AsyncAPI                            | catalog     | 12m  | ⚪ Low | Already exists in catalog/openapi/ |
| 115 | Generate `llms.txt` alongside EventCatalog output                            | catalog     | 10m  | ⚪ Low | AI-friendly docs                   |
| 116 | Schema: support nullable/deprecated/pattern/minimum/maximum struct tags      | catalog     | 12m  | ⚪ Low | Richer schema generation           |
| 117 | Add HLC (Hybrid Logical Clock) implementation in sync/                       | sync        | 12m  | ⚪ Low | Offline-first advanced             |
| 118 | Implement pull-before-push sync protocol                                     | sync        | 12m  | ⚪ Low | Offline-first advanced             |
| 119 | Implement rebase mechanism for local events                                  | sync        | 12m  | ⚪ Low | Offline-first advanced             |
| 120 | Build network simulator for testing                                          | sync        | 12m  | ⚪ Low | Testing infrastructure             |
| 121 | Build multi-client test harness                                              | sync        | 12m  | ⚪ Low | Testing infrastructure             |
| 122 | Create Watermill module for real message broker integration                  | watermill   | 12m  | ⚪ Low | Already thin adapter exists        |
| 123 | Build thin PostgreSQL store adapter (no Watermill dep)                       | storage     | 12m  | ⚪ Low | Already exists                     |
| 124 | Build thin NATS bus adapter (no Watermill dep)                               | storage     | 12m  | ⚪ Low | New adapter                        |
| 125 | Add circuit breaker middleware                                               | middleware  | 12m  | ⚪ Low | Resilience pattern                 |
| 126 | Add OpenTelemetry tracing middleware — extract to separate module            | middleware  | 12m  | ⚪ Low | Already exists; isolate dep        |
| 127 | Add distributed tracing middleware                                           | middleware  | 12m  | ⚪ Low | Observability                      |
| 128 | Rewrite `example/user/` to demonstrate full CQRS stack                       | example     | 12m  | ⚪ Low | Example quality                    |
| 129 | Add `example/user/` smoke test (`TestExampleRuns`)                           | example     | 10m  | ⚪ Low | CI validation                      |
| 130 | Add hybrid service example                                                   | example     | 12m  | ⚪ Low | Both aggregate and context mode    |
| 131 | Add `.goreleaser.yml` for multi-module releases                              | CI          | 10m  | ⚪ Low | Release automation                 |
| 132 | Performance regression CI — benchmark comparison on each PR                  | CI          | 12m  | ⚪ Low | Quality gate                       |
| 133 | Parallelize CI matrix — one job per module                                   | CI          | 10m  | ⚪ Low | Speed                              |
| 134 | Migrate gomodguard → gomodguard_v2 in .golangci.yml                          | CI          | 8m   | ⚪ Low | Linter update                      |
| 135 | Change LICENSE from proprietary to MIT or Apache-2.0                         | legal       | 5m   | ⚪ Low | Already discussed                  |
| 136 | Add distributed consensus capability (Raft/CRDT overlay)                     | sync        | 12m  | ⚪ Low | Research feature                   |
| 137 | Add time-series event query language                                         | core/event  | 12m  | ⚪ Low | Research feature                   |
| 138 | Integrate TypeSpec types → catalog.Registry                                  | catalog     | 12m  | ⚪ Low | Type system integration            |
| 139 | Create documentation site (Docusaurus/MkDocs/Hugo)                           | docs        | 12m  | ⚪ Low | Marketing/docs                     |
| 140 | Set up pkg.go.dev documentation hosting                                      | docs        | 8m   | ⚪ Low | Go ecosystem standard              |
| 141 | Write CHANGELOG.md — 61+ sessions of changes                                 | docs        | 12m  | ⚪ Low | Release hygiene                    |
| 142 | Prune `docs/status/` — 100+ archived reports                                 | docs        | 10m  | ⚪ Low | Repository hygiene                 |

**Tier 4 cumulative time:** ~7.5 hours  
**Total plan time:** ~23.5 hours for 142 tasks

---

## Stale / Completed Items (Excluded from Plan)

These items from TODO_LIST.md are already done or obsolete:

| Original TODO                                           | Status      | Why excluded                            |
| ------------------------------------------------------- | ----------- | --------------------------------------- |
| Implement SQL-backed SnapshotStore                      | ✅ DONE     | `storage/snapshot.go` exists            |
| Implement SQL-backed CheckpointStore                    | ✅ DONE     | `storage/checkpoint.go` exists          |
| Add SQL-backed transactional outbox                     | ✅ DONE     | `storage/transactional_store.go` exists |
| Implement Saga/Process Manager                          | ✅ DONE     | `saga/` module created Session 108      |
| Create saga/ module                                     | ✅ DONE     | `saga/` module exists                   |
| Fix outbox transaction co-participation                 | ✅ DONE     | `SQLBackend` added Session 108          |
| Update stale AGENTS.md                                  | ✅ DONE     | Saga section added Session 108          |
| Update stale FEATURES.md                                | ✅ DONE     | Saga status updated Session 108         |
| Fix storage/dialect.go using `any`                      | ✅ VERIFIED | Intentional for database/sql interop    |
| Fix catalog/asyncapi/exporter.go missing CommandMessage | ✅ VERIFIED | Already handled at builder.go:142       |

---

## Dependency Graph (Critical Path)

```
Tier 1:
  Tags (7) → Remove replace directives (8)  [sequential]
  SQLBackend.SagaStore() (1) → Saga example (19)  [sequential]
  GOWORK=off CI (9) → Coverage gate (11) → Lint all modules (27)  [sequential]
  Hydrate test (2) → Split saga_test.go (20)  [parallel safe]
  PostgreSQL tests (13) → Outbox cycle test (14)  [parallel safe]

Tier 2:
  Query generic (10) → Migration guide ( docs )  [parallel]
  core→memory fix (12) → Publish core independently  [unblocks]
  Pebble optimization (25) → Benchmark comparison  [parallel]

Tier 3:
  All docs (55-60) → Documentation site (139)  [sequential]
  Test splitting (20, 93-96) → Enforce 350-line limit (282)  [parallel]
```

---

## Reuse Checklist

Before implementing any task, check if existing code fits:

| Pattern Needed        | Existing Code That Fits                                         |
| --------------------- | --------------------------------------------------------------- |
| SQL store constructor | `NewSQLSnapshotStore`, `NewSQLCheckpointStore` pattern ✅       |
| Schema DDL            | `Dialect.SnapshotSchema()`, `Dialect.OutboxSchema()` pattern ✅ |
| SQL test pattern      | `snapshot_test.go`, `checkpoint_test.go` sqlmock pattern ✅     |
| Branded ID            | `id.Of[T]`, `id.AggregateID` ✅                                 |
| Error wrapping        | `fmt.Errorf` with `%w` ✅                                       |
| Time formatting       | `Dialect.FormatTime` / `ParseTime` ✅                           |
| Transaction handling  | `saveWithOutboxTx` pattern ✅                                   |
| Middleware chain      | `MiddlewareChain[H, M]` in `core/pkg/dispatcher/` ✅            |
| Projection builder    | `projection.Builder` with `On[T]()` ✅                          |
| Catalog walker        | `catalog.Registry.Walk()` ✅                                    |

**No new third-party libraries needed for Tier 1–3.**

---

## Recommended Next Session (Tier 1 Execution)

If starting immediately, execute Tier 1 in this order:

1. **Tasks 7–8** (tags + replace directives) — 20 min — unblocks all external consumers
2. **Tasks 1–6** (saga coverage + SQLBackend completion) — 43 min — closes Session 108 gaps
3. **Tasks 9–11** (CI hardening) — 26 min — prevents regression
4. **Tasks 13–15** (integration tests + middleware tests) — 34 min — quality gates
5. **Tasks 16–18** (correctness fixes) — 28 min — bug fixes

**Total: ~2.5 hours for 51% of total project value.**
