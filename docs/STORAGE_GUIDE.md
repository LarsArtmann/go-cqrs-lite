# Storage Backend Guide

go-cqrs-lite separates **consumer code** (engine-agnostic) from **deployer choice**
(where data lives). Two paths exist: **presets** (one engine for everything) and
**manual assembly** (mix engines per concern).

For the full deployer-first architecture, see
[`docs/research/2026-06-23_DEPLOYER_FIRST_ARCHITECTURE_AUDIT.md`](research/2026-06-23_DEPLOYER_FIRST_ARCHITECTURE_AUDIT.md).

---

## Path 1: Presets (recommended)

One line picks the engine; all six concerns (events, commands, queries, snapshots,
checkpoints, read models) use it. Consumer code is identical regardless of preset.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/stack/memory"
    // or: stack/sqlite, stack/pebble, stack/postgres, stack/turso
)

bundle, err := memory.New()           // in-memory (tests, dev)
// bundle, err := sqlite.New("app.db") // embedded SQL (single-process prod)
// bundle, err := pebble.New("data")   // embedded KV (high-throughput)
// bundle, err := postgres.New(dsn)    // distributed (multi-process prod)
```

### Multi-DB split (SQLite/Postgres only)

Isolate concerns across separate database files:

```go
bundle, err := sqlite.New("primary.db",
    sqlite.WithDSN(
        sqlopt.WithEventDB("events.db"),   // events + snapshots + checkpoints
        sqlopt.WithQueryDB("queries.db"),  // command + query audit logs
        sqlopt.WithViewDB("views.db"),     // materialized views (cqrs_kv)
    ),
)
```

Postgres supports the same split against separate databases on the same server.

---

## Path 2: Manual assembly (heterogeneous)

Mix engines per concern via `stack.New` with `With*` options:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/stack"
    "github.com/larsartmann/go-cqrs-lite/storage/pebble"
    "github.com/larsartmann/go-cqrs-lite/storage"
)

pebbleBackend, _ := pebble.Open("data", pebble.DefaultOptions(), logger)
sqlDB, _ := storage.OpenSQLite("views.db")
sqlKV, _ := storage.NewSQLiteKVStore(sqlDB)

bundle, err := stack.New(
    stack.WithEventStore(pebbleBackend.EventStore()),
    stack.WithSnapshotStore(pebbleBackend.SnapshotStore()),
    stack.WithReadModels(sqlKV), // SQL for queryable views, Pebble for events
)
```

---

## Backend facades

When you need individual stores (not a full `Bundle`), use a backend facade.
All stores share one connection; `Close()` closes all stores (not the `*sql.DB`).

### SQLite / Postgres

```go
db, _ := storage.OpenSQLite("events.db")
storage.SQLiteInitSchema(ctx, db)

backend, _ := storage.NewSQLiteBackend(db) // or NewSQLBackend for Postgres
defer backend.Close()

eventStore  := backend.EventStore()              // *SQLEventStore
cmdStore, _ := backend.CommandStore()            // *SQLCommandStore (lazy)
qStore, _   := backend.QueryStore()              // *SQLQueryStore (lazy)
snapStore,_ := backend.SnapshotStore()           // *SQLSnapshotStore (lazy)
cpStore, _  := backend.CheckpointStore()         // *SQLCheckpointStore (lazy)
kvStore, _  := backend.KVStore()                 // kv.Store (cqrs_kv table)
```

### Pebble (single-DB full stack)

```go
import "github.com/larsartmann/go-cqrs-lite/storage/pebble"

backend, _ := pebble.Open(dir, pebble.DefaultOptions(), logger)
defer backend.Close() // closes DB AND all stores

eventStore  := backend.EventStore()
snapStore   := backend.SnapshotStore()
cpStore     := backend.CheckpointStore()
kvStore     := backend.ReadModels()
```

All three share one Pebble DB via disjoint key prefixes.

### Turso (Embedded Database)

```go
import "github.com/larsartmann/go-cqrs-lite/storage/turso"

db, _ := turso.Open(turso.DbPath("app.db")) // or turso.OpenInMemory() for tests
backend, _ := turso.NewBackend(db)
```

### DuckDB (analytical / CGo)

DuckDB is an embedded **columnar (OLAP)** engine, wrapped as a `stack.Bundle`
preset (`stack/duckdb`). It requires CGo (statically links a C++ engine, ~30-50MB
binary) — see ADR-0071. Use it for dashboard/reporting read models that need
GROUP BY, window functions, and fast columnar scans; prefer SQLite/Pebble for the
transactional event store.

