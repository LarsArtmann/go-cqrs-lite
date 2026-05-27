# Session 86 — Storage Module: SQLite + Turso Superb Implementation

**Date:** 2026-05-21 01:41
**Session Type:** READ → UNDERSTAND → RESEARCH → REFLECT → EXECUTE → VERIFY
**Branch:** master
**Commits since May 1:** 514

---

## Executive Summary

Deep audit and improvement of the `storage` module's SQLite and Turso persistent storage implementations. Goal: give the END-USER (not developer) choice at deployment time. Added 6 Turso convenience constructors, 25 comprehensive Turso integration tests, 6 real-DB benchmarks, `TursoSyncDB.Close()`, and `TursoInitSchema`. All 16 test packages pass, zero vet, 89.3% storage coverage.

---

## a) FULLY DONE

### Session 86 Executed Changes

| #   | Change                                                                                                   | Files                                       | Impact                     |
| --- | -------------------------------------------------------------------------------------------------------- | ------------------------------------------- | -------------------------- |
| 1   | Added `TursoSyncDB.Close()` — was missing, callers had no way to release the database connection         | `turso_connector.go`                        | API completeness           |
| 2   | Added `TursoInitSchema(ctx, db)` — convenience wrapper for `SQLiteInitSchema`                            | `turso_connector.go`                        | Developer experience       |
| 3   | Added 5 `NewTurso*` constructors: EventStore, SnapshotStore, Outbox, CheckpointStore, TransactionalStore | `turso_connector.go` (+47 lines)            | End-user discoverability   |
| 4   | Rewrote `turso_connector_test.go`: 3 → 25 tests covering all stores, time-travel, metadata, concurrency  | `turso_connector_test.go` (153→880 lines)   | Test quality               |
| 5   | Added 6 real-DB benchmarks: SQLite (Save/Load/LoadAll/LoadToVersion) + Turso (Save/Load)                 | `sqlite_bench_test.go` (new, 263 lines)     | Performance visibility     |
| 6   | Added godoc to `SQLiteCheckpointSchema`, `SQLiteSnapshotSchema`, `SQLiteOutboxSchema`                    | `checkpoint.go`, `snapshot.go`, `outbox.go` | Documentation completeness |

### Storage Module — Persistent Backends (End-User Choice at Deployment)

| Backend           | Package/Driver                | Status           | What the end-user gets                                               |
| ----------------- | ----------------------------- | ---------------- | -------------------------------------------------------------------- |
| **PostgreSQL**    | `database/sql` + `lib/pq`     | Fully functional | `NewSQLEventStore(db)`, `NewSQLSnapshotStore(db)`, etc.              |
| **SQLite**        | `modernc.org/sqlite`          | **Superb**       | `NewSQLiteEventStore(db)`, 25 integration tests, 4 benchmarks        |
| **Turso (local)** | `turso.tech/database/tursogo` | **Superb**       | `OpenTurso(path)` → `NewTursoEventStore(db)`, 25 tests, 2 benchmarks |
| **Turso (sync)**  | `turso.tech/database/tursogo` | **Implemented**  | `OpenTursoSync(ctx, path, url, token)` → Push/Pull/Checkpoint/Stats  |
| **Pebble**        | `cockroachdb/pebble`          | Functional       | Embedded KV store for events                                         |
| **In-Memory**     | `memory` module               | Production-ready | `MemoryStore`, `MemoryBus`, `MemorySnapshotStore`                    |

### Turso API Surface (what the end-user sees)

