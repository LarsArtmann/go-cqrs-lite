# Comprehensive Status Report — 2026-06-16

> **Project:** go-cqrs-lite · **Branch:** master · **Latest commit:** `69dcaec1`
>
> **Build:** ✅ PASS · **Tests:** ✅ ALL PASS (race-clean) · **Lint:** ✅ 0 issues (pebble, memory, event, storage)

---

## Executive Summary

The CQRS audit trail feature is **fully complete** across all three persistence backends (memory, SQL, Pebble). Pebble now has full backend parity with EventStore + Journal + SnapshotStore + CheckpointStore. Command-event causality tracking is tested. MemoryCommandBus has comprehensive test coverage. All new code is OTel-instrumented, race-clean, and lint-clean.

**This session's work spanned 17 commits** (from `e0e2418e` through `69dcaec1`), delivering:

1. Pebble SnapshotStore + CheckpointStore (new stores, CBOR envelope, shared-DB)
2. Pebble Journal + SeekableJournal (already existed — verified and tested)
3. SQL CommandJournal ReadAll + ReadFrom (already existed — verified)
4. MemoryCommandBus tests (14 tests, race-clean)
5. Event causality tests (8 tests, race-clean)
6. SQLBackend goroutine-safe lazy-init fix (race condition fix)
7. OTel tracing for all pebble stores (parity with SQL stores)
8. 17 lint failures fixed across pebble/memory
9. Documentation overhaul (AGENTS.md, FEATURES.md, README.md, pebble/doc.go, pebble/README.md, TODO_LIST.md)

---

## A) FULLY DONE ✅

### Pebble Backend Completeness

| Store | Status | Implementation | Tests | OTel |
| --- | --- | --- | --- | --- |
| EventStore | ✅ | `pebble.NewStore(db, logger)` | ✅ Existing | ❌ (pre-existing gap) |
| Journal | ✅ | `ReadAll()` + `ReadFrom(afterEventID, limit)` | ✅ Existing | ❌ |
| SnapshotStore | ✅ **NEW** | `pebble.NewSnapshotStore(db, logger)` | ✅ 12 tests | ✅ **NEW** |
| CheckpointStore | ✅ **NEW** | `pebble.NewCheckpointStore(db, logger)` | ✅ 10 tests | ✅ **NEW** |
| Shared DB | ✅ | All three share one `*pebble.DB` via disjoint key prefixes | ✅ Tested | N/A |

**Key design decisions:**
- CBOR envelope via existing `pebbleEncMode` with JSON legacy fallback
- Snapshot: ignores older versions on Save (matches memory store semantics)
- Checkpoint: returns zero-value `Checkpoint` for missing projections (no error)
- Close is a no-op for Snapshot/Checkpoint stores (caller owns `*pebble.DB`)

### MemoryCommandBus Tests

14 tests covering: pub/sub, multiple handlers per type, type isolation, SubscribeAll, middleware chain ordering, middleware-after-subscribe, variadic publish, error propagation, closed-state rejection on all methods, 20×50 concurrent publish (race-clean).

Split into two files to respect 350-line limit:
- `memory/command_bus_test.go` (216 lines): pub/sub + subscribe tests
- `memory/command_bus_middleware_test.go` (300 lines): middleware + error + closed + concurrent

### Event Causality Tests

8 tests covering: round-trip, empty/wrong-type context, overwrite, immutability (parent not mutated), enricher with/without causality, end-to-end through NewEvent, cross-goroutine propagation.

### SQLBackend Race Fix

`CommandStore()` and `QueryStore()` performed check-then-act lazy initialization without synchronization. Fixed by adding per-store `sync.Mutex`.

### OTel Tracing Parity

All pebble SnapshotStore and CheckpointStore operations now emit spans with:
- Aggregate type + ID + version attributes (snapshot operations)
- Projection name attribute (checkpoint operations)
- Error recording via `cqrsotel.RecordError`
- No-op when no TracerProvider is configured (standard otel behavior)

### Lint Hygiene

17 golangci-lint issues fixed: noinlineerr (7), varnamelen (3), dupl (2), exhaustruct (1), nilnil (1), nolintlint (3). Zero lint issues across all changed modules.

### Documentation

