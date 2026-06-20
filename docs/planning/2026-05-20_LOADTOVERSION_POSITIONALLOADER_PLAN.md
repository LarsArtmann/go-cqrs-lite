# Implementation Plan: LoadToVersion + PositionalLoader

> Comprehensive task breakdown for the two highest-impact time-travel features.
> Each task: max 12 minutes. Sorted by importance/impact/effort/customer-value.

**Date:** 2026-05-20

---

## Task List (Execution Order)

### Phase 1: LoadToVersion — Store Interface + Core Implementations

| #   | Task                                                                                               | Files Changed               | Effort | Impact     | Customer Value                          |
| --- | -------------------------------------------------------------------------------------------------- | --------------------------- | ------ | ---------- | --------------------------------------- |
| 1   | Add `LoadToVersion` method to `event.Store` interface in `core/event/store.go`                     | `core/event/store.go`       | 5min   | ⭐⭐⭐⭐⭐ | Enables all downstream implementations  |
| 2   | Add `ErrVersionRequired` sentinel error (version must be > 0) in `core/event/errors.go`            | `core/event/errors.go`      | 3min   | ⭐⭐⭐     | Input validation consistency            |
| 3   | Implement `MemoryStore.LoadToVersion` in `memory/store.go` — slice `events[:min(maxVersion, len)]` | `memory/store.go`           | 5min   | ⭐⭐⭐⭐⭐ | In-memory time travel, test utility     |
| 4   | Implement `SQLEventStore.LoadToVersion` — `WHERE version <= $3 ORDER BY version ASC`               | `storage/event_store.go`    | 8min   | ⭐⭐⭐⭐⭐ | Production SQL time travel              |
| 5   | Update `testhelpers/fake_store.go` `FakeStore` to implement `LoadToVersion`                        | `testhelpers/fake_store.go` | 5min   | ⭐⭐⭐⭐   | Unblocks middleware + integration tests |

### Phase 2: LoadToVersion — Pebble + Tests

| #   | Task                                                                                                                 | Files Changed                   | Effort | Impact   | Customer Value                          |
| --- | -------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ------ | -------- | --------------------------------------- |
| 6   | Implement `CQRSAdapter.LoadToVersion` in Pebble store — range scan to maxVersion                                     | `storage/pebble_event_store.go` | 8min   | ⭐⭐⭐   | Complete store coverage                 |
| 7   | Write MemoryStore `TestLoadToVersion` tests: basic, at-exact-version, beyond-stream, empty-stream, not-found, closed | `memory/store_test.go`          | 10min  | ⭐⭐⭐⭐ | Confidence in reference implementation  |
| 8   | Write SQLEventStore `TestLoadToVersion` tests with go-sqlmock: success, empty, query-error                           | `storage/event_store_test.go`   | 10min  | ⭐⭐⭐⭐ | Confidence in production implementation |
| 9   | Write PebbleStore `TestLoadToVersion` tests if Pebble test infra exists                                              | `storage/`                      | 8min   | ⭐⭐     | Complete test coverage                  |
| 10  | Update `testhelpers/fake_store.go` tests for `LoadToVersion`                                                         | `testhelpers/`                  | 5min   | ⭐⭐⭐   | Test helper reliability                 |

### Phase 3: LoadToVersion — Repository Convenience Methods

| #   | Task                                                                                              | Files Changed                                                    | Effort | Impact     | Customer Value                                      |
| --- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ------ | ---------- | --------------------------------------------------- |
| 11  | Add `decider.Repository.LoadAtVersion(ctx, aggID, aggType, version)` method                       | `core/decider/decider.go`                                        | 8min   | ⭐⭐⭐⭐⭐ | Single-call temporal state access for decider users |
| 12  | Add `decider.Repository.LoadAtTime(ctx, aggID, aggType, time)` method (uses LoadToTimestamp)      | `core/decider/decider.go`                                        | 8min   | ⭐⭐⭐     | Temporal access by timestamp                        |
| 13  | Add `aggregate.Repository.LoadAtVersion(ctx, root, version)` method to interface + implementation | `core/aggregate/repository.go`, `core/aggregate/load_helpers.go` | 10min  | ⭐⭐⭐⭐   | Single-call temporal state for aggregate users      |
| 14  | Write `decider.Repository.LoadAtVersion` test: basic, empty-stream, version-0                     | `core/decider/decider_test.go` or new                            | 8min   | ⭐⭐⭐⭐   | Confidence in consumer API                          |
| 15  | Write `aggregate.Repository.LoadAtVersion` test                                                   | `core/aggregate/` test file                                      | 8min   | ⭐⭐⭐     | Confidence in consumer API                          |

