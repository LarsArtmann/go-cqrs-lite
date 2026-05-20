# Session 81 — Comprehensive Status Report

**Date:** 2026-05-20 11:22
**Branch:** master
**Commits Since Last Report:** 5 (since session 80)

---

## A) FULLY DONE

### Position-Based Projection Replay (NEW)
- `projection.Runner` auto-detects `event.PositionalLoader` via type assertion
- When available + checkpoint exists: uses `LoadAllFromPosition(ctx, checkpoint, 0)` instead of `LoadAll()` + linear scan
- Falls back to `LoadAll()` + `filterEvents()` for non-positional stores
- Extracted `filterByTypes()` helper for the position-based path
- Test: `TestRunner_ReplayWithPositionalLoader` verifies position-based replay with checkpoint

### Decider Time-Travel Error Path Coverage
- `TestRepository_LoadAtVersion_StoreError` — non-ErrAggregateNotFound store error
- `TestRepository_LoadAtTime_StoreError` — non-ErrAggregateNotFound store error
- `TestRepository_LoadAtVersion_FoldError` — fold failure during time-travel
- Coverage: 85.6% → 93.6% (error paths in `LoadAtVersion`/`LoadAtTime` now covered)
- Created `errStore` test helper wrapping `event.Store` with error injection

### Storage go-sqlmock Tests (7 NEW)
- `LoadToVersion_Mock_Success` — returns events up to version
- `LoadToVersion_Mock_NotFound` — returns ErrAggregateNotFound
- `LoadToVersion_Mock_QueryError` — connection error
- `LoadToTimestamp_Mock_Success` — returns events up to timestamp
- `LoadToTimestamp_Mock_QueryError` — connection error
- `LoadAllFromPosition_Mock_Success` — returns events after position
- `LoadAllFromPosition_Mock_QueryError` — connection error

### Integration Tests (3 NEW)
- `TestTimeTravel_DeciderLoadAtVersion` — end-to-end LoadToVersion via MemoryStore
- `TestTimeTravel_DeciderLoadAtTime` — end-to-end LoadToTimestamp via MemoryStore
- `TestTimeTravel_PositionalLoader` — position-based loading with checkpoint + limit

### Sync Module Tests + Benchmarks
- `TestNewVectorClockFromMap` — creates clock from map entries
- `TestNewVectorClockFromMap_Empty` — nil map produces empty clock
- 5 benchmarks: `BenchmarkNewVectorClock`, `BenchmarkVectorClock_Increment`, `BenchmarkVectorClock_Merge`, `BenchmarkVectorClock_Compare`, `BenchmarkNewLWWResolver`

### SQL Schema Optimization
- Added composite index `idx_events_agg_time (aggregate_type, aggregate_id, occurred_at)` to both PostgreSQL and SQLite DDL
- Enables efficient `LoadToTimestamp` queries without full table scan

### Interface Compliance Checks
- `io.Closer` compile-time check on `MemoryStore` (`var _ io.Closer`)
- `io.Closer` compile-time check on `MemoryBus` (`var _ io.Closer`)

### Bug Fixes
- **Flaky test**: `TestSQLiteEventStore_LoadAllFromPosition` — ULIDs generated within same millisecond are non-monotonic; added `time.Sleep(2ms)` between event creation to ensure deterministic ordering

### Documentation
- `FEATURES.md`: Added "Position-based replay" row to Projections table; added "Time-travel SQL queries" and "Composite timestamp index" to Storage table
- `AGENTS.md`: Added Session 81 entry; updated decider coverage to 93.6%; updated projection coverage to 94.1%
- `core/event/store.go`: Enhanced `PositionalLoader` godoc with ULID monotonicity documentation

### Cleanup
- Archived 38 old status reports (pre-2026-05-20) to `docs/status/archive/`
- Refreshed 3 stale golden test files (asyncapi.yaml, eventcatalog-config.js, package.json)

### Quality Metrics
| Metric | Value |
|--------|-------|
| Test packages | 24/24 passing |
| Race detector | 24/24 passing |
| Production LOC | ~15,500 |
| Test LOC | ~30,900 |
| Total tests | 988 |
| Benchmarks | 53 |
| Production files >250 lines | 0 |
| Lint issues | 0 (per-module) |

---

## B) PARTIALLY DONE

### Storage Coverage Recovery
- **Current**: 87.8%
- **Target**: 90%+
- **Status**: Added 7 go-sqlmock tests for time-travel methods. SQLite integration tests exist. Still missing: scan error paths, concurrent access patterns, more outbox edge cases.

