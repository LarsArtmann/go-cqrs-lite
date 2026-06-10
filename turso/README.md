# turso — Turso Database Connectors

CQRS storage adapters for [Turso](https://turso.tech/) databases (embedded LibSQL/SQLite).

```bash
go get github.com/larsartmann/go-cqrs-lite/turso/v2
```

## Constructors

| Function                            | Returns               | Description                      |
| ----------------------------------- | --------------------- | -------------------------------- |
| `Open(dbPath)`                      | `*sql.DB`             | Open a local Turso database file |
| `OpenInMemory()`                    | `*sql.DB`             | In-memory database for testing   |
| `OpenSync(ctx, dbPath, url, token)` | `*SyncDB`             | Open with remote sync            |
| `InitSchema(ctx, db)`               | —                     | Create all tables                |
| `InitSchemaWithIndexes(ctx, db)`    | —                     | Create tables + CQRS indexes     |
| `NewEventStore(db)`                 | `*SQLEventStore`      | Event store backed by Turso      |
| `NewSnapshotStore(db)`              | `*SQLSnapshotStore`   | Snapshot store                   |
| `NewCheckpointStore(db)`            | `*SQLCheckpointStore` | Checkpoint store                 |

## Quick Start

```go
db, _ := turso.OpenInMemory()
turso.InitSchema(ctx, db)

store, _ := turso.NewEventStore(db)
store.Save(ctx, ref, events, 0)
```

## Sync

For offline-first with remote sync:

```go
syncDB, _ := turso.OpenSync(ctx, "local.db", "libsql://my-db.turso.io", "token")
syncDB.Push(ctx)  // send local changes
syncDB.Pull(ctx)  // receive remote changes
```

All constructors delegate to the equivalent SQLite implementations in the `storage` module.

---

## Auto-Smart Indexing (turso/indexing)

The `turso/indexing` sub-package provides **auto-smart index management** for Turso/LibSQL databases. It analyzes `EXPLAIN QUERY PLAN` output, detects full-table scans, and recommends or automatically creates indexes optimized for CQRS event-sourcing workloads.

```bash
# No extra dependency — it's part of the turso module.
```

### Quick Start — Apply CQRS Indexes

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v2"

// One-shot: schema + all recommended CQRS indexes
db, _ := turso.OpenInMemory()
turso.InitSchemaWithIndexes(ctx, db)

// Or apply indexes to an existing database:
turso.ApplyCQRSIndexes(ctx, db)
```

### Recommended CQRS Indexes

`indexing.RecommendedCQRSIndexes()` returns pre-calculated indexes for common CQRS patterns:

| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_events_cursor` | `(occurred_at, id)` | Cursor pagination for `ReadFrom` / journal replay |
| `idx_events_agg_ver` | `(aggregate_type, aggregate_id, version)` | Covering index for `LoadFromVersion` / `LoadToVersion` |
| `idx_events_type_time` | `(event_type, occurred_at)` | Projection filters by event type with ordering |
| `idx_commands_agg_time` | `(aggregate_type, aggregate_id, received_at)` | Command audit trail with time ordering |
| `idx_commands_type_time` | `(command_type, received_at)` | Command type analytics |

### Index Advisor

Analyze any query for missing indexes:

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"

advisor := indexing.NewAdvisor(db)
recs, _ := advisor.AnalyzeQuery(ctx,
    "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ?",
    "User", "user-id")
for _, r := range recs {
    fmt.Println(r.Reason, r.Index.DDL())
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