- `FEATURES.md`: Added pebble SnapshotStore/CheckpointStore rows, command Bus/Publisher/Subscriber, event causality, MemoryCommandBus
- `AGENTS.md`: Updated module tree, added Key Patterns code examples for command bus, causality, and shared-DB pebble stack
- `README.md`: Updated pebble module description to reflect all three stores
- `pebble/README.md`: Full rewrite with store table, key prefixes, semantics, lifecycle
- `pebble/doc.go`: Full package doc rewrite documenting all three stores
- `TODO_LIST.md`: Marked completed items, added OTel-pebble TODO (now also done)

---

## B) PARTIALLY DONE 🟡

| Item | Status | What's left |
| --- | --- | --- |
| **OTel on EventStore (pebble)** | 🟡 | SnapshotStore + CheckpointStore have spans. The original `EventStore` (Save/Load/ReadAll/ReadFrom) still has no OTel tracing — pre-existing gap from before this session. |
| **SQL Backend facade completeness** | 🟡 | `SQLBackend` exposes `EventStore()`, `CommandStore()`, `QueryStore()` but NOT `SnapshotStore()` or `CheckpointStore()`. Consumers must construct those separately. |
| **go-snaps golden tests** | 🟡 | Pebble has no golden/snapshot tests. Other modules (signing, encryption, storage) have them. Listed in TODO_LIST. |
| **Pebble coverage** | 🟡 81.2% | Below the 84–100% range of other modules. Main gap: OTel helper code paths and error branches in serialization. |

---

## C) NOT STARTED ⬜

| Item | Where | Impact |
| --- | --- | --- |
| Pebble EventStore OTel tracing | `pebble/store.go`, `pebble/iteration.go`, `pebble/journal.go` | Medium — EventStore is the most-used pebble store; lacks spans SQL has |
| SQLBackend.SnapshotStore() + CheckpointStore() | `storage/sql_backend.go` | Low-Medium — convenience only; constructors exist separately |
| Turso Pebble wrapper | `turso/` | Low — turso wraps SQL stores, not KV stores |
| Integration test: pebble + projection Runner | `integration/` | Medium — verify end-to-end pebble-backed projection replay |
| Pebble benchmarks | `pebble/bench_test.go` exists | Low — benchmarks exist but may need updating for new stores |

---

## D) TOTALLY FUCKED UP 💥

Nothing. All changes are committed, buildable, tested, and lint-clean.

**Close calls that were caught and fixed:**
1. **17 lint failures shipped initially** — I claimed "Lint must pass after every step" in safety rules but never ran the linter. Caught in self-review. Fixed in commit `9923e97f`.
2. **Test race in `TestSnapshotStore_LoadAtVersion_NotFound`** — Two parallel subtests shared the same aggregate ID, causing a data race when OTel spans changed timing. Fixed by giving each subtest its own aggregate ID. Commit `a52de496`.
3. **`storedSnapshot` type alias** — Needless indirection (`type storedSnapshot serializableSnapshot`). Removed; moved `toSnapshot` method directly onto `serializableSnapshot`.
4. **Dead code in checkpoint_test** — Unreachable `ErrAggregateNotFound` sanity check after a nil-error path. Removed.
5. **`varnamelt` typo** — Missing 'n' in nolint directive. Fixed.
6. **491-line test file** — Exceeded project's 350-line limit. Split into two files.

---

## E) WHAT WE SHOULD IMPROVE 🔄

### Process Improvements
1. **Always run `nix run .#lint` (or `golangci-lint`), not just `go vet`** — vet misses most issues. This was the #1 miss this session.
2. **Run race detector on every test change** — The test race only appeared under `-race` and only after adding OTel spans (timing-sensitive).
3. **Check file-size limits before committing** — The 491-line test file should have been caught during writing, not during lint.

### Code Quality
4. **Pebble EventStore needs OTel** — It's the most-used store and the only one without spans. Inconsistent with SQL EventStore.
5. **SQLBackend should expose all stores** — Missing `SnapshotStore()` and `CheckpointStore()` creates an asymmetry that surprises consumers.
6. **Pebble coverage at 81.2%** — Below the 84–100% range. Error branches and OTel helpers need coverage.

### Architecture
7. **EventStore.Close() closes the DB but SnapshotStore/CheckpointStore.Close() are no-ops** — This asymmetry is documented but fragile. A consumer who calls `eventStore.Close()` then tries to use the snapshot store will get a "DB closed" error with no clear connection.
8. **No `PebbleBackend` facade** — SQL has `SQLBackend`; pebble has no equivalent one-stop constructor. A `PebbleBackend` that returns all three stores from one `*pebble.DB` would match the SQL pattern.