```go
// Opening a database
db, err := storage.OpenTurso("myapp.db")
db, err := storage.OpenTursoInMemory()

// Synced database (push/pull with remote)
syncDB, err := storage.OpenTursoSync(ctx, "myapp.db", "libsql://db.turso.io", "token")
syncDB.Push(ctx)       // send local writes to remote
syncDB.Pull(ctx)       // fetch remote changes
syncDB.Checkpoint(ctx) // compact WAL
stats, _ := syncDB.Stats(ctx)
syncDB.Close()         // NEW: was missing before

// Schema initialization
storage.TursoInitSchema(ctx, db)

// Store constructors — identical API shape to SQLite/PostgreSQL
store, _ := storage.NewTursoEventStore(db)
snapStore, _ := storage.NewTursoSnapshotStore(db)
outbox, _ := storage.NewTursoOutbox(db)
checkpoint, _ := storage.NewTursoCheckpointStore(db)
txStore, _ := storage.NewTursoTransactionalStore(store, outbox)
```

### Storage Module Quality Metrics

| Metric               | Value                      |
| -------------------- | -------------------------- |
| Production files     | 25 files, 2,544 LOC        |
| Test files           | 14 files, ~6,600 LOC       |
| Total tests          | 173                        |
| Benchmarks           | 11 (5 sqlmock + 6 real DB) |
| Coverage             | 89.3%                      |
| Sentinel errors      | 10 (all classified)        |
| Files over 250 lines | 0 production files         |
| `go vet`             | Clean                      |

### Project-Wide Quality Metrics

| Metric               | Value                                |
| -------------------- | ------------------------------------ |
| Test packages        | 16 (all passing)                     |
| Production LOC       | 15,518                               |
| Test LOC             | 31,425                               |
| Total benchmarks     | 59 across 14 files                   |
| Total test functions | ~1,053                               |
| Sentinel errors      | 64 (all classified with errorfamily) |
| TODO/FIXME markers   | 0                                    |
| Commits since May 1  | 514                                  |

### Per-Package Coverage

| Package               | Coverage |
| --------------------- | -------- |
| `core/query`          | 100.0%   |
| `core/pkg/dispatcher` | 100.0%   |
| `middleware`          | 100.0%   |
| `memory`              | 99.6%    |
| `core/pkg/id`         | 97.8%    |
| `core/aggregate`      | 95.9%    |
| `core/command`        | 94.7%    |
| `projection`          | 93.9%    |
| `core/decider`        | 93.3%    |
| `core/event`          | 90.9%    |
| `storage`             | 89.3%    |
| `testhelpers`         | 10.5%    |

---

## b) PARTIALLY DONE

### Turso Sync Integration Tests

The `TursoSyncDB.Push/Pull/Checkpoint/Stats` methods exist and have API tests (`TestTurso_SyncRejectsMemoryDB`), but **no live sync tests** — would require a running Turso sync server. The `OpenTursoSync` constructor validates in-memory rejection and delegates to `turso.NewTursoSyncDb`.

**What exists:**

- `OpenTursoSync` constructor with validation
- `Push/Pull/Checkpoint/Stats` methods wrapping `turso.TursoSyncDb`
- `ErrTursoMemorySync` sentinel error
- 1 test (sync rejects in-memory DB)

**What's missing:**

- Live Push/Pull roundtrip test (needs `npx turso@latest --sync-server`)
- Checkpoint-after-write test
- Stats verification test
- Conflict resolution test (two nodes writing simultaneously)

### PostgreSQL Integration Tests

All PostgreSQL store code uses `go-sqlmock` — no real PostgreSQL integration tests. The SQLite/Turso tests provide confidence the SQL logic is correct, but PostgreSQL-specific behavior (e.g., `BYTEA` vs `BLOB`, `TIMESTAMPTZ` vs `TEXT`) is only verified via mock.

---

## c) NOT STARTED

### 1. PostgreSQL Real Integration Tests

No `testcontainers` or real PostgreSQL tests. DDL exists, dialect exists, mock tests pass, but no end-to-end validation.

### 2. Turso Sync Live Tests

No tests against a real Turso sync server. The blog post benchmarks show 8.9x faster sync, 16.3x less data — we can't verify these claims without a running server.

### 3. Storage Module Documentation

