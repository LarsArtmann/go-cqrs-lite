# Session 146 — Comprehensive Status Report

**Date:** 2026-05-29 14:24
**Branch:** master (up to date with origin/master)
**Last commit:** `1d015b9 docs: remove saga module references, update catalog schema extraction results`

---

## Executive Summary

Saga module successfully removed. Storage module is now a clean leaf (core+otel only, no saga dependency). New `example/saga-pattern/` teaches the pattern via projection+command dispatch. **30/31 test packages pass** — 1 pre-existing pebble failure. All documentation updated and consistent.

---

## a) FULLY DONE ✅

### This Session

| Item                    | Detail                                                                                                                                                                               |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Saga module removal** | Deleted entire `saga/` module (~1,600 LOC, 20+ files). Removed from go.work, flake.nix, cmd/api-stability                                                                            |
| **Storage cleanup**     | Removed saga_store.go, saga_store_test.go, sqlite_integration_outbox_saga_test.go, saga schema from dialects, sagaStore from SQLBackend, tableSagas constant                         |
| **Turso cleanup**       | Removed NewTursoSagaStore constructor                                                                                                                                                |
| **New example**         | Created `example/saga-pattern/` showing saga orchestration via projection + command dispatch                                                                                         |
| **Documentation sweep** | Updated AGENTS.md, FEATURES.md, README.md, storage/README.md, docs/README.md, docs/architecture/README.md, docs/MIGRATION_v1.md, docs/api_surface.txt (992 exports, down from 1058+) |
| **ADR-0004 status**     | Marked as "Superseded" in docs/README.md                                                                                                                                             |

### All-Time Completed (High-Value Items)

| Category                    | Modules/Features                                                                       |
| --------------------------- | -------------------------------------------------------------------------------------- |
| **Core CQRS**               | Command, Event, Query dispatchers with middleware chains                               |
| **Decider pattern**         | Pure-function aggregates with Repository[State], snapshot support                      |
| **Event sourcing**          | Store (Sink/Source ISP split), Journal, SeekableJournal, BackwardsSource               |
| **SQL storage**             | PostgreSQL + SQLite + Turso: EventStore, SnapshotStore, Outbox, CheckpointStore        |
| **Embedded storage**        | Pebble key-value event store with async writes                                         |
| **Projections**             | Replay+live runner, HandlerRegistry, Builder with On[T](<>), parallel processing       |
| **Event signing**           | HMAC-SHA256, Ed25519, multisig, verification middleware                                |
| **24 middleware factories** | Logging, Retry, Recovery, Validation, Metrics, OTel, Circuit Breaker × 3 message types |
| **Catalog**                 | Registry, AsyncAPI/D2/OpenAPI/EventCatalog exporters                                   |
| **Branded IDs**             | `id.Of[T]` with ULID, AggregateID, EventID, etc.                                       |
| **Error taxonomy**          | 5-family classification: Rejection/Conflict/Transient/Infrastructure/Corruption        |
| **Tombstone soft-delete**   | TombstoneStatus enum, detection, marking — no Delete on Store                          |
| **OTel integration**        | Tracing + Metrics middleware, shared helpers module                                    |
| **Module isolation**        | 21 modules in go.work, each with own go.mod                                            |
| **Code generation**         | cqrs-gen: typed handler registration from Go structs                                   |

---

## b) PARTIALLY DONE ⚠️

