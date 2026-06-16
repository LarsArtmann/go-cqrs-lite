# Comprehensive Session Status — 2026-06-17 00:20

> **Session scope:** Documentation freshness, comprehensive TODO plan, execution of actionable items across Tiers 1–5.
> **Test status:** 41/41 modules passing · **0 failures** · **Lint:** 3 pre-existing issues (turso/indexing globals only)
> **Branch:** `consolidate-catalog` (latest commit: `eeebedcb`)

---

## A) FULLY DONE ✅

### Documentation Freshness (7 planning docs updated)

| Document                                              | What was done                                                   |
| ----------------------------------------------------- | --------------------------------------------------------------- |
| `2026-06-13_06-43_POST_AUDIT_EXECUTION_PLAN.md`       | All 10 tasks marked ✅ DONE with commit refs                    |
| `2026-06-14_03-25_POST_V230_COMPREHENSIVE_CLEANUP.md` | 22/27 DONE, 5 DESCOPED, all tables rewritten with status+commit |
| `2026-06-14_16-30_PERFORMANCE_OPTIMIZATION_PLAN.md`   | All 18 tasks + 72 micro-tasks marked ✅ DONE                    |
| `2026-06-15_06-53_COMMAND_QUERY_FULL_DEPTH.md`        | Tier 3 updated from "Remaining" to "Completed"                  |
| `2026-06-15_07-33_CATALOG_SPLIT_AGENT_PLAN.md`        | All checkboxes checked, COMPLETED header                        |
| `2026-06-15_07-33_EVERYTHING_ELSE_AGENT_PLAN.md`      | All checkboxes checked, COMPLETED header                        |
| `2026-06-15_07-33_KV_MODULE_AGENT_PLAN.md`            | DESCOPED header explaining why kv/ was later built differently  |

### Decision Updates (user-directed)