No `docs/planning/STORAGE_BACKEND_GUIDE.md` explaining deployment choices, connection pooling, migration, or recommended patterns for each backend.

### 4. WAL Mode Configuration for SQLite

SQLite's default journal mode is DELETE. For production use, WAL mode is strongly recommended. No `SQLiteEnableWAL(db)` helper exists.

### 5. Connection Pool Configuration Guide

SQLite needs `SetMaxOpenConns(1)` for write safety. Turso has its own `BusyTimeout`. No guidance exists for end-users.

### 6. Database Migration Strategy

No migration tooling. Schema versioning is manual (`TursoInitSchema` is all-or-nothing `CREATE TABLE IF NOT EXISTS`).

---

## d) TOTALLY FUCKED UP

### Nothing is broken.

All 16 test packages pass. Zero `go vet` issues. Zero TODO/FIXME markers. All production files under 250 lines. All sentinel errors classified. The storage module is in the best shape it's ever been.

### One marginal concern

The `testhelpers` package has 10.5% coverage. This is **by design** — test helpers are utility code used by other packages' tests, not independently tested. But it looks bad on a coverage report.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact, Low Effort

1. **Add `SQLiteEnableWAL(db)` helper** — 5 lines, huge production safety improvement
2. **Add `PostgresInitSchema(ctx, db)` convenience function** — mirrors `SQLiteInitSchema`/`TursoInitSchema`
3. **Add connection pool helpers** — `ConfigureSQLitePool(db)`, `ConfigureTursoPool(db, timeout)` with correct defaults
4. **Add `OpenSQLite(path)` constructor** — mirrors `OpenTurso`, removes need for raw `sql.Open("sqlite", ...)`
5. **Improve `testhelpers` coverage** — add basic smoke tests for helpers

### High Impact, Medium Effort

6. **PostgreSQL integration tests with testcontainers** — highest confidence for production PostgreSQL use
7. **Storage backend guide** — document when to use SQLite vs Turso vs PostgreSQL vs Pebble
8. **Schema migration support** — at minimum, detect if schema is outdated and warn
9. **Turso sync live tests** — spin up local sync server, test Push/Pull roundtrip
10. **Benchmark comparison report** — run SQLite vs Turso vs Pebble benchmarks and document results

### Medium Impact, Low Effort

11. **Add `OpenTursoSync` options pattern** — `WithSyncBusyTimeout`, `WithSyncClientName`, `WithPartialSync`
12. **Add `TursoSyncDB` integration with `OutboxPublisher`** — auto-push after outbox processing
13. **Document Turso sync best practices** — when to Push (every write? batch? periodic?)

---

## f) Top #25 Things We Should Get Done Next

### Storage Completion (items 1-8)

| #   | Task                                                         | Effort | Impact |
| --- | ------------------------------------------------------------ | ------ | ------ |
| 1   | Add `SQLiteEnableWAL(db)` helper                             | 15 min | HIGH   |
| 2   | Add `OpenSQLite(path)` + `OpenSQLiteInMemory()` constructors | 15 min | HIGH   |
| 3   | Add `ConfigureSQLitePool(db)` + `ConfigureTursoPool(db)`     | 15 min | HIGH   |
| 4   | Add `PostgresInitSchema(ctx, db)` convenience function       | 10 min | MEDIUM |
| 5   | Add storage backend guide (`docs/STORAGE_GUIDE.md`)          | 2h     | HIGH   |
| 6   | PostgreSQL integration tests with testcontainers             | 4h     | HIGH   |
| 7   | Turso sync live tests (local sync server)                    | 3h     | HIGH   |
| 8   | Benchmark comparison: SQLite vs Turso vs Pebble              | 1h     | MEDIUM |

### Turso Sync Polish (items 9-13)