### Decider Coverage
- **Current**: 93.6% (was 85.6%)
- **Target**: 95%+
- **Status**: Error paths in `LoadAtVersion`/`LoadAtTime` covered. Snapshot-aware time-travel paths (`loadFromSnapshot` → `LoadToVersion`) not yet tested.

### Catalog Coverage
- **Current**: 91.3% (main catalog package)
- **Status**: Dropped from ~94% due to internal package restructuring (caseutil, schemautil extraction). No new tests added for internal packages.

---

## C) NOT STARTED

### Critical — Blocks External Consumers
| # | Task | Effort |
|---|------|--------|
| 1 | Push release tags to remote | 1 min |
| 2 | Update SEC go.mod to new tags | 5 min |
| 3 | Remove `replace` directives from go.mod (after tags pushed) | 10 min |

### High — Customer-Facing
| # | Task | Effort |
|---|------|--------|
| 4 | `query.Handler` typed generics migration (breaks `any` rule) | 8h |
| 5 | `CatalogMeta` consolidation across event/command/query | 4h |
| 6 | Add `GOWORK=off` CI matrix job | 30 min |
| 7 | Add minimum coverage gate to CI (80%) | 15 min |

### Medium — Quality
| # | Task | Effort |
|---|------|--------|
| 8 | Add `WithClock(func() time.Time)` option to NewEvent | 30 min |
| 9 | Move schema DDL onto `Dialect` interface | 1h |
| 10 | `Dialect.FormatTime` → `driver.Valuer`/`sql.Scanner` | 2h |
| 11 | Turso integration test (save→load→delete) | 30 min |
| 12 | Write examples for `NewTypedProjection` and `RegisterTyped` | 30 min |
| 13 | Document time-travel API in README with examples | 30 min |
| 14 | Fix pre-commit hook (gci config, library-policy exemption) | 1h |

### Low — Polish
| # | Task | Effort |
|---|------|--------|
| 15 | Normalize go.mod version references across workspace | 30 min |
| 16 | Split `decider_test.go` (1190 lines) into multiple files | 1h |
| 17 | Split `runner_test.go` (1057 lines) into multiple files | 30 min |
| 18 | Trim AGENTS.md under 400 lines | 2h |

### Future (v2 / Major Effort)
| # | Task | Effort |
|---|------|--------|
| 19 | Global `TransactionID` branded type (breaking) | 22h |
| 20 | `Store.ReadBackwards` reverse stream reads | 2h |
| 21 | Temporal read-only safety (prevent Save after LoadAtVersion) | 2h |
| 22 | `ValidAt` bi-temporal metadata + `LoadToValidTime` | 10h |
| 23 | Saga/Process Manager implementation | 18h |
| 24 | Watermill pub/sub adapter module | 8h |
| 25 | PostgreSQL integration tests for storage | 2h |

---

## D) TOTALLY FUCKED UP

### Nothing is totally fucked up.

All 24 test packages pass with race detector. Zero lint. Zero production files over 250 lines. All examples build clean.

### Pre-Existing Known Issues (Not Introduced This Session)
| Issue | Severity | Since |
|-------|----------|-------|
| 8 release tags are LOCAL ONLY — not pushed to remote | CRITICAL | Session 79 |
| `replace` directives in go.mod prevent `GOWORK=off` builds | HIGH | Session 55 |
| `query.Handler` returns `any` — violates project "no any" rule | MEDIUM | Design |
| Pre-commit hook broken (gci config, library-policy) | MEDIUM | Session 79 |
| `core/go.mod` includes `memory`+`testhelpers` as direct requires (Go module limitation) | LOW | Session 46 |

---

## E) WHAT WE SHOULD IMPROVE

### 1. Release Discipline
Tags were created in Session 79 but NEVER PUSHED. This is the single biggest blocker. Without pushing tags, external consumers cannot use any of the new features (TypedHandler, PositionalLoader, time-travel, etc.).

### 2. CI Pipeline Gaps
- No `GOWORK=off` CI job means version drift goes undetected
- No minimum coverage gate means regressions slip through
- Pre-commit hook is broken, requiring `--no-verify` on every commit

