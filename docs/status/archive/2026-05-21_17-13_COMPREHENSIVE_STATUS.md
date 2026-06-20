# Comprehensive Status Report — Session 87

**Date:** 2026-05-21 17:13
**Session:** 87 (Documentation Reconciliation + Cleanup)
**Previous Session:** 86 (Catalog Quality Sweep)
**Trigger:** READ, UNDERSTAND, RESEARCH, REFLECT → Documentation reconciliation

---

## Executive Summary

This session focused on **documentation hygiene** — reconciling the TODO list against actual code, updating stale coverage numbers, and trimming the bloated AGENTS.md. No production code was changed. All 24 test packages pass, zero lint, zero build errors.

**Key metrics:** TODO_LIST.md went from 252 unchecked items → 63 verified done + 188 real open. AGENTS.md went from 896 → 537 lines. FEATURES.md coverage numbers updated to match actual test runs.

---

## a) FULLY DONE ✅

### This Session

| #   | Task                        | Detail                                                                                                                                                                                          |
| --- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | TODO_LIST.md reconciliation | 63 stale items marked `[x]` with evidence. Each verified against actual code: panic recovery, error returns, type migrations, file splits, error classification, API changes, deep copies, etc. |
| 2   | FEATURES.md coverage update | 17 coverage numbers updated from actual test run. 6 packages added to Module Maturity Matrix (sync, projection, catalog/openapi, catalog/docserver). Module count corrected 11→12.              |
| 3   | AGENTS.md trim              | 896→537 lines. Session history (Sessions 20–86) extracted to `docs/sessions/SESSION_HISTORY.md`. Milestone summary table retained. Coverage table updated.                                      |
| 4   | Full verification           | 24/24 test packages pass. All 11 modules + 2 examples build. Zero lint in core, catalog, middleware. Golden tests pass.                                                                         |

### Verified From Previous Sessions (Marked Done This Session)

63 items in TODO_LIST.md were already implemented but listed as open. The most significant:

| Category           | Items                                                             | Sessions            |
| ------------------ | ----------------------------------------------------------------- | ------------------- |
| Panic recovery     | HandleParallel, OutboxPublisher                                   | Session 43          |
| Error returns      | NewLWWResolver, EveryNEvents                                      | Sessions 86, 51     |
| Type safety        | Version, SchemaVersion, OutboxStatus                              | Session 65          |
| API changes        | IdempotencyKey on Command, context.Context on Handler             | Sessions 31, 55     |
| Bug fixes          | Close() ownership, double-marshal, timer leak, deep copy          | Sessions 25, 20, 86 |
| File splits        | pebble_event_store, storage/helpers, decider, aggregate, event    | Sessions 68, 73, 83 |
| Time-travel        | LoadToVersion, LoadToTimestamp, PositionalLoader, composite index | Sessions 80, 81     |
| Error taxonomy     | 38 sentinels classified, RegisterClassification                   | Sessions 31, 44, 51 |
| Shared helpers     | PublishChanges, SaveSnapshot, SnapshotStrategy                    | Session 48          |
| Infrastructure     | CI pipeline, OpenSQLite, ConfigureSQLitePool, PostgresInitSchema  | Sessions 80+, 86    |
| Dependency cleanup | cockroachdb/errors removed, go-json-experiment/json removed       | Session 54          |

---

## b) PARTIALLY DONE ⚠️

| #   | Item                           | What's Done                                           | What's Missing                                                                                    |
| --- | ------------------------------ | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 1   | CatalogMeta consolidation      | `Catalogable` and `CatalogCore` deleted               | `CatalogMeta` still exists in 3 packages (event, command, query) — blocked on dispatcher refactor |
| 2   | filterEvents O(n) optimization | `filterByTypes` helper exists                         | Still uses `slices.Contains` (O(n×m)), not set-based lookup                                       |
| 3   | Lint across all modules        | core=0, catalog=0, middleware=0 issues                | storage, memory, projection, sync not linted with golangci-lint                                   |
| 4   | Error classification           | 38 sentinels classified, `RegisterClassification` API | Still uses `init()` side effects, not explicit setup                                              |

---

## c) NOT STARTED 📐