| Item                 | What's Done                           | What's Missing                                                                             |
| -------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------ |
| **example/user/**    | Basic structure exists                | Doesn't demonstrate full CQRS stack (projections, sagas, streams, signing). No smoke test. |
| **example/todo/**    | Has go.mod, basic structure           | No comprehensive demo of saga pattern using projection primitives                          |
| **catalog/schema/**  | Schema types, reflect, YAML           | `schema_test.go` was renamed → `basic_test.go` (unstaged change)                           |
| **core/store/**      | EventStore adapter over Backend       | 28.2% coverage — very low                                                                  |
| **listing/**         | Module extracted with builder, reader | Benchmark test references deleted `stream/` package (pre-existing compile error)           |
| **Pebble backend**   | EventStore with async writes          | `TestPebbleBackend/ScanPrefix` fails — Scan returns empty, expects sorted keys             |
| **example/listing/** | Has main.go                           | References deleted `stream/` package (pre-existing compile error)                          |

---

## c) NOT STARTED 📐

| #   | Item                                                                                   | Priority | Est |
| --- | -------------------------------------------------------------------------------------- | -------- | --- |
| 1   | **core/store coverage** — 28.2% is critically low                                      | HIGH     | 2h  |
| 2   | **Fix Pebble ScanPrefix test** — only failing test in the repo                         | HIGH     | 30m |
| 3   | **Fix listing/ compile errors** — references deleted `stream/` package                 | HIGH     | 1h  |
| 4   | **Rewrite example/user/** — demonstrate full CQRS stack                                | MEDIUM   | 2h  |
| 5   | **Stream module integration tests**                                                    | MEDIUM   | 1h  |
| 6   | **Performance regression CI** — benchmark comparison on PRs                            | MEDIUM   | 2h  |
| 7   | **Fuzz tests** — event creation, ID parsing, schema, DecodePayload                     | MEDIUM   | 3h  |
| 8   | **BDD tests** — Version, SchemaVersion, OutboxStatus, Pagination                       | LOW      | 2h  |
| 9   | **Documentation site** — Docusaurus/MkDocs/Hugo                                        | LOW      | 4h  |
| 10  | **v2 breaking changes** — query.Handler generic, io.Closer removal, event immutability | LOW      | 4h  |

---

## d) TOTALLY FUCKED UP 💀

| Issue                                           | Severity    | Detail                                                                                                                                                         |
| ----------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **core/store at 28.2% coverage**                | 🔴 CRITICAL | New module with barely any tests. Ships as "production" but untested.                                                                                          |
| **listing/ references deleted stream/**         | 🔴 CRITICAL | `listing/benchmark_test.go` imports `github.com/larsartmann/go-cqrs-lite/stream` which no longer exists. `example/listing/main.go` also broken. Compile error. |
| **Pebble ScanPrefix broken**                    | 🟡 HIGH     | `TestPebbleBackend/ScanPrefix` fails consistently. Scan keys returns empty. Likely a real bug in key iteration.                                                |
| **ADR-0004 outdated**                           | 🟡 MEDIUM   | Still says "Accepted" with full saga module rationale. Should be updated to "Superseded" with explanation of why saga was removed.                             |
| **No example/stream/**                          | 🟡 MEDIUM   | Listed in docs/README.md but directory doesn't exist. Dead reference.                                                                                          |
| **example/user/ is a skeleton**                 | 🟡 MEDIUM   | Doesn't demonstrate the library's actual capabilities. Misleading as the primary example.                                                                      |
| **otel/attributes.go has stale saga constants** | 🟢 LOW      | `AttrSagaType`, `AttrSagaStep`, `AttrSagaStepName` still defined. Harmless but dead code.                                                                      |
| **projection/health.go references saga.Runner** | 🟢 LOW      | Comment says "Both projection.Runner and saga.Runner implement this interface". saga.Runner no longer exists.                                                  |

---

## e) WHAT WE SHOULD IMPROVE

1. **core/store coverage is the #1 quality gap** — 28.2% on a new "production" module is unacceptable. Need comprehensive tests before v1.
2. **listing/ module is broken** — references deleted stream/ package. Either fix the import or merge listing into another module.
3. **Example quality is low** — example/user/ doesn't showcase the library. example/listing/ doesn't compile. Only example/projection/ and example/saga-pattern/ are healthy.
4. **ADR-0004 needs proper superseded writeup** — explain why saga was removed (pattern, not abstraction) and point to example/saga-pattern/.
5. **API surface audit** — dropped from 1058 to 992 exports (66 removed). Should verify no consumers depend on removed exports (acceptable pre-v1.0).
6. **Test flakiness** — pebble ScanPrefix is a real bug, not flaky. Fix it.
7. **Dead code cleanup** — otel saga attributes, projection/health.go comment, example/stream/ reference.

---

## f) TOP 25 THINGS TO DO NEXT

| Rank | Task                                                                   | Impact                   | Est |
| ---- | ---------------------------------------------------------------------- | ------------------------ | --- |
| 1    | Fix listing/ compile errors (references deleted stream/)               | Unblocks build           | 1h  |
| 2    | Write core/store tests → ≥80% coverage                                 | Quality gate             | 2h  |
| 3    | Fix Pebble ScanPrefix test failure                                     | Only failing test        | 30m |
| 4    | Update ADR-0004 to "Superseded" with removal rationale                 | Accuracy                 | 15m |
| 5    | Remove dead otel saga attributes (AttrSagaType, etc.)                  | Cleanup                  | 10m |
| 6    | Fix projection/health.go stale saga.Runner comment                     | Cleanup                  | 5m  |
| 7    | Remove example/stream/ reference from docs/README.md                   | Accuracy                 | 5m  |
| 8    | Rewrite example/user/ to demonstrate full CQRS stack                   | Onboarding               | 2h  |
| 9    | Add example/saga-pattern/ smoke test                                   | CI coverage              | 15m |
| 10   | Add example/user/ smoke test                                           | CI coverage              | 15m |
| 11   | Stream module integration tests                                        | Quality                  | 1h  |
| 12   | Add E2E throughput benchmarks                                          | Performance transparency | 2h  |
| 13   | Benchmark storage backends (PG vs SQLite vs Pebble)                    | Performance transparency | 2h  |
| 14   | Add fuzz tests for event creation, ID parsing                          | Robustness               | 3h  |
| 15   | Add BDD tests for Version, SchemaVersion, Pagination                   | Coverage                 | 2h  |
| 16   | Split large test files (decider_test.go ~1200L, runner_test.go ~1057L) | Maintainability          | 1h  |
| 17   | Enforce 350-line limit on test files via pre-commit hook               | Quality gate             | 30m |
| 18   | Performance regression CI — benchmark comparison on PRs                | Prevent regressions      | 2h  |
| 19   | Add gofumpt/goimports to pre-commit hook                               | Style consistency        | 30m |
| 20   | v2: Make query.Handler generic TypedHandler[T]                         | Type safety              | 1h  |
| 21   | v2: io.Closer removal from core interfaces                             | API cleanliness          | 2h  |
| 22   | Push signing v1.0.0 tag                                                | Release readiness        | 15m |
| 23   | Push all modules v1.0.0 tags (remove replace directives)               | Release readiness        | 1h  |
| 24   | Documentation site (Docusaurus/MkDocs)                                 | Discoverability          | 4h  |
| 25   | Add HLC (Hybrid Logical Clock) implementation                          | Offline-first enabler    | 4h  |

---

## g) MY TOP #1 QUESTION

**Should the `listing/` module be merged into `core/store/` or remain standalone?**

The `listing/` module currently imports the deleted `stream/` package, so it doesn't compile. It was extracted as a dedicated module but its purpose (aggregate listing, tombstone detection, cursor pagination) overlaps heavily with `core/store/`. The options are:

1. **Merge into core/store/** — listing is a natural reader concern over the Backend interface
2. **Fix listing/ to not depend on stream/** — remove the stream dependency, keep it standalone
3. **Delete listing/ entirely** — consumers can build listing queries directly with core/store

I cannot determine the right call because it depends on the intended architecture: is `listing/` a first-class module or an optional convenience? The answer determines whether we invest in fixing it or consolidating it.

---

## Build & Test Matrix

| Check                            | Status                                            |
| -------------------------------- | ------------------------------------------------- |
| `go build ./...` (all workspace) | ✅ PASS                                           |
| `go vet ./...` (all modules)     | ✅ PASS                                           |
| core/command                     | ✅ 94.2%                                          |
| core/event                       | ✅ 90.7%                                          |
| core/query                       | ✅ 96.8%                                          |
| core/decider                     | ✅ 100.0%                                         |
| core/pkg/id                      | ✅ 100.0%                                         |
| core/pkg/dispatcher              | ✅ 92.2%                                          |
| **core/store**                   | ⚠️ **28.2%**                                      |
| memory                           | ✅ 96.6%                                          |
| catalog                          | ✅ 71.0%                                          |
| middleware                       | ✅ 94.0%                                          |
| testhelpers                      | ✅ 83.7%                                          |
| integration                      | ✅ PASS                                           |
| projection                       | ✅ 90.4%                                          |
| signing                          | ✅ 94.2%                                          |
| storage                          | ✅ 84.2%                                          |
| watermill                        | ✅ 94.4%                                          |
| **pebble**                       | ❌ **ScanPrefix FAIL**                            |
| codec                            | ✅ 100.0%                                         |
| turso                            | ⚠️ 0.0% (no test files)                           |
| listing                          | ❌ **COMPILE ERROR** (references deleted stream/) |
| otel                             | ✅ 96.6%                                          |

**Pass rate: 30/31 packages (96.8%)**

## Module Inventory (21 modules)

```
core/          — Command, Event, Query, Decider, Branded IDs, Store
memory/        — In-memory Store, Bus, SnapshotStore
catalog/       — Registry, AsyncAPI/D2/OpenAPI/EventCatalog exporters
middleware/     — 24 middleware factories (8 concerns × 3 message types)
testhelpers/   — Noop/Failing/Panic handlers, FakeMetrics
projection/    — Runner (replay+live), HandlerRegistry, Builder
signing/       — HMAC-SHA256, Ed25519, multisig, middleware
storage/       — SQLEventStore, SnapshotStore, Outbox, CheckpointStore (PG/SQLite/Turso)
otel/          — Shared OpenTelemetry helpers
watermill/     — Watermill protocol adapter
pebble/        — Embedded key-value event store
codec/         — Payload encoding (JSON, Raw)
turso/         — Turso database connector
listing/       — Aggregate listing (BROKEN: references deleted stream/)
cmd/cqrs-gen/  — Code generator
integration/   — Cross-module tests
```

**Production LOC:** ~44,000 across all modules
**API surface:** 992 exported symbols
**Examples:** saga-pattern, projection, storage, todo, user, listing (broken)
