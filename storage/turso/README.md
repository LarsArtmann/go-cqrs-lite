# turso — Turso Database Connectors

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/turso/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/turso/v3)

CQRS storage adapters for [Turso](https://turso.tech/) databases (embedded LibSQL/SQLite with optional remote sync).

```bash
go get github.com/larsartmann/go-cqrs-lite/turso/v3
```

## Constructors

| Function                                         | Returns                        | Description                                                           |
| ------------------------------------------------ | ------------------------------ | --------------------------------------------------------------------- |
| `Open(dbPath)`                                   | `(*sql.DB, error)`             | Open a local Turso database file                                      |
| `OpenInMemory()`                                 | `(*sql.DB, error)`             | In-memory database for testing                                        |
| `OpenSync(ctx, dbPath, url, token)`              | `(*SyncDB, error)`             | Open with remote sync                                                 |
| `OpenSyncWithConfig(ctx, ..., opts)`             | `(*SyncDB, error)`             | Open with remote sync + advanced [SyncOption] tuning                  |
| `InitSchema(ctx, db)`                            | `error`                        | Create all tables (events, commands, queries, snapshots, checkpoints) |
| `InitSchemaWithIndexes(ctx, db)`                 | `error`                        | Create tables + CQRS-optimized indexes                                |
| `InitSchemaWithIndexesAndOptimizations(ctx, db)` | `error`                        | Tables + indexes + performance PRAGMAs (one-shot production setup)    |
| `ConfigurePool(db)`                              | —                              | Cap connection pool at 1 (required for embedded LibSQL)               |
| `NewBackend(db)`                                 | `(*Backend, error)`            | Facade exposing all 5 stores sharing one `*sql.DB`                    |
| `NewEventStore(db)`                              | `(*SQLEventStore, error)`      | Event store backed by Turso                                           |
| `NewCommandStore(db)`                            | `(*SQLCommandStore, error)`    | Command audit store                                                   |
| `NewQueryStore(db)`                              | `(*SQLQueryStore, error)`      | Query audit store                                                     |
| `NewSnapshotStore(db)`                           | `(*SQLSnapshotStore, error)`   | Snapshot store                                                        |
| `NewCheckpointStore(db)`                         | `(*SQLCheckpointStore, error)` | Projection checkpoint store                                           |

All constructors take phantom-typed inputs (`DbPath`, `RemoteURL`, `AuthToken`) for compile-time type safety, and delegate to the SQLite implementations in the `storage` module (Turso uses the same SQL dialect as SQLite).

## Quick Start — Local Database

```go
ctx := context.Background()

db, _ := turso.Open(turso.DbPath("app.db"))
defer db.Close()
turso.ConfigurePool(db) // cap pool at 1 — required for embedded LibSQL

turso.InitSchema(ctx, db)

store, _ := turso.NewEventStore(db)
store.Save(ctx, ref, events, 0)
```

## Quick Start — Full Stack via Backend Facade

The `Backend` facade exposes all five CQRS stores (event, command, query, snapshot, checkpoint) sharing a single database connection. All accessors are goroutine-safe; command/query/snapshot/checkpoint stores are lazily created on first call.

```go
db, _ := turso.Open(turso.DbPath("app.db"))
defer db.Close()
turso.ConfigurePool(db)

backend, _ := turso.NewBackend(db)
defer backend.Close() // closes derived stores, NOT the *sql.DB

eventStore := backend.EventStore()             // eager
cmdStore, _ := backend.CommandStore()           // lazy, goroutine-safe
qStore, _   := backend.QueryStore()             // lazy, goroutine-safe
snapStore, _ := backend.SnapshotStore()          // lazy, goroutine-safe
cpStore, _  := backend.CheckpointStore()         // lazy, goroutine-safe
```

## Production Setup — Schema + Indexes + Pragmas

For new databases, create all tables, apply CQRS-optimized indexes, and set performance PRAGMAs in one call:

```go
db, _ := turso.Open(turso.DbPath("app.db"))
turso.ConfigurePool(db)

_ = turso.InitSchemaWithIndexesAndOptimizations(ctx, db)
```

## Remote Sync (Offline-First)

For offline-first applications with remote sync:

```go
syncDB, _ := turso.OpenSync(ctx,
    turso.DbPath("local.db"),
    turso.RemoteURL("libsql://my-db.turso.io"),
    turso.AuthToken("token"),
)
defer syncDB.Close()

syncDB.Push(ctx)                   // send local writes to remote
changed, _ := syncDB.Pull(ctx)     // receive remote changes (true if changed)
_ = syncDB.HealthCheck(ctx)        // verify connection for health probes
stats, _ := syncDB.Stats(ctx)      // WAL size, bytes sent/received
```

### Advanced Sync Configuration

Use `OpenSyncWithConfig` with `SyncOption` functions for fine-grained control:

```go
syncDB, _ := turso.OpenSyncWithConfig(ctx,
    turso.DbPath("local.db"),
    turso.RemoteURL("libsql://my-db.turso.io"),
    turso.AuthToken("token"),
    turso.WithSyncClientName("my-app"),
    turso.WithSyncLongPollTimeout(30*time.Second),
    turso.WithSyncBusyTimeout(10*time.Second),
    turso.WithSyncBootstrapIfEmpty(false),  // skip initial bootstrap
    turso.WithSyncNamespace("production"),
)
```

Available `SyncOption` functions:

| Option                        | Description                                                         |
| ----------------------------- | ------------------------------------------------------------------- |
| `WithSyncNamespace(s)`        | Isolate sync state between applications sharing a database          |
| `WithSyncClientName(s)`       | Unique client identifier for Turso sync diagnostics                 |
| `WithSyncLongPollTimeout(d)`  | Long-poll timeout for change detection (longer = lower latency)     |
| `WithSyncBootstrapIfEmpty(b)` | Skip initial full-state bootstrap (call `Pull` manually instead)    |
| `WithSyncBusyTimeout(d)`      | Busy timeout for write-lock acquisition (default 5s, -1 to disable) |

---

## Auto-Smart Indexing (turso/indexing)

The `turso/indexing` sub-package provides **auto-smart index management** for Turso/LibSQL databases. It analyzes `EXPLAIN QUERY PLAN` output, detects full-table scans, and recommends or automatically creates indexes optimized for CQRS event-sourcing workloads.

```bash
# No extra dependency — it's part of the turso module.
```

### Quick Start — Apply CQRS Indexes

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v3"

// One-shot: schema + all recommended CQRS indexes
db, _ := turso.OpenInMemory()
turso.InitSchemaWithIndexes(ctx, db)

// Or apply indexes to an existing database:
turso.ApplyCQRSIndexes(ctx, db)
```

### Recommended CQRS Indexes

`indexing.RecommendedCQRSIndexes()` returns pre-calculated indexes for common CQRS patterns:

| Index                    | Columns                                       | Purpose                                                |
| ------------------------ | --------------------------------------------- | ------------------------------------------------------ |
| `idx_events_cursor`      | `(occurred_at, id)`                           | Cursor pagination for `ReadFrom` / journal replay      |
| `idx_events_agg_ver`     | `(aggregate_type, aggregate_id, version)`     | Covering index for `LoadFromVersion` / `LoadToVersion` |
| `idx_events_type_time`   | `(event_type, occurred_at)`                   | Projection filters by event type with ordering         |
| `idx_commands_agg_time`  | `(aggregate_type, aggregate_id, received_at)` | Command audit trail with time ordering                 |
| `idx_commands_type_time` | `(command_type, received_at)`                 | Command type analytics                                 |

### Index Advisor

Analyze any query for missing indexes:

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v3/indexing"

advisor := indexing.NewAdvisor(db)
recs, _ := advisor.AnalyzeQuery(ctx,
    "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ?",
    "User", "user-id")
for _, r := range recs {
    fmt.Println(r.Priority, r.Index.DDL())
}
```

Analyze all CQRS tables at once:

```go
recs, _ := advisor.MissingIndexes(ctx)
```

### Auto-Indexer

Enable automatic index creation based on detected query-plan issues:

```go
auto := indexing.NewAutoIndexer(db)
auto.Enable()

// Apply all current recommendations
_ = auto.ApplyRecommended(ctx)

// Or apply the predefined CQRS indexes directly
_ = auto.ApplyCQRSIndexes(ctx)
```

The auto-indexer is **disabled by default** — consumers must explicitly `Enable()` it.

### Turso Performance Optimizations

Apply recommended PRAGMAs for CQRS workloads:

```go
_ = indexing.ApplyOptimizations(ctx, db)
```

This sets WAL mode, normal synchronous, 64MB page cache, and memory temp store.

Individual helpers are also available:

```go
_ = indexing.ApplyWAL(ctx, db)
_ = indexing.SetCacheSize(ctx, db, -64000)
_ = indexing.Analyze(ctx, db)          // update query planner statistics
_ = indexing.AnalyzeTable(ctx, db, "events")
```

Pragmas not supported by a specific LibSQL/Turso variant are silently skipped.

### Index Definition Helpers

Build and inspect index definitions programmatically:

```go
idx := indexing.Index{
    Name:    "idx_my_index",
    Table:   "events",
    Columns: []string{"event_type", "occurred_at"},
}
fmt.Println(idx.DDL())      // CREATE INDEX IF NOT EXISTS ...
fmt.Println(idx.DropDDL())  // DROP INDEX IF EXISTS ...
```

[SyncOption]: https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/turso/v3#SyncOption

## Related Modules

- [**storage/v2**](../storage/README.md) — Turso delegates SQL store implementations to this module
- [**pebble/v2**](../pebble/README.md) — Sibling embedded backend (PebbleDB)
- [**memory/v2**](../memory/README.md) — In-memory implementations for tests
- [**event/v2**](../event/README.md) — Event store interface
- [**command/v2**](../command/README.md) — Command store interface
- [**query/v2**](../query/README.md) — Query store interface
- [**snapshot/v2**](../snapshot/README.md) — Snapshot store interface
- [**otel/v2**](../otel/README.md) — Index analysis emits spans via `otel/` re-exports