**DSN format:**

| DSN                | Meaning                                           |
| ------------------ | ------------------------------------------------- |
| `""` (empty)       | In-memory database (process-local, lost on close) |
| `/path/to/file.db` | Persistent single-file database                   |
| `:memory:`         | Explicit in-memory                                |

Query-string options tune the engine and are appended to the DSN automatically
by the helpers:

| Option         | Helper                          | Effect                                 |
| -------------- | ------------------------------- | -------------------------------------- |
| `threads=N`    | `duckdb.WithThreads(4)`         | DuckDB worker threads                  |
| `memory_limit` | `duckdb.WithMemoryLimit("1GB")` | Memory cap (DuckDB syntax, e.g. `1GB`) |

```go
import "github.com/larsartmann/go-cqrs-lite/stack/duckdb"

bundle, _ := duckdb.New("analytics.db")                       // persistent
bundle, _ := duckdb.New("", duckdb.WithThreads(4),
    duckdb.WithMemoryLimit("1GB"))                            // in-memory, tuned

// Multi-DB split (events/queries/views in separate files), like SQLite/Postgres:
bundle, _ := duckdb.New("primary.db",
    duckdb.WithDSN(
        sqlopt.WithEventDB("events.db"),
        sqlopt.WithViewDB("views.db"),
    ))
```

Notes:

- The `metadata` column is `BLOB` (not VARCHAR) to avoid byte-slice escaping on
  roundtrip; the dialect uses `$1` placeholders (Postgres-compatible) and returns
  `time.Time` natively.
- Analytical read models: `duckdb.SQLViewModel[V,K]` builds a real columnar view
  table enabling server-side WHERE/ORDER BY and native GROUP BY.

---

## Event Store operations

| Method                                               | Interface               | Description                               |
| ---------------------------------------------------- | ----------------------- | ----------------------------------------- |
| `Save(ctx, aggType, aggID, events, expectedVersion)` | `event.EventSink`       | Append events with optimistic concurrency |
| `AppendBatch(ctx, aggType, aggID, events)`           | `event.EventSink`       | Append without concurrency check          |
| `Load(ctx, aggType, aggID)`                          | `event.EventSource`     | Load all events for an aggregate          |
| `LoadFromVersion(ctx, aggType, aggID, version)`      | `event.EventSource`     | Load events starting from version         |
| `LoadToVersion(ctx, aggType, aggID, maxVersion)`     | `event.EventSource`     | Load events up to version (time-travel)   |
| `LoadToTimestamp(ctx, aggType, aggID, maxTime)`      | `event.EventSource`     | Load events up to timestamp (time-travel) |
| `ReadAll(ctx)`                                       | `event.Journal`         | Load all events across aggregates         |
| `ReadFrom(ctx, afterEventID, limit)`                 | `event.SeekableJournal` | Cursor-based global load                  |

---

## Schema DDL

Schema initialization is handled by `SQLiteInitSchema` / `PostgresInitSchema`,
which create all tables (`events`, `snapshots`, `checkpoints`, `commands`,
`queries`, `cqrs_kv`) if they don't exist. For raw DDL access:

```go
// Individual table schemas (from the Dialect)
sqlpkg.PostgresDialect{}.EventSchema()     // events table DDL
sqlpkg.PostgresDialect{}.SnapshotSchema()  // snapshots table DDL
sqlpkg.PostgresDialect{}.CheckpointSchema()// checkpoints table DDL
sqlpkg.SQLiteDialect{}.EventSchema()       // SQLite equivalent
```

Embedded DDL files live in `storage/migrations/` (`postgres.sql`, `sqlite.sql`).

---

## SQLite pragmas

```go
_ = storage.SQLiteEnableWAL(ctx, db)          // WAL mode + busy_timeout=5000
_ = storage.SQLiteEnableForeignKeys(ctx, db)  // PRAGMA foreign_keys=ON
```

---

## See also

- [Deployer-first architecture audit](research/2026-06-23_DEPLOYER_FIRST_ARCHITECTURE_AUDIT.md)
- [Storage environment mapping](research/storage-environment-mapping.md)
- [Storage first-principles analysis](research/storage-first-principles-analysis.md)
- [Database architecture taxonomy](research/database-architecture-taxonomy.md)
