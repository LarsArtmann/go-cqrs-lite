# Session 79: Time Travel Implementation — Status Report

**Date:** 2026-05-20 03:47 CEST
**Branch:** master (1 commit ahead of origin)
**Test Status:** 22/22 packages PASS, 0 FAIL

---

## Executive Summary

Implemented the **foundational time-travel API surface** for go-cqrs-lite: `Store.LoadToVersion`, `Store.LoadToTimestamp`, and `PositionalLoader.LoadAllFromPosition` across all 4 store implementations. All existing tests pass. The new API is **in the codebase and compiling** but **not yet tested** with new dedicated tests.

---

## A) FULLY DONE ✅

### Core Interfaces (core/event/store.go)

- [x] `LoadToVersion(ctx, aggType, aggID, maxVersion) ([]Event, error)` — added to `event.Store`
- [x] `LoadToTimestamp(ctx, aggType, aggID, maxTime) ([]Event, error)` — added to `event.Store`
- [x] `PositionalLoader` interface — new interface extending `GlobalLoader` with `LoadAllFromPosition(ctx, afterEventID, limit)`
- [x] Godoc on all new methods

### Store Implementations

- [x] **MemoryStore** — `LoadToVersion` (O(1) slice), `LoadToTimestamp` (filter), `LoadAllFromPosition` (scan + limit)
- [x] **SQLEventStore** — `LoadToVersion` (`WHERE version <= $3`), `LoadToTimestamp` (`WHERE occurred_at <= $3`), `LoadAllFromPosition` (`WHERE id > $1 ORDER BY occurred_at ASC LIMIT $2`)
- [x] **PebbleStore** — `LoadToVersion` (range scan to maxVersion+1), `LoadToTimestamp` (load all + filter)
- [x] **FakeStore** (testhelpers) — `LoadToVersion`, `LoadToTimestamp` (matches MemoryStore pattern)

### Interface Compliance Checks

- [x] `var _ event.Store = (*MemoryStore)(nil)`
- [x] `var _ event.PositionalLoader = (*MemoryStore)(nil)`
- [x] `var _ event.Store = (*SQLEventStore)(nil)`
- [x] `var _ event.PositionalLoader = (*SQLEventStore)(nil)`

### Existing Test Compatibility

- [x] `core/decider/decider_test.go` — updated `failingLoadStore` stub with new methods
- [x] `integration/aggregate/repository_test.go` — updated `failingStore` stub with new methods
- [x] All 22 test packages pass (core/_, memory, catalog/_, middleware, testhelpers, projection, storage, integration/\*)

### Research & Planning Documents

- [x] `docs/research/2026-05-20_TIME_TRAVEL_INDUSTRY_SURVEY.md` — 700+ line survey of 13 event sourcing systems
- [x] `docs/planning/2026-05-20_LOADTOVERSION_POSITIONALLOADER_PLAN.md` — 36-task execution plan

### Code Changes Summary

| File                                       | Lines Changed | What                                     |
| ------------------------------------------ | ------------- | ---------------------------------------- |
| `core/event/store.go`                      | +30           | New interface methods + PositionalLoader |
| `memory/store.go`                          | +120          | 3 new methods + interface check          |
| `storage/event_store.go`                   | +155          | 3 new methods + helper + interface check |
| `storage/pebble_event_store.go`            | +57           | 2 new methods                            |
| `testhelpers/fake_store.go`                | +46           | 2 new methods                            |
| `core/decider/decider_test.go`             | +18           | Stub updates                             |
| `integration/aggregate/repository_test.go` | +19           | Stub updates                             |
| **TOTAL**                                  | **+447, -4**  |                                          |

---

## B) PARTIALLY DONE 🔧

Nothing partially done — all started work is complete.

---

## C) NOT STARTED ❌

### Tests for New Methods (HIGH PRIORITY)

- [ ] MemoryStore `LoadToVersion` tests (6 cases: basic, exact-version, beyond-stream, empty, not-found, closed)
- [ ] MemoryStore `LoadToTimestamp` tests (4 cases: basic, exact-match, future-time, empty)
- [ ] MemoryStore `LoadAllFromPosition` tests (4 cases: basic, zero-position, limit, beyond-end)
- [ ] SQLEventStore `LoadToVersion` tests (go-sqlmock: success, empty, query-error)
- [ ] SQLEventStore `LoadToTimestamp` tests (go-sqlmock: success, empty, query-error)
- [ ] SQLEventStore `LoadAllFromPosition` tests (go-sqlmock: with-position, no-position, limit)
- [ ] PebbleStore `LoadToVersion` tests
- [ ] FakeStore `LoadToVersion`/`LoadToTimestamp` tests

### Repository Convenience Methods (HIGH PRIORITY)

- [ ] `decider.Repository.LoadAtVersion(ctx, aggID, aggType, version)` — single-call temporal state
- [ ] `decider.Repository.LoadAtTime(ctx, aggID, aggType, time)` — single-call temporal state by timestamp
- [ ] `aggregate.Repository.LoadAtVersion(ctx, root, version)` — interface addition + implementation
- [ ] Tests for decider `LoadAtVersion`/`LoadAtTime`
- [ ] Tests for aggregate `LoadAtVersion`

### Projection Runner Integration (HIGH PRIORITY)

- [ ] Auto-detect `PositionalLoader` in `projection.Runner` via type assertion
- [ ] Rewrite `Runner.replay()` to use `LoadAllFromPosition` when available
- [ ] Keep `filterEvents()` as fallback when `PositionalLoader` unavailable
- [ ] Test: position-based replay
- [ ] Test: fallback to LoadAll

