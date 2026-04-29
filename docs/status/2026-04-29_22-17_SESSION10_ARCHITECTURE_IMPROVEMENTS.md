# Session 10: Architecture Improvements Report

**Date:** 2026-04-29 22:17  
**Branch:** master  
**Commits ahead of origin:** 6  
**Scope:** improve-codebase-architecture skill execution

---

## Executive Summary

This session applied the `improve-codebase-architecture` skill to deepen modules, eliminate shallow pass-throughs, and improve type safety. **6 commits** across 5 modules. **1 module deleted** (xtypes). **1 orphaned interface integrated** (SnapshotStore). **1 runtime error class eliminated** (HistoryLoader → compile-time Root.LoadEvents).

**Build status:** ALL PASS (14 packages, zero lint issues)  
**Coverage delta:** Aggregate dropped from 95.1% → 82.4% (new untested snapshot path)

---

## a) FULLY DONE ✅

### 1. SnapshotStore Integration (commit `f150708`)

**What:** `EventSourcedRepository` now accepts an optional `SnapshotStore` via `NewRepositoryWithSnapshot()`. `Load()` loads snapshot first, sets version via `SetVersion()`, replays events from `snapshot.Version` onward.

**Files:**
- `core/aggregate/aggregate.go` — Added `SetVersion` to `Core` and `Root`
- `core/aggregate/repository.go` — Added `snapshotStore` field, `NewRepositoryWithSnapshot`, snapshot-aware `Load`

**Impact:** HIGH — orphaned interface becomes real. All aggregates with large histories benefit.

**Coverage note:** New code paths (snapshot branch in Load, SetVersion, NewRepositoryWithSnapshot) are untested → aggregate coverage dropped from 95.1% → 82.4%. This is expected and will be addressed in Task 3.3.

---

### 2. Compile-time LoadEvents (commit `f150708`)

**What:** Eliminated `HistoryLoader` runtime side-interface. `LoadEvents([]event.Event) error` and `SetVersion(event.Version)` are now required methods on `Root`. `Root` implementations that embed `Core` delegate in one line.

**Before:** Runtime type assertion + verbose error message for missing `HistoryLoader`  
**After:** Compile-time enforcement. The `rootWithoutHistoryLoader` test type and its test are deleted because the failure mode is now impossible.

**Impact:** HIGH — entire class of runtime errors eliminated. Developer experience improved.

---

### 3. Type-safe Validators (commit `d3b27c3`)

**What:** Replaced `Validator func(any) error` with three typed variants:
- `CommandValidator func(command.Command) error`
- `EventValidator func(event.Event) error`
- `QueryValidator func(query.Query) error`

**Impact:** MEDIUM — no more type assertions in validation middleware. Compile-time safety.

---

### 4. EventBuilder Migration (commit `a6755ab`)

**What:** Moved `xtypes.EventBuilder` to `core/event.Builder` with fluent API:
```go
evt, err := event.NewBuilder("UserCreated", aggID, "User", 1).
    WithPayload(data).
    WithCorrelationID(correlationID).
    Build()
```

**Files:**
- `core/event/builder.go` — New Builder type (123 lines)
- `core/event/builder_test.go` — 7 test cases, 100% builder coverage

**Impact:** MEDIUM — useful builder lives where events are defined.

---

### 5. Generic Typed Interface (commit `f3532ad`)

**What:** Added `dispatcher.Typed` interface for generic message handling:
```go
type Typed interface { Type() string }
```

Command, Event, and Query all satisfy this implicitly. Enables generic middleware that only needs the type.

**Impact:** MEDIUM — foundation for generic middleware core (Task 2.1).

---

### 6. Remove internal testhelpers shim (commit `63b39a5`)

**What:** Deleted 37-line `core/internal/testhelpers` pass-through. Migrated 9 test files to import shared `testhelpers` directly. Moved `AssertCallOrder` to shared `testhelpers`.

**Files deleted:** `core/internal/testhelpers/helpers.go`  
**Files modified:** 9 core test packages

**Impact:** MEDIUM — one place for test utilities. No indirection seam that added nothing.

---

### 7. Delete xtypes module (commit `51b1d95`)

**What:** Deleted the entire `xtypes/` module (6 files, ~250 lines). Zero modules depended on it.

**Rationale:** After EventBuilder migration, xtypes provided only thin wrappers:
- `TypedCommand` → just called `command.New`
- `TypedAggregate` → just wrapped `aggregate.Core`
- `TypedEvent` → just wrapped `event.Core`
- ID aliases → already in `core/pkg/id`

**Files deleted:** `xtypes/aggregate.go`, `command.go`, `event.go`, `id.go`, `go.mod`, `go.sum`, `xtypes_test.go`