| #   | Task                                                      | Effort | Impact |
| --- | --------------------------------------------------------- | ------ | ------ |
| 9   | Options pattern for `OpenTursoSync`                       | 30 min | MEDIUM |
| 10  | Add `WithBusyTimeout` option for Turso connections        | 15 min | MEDIUM |
| 11  | Add `WithPartialSync` option for large databases          | 30 min | MEDIUM |
| 12  | Turso sync + OutboxPublisher integration pattern          | 2h     | HIGH   |
| 13  | Document sync strategies (per-write vs batch vs periodic) | 1h     | MEDIUM |

### Project-Wide (items 14-25)

| #   | Task                                                          | Effort | Impact |
| --- | ------------------------------------------------------------- | ------ | ------ |
| 14  | Fix `testhelpers` coverage (currently 10.5%)                  | 1h     | LOW    |
| 15  | Add `integration/storage/` cross-module integration tests     | 2h     | MEDIUM |
| 16  | Example app: show Turso sync deployment pattern               | 2h     | HIGH   |
| 17  | Update `example/user/` to demonstrate storage backend choice  | 1h     | MEDIUM |
| 18  | Add `StorageBackend` enum + `NewEventStoreFromConfig` factory | 1h     | HIGH   |
| 19  | Pebble store: add benchmarks matching SQLite/Turso            | 30 min | LOW    |
| 20  | Schema migration: detect version mismatch, warn/error         | 2h     | MEDIUM |
| 21  | Add `nix run .#bench` for comprehensive benchmarking          | 30 min | LOW    |
| 22  | Update AGENTS.md with storage backend documentation           | 15 min | MEDIUM |
| 23  | Review `core/event` god-package (12 concerns, ~75 exports)    | 4h     | HIGH   |
| 24  | Consolidate `CatalogMeta` across 3 packages                   | 2h     | MEDIUM |
| 25  | Add `io.Closer` removal plan for interfaces                   | 1h     | LOW    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the storage module provide a unified factory function like `NewEventStoreFromConfig(backend string, dsn string)` that auto-detects and creates the right store?**

Arguments for:

- End-users get a single entry point
- Deployment-time choice becomes a config file change
- Aligns with "end-user chooses at deployment time"

Arguments against:

- Violates "library, not framework" principle (AGENTS.md)
- Adds unnecessary abstraction over 4 `NewXxxStore()` constructors
- Forces all backend dependencies into a single binary (Pebble + Turso + SQLite + PostgreSQL even if user only needs one)
- Go's explicit dependency model prefers explicit imports

**I lean toward "no factory" — the current approach with explicit constructors (`NewSQLiteEventStore`, `NewTursoEventStore`, `NewSQLEventStore`) is more Go-idiomatic. But I'd like your call.**

---

## Test Results

```
ok  github.com/larsartmann/go-cqrs-lite/core/aggregate       0.003s
ok  github.com/larsartmann/go-cqrs-lite/core/command          0.002s
ok  github.com/larsartmann/go-cqrs-lite/core/decider          0.004s
ok  github.com/larsartmann/go-cqrs-lite/core/event            0.014s
ok  github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher   0.003s
ok  github.com/larsartmann/go-cqrs-lite/core/pkg/id           0.002s
ok  github.com/larsartmann/go-cqrs-lite/core/query            0.002s
ok  github.com/larsartmann/go-cqrs-lite/memory                0.006s
ok  github.com/larsartmann/go-cqrs-lite/middleware            0.143s
ok  github.com/larsartmann/go-cqrs-lite/testhelpers           0.001s
ok  github.com/larsartmann/go-cqrs-lite/projection            0.129s
ok  github.com/larsartmann/go-cqrs-lite/storage               0.816s
ok  github.com/larsartmann/go-cqrs-lite/integration/aggregate 0.005s
ok  github.com/larsartmann/go-cqrs-lite/integration/command   0.002s
ok  github.com/larsartmann/go-cqrs-lite/integration/event     0.007s
ok  github.com/larsartmann/go-cqrs-lite/integration/query     0.004s
```

**16/16 packages pass. Zero failures.**