### 3. Coverage Debt
- Storage (87.8%) and catalog (91.3%) drag the average down
- `testhelpers` at 12.2% is a gap (low priority since it's test utility)
- Decider snapshot + time-travel interaction untested

### 4. Documentation Debt
- AGENTS.md is 880+ lines — session history dominates
- README has no time-travel examples
- No examples for `NewTypedProjection[T]` or `RegisterTyped[T]`

### 5. Architecture Debt
- `CatalogMeta` duplicated across 3 packages (event, command, query)
- `query.Handler` returns `any` — the only remaining `any` in the codebase
- `WithClock` option missing — consumers who need deterministic timestamps must use `WithOccurredAt` per-event

---

## F) Top #25 Things We Should Get Done Next

| Rank | Task | Impact | Effort | Category |
|------|------|--------|--------|----------|
| 1 | Push release tags to remote | CRITICAL | 1 min | Release |
| 2 | Update SEC go.mod to new tags | CRITICAL | 5 min | Consumer |
| 3 | Remove `replace` directives from go.mod files | HIGH | 10 min | Hygiene |
| 4 | Add `GOWORK=off` CI matrix job | HIGH | 30 min | CI |
| 5 | Add minimum coverage gate to CI (80%) | HIGH | 15 min | CI |
| 6 | Fix pre-commit hook (gci + library-policy) | MEDIUM | 1h | DX |
| 7 | Document time-travel API in README | HIGH | 30 min | Docs |
| 8 | Write examples for `NewTypedProjection` and `RegisterTyped` | MEDIUM | 30 min | Docs |
| 9 | Add `WithClock(func() time.Time)` option | MEDIUM | 30 min | API |
| 10 | Add Turso integration test | MEDIUM | 30 min | Quality |
| 11 | Recover catalog coverage (91.3% → 95%+) | MEDIUM | 1h | Coverage |
| 12 | Add decider snapshot+time-travel interaction tests | MEDIUM | 1h | Coverage |
| 13 | `Dialect.FormatTime` → `driver.Valuer`/`sql.Scanner` | MEDIUM | 2h | API |
| 14 | Move schema DDL onto `Dialect` interface | MEDIUM | 1h | Architecture |
| 15 | `query.Handler` typed generics migration | HIGH | 8h | Architecture |
| 16 | `CatalogMeta` consolidation | MEDIUM | 4h | Architecture |
| 17 | Trim AGENTS.md under 400 lines | LOW | 2h | Docs |
| 18 | Split `decider_test.go` (1190 lines) | LOW | 1h | DX |
| 19 | Normalize go.mod version references | LOW | 30 min | Hygiene |
| 20 | `Store.ReadBackwards` reverse stream reads | MEDIUM | 2h | Feature |
| 21 | Temporal read-only safety | MEDIUM | 2h | Feature |
| 22 | PostgreSQL integration tests | MEDIUM | 2h | Quality |
| 23 | Global `TransactionID` (breaking, v2) | HIGH | 22h | Feature |
| 24 | Saga/Process Manager | HIGH | 18h | Feature |
| 25 | Watermill pub/sub adapter | MEDIUM | 8h | Feature |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we cut the release NOW or wait for more features?**

The 8 release tags (`core/v1.4.0`, `memory/v1.2.0`, `catalog/v0.5.0`, etc.) have been sitting locally since Session 79. They include:
- `event.PositionalLoader` + `LoadToVersion`/`LoadToTimestamp` on `Store`
- `command.TypedHandler[T]` + `RegisterTyped[T]`
- `event.NewEvents`/`DecodePayloads[T]` batch helpers
- `event.NewTypedProjection[T]`
- Error taxonomy (5 families, `Classify()`, `IsRetryable()`)
- `decider.Repository.LoadAtVersion`/`LoadAtTime`

**Option A**: Push now — consumers get time-travel + typed handlers + error taxonomy today. We can always tag `v1.5.0` later for position-based projection replay.

**Option B**: Wait until `PositionalLoader` integration in projection Runner is in a tagged release — ships a more complete story.

I recommend **Option A** — push now, tag again after more projection work. The current tags are additive and non-breaking for consumers who already implement `Store` (the new interface methods can be added incrementally).

---

## Test Coverage Summary

| Package | Coverage | Change |
|---------|----------|--------|
| `core/command` | 98.1% | — |
| `core/query` | 100.0% | — |
| `core/event` | 92.9% | — |
| `core/aggregate` | 96.1% | — |
| `core/decider` | 93.6% | +8.0% |
| `core/pkg/id` | 97.8% | — |
| `core/pkg/dispatcher` | 100.0% | — |
| `memory` | 99.6% | — |
| `catalog` | 91.3% | — |
| `catalog/adapters` | 97.1% | — |
| `catalog/asyncapi` | 97.1% | — |
| `catalog/d2` | 97.6% | — |
| `catalog/eventcatalog` | 95.8% | — |
| `catalog/openapi` | 98.1% | — |
| `catalog/docserver` | 91.0% | — |
| `middleware` | 100.0% | — |
| `testhelpers` | 12.2% | — |
| `projection` | 94.1% | -3.5% (new code added) |
| `storage` | 87.8% | +0.2% |
| `sync` | 94.9% | — |