**Impact:** MEDIUM — simpler monorepo (5 modules instead of 6). Less code to maintain.

---

### 8. Lint and Format Fixes (commit `51b1d95`)

- Fixed 9 gci import ordering issues in test files
- Fixed 1 gofumpt issue in `repository_test.go`
- Fixed 2 staticcheck QF1008 (redundant `.Core` selector)
- Fixed 2 exhaustruct issues (explicit zero fields)
- Updated `flake.nix` to remove xtypes from `testModules`
- Updated `go.work` to remove xtypes from workspace

**Result:** Zero lint issues across all 5 modules.

---

## b) PARTIALLY DONE 🟡

### Snapshot Integration Tests (Task 3.3)

**Status:** SnapshotStore is wired into Repository but the snapshot+replay path has **zero test coverage**.

**Untested code:**
- `NewRepositoryWithSnapshot` — 0% coverage
- `Load` snapshot branch (lines 80-93) — 0% coverage
- `SetVersion` — 0% coverage

**This is the biggest gap in the session.** The coverage drop from 95.1% → 82.4% is entirely due to this untested new code.

---

## c) NOT STARTED ⏳

From the execution plan (`docs/planning/2026-04-29_22-04-IMPROVE_CODEBASE_ARCHITECTURE.md`):

| # | Task | Phase | Why Deferred |
|---|------|-------|-------------|
| 2.1 | Generic middleware core for Command/Event | 4% | Requires careful design to not over-abstract |
| 2.4 | CatalogBuilder wraps Registry | 4% | Medium effort, medium impact — lower priority |
| 3.1 | Query generic result types | 20% | Breaking change across many files; high risk |
| 3.2 | Outbox seam for atomic save+publish | 20% | New interface + memory adapter + tests; large |
| 3.3 | Snapshot integration tests | 20% | Needs time; not started yet |

---

## d) TOTALLY FUCKED UP! 🔴

**NONE.** All builds pass, all tests pass, zero lint issues.

**However, one concern:** The `core/aggregate` coverage dropped from 95.1% to 82.4%. The new snapshot code is **production code without tests**. This is a temporary state that must be resolved before the next release.

---

## e) WHAT WE SHOULD IMPROVE 🟢

### Immediate (next session)

1. **Snapshot integration tests** — The untested snapshot path is the #1 priority. Test:
   - Save events, save snapshot, load more events, load via snapshot → verify state
   - `SetVersion` direct test
   - `NewRepositoryWithSnapshot` construction test
   - Snapshot not found fallback (load all events)

2. **Generic middleware core** — Extract `ErrorRecovery`, `ErrorValidation` for Command/Event. Query has a different return type `(any, error)` so it stays separate.

3. **Query generic result types** — `Handler` returns `(any, error)` with runtime type assertion. Making it generic (`Handler[T any]`) is a transformative change but touches the dispatcher, all query handlers, and tests.

### Medium-term

4. **Outbox seam** — The `store.Save` + `bus.Publish` pattern is not atomic. A real production system needs an outbox table. Design the interface now, implement later.

5. **CatalogBuilder wraps Registry** — Two builders with identical internals. Consolidate for locality.

6. **MemorySnapshotStore deep copy** — `Snapshot.State []byte` is shared on load (shallow copy). Should deep-copy the byte slice.

### Architectural questions

7. **EventSourcedRepository.Save partial failure** — If `store.Save` succeeds but `bus.Publish` fails, events are persisted but never published. No compensation mechanism exists. This is the most critical production-readiness gap.

8. **go.work replace directives** — `core` still requires `memory` and `testhelpers` via `replace`. These modules are not independently publishable without the workspace file.

---

## f) Top #25 Things To Get Done Next