### Phase 4: LoadToTimestamp — Store Interface + Implementations

| #   | Task                                                                                       | Files Changed                 | Effort | Impact   | Customer Value                   |
| --- | ------------------------------------------------------------------------------------------ | ----------------------------- | ------ | -------- | -------------------------------- |
| 16  | Add `LoadToTimestamp` method to `event.Store` interface in `core/event/store.go`           | `core/event/store.go`         | 5min   | ⭐⭐⭐⭐ | Enables "as-at" queries          |
| 17  | Implement `MemoryStore.LoadToTimestamp` — filter by `OccurredAt() <= maxTime`              | `memory/store.go`             | 8min   | ⭐⭐⭐⭐ | In-memory timestamp time travel  |
| 18  | Implement `SQLEventStore.LoadToTimestamp` — `WHERE occurred_at <= $3 ORDER BY version ASC` | `storage/event_store.go`      | 8min   | ⭐⭐⭐⭐ | Production timestamp time travel |
| 19  | Implement `FakeStore.LoadToTimestamp` in testhelpers                                       | `testhelpers/fake_store.go`   | 5min   | ⭐⭐⭐   | Test helper completeness         |
| 20  | Write MemoryStore `TestLoadToTimestamp` tests: basic, exact-match, future-time, empty      | `memory/store_test.go`        | 8min   | ⭐⭐⭐⭐ | Confidence                       |
| 21  | Write SQLEventStore `TestLoadToTimestamp` tests with go-sqlmock                            | `storage/event_store_test.go` | 8min   | ⭐⭐⭐⭐ | Confidence                       |
| 22  | Add timestamp index to SQL DDL in `storage/helpers.go`                                     | `storage/helpers.go`          | 5min   | ⭐⭐⭐⭐ | Query performance                |

### Phase 5: PositionalLoader — Interface + Core Implementations

| #   | Task                                                                                                        | Files Changed                 | Effort | Impact     | Customer Value                             |
| --- | ----------------------------------------------------------------------------------------------------------- | ----------------------------- | ------ | ---------- | ------------------------------------------ |
| 23  | Add `PositionalLoader` interface to `core/event/store.go` — `LoadAllFromPosition(ctx, afterEventID, limit)` | `core/event/store.go`         | 5min   | ⭐⭐⭐⭐⭐ | Enables production-scale projection replay |
| 24  | Implement `MemoryStore.LoadAllFromPosition` — binary search by event ID + slice + limit                     | `memory/store.go`             | 10min  | ⭐⭐⭐⭐⭐ | In-memory position-based loading           |
| 25  | Implement `SQLEventStore.LoadAllFromPosition` — `WHERE id > $1 ORDER BY occurred_at ASC LIMIT $2`           | `storage/event_store.go`      | 8min   | ⭐⭐⭐⭐⭐ | Production position-based loading          |
| 26  | Write MemoryStore `TestLoadAllFromPosition` tests: basic, zero-position, limit, beyond-end                  | `memory/store_test.go`        | 10min  | ⭐⭐⭐⭐   | Confidence                                 |
| 27  | Write SQLEventStore `TestLoadAllFromPosition` tests with go-sqlmock                                         | `storage/event_store_test.go` | 8min   | ⭐⭐⭐⭐   | Confidence                                 |

### Phase 6: Projection Runner — Use PositionalLoader

