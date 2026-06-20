# Status Report — Session 126: Post-Sink/Source Comprehensive Health Check

**Date:** 2026-05-29 03:40 CEST
**Branch:** master (up to date with origin)
**Working tree:** CLEAN (untracked: `cmd/cqrs-gen/cqrs-gen` binary)
**Total coverage:** 90.9% across 23 measured packages
**Test suite:** 29 packages, 0 failures, 0 vet issues

---

## A. FULLY DONE ✅

These items are complete, tested, committed, and pushed.

### A1. Sink/Source Interface Split (ADR-0006)

| File                             | Change                                                                                                                                                     |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core/event/store.go`            | `EventSink` (write) + `EventSource` (read) + `Store = EventSink + EventSource`. Added `Journal`, `SeekableJournal`, `BackwardsSource`, `TransactionalSink` |
| `core/event/tombstone.go`        | `TombstoneStatus` enum (Active/Tombstoned/Undetermined), `DetectTombstone()`, `MarkTombstone()`, `MarkRebirth()`                                           |
| `memory/store.go`                | Implements `Journal`, `SeekableJournal`, `BackwardsSource`. No `Delete`.                                                                                   |
| `storage/event_store.go`         | `BackwardsSource` assertion, OTel tracing                                                                                                                  |
| `storage/event_store_load.go`    | OTel attributes on Load                                                                                                                                    |
| `storage/sql_backend.go`         | `TransactionalSink()` method (deprecated `TransactionalStore`)                                                                                             |
| `storage/transactional_store.go` | `TransactionalSink` assertion                                                                                                                              |
| `storage/pebble_helpers.go`      | Removed `Delete` method                                                                                                                                    |
| `decider/decider.go`             | `TransactionalSink` type assertion                                                                                                                         |
| `testhelpers/fake_store.go`      | No `Delete` method or `deleteFn` field                                                                                                                     |

### A2. Stream Read Model Module

| Component                    | Status                                                                                                                                                                                                                                                                                                                                         |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `stream/types.go`            | `AggregateRef`, `AggregateStatus`, `Page[T]`, `TombstonePolicy`, `ListOptions`, `TombstonePolicy.String()`, `AggregateStatus.MarshalJSON()`, `validateTablePrefix()`                                                                                                                                                                           |
| `stream/aggregate_reader.go` | `AggregateReader` interface                                                                                                                                                                                                                                                                                                                    |
| `stream/builder.go`          | `ListBuilder` fluent API                                                                                                                                                                                                                                                                                                                       |
| `stream/in_memory.go`        | `InMemoryAggregateReader` using `Journal.ReadAll()`                                                                                                                                                                                                                                                                                            |
| `stream/sql_reader.go`       | `SQLAggregateReader` with cursor pagination, tombstone filtering, SQL injection protection                                                                                                                                                                                                                                                     |
| `stream/projection.go`       | `AggregateProjection` (SQL read model maintainer with tombstone detection)                                                                                                                                                                                                                                                                     |
| `stream/middleware.go`       | `StatusMiddleware` for event bus                                                                                                                                                                                                                                                                                                               |
| Tests                        | `builder_test.go` (chaining + defaults + `TombstonePolicy.String` + `AggregateStatus.MarshalJSON`), `sql_reader_test.go` (prefix, empty, requires-type, tombstone filtering, cursor pagination), `projection_test.go` (prefix, normal event, tombstone, rebirth, tombstone preservation), `in_memory_test.go` (5 scenarios), `sql_bdd_test.go` |

### A3. OTel Instrumentation Module

| Component                     | Status                                                                |
| ----------------------------- | --------------------------------------------------------------------- |
| `otel/attributes.go`          | 15+ attribute constants (`AttrEventCount`, `AttrAggregateType`, etc.) |
| `otel/instrumentation.go`     | `Instrumentation` struct                                              |
| `otel/spans.go`               | `StartSpan`, `RecordError`                                            |
| `otel/tracer.go`              | `NewTracer` factory                                                   |
| `otel/meter.go`               | `NewMeter` factory                                                    |
| `storage/otel.go`             | Per-module tracer + helper functions                                  |
| `storage/event_store.go`      | Tracing on Save                                                       |
| `storage/event_store_load.go` | Tracing on Load, LoadFromVersion, LoadToVersion, LoadToTimestamp      |
| `storage/checkpoint.go`       | Tracing on Checkpoint Load/Save                                       |
| `middleware/tracing.go`       | Command/event/query tracing middleware                                |
| `middleware/metrics_otel.go`  | OTel metrics middleware                                               |

### A4. Documentation

| Document                             | Status                                                                                         |
| ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `docs/adr/0006-sink-source-split.md` | Full ADR with context, decision, consequences, naming collision                                |
| `docs/STORAGE_GUIDE.md`              | Updated method references (ReadAll, ReadFrom, AppendBatch)                                     |
| `docs/ARCHITECTURE_PATTERNS.md`      | Fixed `LoadAllFromPosition` → `ReadFrom`, tombstone guidance                                   |
| `core/README.md`                     | Store = EventSink + EventSource section                                                        |
| `storage/README.md`                  | Shows `TransactionalSink()` as primary API                                                     |
| `AGENTS.md`                          | Module count 15, otel + stream in tree, ISP design principles, Sink/Source patterns, otel deps |
| `FEATURES.md`                        | Stream section (10 features), otel + stream in maturity matrix                                 |

### A5. BDD Test Suite Expansion

| Module         | Tests Added                                                                     |
| -------------- | ------------------------------------------------------------------------------- |
| `core/command` | BDD suite scaffolding                                                           |
| `core/event`   | 39 Ginkgo specs (creation, store, schema evolution, errors, versions, metadata) |
| `core/query`   | BDD suite for pagination + paginated results                                    |
| `core/decider` | BDD nil-input validation descriptions improved                                  |
| `middleware`   | BDD suite for recovery, retry, circuit breaker                                  |
| `saga`         | 12 Ginkgo specs with user-story descriptions, atomic dispatcher                 |
| `stream`       | SQL BDD test + cursor pagination + TombstonePolicy/AggregateStatus tests        |

### A6. Delete Removal

`Delete` method removed from ALL implementations: `MemoryStore`, `SQLEventStore`, `PebbleEventStore`, `FakeStore`. 15+ Delete test functions removed across memory, storage, testhelpers, decider, integration.

### A7. Catalog Golden Files

Regenerated and verified. All 9 catalog packages pass.

---

## B. PARTIALLY DONE 🔶

### B1. OTel Tracing Coverage

**What's done:** Storage (Save, Load, LoadFromVersion, LoadToVersion, LoadToTimestamp, Checkpoint), middleware (command/event/query tracing).

**What's missing:**

- `storage/outbox.go` — no spans on `Append`, `LoadPending`, `MarkPublished`
- `storage/snapshot.go` — no spans on `Save`, `Load`
- `projection/runner.go` — no spans on replay or live handler
- `saga/runner.go` — has imports but **no actual span creation** (imports added then emptied)
- `stream/` — no tracing at all
- No OTel **metrics** on any module (only middleware has metrics wiring, no actual counter/histogram recording)

### B2. Example Applications

- `example/user` — Updated for Sink/Source split
- `example/storage` — go.mod updated with otel dep
- `example/saga` — go.sum updated
- **Missing:** No example for `stream/` module, no example for `otel/` module, no example showing tombstone/rebirth

### B3. Deprecation Migration Path

- `BackwardsLoader = BackwardsSource` alias exists ✅
- `TransactionalStore` interface exists with deprecation ✅
- `GlobalLoader` / `PositionalLoader` deprecated ✅
- **Missing:** No `staticcheck` or `govet` directives to flag deprecated usage at compile time (e.g., `// Deprecated:` comments exist but no `SA1019` suppression plan)