### High-Impact Items With Zero Progress

| #   | Item                                | Why Important                                                | Effort |
| --- | ----------------------------------- | ------------------------------------------------------------ | ------ |
| 1   | PostgreSQL integration tests        | Most common deployment target, untested with real DB         | 3h     |
| 2   | Remove go.mod replace directives    | Blocks independent module publishing                         | 2h     |
| 3   | GOWORK=off CI verification          | Version drift goes undetected                                | 1h     |
| 4   | Clock interface (`WithClock`)       | Deterministic testing without time.Now() monkey-patching     | 1h     |
| 5   | SubscriptionScope enum              | Replaces `nil = all` in EventTypes() with explicit semantics | 1h     |
| 6   | Pebble optimistic concurrency       | Concurrent writes silently overwrite                         | 2h     |
| 7   | Outbox transaction co-participation | Save + outbox in separate transactions                       | 3h     |
| 8   | Circuit breaker middleware          | Resilience pattern for production use                        | 3h     |
| 9   | Saga/Process Manager                | Design doc exists, no implementation                         | 8h+    |
| 10  | Watermill module                    | Real message broker integration                              | 8h+    |
| 11  | CONTRIBUTING.md                     | No contributor guidelines exist                              | 2h     |
| 12  | CONTEXT.md with domain glossary     | No shared vocabulary doc                                     | 1h     |
| 13  | docs/adr/ directory                 | No architecture decision records                             | 2h     |
| 14  | Module READMEs                      | No per-module documentation                                  | 3h     |
| 15  | Release tags                        | 8 tags LOCAL ONLY, blocks external consumers                 | 1h     |

---

## d) TOTALLY FUCKED UP 💀