| # | Task | Module | Effort | Impact | Priority |
|---|------|--------|--------|--------|----------|
| 1 | Snapshot integration tests | core/aggregate | 30min | HIGH | P0 🔴 |
| 2 | Generic ErrorRecovery/ErrorValidation | middleware | 45min | MEDIUM | P1 |
| 3 | Query generic result types | core/query | 3h | HIGH | P1 |
| 4 | Outbox seam interface + memory adapter | core/event | 3h | HIGH | P2 |
| 5 | MemorySnapshotStore deep copy | memory | 15min | LOW | P2 |
| 6 | CatalogBuilder wraps Registry | catalog | 40min | MEDIUM | P2 |
| 7 | Remove core→memory replace directive | core/go.mod | 30min | MEDIUM | P2 |
| 8 | Add `event.Builder` benchmark | core/event | 15min | LOW | P3 |
| 9 | Add `Root.SetVersion` test | core/aggregate | 10min | LOW | P3 |
| 10 | EventRetry context cancellation test | middleware | 15min | LOW | P3 |
| 11 | Repository.Save partial failure test | core/aggregate | 20min | MEDIUM | P3 |
| 12 | Remove stale `replace` from middleware/go.mod | middleware | 10min | LOW | P3 |
| 13 | Projection module design | new module | 2d | HIGH | P4 |
| 14 | SQL-backed event store | new module | 3d | HIGH | P4 |
| 15 | SQL-backed snapshot store | new module | 1d | MEDIUM | P4 |
| 16 | Watermill pub/sub adapter | new module | 2d | MEDIUM | P4 |
| 17 | Event upcasting infrastructure | core/event | 1d | MEDIUM | P4 |
| 18 | Aggregate snapshot scheduling | core/aggregate | 1d | MEDIUM | P4 |
| 19 | Command deduplication via idempotency key | core/command | 1d | MEDIUM | P4 |
| 20 | Event metadata enrichment middleware | middleware | 30min | LOW | P5 |
| 21 | AsyncAPI 3.0.0 spec compliance audit | catalog/asyncapi | 2h | LOW | P5 |
| 22 | Golden file tests for EventCatalog output | catalog/eventcatalog | 1h | LOW | P5 |
| 23 | Benchmark suite for aggregate operations | core/aggregate | 30min | LOW | P5 |
| 24 | Context propagation through event metadata | core/event | 1h | LOW | P5 |
| 25 | OpenTelemetry tracing middleware | middleware | 2h | MEDIUM | P5 |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How do we make `EventSourcedRepository.Save` atomic without imposing a database transaction abstraction on the core module?**

The current implementation:
```go
err := r.store.Save(ctx, ...)   // succeeds
err = r.bus.Publish(ctx, ...)   // fails → events persisted, never published
```

An outbox pattern requires:
1. A transactional store that can write events + outbox records atomically
2. A background publisher that reads from the outbox
3. The core module should not depend on SQL or specific storage

**Options considered:**
- **Option A:** Add `OutboxStore` interface to `core/event`. `Save` checks if the store implements `OutboxStore` and uses it. Problem: core module grows a new interface for a pattern not everyone needs.
- **Option B:** Create a new `outbox` module. Wrap `Store` + `Bus` in an outbox decorator. Problem: where does the background publisher live? Who starts/stops it?
- **Option C:** Document the limitation. Let application code handle the outbox pattern. Problem: every user reinvents the wheel.

**What is the right seam for atomic save+publish?**

---

## Appendix: Coverage Summary

| Package | Coverage | Delta vs Session 9 |
|---------|----------|-------------------|
| `core/command` | 100.0% | — |
| `core/query` | 100.0% | — |
| `core/pkg/dispatcher` | 100.0% | — |
| `memory` | 99.4% | — |
| `middleware` | 99.2% | — |
| `catalog/adapters` | 98.8% | — |
| `core/event` | 98.2% | +0.3% (builder tests) |
| `core/pkg/id` | 97.1% | — |
| `catalog/asyncapi` | 97.6% | — |
| `catalog/eventcatalog` | 95.5% | — |
| `catalog` | 94.3% | +0.1% |
| `core/aggregate` | **82.4%** | **−12.7%** (snapshot code untested) |

**Overall:** 84.7% (down from ~91% due to aggregate snapshot path)

---

## Appendix: Module Graph (Post-Cleanup)

```
                    ┌─────────────┐
                    │  testhelpers│
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   ┌─────────┐       ┌─────────┐       ┌──────────┐
   │  core   │◄──────│ memory  │       │ middleware│
   └────┬────┘       └─────────┘       └──────────┘
        │
        ├─────────────┬─────────────┐
        ▼             ▼             ▼
   ┌─────────┐  ┌──────────┐  ┌──────────┐
   │ catalog │  │ (future) │  │ (future) │
   └─────────┘  │ storage  │  │projection│
                └──────────┘  └──────────┘
```

Modules: **5** (was 6, xtypes deleted)

---

## Appendix: Commit History (This Session)

| Commit | Message | Files |
|--------|---------|-------|
| `f150708` | feat(aggregate): integrate SnapshotStore and make LoadEvents compile-time safe | 8 |
| `d3b27c3` | feat(middleware): type-safe validators for command, event, and query | 3 |
| `a6755ab` | feat(event): add fluent Builder for event construction | 2 |
| `f3532ad` | feat(dispatcher): add Typed interface for generic message handling | 1 |
| `63b39a5` | cleanup(testhelpers): remove core/internal/testhelpers pass-through shim | 12 |
| `51b1d95` | cleanup(xtypes): remove shallow xtypes module; fix lint; update flake.nix | 22 |

**Total:** 48 files changed, ~800 insertions, ~1100 deletions

---

_End of report. Waiting for instructions._