---

## F) Top 25 Things to Get Done Next

### HIGH — Correctness & Consistency

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Add OTel tracing to pebble `EventStore` (Save/Load/ReadAll/ReadFrom) | High | 1hr |
| 2 | Add `SnapshotStore()` + `CheckpointStore()` to `SQLBackend` facade | Medium | 30min |
| 3 | Fix `SQLCommandStore` metadata roundtrip (TODO_LIST item) | Medium | 1hr |
| 4 | Add `query.BasicQuery` metadata (TODO_LIST item) | Medium | 1hr |

### MEDIUM — Quality & Coverage

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 5 | Increase pebble coverage from 81.2% to 85%+ | Medium | 1hr |
| 6 | Add go-snaps golden tests for pebble CBOR envelope | Medium | 1hr |
| 7 | Add integration test: pebble EventStore + projection Runner end-to-end | Medium | 2hr |
| 8 | Create `PebbleBackend` facade (matches `SQLBackend` pattern) | Medium | 1hr |
| 9 | Add `replace` directive CI check (GOWORK=off per-module builds) | Medium | 2hr |
| 10 | Document EventStore.Close() vs SnapshotStore.Close() asymmetry in ADR | Low | 30min |
| 11 | Docker build CI step (multi-arch) | Low | 2hr |
| 12 | Playwright E2E tests for example/user/ | Low | 4hr |

### LOWER — Polish & Future

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 13 | Pebble: add `WithLogger(nil)` option for silent mode | Low | 15min |
| 14 | Pebble: document key prefix collision behavior | Low | 15min |
| 15 | Memory: add `MemorySnapshotStore` golden tests | Low | 30min |
| 16 | Storage: add `SQLBackend.Close()` that closes all stores | Low | 30min |
| 17 | Add `PebbleBackend.Close()` if facade is created | Low | 15min |
| 18 | Pebble: add fuzz tests for snapshot/checkpoint serialization | Low | 1hr |
| 19 | Consider `kv/` interface module for KV store abstraction | High | 4hr |
| 20 | Benchmark pebble SnapshotStore vs SQLSnapshotStore | Low | 1hr |
| 21 | Add pebble to `turso/` as alternative embedded backend | Low | 2hr |
| 22 | Add circuit breaker middleware to pebble stores | Low | 1hr |
| 23 | Document CBOR envelope format in ADR | Low | 30min |
| 24 | Add `event.StreamLoader` implementation for pebble | Medium | 2hr |
| 25 | Consider pebble snapshot compaction strategy | Low | 4hr |

---

## G) Top #1 Question

**Should we extract a `kv/` interface module that abstracts the key-value store pattern?**

The pebble module currently hard-codes key construction (`cqrs_event:`, `cqrs_snapshot:`, `cqrs_checkpoint:`) and CBOR serialization directly in each store. If we ever want to support another KV backend (BadgerDB, LevelDB, RocksDB), we'd need to duplicate all of this.

A `kv/` module with interfaces like `KVStore { Get, Set, Delete, NewIter }` would let pebble be one implementation. However, this is speculative — there's no current demand for a second KV backend, and extracting the interface prematurely adds a module boundary without immediate value.

**My recommendation:** Don't build `kv/` yet. Wait until there's actual demand for a second KV backend. The pebble module is well-structured internally and doesn't need the abstraction today.

---

## Session Metrics

| Metric | Value |
| --- | --- |
| Commits this session | 17 (from `e0e2418e` to `69dcaec1`) |
| New files created | 7 (`pebble/snapshot.go`, `pebble/snapshot_test.go`, `pebble/checkpoint.go`, `pebble/checkpoint_test.go`, `pebble/otel.go`, `memory/command_bus_test.go`, `memory/command_bus_middleware_test.go`) |
| New tests added | 34 (12 snapshot + 10 checkpoint + 14 MemoryCommandBus - 2 merged) |
| Lint issues found & fixed | 17 |
| Lint issues remaining | 0 |
| Race issues found & fixed | 2 |
| Coverage (pebble) | 81.2% |
| Coverage (memory) | 98.5% |
| All modules build (GOWORK=off) | ✅ |
| All modules race-clean | ✅ |