### SQL Schema (MEDIUM PRIORITY)

- [ ] Add timestamp index to SQL DDL for `LoadToTimestamp` performance

### Cross-Module Tests (MEDIUM PRIORITY)

- [ ] Integration test: `LoadToVersion` + decider `LoadAtVersion` end-to-end

### Documentation (MEDIUM PRIORITY)

- [ ] Update `AGENTS.md` with new APIs
- [ ] Update `FEATURES.md` with time-travel capabilities

### Quality Gate

- [ ] Full lint check (`nix run .#lint`)

---

## D) TOTALLY FUCKED UP 💀

Nothing. All code compiles, all 22 existing test packages pass, no regressions introduced.

---

## E) WHAT WE SHOULD IMPROVE

1. **`LoadToTimestamp` in Pebble is O(n)** — loads entire stream then filters. Could be optimized with a secondary index by timestamp, but Pebble's key scheme is version-based. Acceptable for now since Pebble is primarily for testing/niche use.

2. **`LoadAllFromPosition` in MemoryStore is O(n)** — scans all events to find the position. Could be improved with a sorted index, but MemoryStore is for testing. SQL store is O(k) which is the production path.

3. **SQL `LoadAllFromPosition` uses `id > $1`** — this works because event IDs are ULIDs which are time-sortable. But if event stores use non-ordered IDs, this would break. Should document the ULID dependency.

4. **No timestamp index in SQL DDL** — `LoadToTimestamp` does a full table scan per aggregate. Need `CREATE INDEX idx_events_occurred_at ON events (aggregate_type, aggregate_id, occurred_at)`.

5. **Pebble doesn't implement `PositionalLoader`** — no global index. Pebble users can't use position-based projection replay. This is acceptable since Pebble is for embedded/niche use.

---

## F) Top 25 Things to Get Done Next

Ranked by impact × effort:

| #   | Task                                                    | Effort | Impact     |
| --- | ------------------------------------------------------- | ------ | ---------- |
| 1   | MemoryStore `LoadToVersion` tests                       | 10min  | ⭐⭐⭐⭐   |
| 2   | MemoryStore `LoadToTimestamp` tests                     | 8min   | ⭐⭐⭐⭐   |
| 3   | MemoryStore `LoadAllFromPosition` tests                 | 10min  | ⭐⭐⭐⭐   |
| 4   | SQLEventStore `LoadToVersion` tests                     | 10min  | ⭐⭐⭐⭐   |
| 5   | SQLEventStore `LoadToTimestamp` tests                   | 8min   | ⭐⭐⭐⭐   |
| 6   | SQLEventStore `LoadAllFromPosition` tests               | 8min   | ⭐⭐⭐⭐   |
| 7   | `decider.Repository.LoadAtVersion` method               | 8min   | ⭐⭐⭐⭐⭐ |
| 8   | `decider.Repository.LoadAtTime` method                  | 8min   | ⭐⭐⭐⭐   |
| 9   | Decider `LoadAtVersion`/`LoadAtTime` tests              | 10min  | ⭐⭐⭐⭐   |
| 10  | `aggregate.Repository.LoadAtVersion` (interface + impl) | 10min  | ⭐⭐⭐⭐   |
| 11  | Aggregate `LoadAtVersion` tests                         | 8min   | ⭐⭐⭐     |
| 12  | Auto-detect `PositionalLoader` in projection Runner     | 5min   | ⭐⭐⭐⭐⭐ |
| 13  | Rewrite `Runner.replay()` with position-based loading   | 10min  | ⭐⭐⭐⭐⭐ |
| 14  | Keep `filterEvents()` as fallback                       | 3min   | ⭐⭐⭐     |
| 15  | Test: position-based replay                             | 10min  | ⭐⭐⭐⭐⭐ |
| 16  | Test: fallback to LoadAll                               | 8min   | ⭐⭐⭐⭐   |
| 17  | Add timestamp index to SQL DDL                          | 5min   | ⭐⭐⭐⭐   |
| 18  | PebbleStore `LoadToVersion` tests                       | 8min   | ⭐⭐       |
| 19  | FakeStore `LoadToVersion`/`LoadToTimestamp` tests       | 5min   | ⭐⭐⭐     |
| 20  | Integration test: LoadToVersion + decider end-to-end    | 8min   | ⭐⭐⭐     |
| 21  | Update `AGENTS.md`                                      | 5min   | ⭐⭐⭐     |
| 22  | Update `FEATURES.md`                                    | 5min   | ⭐⭐⭐     |
| 23  | Full lint check                                         | 5min   | ⭐⭐⭐⭐   |
| 24  | Commit everything                                       | 5min   | ⭐⭐⭐     |
| 25  | Clean up plan doc (mark completed items)                | 3min   | ⭐⭐       |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `aggregate.Repository.LoadAtVersion` be added to the `Repository` interface (breaking change) or added as a method on the `EventSourcedRepository` struct only (non-breaking but requires type assertion)?**

Context: The `aggregate.Repository` interface currently has 3 methods: `Save`, `Load`, `Delete`. Adding `LoadAtVersion` to the interface would break every consumer who implements `Repository` directly. But keeping it off the interface means consumers can't use it without knowing the concrete type.

The `decider.Repository` is a concrete struct (not an interface), so adding methods there is non-breaking. But `aggregate.Repository` is an interface — this is a real design decision.

**My recommendation:** Add it only to `EventSourcedRepository` for now (non-breaking). Consumers who need it type-assert: `if r, ok := repo.(*aggregate.EventSourcedRepository); ok { r.LoadAtVersion(...) }`. We can add it to the interface in a future major version.