| #   | Task                                                                                                | Files Changed               | Effort | Impact     | Customer Value                       |
| --- | --------------------------------------------------------------------------------------------------- | --------------------------- | ------ | ---------- | ------------------------------------ |
| 28  | Add `PositionalLoader` field to `projection.Runner` + detect in `NewRunner` via type assertion      | `projection/runner.go`      | 5min   | ⭐⭐⭐⭐⭐ | Runner auto-detects optimized loader |
| 29  | Rewrite `Runner.replay()` to use `LoadAllFromPosition` when available, fall back to `LoadAll`       | `projection/runner.go`      | 10min  | ⭐⭐⭐⭐⭐ | O(k) replay instead of O(n)          |
| 30  | Remove or deprecate `filterEvents()` (only needed for fallback path)                                | `projection/runner.go`      | 3min   | ⭐⭐⭐     | Code cleanup                         |
| 31  | Write `TestRunner_ReplayWithPositionalLoader` test — verify position-based replay works             | `projection/runner_test.go` | 10min  | ⭐⭐⭐⭐⭐ | Confidence in production replay      |
| 32  | Write `TestRunner_ReplayFallbackToLoadAll` test — verify fallback when PositionalLoader unavailable | `projection/runner_test.go` | 8min   | ⭐⭐⭐⭐   | Backward compatibility confidence    |

### Phase 7: Cross-Module Tests + Documentation

| #   | Task                                                                              | Files Changed  | Effort | Impact   | Customer Value           |
| --- | --------------------------------------------------------------------------------- | -------------- | ------ | -------- | ------------------------ |
| 33  | Add integration test: LoadToVersion + decider Repository.LoadAtVersion end-to-end | `integration/` | 8min   | ⭐⭐⭐⭐ | Cross-module correctness |
| 34  | Update `AGENTS.md` with new APIs and coverage numbers                             | `AGENTS.md`    | 5min   | ⭐⭐⭐   | Documentation freshness  |
| 35  | Update `FEATURES.md` with new time-travel capabilities                            | `FEATURES.md`  | 5min   | ⭐⭐⭐   | Feature inventory        |
| 36  | Run full test suite across all modules and verify zero lint                       | all modules    | 5min   | ⭐⭐⭐⭐ | Quality gate             |

---

## Summary

| Phase                                      | Tasks        | Total Effort | Cumulative |
| ------------------------------------------ | ------------ | ------------ | ---------- |
| **1: LoadToVersion — Interface + Core**    | 5            | 26min        | 26min      |
| **2: LoadToVersion — Pebble + Tests**      | 5            | 41min        | 67min      |
| **3: LoadToVersion — Repository Methods**  | 5            | 42min        | 109min     |
| **4: LoadToTimestamp — Interface + Impl**  | 7            | 47min        | 156min     |
| **5: PositionalLoader — Interface + Impl** | 5            | 41min        | 197min     |
| **6: Projection Runner Integration**       | 5            | 36min        | 233min     |
| **7: Cross-Module Tests + Docs**           | 4            | 23min        | 256min     |
| **TOTAL**                                  | **36 tasks** | **~4.3h**    | —          |

### Top 10 by Impact × Speed (Do These First)

| Priority | #   | Task                                                    | Effort |
| -------- | --- | ------------------------------------------------------- | ------ |
| 1        | 1   | Add `LoadToVersion` to `event.Store` interface          | 5min   |
| 2        | 3   | Implement `MemoryStore.LoadToVersion`                   | 5min   |
| 3        | 4   | Implement `SQLEventStore.LoadToVersion`                 | 8min   |
| 4        | 5   | Update `FakeStore` with `LoadToVersion`                 | 5min   |
| 5        | 23  | Add `PositionalLoader` interface                        | 5min   |
| 6        | 24  | Implement `MemoryStore.LoadAllFromPosition`             | 10min  |
| 7        | 25  | Implement `SQLEventStore.LoadAllFromPosition`           | 8min   |
| 8        | 28  | Auto-detect `PositionalLoader` in projection Runner     | 5min   |
| 9        | 29  | Rewrite `Runner.replay()` to use position-based loading | 10min  |
| 10       | 11  | Add `decider.Repository.LoadAtVersion`                  | 8min   |