---

## C. NOT STARTED ⬜

### C1. No v1.0.0 Release

All modules use `replace` directives in go.mod files. No version tags pushed. Consumers must use `replace` directives themselves or clone the monorepo.

### C2. No CI Pipeline Updates

The `.github/workflows/ci.yml` was not updated to:

- Test the `otel/` module (it was added to go.work but CI may skip it)
- Test the `stream/` module separately
- Add OTel-specific lint rules

### C3. No Benchmark Suite

No Go benchmarks exist for:

- Stream cursor pagination throughput
- SQL reader vs in-memory reader performance
- OTel tracing overhead on hot paths
- Tombstone detection cost on large streams

### C4. No Integration Test for OTel

No test verifies that spans are actually emitted with correct names, attributes, and parent linkage.

### C5. No Migration Guide

No document explains how existing consumers should migrate from `Store.Delete()` → tombstone pattern, or from `GlobalLoader` → `Journal`.

### C6. No API Stability Guarantees

No `go_api.yaml` or `apidiff` tooling to detect breaking changes between commits.

### C7. No Structured Logging

No `slog` integration beyond what pebble uses internally. No log correlation with trace IDs.

### C8. No Health Check Endpoints

No `/healthz` or readiness probes. No liveness checking for projection runner, outbox publisher, or saga runner.

