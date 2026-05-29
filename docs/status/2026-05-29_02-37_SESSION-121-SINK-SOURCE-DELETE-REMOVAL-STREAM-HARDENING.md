# Session 121 — Sink/Source Split + Delete Removal + Stream Module Hardening

**Date:** 2026-05-29 02:37 CEST
**Branch:** master (1 commit ahead of origin)

---

## What Was Done

### 1. Sink/Source Decomposition of `event.Store` — DONE

`core/event/store.go` now defines:

| Interface | Extends | Purpose |
|-----------|---------|---------|
| `EventSink` | `io.Closer` | Write side: `Save`, `AppendBatch` |
| `EventSource` | `io.Closer` | Read side: `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` |
| `Store` | `EventSink + EventSource` | Composite — all existing impls satisfy this |
| `BackwardsSource` | `EventSource` | `LoadBackwards` — renamed from `BackwardsLoader` (kept as alias) |
| `TransactionalSink` | `EventSink` | `SaveWithOutbox` — renamed from `TransactionalStore` (kept as deprecated) |

`Delete` **removed** from `Store`. History is immutable.

### 2. Delete Removal Across All Implementations — DONE

Removed `Delete` method from:
- `memory.MemoryStore`
- `storage.SQLEventStore`
- `testhelpers.FakeStore` (+ `DeleteFn` setter)
- `testhelpers.FakeStoreSetters.DeleteFn`

Removed 15 Delete test functions across 13 test files.
`SnapshotStore.Delete` kept — it's snapshot cache invalidation, not event history deletion.

### 3. Stream Module Hardening — DONE

- **SQL injection fix**: `validTablePrefix` regex on `NewAggregateProjection` + `NewSQLAggregateReader`
- **Tombstone detection fix**: `projection.go` no longer overwrites tombstone status on normal events
- **Rebirth support**: Tombstone → Rebirth transition works via metadata
- **New tests**: `builder_test.go`, `projection_test.go`, `sql_reader_test.go` (all passing)

### 4. Naming Consistency — DONE

- `BackwardsLoader` → `BackwardsSource` in interface assertions and tests
- `TransactionalStore` → `TransactionalSink` in `decider.go` type assertion
- Deprecated aliases preserved for backward compatibility

---

## Test Results

All 17 modified packages pass:

```
ok  core/command       core/decider       core/event
ok  core/pkg/dispatcher core/pkg/id       core/query
ok  memory             storage            testhelpers
ok  stream             integration/*      projection          signing
```

Pre-existing failures (unrelated): `saga` (2 BDD tests), `catalog/asyncapi`, `catalog/eventcatalog` (golden files).

---

## What's Still Wrong / Needs Fixing

### a) FULLY DONE

- Sink/Source interface split in `core/event/store.go`
- Delete removal from all event store implementations
- Delete test removal across all test files
- `decider.go` updated for `TransactionalSink`
- `BackwardsLoader` → `BackwardsSource` in interface assertions
- SQL injection validation in stream/
- Tombstone detection fix in projection.go
- New stream/ tests (builder, projection, sql_reader)
- `sql_bdd_test.go` updated for new `NewSQLAggregateReader` signature

### b) PARTIALLY DONE

- **Naming migration**: `BackwardsLoader` still used in FEATURES.md, docs. `TransactionalStore` still used in `storage/sql_backend.go`, `storage/README.md`, `storage/transactional_store.go` assertion. These compile but use deprecated names.

### c) NOT STARTED

- **FEATURES.md update**: Lines 80, 84, 85, 116, 184, 408, 409, 417, 433 have stale references
- **docs/STORAGE_GUIDE.md**: Still lists `Delete`, `LoadAll`, `LoadAllFromPosition`
- **docs/ARCHITECTURE_PATTERNS.md**: References `Store.Delete`, deprecated names
- **core/README.md**: Doesn't show Sink/Source split
- **AGENTS.md**: Missing `EventSink`, `EventSource`, `TransactionalSink`, `BackwardsSource`
- **storage/pebble_helpers.go**: Comment says "implements event.Store.Delete" — wrong

### d) TOTALLY FUCKED UP

Nothing. No broken code, no compilation errors, no test regressions from our changes.

### e) WHAT WE SHOULD IMPROVE