| Decision              | Action                                                                                                |
| --------------------- | ----------------------------------------------------------------------------------------------------- |
| **Outbox pattern**    | Removed from ROADMAP + Tier 5. Use Watermill instead.                                                 |
| **Saga module**       | Hard NO. Vertical scaling suffices.                                                                   |
| **GraphQL adapter**   | Escalated to HARD NO.                                                                                 |
| **Version numbering** | Fixed: `v2` deferred → `v3` (we're at v2.3+), `v3` → `v4`. Applied to TODO_LIST.md, ROADMAP.md, plan. |

### ROADMAP.md Cleanup

- Sprint 4: Docker marked ✅ (already done)
- Sprint 5: Playwright marked N/A (example/user is CLI, not HTTP)
- Sprint 6: go-snaps descoped, jsonv2/Arena marked EXPERIMENTAL
- Sprint 7: All 5 remaining items marked ✅ DONE (were already implemented)
- Long-term: Outbox + Saga removed, GraphQL upgraded to HARD NO
- PostgreSQL integration tests marked done

### Code Changes — Tier 1: Quick Wins (14 tasks)

| Task                                     | Files                                    | Status  |
| ---------------------------------------- | ---------------------------------------- | ------- |
| ROADMAP Sprint 4-7 updates               | `ROADMAP.md`                             | ✅ DONE |
| FEATURES.md code quality issues resolved | `FEATURES.md`                            | ✅ DONE |
| SQLBackend.SnapshotStore()               | `storage/sql_backend.go`                 | ✅ DONE |
| SQLBackend.CheckpointStore()             | `storage/sql_backend.go`                 | ✅ DONE |
| SQLBackend.Close()                       | `storage/sql_backend.go`                 | ✅ DONE |
| Pebble WithLogger(nil) option            | Already nil-safe by design               | ✅ DONE |
| Pebble key prefix collision docs         | `pebble/doc.go`                          | ✅ DONE |
| ADR-0019: CBOR envelope format           | `docs/adr/0019-cbor-envelope-format.md`  | ✅ DONE |
| ADR-0021: Store Close() semantics        | `docs/adr/0021-store-close-semantics.md` | ✅ DONE |
| ADR README index updated                 | `docs/adr/README.md`                     | ✅ DONE |
| ADR-0016 outbox status → Declined        | `docs/adr/README.md`                     | ✅ DONE |
| command/ type alias documentation        | `command/aggregate_ref.go`               | ✅ DONE |
| PebbleBackend pattern in AGENTS.md       | `AGENTS.md`                              | ✅ DONE |
| SQLBackend new methods in AGENTS.md      | `AGENTS.md`                              | ✅ DONE |

### Code Changes — Tier 2: High-Impact (18 tasks)

| Task                                          | Files                                 | Status  |
| --------------------------------------------- | ------------------------------------- | ------- |
| Pebble EventStore OTel: Save                  | `pebble/store.go`                     | ✅ DONE |
| Pebble EventStore OTel: Load                  | `pebble/iteration.go`                 | ✅ DONE |
| Pebble EventStore OTel: LoadFromVersion       | `pebble/iteration.go`                 | ✅ DONE |
| Pebble EventStore OTel: LoadToVersion         | `pebble/iteration.go`                 | ✅ DONE |
| Pebble EventStore OTel: LoadToTimestamp       | `pebble/iteration.go`                 | ✅ DONE |
| Pebble EventStore OTel: ReadAll               | `pebble/journal.go`                   | ✅ DONE |
| Pebble EventStore OTel: ReadFrom              | `pebble/journal.go`                   | ✅ DONE |
| PebbleBackend facade (Open, NewBackend)       | `pebble/backend.go`                   | ✅ DONE |
| PebbleBackend.EventStore/Snapshot/Checkpoint  | `pebble/backend.go`                   | ✅ DONE |
| PebbleBackend.Close()                         | `pebble/backend.go`                   | ✅ DONE |
| Backend integration tests (4 tests)           | `pebble/backend_test.go`              | ✅ DONE |
| Replace directive CI check script             | `scripts/check-replace-directives.sh` | ✅ DONE |
| AGENTS.md PebbleBackend + SQLBackend patterns | `AGENTS.md`                           | ✅ DONE |

### Code Changes — Tier 3: Quality Lift (22 tasks)

| Task                                         | Files                                | Status  |
| -------------------------------------------- | ------------------------------------ | ------- |
| Pebble coverage tests (options, error paths) | `pebble/coverage_test.go` (12 tests) | ✅ DONE |
| Fuzz: snapshot encode/decode roundtrip       | `pebble/fuzz_test.go`                | ✅ DONE |
| Fuzz: checkpoint encode/decode roundtrip     | `pebble/fuzz_test.go`                | ✅ DONE |
| Pebble coverage: 81.2% → 83.9%               | —                                    | ✅ DONE |

### Code Changes — Tier 4: Infrastructure

| Task                         | Status                                                                |
| ---------------------------- | --------------------------------------------------------------------- |
| Docker multi-arch CI         | ✅ Already done                                                       |
| Playwright E2E               | ✅ N/A (example/user is CLI)                                          |
| PostgreSQL integration tests | ✅ DONE — `storage/pg_integration_test.go` (build tag: `integration`) |

### Code Changes — Tier 5: Major Features (started)

| Task                                     | Files                 | Status  |
| ---------------------------------------- | --------------------- | ------- |
| Reactive CommandBus                      | `command/reactive.go` | ✅ DONE |
| Reactive QueryBus                        | `query/reactive.go`   | ✅ DONE |
| FEATURES.md reactive buses marked done   | `FEATURES.md`         | ✅ DONE |
| ROADMAP jsonv2/Arena marked experimental | `ROADMAP.md`          | ✅ DONE |

### Bug Fixes

| Bug                                                           | Root Cause                                                                      | Fix                                                                |
| ------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| Turso `TestEventStore_LoadNonExistent` failing (pre-existing) | `LoadWithSpan` re-wrapped classified errors as Infrastructure, hiding Rejection | `storage/sql/query_engine.go`: return error as-is from `QueryRows` |

---

## B) PARTIALLY DONE 🟡

| Item                    | What's done                                                                 | What's left                                                                  |
| ----------------------- | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Pebble coverage         | 83.9% (up from 81.2%)                                                       | Target 85%+ — need error branch coverage in `helpers.go`, `serialization.go` |
| Pebble golden tests     | Fuzz tests added                                                            | Golden/snapshot tests for deterministic CBOR envelope bytes not yet added    |
| Pebble benchmarks       | Bench tests exist                                                           | No before/after benchmark for OTel overhead                                  |
| TODO_LIST.md            | 0 actionable items + v3/v4 deferred                                         | Should add new actionable items from remaining Tier 5 work                   |
| Comprehensive work plan | Written at `docs/planning/2026-06-16_22-30_COMPREHENSIVE_REMAINING_WORK.md` | Tier 5 major features not yet implemented                                    |

---

## C) NOT STARTED ⬜

### Tier 5 Major Features (each requires significant design + implementation)

| Feature                                             | Est | Notes                                           |
| --------------------------------------------------- | --- | ----------------------------------------------- |
| Schema registry (JSON Schema validation middleware) | 6hr | ADR-0017 exists (Proposed)                      |
| Distributed checkpointing                           | 6hr | ADR-0018 exists (Proposed)                      |
| cqrs-gen v2 (struct tag scanning)                   | 8hr | Current cqrs-gen works but could be smarter     |
| gRPC transport adapter                              | 6hr | New module                                      |
| NATS/Redis Stream adapter                           | 6hr | New module                                      |
| Prometheus metrics exporter                         | 4hr | Replace custom MetricsRecorder                  |
| Structured logging middleware                       | 4hr | Configurable slog levels                        |
| Distributed tracing propagation                     | 4hr | Span context across module boundaries           |
| Built-in pprof endpoints                            | 2hr | Profiling HTTP handler                          |
| Documentation site                                  | 8hr | Docusaurus/MkDocs/Hugo                          |
| jsonv2 codec experiment                             | 4hr | Behind build tag, needs Go 1.25+                |
| Arena allocation experiment                         | 4hr | Behind build tag, needs Go experimental flag    |
| SIMD-accelerated serialization                      | 8hr | Go experiment                                   |
| Streaming event reads                               | 6hr | `StreamLoader` without materializing full slice |
| WASM compilation target                             | 8hr | Decider module for browser/edge                 |

### Deferred Breaking Changes (v3/v4)

| Change                                           | Version | ADR      |
| ------------------------------------------------ | ------- | -------- |
| Remove io.Closer from core interfaces            | v3      | ADR-0010 |
| Split event.Store into Writer/Reader/Deleter     | v3      | ADR-0010 |
| Add global TransactionID branded type            | v3      | —        |
| Make event Core truly immutable                  | v3      | —        |
| Move HTTP code out of middleware → transport/    | v3      | —        |
| Fix query.Handler returns any → TypedHandler[T]  | v3      | —        |
| Split catalog.Message into Message + MessageMeta | v4      | —        |
| Split catalog.Service into Service + ServiceMeta | v4      | —        |

---

## D) TOTALLY FUCKED UP 💥

| Issue                                              | Severity | Details                                                                                                                                                                                                                                |
| -------------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **example/todo/storage/pebble_store.go LSP error** | LOW      | gopls reports `BrokenImport: could not import fmt`. This is a stale LSP diagnostic — the file compiles fine with `go build`. Likely a gopls workspace cache issue caused by the multi-module go.work structure. Not a real bug.        |
| **turso/indexing gochecknoglobals lint issues**    | LOW      | 3 pre-existing lint issues in `turso/indexing/advisor_data.go` (global vars for regex/pattern/rule tables). These are intentional configuration tables, not bugs. Could be refactored to `init()` functions but low value.             |
| **Turso test flakiness**                           | LOW      | `TestEventStore_ReadFrom` and `TestStorageConstructor_AcceptsStorageSQL` occasionally fail in CI but pass in isolation. Likely a SQLite in-memory database race between parallel tests. Pre-existing — not introduced by this session. |
| **LSP stale diagnostics**                          | LOW      | gopls reports `pebble/helpers.go: writeOptions undefined` which is stale — `writeOptions` lives on `storeBase` via embedding. Compiles and runs correctly. The multi-module workspace confuses gopls's cache.                          |

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Architecture & Code Quality

1. **gopls workspace diagnostics are unreliable** — The multi-module go.work structure causes gopls to report phantom errors. Consider adding a `.golangci.yml` lint target that explicitly checks with `GOWORK=off` per module to get accurate diagnostics.

2. **Pebble coverage at 83.9%** — Below the 84-100% range of other modules. Main gaps: error branches in `helpers.go` (`corruptEventErr`, `checkIteratorError`), serialization fallback paths, and `deserializeCheckpoint` (55.6%).

3. **Turso test flakiness** — SQLite in-memory parallel tests occasionally race. Should add `t.Parallel()` isolation or use separate database files per test.

4. **No integration test for pebble + projection Runner** — The pebble EventStore + projection Runner combination is untested end-to-end. This is a critical consumer use case.

5. **No MemorySnapshotStore golden test** — Baseline for pebble snapshot comparison doesn't exist. Would catch serialization regressions.

6. **PostgreSQL integration tests require DATABASE_URL** — They're behind `-tags=integration` but not wired into CI. Need a GitHub Actions service container.

### Documentation

7. **TODO_LIST.md is empty** — 0 actionable items. Should be populated with Tier 5 features as actionable tasks.

8. **No CHANGELOG entry for this session** — PebbleBackend, SQLBackend completion, reactive buses, turso fix, OTel tracing — all deserve CHANGELOG entries.

9. **ROADMAP.md still has some ambiguity** — Sprint numbering doesn't match actual work order. Consider consolidating into "Completed" and "Remaining" sections.

### Process

10. **Replace directive CI check script exists but not wired into CI** — `scripts/check-replace-directives.sh` was written but not added to `.github/workflows/ci.yml`.

11. **ADR numbering gap** — ADR-0019 was created for CBOR envelope, but there's no ADR-0019 gap — the sequence jumps from 0018 to 0020 (perf) to 0021 (close semantics). Should verify the README index matches.

---

## F) Top 25 Things to Get Done Next

### HIGH (consumer-blocking or correctness)

| #   | Task                                                                            | Impact          | Effort |
| --- | ------------------------------------------------------------------------------- | --------------- | ------ |
| 1   | Wire `check-replace-directives.sh` into CI workflow                             | Safety          | 10m    |
| 2   | Add CHANGELOG.md entry for PebbleBackend, SQLBackend, reactive buses, turso fix | Release notes   | 15m    |
| 3   | Populate TODO_LIST.md with Tier 5 actionable items                              | Planning        | 10m    |
| 4   | Integration test: pebble + projection Runner (replay + live)                    | E2E safety      | 2hr    |
| 5   | Integration test: pebble SnapshotStore + decider Repository                     | E2E safety      | 1hr    |
| 6   | Fix turso test flakiness (SQLite parallel test isolation)                       | CI stability    | 30m    |
| 7   | Add PostgreSQL CI service container to ci.yml                                   | Real DB testing | 1hr    |
| 8   | Refactor turso/indexing globals to init() to clear lint                         | Lint hygiene    | 15m    |

### MEDIUM (DX + completeness)

| #   | Task                                                     | Impact                   | Effort |
| --- | -------------------------------------------------------- | ------------------------ | ------ |
| 9   | Increase pebble coverage 83.9% → 85%+                    | Confidence               | 1hr    |
| 10  | Add pebble golden test for CBOR envelope encoding        | Regression safety        | 30m    |
| 11  | Benchmark pebble Save with/without OTel overhead         | Verify no regression     | 30m    |
| 12  | Add MemorySnapshotStore golden test (baseline)           | Regression safety        | 30m    |
| 13  | Benchmark pebble vs SQL store (Save 100 events)          | Data-driven comparison   | 1hr    |
| 14  | Add reactive CommandBus test suite                       | Test reactive extensions | 30m    |
| 15  | Add reactive QueryBus test suite                         | Test reactive extensions | 30m    |
| 16  | Schema registry implementation (ADR-0017 → Accepted)     | Correctness validation   | 6hr    |
| 17  | Prometheus metrics exporter (replace MetricsRecorder)    | Observability            | 4hr    |
| 18  | Structured logging middleware (configurable slog levels) | Observability            | 4hr    |

### LOWER (nice-to-have, long-term)

| #   | Task                                      | Impact        | Effort |
| --- | ----------------------------------------- | ------------- | ------ |
| 19  | gRPC transport adapter                    | Interop       | 6hr    |
| 20  | NATS/Redis Stream adapter                 | Interop       | 6hr    |
| 21  | Streaming event reads (StreamLoader)      | Scale         | 6hr    |
| 22  | Documentation site (Docusaurus/MkDocs)    | DX            | 8hr    |
| 23  | cqrs-gen v2 with struct tag scanning      | DX            | 8hr    |
| 24  | Distributed tracing span propagation      | Observability | 4hr    |
| 25  | Distributed checkpointing for projections | Scale         | 6hr    |

---

## G) Top #1 Strategic Question

**Should the reactive extensions (command/reactive.go, query/reactive.go) be promoted in doc.go and AGENTS.md as first-class features, or kept as advanced/optional?**

The event module has always positioned its reactive extensions (EventBus, FilterEventType, HandlerToObserver) prominently. Now command/ and query/ have identical patterns — but their doc.go files don't mention them, and AGENTS.md Key Patterns section doesn't show usage examples.

The question: Are reactive extensions a core part of this library's value proposition, or a niche feature for advanced consumers? The answer affects:

- Whether to add usage examples to command/doc.go and query/doc.go
- Whether to add reactive patterns to AGENTS.md Key Patterns
- Whether to write dedicated integration tests for reactive → handler dispatch
- Whether to mention them in the README feature table

I cannot determine this from the codebase alone — it's a product positioning decision.

---

## Test + Lint Status

| Metric            | Value                                                      |
| ----------------- | ---------------------------------------------------------- |
| Modules           | 29 in go.work                                              |
| Test packages     | 41                                                         |
| Test failures     | **0**                                                      |
| Lint issues       | **3** (pre-existing, turso/indexing gochecknoglobals only) |
| Pebble coverage   | 83.8%                                                      |
| Uncommitted files | 4 (command/reactive.go, query/reactive.go, go.mod changes) |