### C9. No Documentation Site

No generated documentation site (e.g., pkg.go.dev preview, Docusaurus, or similar). READMEs exist but are not rendered anywhere.

### C10. No Chaos/Fault-Injection Testing

No tests for network partitions, database connection drops, or partial write failures beyond basic error cases.

---

## D. TOTALLY FUCKED UP 💣

### D1. `otel/` Module Has ZERO Tests

`otel/` exports 5 files of public API (`StartSpan`, `RecordError`, `NewTracer`, `NewMeter`, attribute constants). None of it is tested. If `TracerProvider` is nil, `StartSpan` silently returns a no-op span — but this behavior is unverified. If the API changes, nothing will catch it.

### D2. `saga/runner.go` Has Ghost OTel Imports

The saga runner has `cqrsotel` and `trace` imports that were added during the OTel push but **no spans are actually created**. The `Start` method runs without any tracing context. The imports exist solely because they were added and then the actual span calls were removed/never written.

### D3. LSP Diagnostics Show 100+ Errors (Stale)

The LSP reports `ErrNilEvent` undefined in `tombstone.go` and missing methods on `MemoryStore`. These are **stale LSP cache issues** — `go build` succeeds cleanly. But it means the development experience in editors is broken and confusing for anyone who opens the project.

### D4. `catalog/internal/cattest` Has 0% Coverage

This entire package (30+ functions) is untested. It's a test helper package for catalog tests, but none of the catalog tests actually import it. Dead code masquerading as infrastructure.

### D5. BuildFlow Pre-commit Hook Has Persistent Failures

- `golangci-lint` exits code 7 (workspace pattern error) — pre-existing, never fixed
- `go-structure-linter` complains about missing `pkg/` and `internal/` directories — by design, never addressed
- These create noise in every commit and train developers to ignore CI failures

---

## E. WHAT WE SHOULD IMPROVE 🔧

### E1. Test the `otel/` Module

Add at least:

- `StartSpan` with nil TracerProvider returns no-op span
- `StartSpan` with real TracerProvider creates span with correct name
- `RecordError` sets span status
- Attribute constants are correct and non-empty

### E2. Fix Saga OTel Ghost Imports

Either:

- Add actual span creation to `saga/runner.go` (`Start`, `ExecuteStep`, compensate), OR
- Remove the unused imports entirely

### E3. Add OTel Spans to Outbox, Snapshot, Projection Runner, Stream

These are hot paths that would benefit from distributed tracing. Currently only the SQL store load/save operations have spans.

### E4. Write a Migration Guide

A single `docs/MIGRATION_v1.md` covering:

- `Store.Delete()` → tombstone pattern
- `GlobalLoader` → `Journal`
- `PositionalLoader` → `SeekableJournal`
- `TransactionalStore` → `TransactionalSink`
- `BackwardsLoader` → `BackwardsSource`

### E5. Kill Dead Code in `catalog/internal/cattest`

Either use it in catalog tests or delete it. 30+ functions at 0% coverage is a smell.

### E6. Add Breakpoint Tests for Public API

Use `internal/apitest` or similar to ensure the public API surface doesn't change without explicit approval. This is critical for a library.

### E7. Fix LSP Stale Cache

Run `gopls check` or restart LSP after major refactors. The 100+ phantom errors confuse IDE users.

### E8. Add Example for Stream Module

`example/stream/` showing cursor pagination, tombstone filtering, and the builder pattern.

### E9. Improve Coverage Gaps

| Package        | Coverage | Target | Gap                                 |
| -------------- | -------- | ------ | ----------------------------------- |
| `core/decider` | 91.3%    | 95%    | `applyEnricher` at 18.2%            |
| `storage`      | 93.4%    | 95%    | `turso_sync.OpenTursoSync` at 22.2% |
| `middleware`   | 95.0%    | 98%    | `circuit_breaker.allow` at 54.5%    |
| `testhelpers`  | 85.0%    | 95%    | Multiple assertion helpers at 60%   |

### E10. Clean Up Pre-commit Hook Noise

Either fix the golangci-lint workspace issue or suppress it explicitly. Same for go-structure-linter opinions.

---

## F. TOP 25 THINGS TO DO NEXT (Priority Order)

| #   | Item                                                      | Impact | Effort | Category       |
| --- | --------------------------------------------------------- | ------ | ------ | -------------- |
| 1   | Test `otel/` module (spans, attributes, nil provider)     | HIGH   | 2h     | Quality        |
| 2   | Fix saga OTel ghost imports (add spans or remove imports) | HIGH   | 1h     | Correctness    |
| 3   | Add OTel spans to `storage/outbox.go`                     | HIGH   | 1h     | Observability  |
| 4   | Add OTel spans to `storage/snapshot.go`                   | HIGH   | 1h     | Observability  |
| 5   | Add OTel spans to `projection/runner.go`                  | MEDIUM | 2h     | Observability  |
| 6   | Write migration guide `docs/MIGRATION_v1.md`              | HIGH   | 2h     | Documentation  |
| 7   | Add `stream/` example application                         | MEDIUM | 3h     | Documentation  |
| 8   | Kill dead code in `catalog/internal/cattest`              | MEDIUM | 30m    | Cleanup        |
| 9   | Add `tombstone` + `rebirth` example to `example/user`     | MEDIUM | 1h     | Documentation  |
| 10  | Fix `applyEnricher` coverage (18.2% → 80%+)               | MEDIUM | 1h     | Quality        |
| 11  | Add circuit breaker state transition tests                | MEDIUM | 1h     | Quality        |
| 12  | Fix `OpenTursoSync` coverage (22.2%)                      | LOW    | 1h     | Quality        |
| 13  | Add OTel integration test (verify actual span emission)   | HIGH   | 3h     | Quality        |
| 14  | Update CI pipeline for `otel/` and `stream/` modules      | MEDIUM | 1h     | Infrastructure |
| 15  | Add benchmarks for stream cursor pagination               | LOW    | 2h     | Performance    |
| 16  | Add API surface stability test (`apidiff`)                | HIGH   | 3h     | Quality        |
| 17  | Remove `replace` directives and tag v1.0.0                | HIGH   | 4h     | Release        |
| 18  | Add structured logging with trace ID correlation          | MEDIUM | 3h     | Observability  |
| 19  | Fix LSP stale cache (document workaround)                 | LOW    | 30m    | DX             |
| 20  | Clean up BuildFlow pre-commit noise                       | LOW    | 2h     | DX             |
| 21  | Add health check utilities for runners                    | LOW    | 2h     | Operations     |
| 22  | Add chaos/fault-injection test suite                      | LOW    | 4h     | Resilience     |
| 23  | Generate documentation site                               | LOW    | 4h     | Documentation  |
| 24  | Add `aggregate` package deprecation path to decider       | MEDIUM | 2h     | Cleanup        |
| 25  | Add `internal/apitest` package for contract testing       | MEDIUM | 3h     | Quality        |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**What is the actual release strategy for v1.0.0?**