1. **FEATURES.md must reflect reality** — Currently lists `Delete` as a Store feature, `Repository.Delete`, etc. This is the #1 priority.
2. **Deprecated name migration** — `TransactionalStore()` on `SQLBackend` should return `TransactionalSink`. `storage/README.md` should show new names.
3. **Docs consistency** — All docs should use `ReadAll`/`ReadFrom` (Journal/SeekableJournal) not `LoadAll`/`LoadAllFromPosition` (GlobalLoader/PositionalLoader).
4. **PebbleEventStore.Delete** is now orphaned — not required by any interface. Should either remove it or document it as a utility method.

---

## Top #25 Things to Do Next (Sorted by Impact × Ease)

### P0 — Must Do (Wrong/misleading information)

| # | Task | Impact | Effort | File(s) |
|---|------|--------|--------|----------|
| 1 | Fix FEATURES.md: remove Delete, update naming | Very High | 15min | FEATURES.md |
| 2 | Fix storage/pebble_helpers.go comment | High | 1min | storage/pebble_helpers.go:14 |
| 3 | Update sql_backend.go return type | High | 2min | storage/sql_backend.go:92 |
| 4 | Update transactional_store.go assertion | Medium | 1min | storage/transactional_store.go:99 |
| 5 | Update storage/README.md | Medium | 5min | storage/README.md |

### P1 — Should Do (Stale docs)

| # | Task | Impact | Effort | File(s) |
|---|------|--------|--------|----------|
| 6 | Update AGENTS.md with Sink/Source concepts | Medium | 5min | AGENTS.md |
| 7 | Fix docs/STORAGE_GUIDE.md method table | Medium | 5min | docs/STORAGE_GUIDE.md |
| 8 | Fix docs/ARCHITECTURE_PATTERNS.md | Medium | 5min | docs/ARCHITECTURE_PATTERNS.md |
| 9 | Update core/README.md with Sink/Source split | Medium | 5min | core/README.md |
| 10 | Fix catalog golden files (pre-existing) | Low | 5min | catalog/testdata/golden/ |

### P2 — Nice to Have (Architecture improvements)

| # | Task | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 11 | Remove PebbleEventStore.Delete (orphaned method) | Medium | 5min | Check if anything calls it |
| 12 | Add stream/ to AGENTS.md module graph detail | Low | 2min | AGENTS.md |
| 13 | Consider removing deprecated aliases in next major version | Low | 0min | Future: v2 |
| 14 | Add TombstoneStatus to FEATURES.md | Low | 5min | FEATURES.md |
| 15 | Add AggregateReader/Projection to FEATURES.md | Low | 5min | FEATURES.md |
| 16 | Fix saga BDD tests (pre-existing) | Medium | 30min | saga/saga_bdd_test.go |
| 17 | Add stream/ integration test with real Journal | Low | 15min | stream/ |
| 18 | Update docs/adr/ for Sink/Source split decision | Low | 15min | docs/adr/ |
| 19 | Add cursor-based pagination test for SQL reader | Low | 10min | stream/ |
| 20 | Verify example/user and example/todo still work with tombstone pattern | Low | 10min | example/ |
| 21 | Add TombstonePolicy string representation | Low | 2min | stream/types.go |
| 22 | Update signing/signer.go for new interfaces | Low | 5min | signing/ |
| 23 | Add AggregateStatus JSON marshaling | Low | 5min | stream/types.go |
| 24 | Review stream/middleware.go for edge cases | Low | 10min | stream/ |
| 25 | Clean up replace directives planning | Low | 0min | Future: v1.0.0 |

---

## Question I Cannot Figure Out Myself

**#1: Should `PebbleEventStore.Delete` be removed entirely, or kept as a utility method?**

PebbleEventStore still has a `Delete` method that removes all events for an aggregate from the Pebble KV store. This method is no longer required by `event.Store` (it was removed during the Sink/Source split). However:
- It could be useful for testing (clearing data)
- The existing pebble test suite test `testEventStore_Delete` was already removed, and a new `TestPebbleEventStore_Delete_Empty` was removed too
- The `example/todo/storage/pebble_store.go` uses `s.db.Delete()` internally (it's the Pebble KV delete, not our Delete)

**My recommendation**: Remove it. If someone needs it, they can implement it locally. But this is a design decision I defer to you.

---

## Git Status

```
On branch master
Your branch is ahead of 'origin/master' by 1 commit.
nothing to commit, working tree clean
```

Recent commits on this work:
- `aaf4119` chore: run go mod tidy on stream module
- `62e3628` refactor: complete Store Sink/Source split downstream + fix stream, example, golden protection
- `7355dc9` test: reorganize event store tests and remove Delete functionality tests