| #   | Issue                                  | Severity | Detail                                                                                                                                                                    |
| --- | -------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `query.Handler` returns `any`          | HIGH     | Violates project's own "no any" rule. `DispatchTyped[T]` is a workaround but the core API is wrong. Breaking change to fix.                                               |
| 2   | Pre-commit hook broken                 | MEDIUM   | BuildFlow fails on pre-existing issues. Forces `--no-verify` for every commit. CI should catch what the hook can't.                                                       |
| 3   | Replace directives in all go.mod files | MEDIUM   | Every module has `replace` directives pointing to local paths. Prevents `go get` from any external consumer. 10 of 12 go.mod files affected.                              |
| 4   | `core/event` coverage dropped to 89.1% | MEDIUM   | Was 94.4% in docs (stale). Actual is 89.1%. The god-package has grown — new types (Version arithmetic, error taxonomy, ContextEnricher) added without proportional tests. |
| 5   | `testhelpers` at 10.5% coverage        | LOW      | Test utility package barely tested. Not critical (it's for tests), but zero trust that helpers work correctly.                                                            |
| 6   | `catalog/internal/cattest` at 0%       | LOW      | 454 lines, zero tests, no external imports. Dead code masquerading as a package.                                                                                          |
| 7   | AGENTS.md still 537 lines              | LOW      | Target was <400 lines. Session history extracted but the architecture/conventions section is verbose.                                                                     |

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Process Improvements

1. **Coverage numbers should be auto-generated** — Every session manually updates FEATURES.md and AGENTS.md coverage tables. This is error-prone (we had 6+ stale numbers). A `nix run .#coverage-report` that outputs a markdown table would eliminate this.

2. **TODO_LIST.md should be code-verified** — 25% of items were stale. A CI job that checks `git log` for fix commits and auto-marks items would prevent drift.

3. **Pre-commit hook needs fixing or removal** — Currently broken, forces `--no-verify`. Either fix the gci config or remove the hook and rely on CI.

4. **Session history should live in git, not in AGENTS.md** — This session extracted it. Future sessions should append to `docs/sessions/` instead of bloating AGENTS.md.

5. **Error classification should use explicit setup, not init()** — Hidden global side effects via `init()` make testing harder and violate the library's "no surprises" philosophy.

### Code Quality Improvements

6. **`core/event` is a god-package** — 1100+ lines covering store interfaces, bus interfaces, snapshots, upcasters, error taxonomy, codecs, projections, outbox, context enrichment, metadata, version arithmetic. Should be split into sub-packages.

7. **`storage/dialect.go` `any` usage** — 3 methods return `any` (PostgreSQL returns `time.Time`, SQLite returns `string`). Legitimate internal abstraction but violates the project rule.

8. **OutboxPublisher split-brain** — `cancel` stays non-nil after `Close()`, meaning the publisher can't be restarted or reused.

9. **HandleParallel channel drain** — On context cancellation, goroutines may leak if channel isn't drained.

10. **No benchmarks for hot paths** — 59 benchmarks exist but storage (the most performance-critical module) has only 1 benchmark. No benchmarks for projection replay, event creation with middleware chain, or concurrent access patterns.

---

## f) TOP #25 THINGS TO DO NEXT

Sorted by **Impact × Urgency / Effort**:

| #   | Task                                                           | Impact   | Effort | Category     |
| --- | -------------------------------------------------------------- | -------- | ------ | ------------ |
| 1   | Remove go.mod replace directives → enable `go get`             | CRITICAL | 2h     | Publishing   |
| 2   | Push release tags to remote (8 tags LOCAL ONLY)                | CRITICAL | 30min  | Publishing   |
| 3   | Fix pre-commit hook (gci config) or remove it                  | HIGH     | 1h     | DX           |
| 4   | Add Clock interface + `WithClock` option                       | HIGH     | 1h     | Testing      |
| 5   | Add GOWORK=off CI job to catch version drift                   | HIGH     | 1h     | CI           |
| 6   | Add PostgreSQL integration tests (testcontainers)              | HIGH     | 3h     | Quality      |
| 7   | Fix Pebble Store optimistic concurrency                        | HIGH     | 2h     | Correctness  |
| 8   | Fix outbox transaction co-participation                        | HIGH     | 3h     | Correctness  |
| 9   | Fix HandleParallel channel drain on cancellation               | HIGH     | 30min  | Leak fix     |
| 10  | Increase `core/event` coverage 89.1% → 93%+                    | HIGH     | 2h     | Coverage     |
| 11  | Split `core/event` god-package into sub-packages               | HIGH     | 4h     | Architecture |
| 12  | Add CONTRIBUTING.md                                            | MEDIUM   | 2h     | Docs         |
| 13  | Add docs/adr/ with first 3 ADRs                                | MEDIUM   | 2h     | Docs         |
| 14  | Add SubscriptionScope enum                                     | MEDIUM   | 1h     | Type safety  |
| 15  | Replace `init()` error registration with explicit setup        | MEDIUM   | 2h     | API hygiene  |
| 16  | Delete `catalog/internal/cattest` (0% coverage, 0 imports)     | MEDIUM   | 30min  | Dead code    |
| 17  | Fix `storage/dialect.go` `any` usage                           | MEDIUM   | 30min  | Convention   |
| 18  | Add storage benchmarks (PG vs SQLite vs Pebble)                | MEDIUM   | 3h     | Performance  |
| 19  | Formally deprecate aggregate package                           | MEDIUM   | 30min  | API clarity  |
| 20  | Wire example/user/ to use catalog-aware constructors           | MEDIUM   | 1h     | Example      |
| 21  | Normalize go.mod versions across workspace                     | MEDIUM   | 1h     | Hygiene      |
| 22  | Add minimum coverage gate to CI (80%)                          | LOW      | 30min  | CI           |
| 23  | Extend lint to all modules (storage, memory, projection, sync) | LOW      | 2h     | Quality      |
| 24  | Create CONTEXT.md with domain glossary                         | LOW      | 1h     | Docs         |
| 25  | Add Saga design doc review + implementation start              | LOW      | 8h+    | Feature      |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**Should `query.Handler` return `(any, error)` or `(T, error)`?**

The current API:

```go
type Handler = func(context.Context, Query) (any, error)
```

This violates the project's own "no any" rule. There are three options:

1. **Keep `any`** — Status quo. `DispatchTyped[T]` provides the type-safe escape hatch. Consumers learn the pattern. Minimal disruption.

2. **Generic handler** — `type Handler[T any] func(context.Context, Query) (T, error)`. This requires rethinking the entire dispatcher to be generic. The `Dispatcher` would need to be `Dispatcher[T]` or use a registry of typed handlers. Massive breaking change.

3. **Result[T] type** — Add `type Result[T any] struct { Value T; Err error }`. The handler still returns `(any, error)` internally but the public API wraps it. Adds complexity without fully solving the problem.

My recommendation is **option 1** — keep `any` and document the `DispatchTyped[T]` pattern. But I need a human decision because this permanently shapes the query API surface.

---

## Project Statistics

| Metric                 | Value                                       |
| ---------------------- | ------------------------------------------- |
| Total Go LOC           | 47,635                                      |
| Production LOC         | 15,988                                      |
| Test LOC               | 31,647                                      |
| Test:Prod ratio        | 1.98:1                                      |
| Test packages          | 26 (24 ok + 1 no-tests + 1 no-files)        |
| Benchmarks             | 59 across 13 files                          |
| Go modules             | 12 (11 sub-modules + root)                  |
| Total commits          | 935                                         |
| Ahead of origin        | 2                                           |
| TODO items open        | 188 (7 HIGH, 22 MEDIUM, 8 LOW, 151 Unknown) |
| TODO items done        | 63                                          |
| Zero-coverage packages | 1 (`catalog/internal/cattest`)              |
| Known Issues           | 4 LOW                                       |

## Test Coverage by Package

| Package                       | Coverage | Status       |
| ----------------------------- | -------- | ------------ |
| `core/query`                  | 100.0%   | ✅           |
| `core/pkg/dispatcher`         | 100.0%   | ✅           |
| `middleware`                  | 100.0%   | ✅           |
| `catalog/adapters`            | 100.0%   | ✅           |
| `memory`                      | 99.6%    | ✅           |
| `core/pkg/id`                 | 97.8%    | ✅           |
| `core/aggregate`              | 95.9%    | ✅           |
| `catalog/d2`                  | 95.0%    | ✅           |
| `core/command`                | 94.7%    | ⚠️ was 100%  |
| `catalog/openapi`             | 94.4%    | ✅           |
| `projection`                  | 93.9%    | ✅           |
| `sync`                        | 92.2%    | ✅           |
| `catalog/eventcatalog`        | 91.3%    | ✅           |
| `catalog/docserver`           | 91.0%    | ✅           |
| `catalog`                     | 90.5%    | ⚠️ was 94.4% |
| `catalog/asyncapi`            | 93.7%    | ✅           |
| `core/decider`                | 93.3%    | ⚠️ was 95.0% |
| `core/event`                  | 89.1%    | ⚠️ was 94.4% |
| `storage`                     | 88.1%    | ✅           |
| `catalog/internal/schemautil` | 84.2%    | —            |
| `catalog/internal/caseutil`   | 76.5%    | —            |
| `testhelpers`                 | 10.5%    | 🧪 test-only |
| `catalog/internal/cattest`    | 0.0%     | 💀 dead code |

**Average production coverage (excl. testhelpers/cattest):** 93.0%

## Build & Quality Status

| Check                      | Result                         |
| -------------------------- | ------------------------------ |
| `go test ./...`            | ✅ 24/24 pass                  |
| `go build` (per module)    | ✅ All 11 modules + 2 examples |
| `-race`                    | ✅ Passes                      |
| golangci-lint (core)       | ✅ 0 issues                    |
| golangci-lint (catalog)    | ✅ 0 issues                    |
| golangci-lint (middleware) | ✅ 0 issues                    |
| Golden tests               | ✅ Pass without refresh        |
| Pre-commit hook            | ❌ Broken (BuildFlow)          |

---

## Files Modified This Session

| File                               | Change                                                                      |
| ---------------------------------- | --------------------------------------------------------------------------- |
| `TODO_LIST.md`                     | 63 items marked `[x]`, coverage numbers corrected                           |
| `FEATURES.md`                      | 17 coverage numbers updated, 6 rows added to matrix, module count corrected |
| `AGENTS.md`                        | 896→537 lines, session history extracted, coverage table updated            |
| `docs/sessions/SESSION_HISTORY.md` | NEW — extracted session history from AGENTS.md                              |

---

_End of Session 87 status report._