All 15 modules use `replace` directives pointing to local paths. This means:

- No consumer can `go get github.com/larsartmann/go-cqrs-lite/core` without their own `replace` directives
- The module graph is fragile — a consumer pinning `core@v1.6.0` may get a different `event` package than what `memory@v1.6.0` expects
- The `replace` directives reference local paths that don't exist in the published module

I cannot determine:

1. Should all 15 modules be tagged `v1.0.0` simultaneously?
2. Should the less-mature modules (`otel`, `stream`, `cqrs-gen`) be held back?
3. Is there a plan for the `example/` modules — are they versioned or not?
4. Should `replace` directives be removed before tagging, or does the tagging process handle it?
5. Is `goproxy` compatibility a concern (multi-module repos have edge cases)?

This decision fundamentally affects the `go.mod` structure and consumer onboarding. It cannot be made by a code agent.

---

## Metrics Summary

| Metric                   | Value                                              |
| ------------------------ | -------------------------------------------------- |
| Total Go source files    | 244                                                |
| Total Go test files      | 221                                                |
| Packages tested          | 29                                                 |
| Packages passing         | 29 (100%)                                          |
| `go vet` issues          | 0                                                  |
| Total code coverage      | 90.9%                                              |
| Packages > 90% coverage  | 19 of 23 measured                                  |
| ADRs                     | 6                                                  |
| Workspace modules        | 15 (in go.work) + example modules                  |
| Commits in last 48h      | 182                                                |
| Uncommitted changes      | 0                                                  |
| Pre-existing CI failures | golangci-lint exit 7, go-structure-linter opinions |

## Per-Package Coverage

| Package                       | Coverage            | Maturity        |
| ----------------------------- | ------------------- | --------------- |
| `core/pkg/id`                 | 100.0%              | ✅ Production   |
| `core/pkg/dispatcher`         | 100.0%              | ✅ Production   |
| `catalog/internal/caseutil`   | 100.0%              | ✅ Production   |
| `memory`                      | 99.8%               | 🧪 Test utility |
| `catalog/openapi`             | 98.1%               | ✅ Production   |
| `projection`                  | 98.1%               | ✅ Production   |
| `catalog/d2`                  | 97.5%               | ✅ Production   |
| `stream`                      | 96.5%               | ✅ Production   |
| `catalog/asyncapi`            | 96.3%               | ✅ Production   |
| `saga`                        | 96.1%               | ✅ Production   |
| `catalog/eventcatalog`        | 95.8%               | ✅ Production   |
| `signing`                     | 95.7%               | ✅ Production   |
| `middleware`                  | 95.0%               | ✅ Production   |
| `catalog/docserver`           | 94.9%               | ✅ Production   |
| `core/query`                  | 93.5%               | ✅ Production   |
| `storage`                     | 93.4%               | ✅ Production   |
| `core/event`                  | 93.1%               | ✅ Production   |
| `core/command`                | 92.3%               | ✅ Production   |
| `core/decider`                | 91.3%               | ✅ Production   |
| `catalog/internal/schemautil` | 89.2%               | ✅ Production   |
| `catalog`                     | 86.8%               | ✅ Production   |
| `testhelpers`                 | 85.0%               | 🧪 Test utility |
| `catalog/internal/cattest`    | 0.0%                | 💀 Dead code    |
| `otel`                        | N/A (no test files) | ⚠️ Untested     |
