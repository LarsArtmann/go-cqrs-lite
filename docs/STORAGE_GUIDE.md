# Storage Backend Guide

go-cqrs-lite supports multiple storage backends for event persistence.

## Backends

### PostgreSQL

Production-ready. Full transactional support.

```go
db, _ := sql.Open("pgx", "postgres://...")
storage.PostgresInitSchema(ctx, db)

store, _ := storage.NewSQLEventStore(db)
snapshots, _ := storage.NewSQLSnapshotStore(db)
checkpoints, _ := storage.NewSQLCheckpointStore(db)
sagas, _ := storage.NewSQLSagaStore(db)
```

### SQLite

Local-first, embedded. Supports WAL mode for concurrent reads.

```go
db, _ := storage.OpenSQLite("events.db")
storage.SQLiteInitSchema(ctx, db)
storage.SQLiteEnableWAL(ctx, db)

store, _ := storage.NewSQLiteEventStore(db)
```

### Pebble (Embedded KV)

High-performance embedded store for single-process deployments. No SQL dependency.

```go
store, _ := storage.OpenPebbleEventStore("data/events",
    storage.WithPebbleLogger(slog.Default()),
)
```

### Turso (libSQL)

Remote SQLite-compatible database.

```go
db, _ := storage.OpenTurso("libsql://...", authToken)
```

## Event Store Operations

| Method                                               | Description                                 |
| ---------------------------------------------------- | ------------------------------------------- |
| `Save(ctx, aggType, aggID, events, expectedVersion)` | Append events with optimistic concurrency   |
| `AppendBatch(ctx, aggType, aggID, events)`           | Append without concurrency check            |
| `Load(ctx, aggType, aggID)`                          | Load all events for an aggregate            |
| `LoadFromVersion(ctx, aggType, aggID, version)`      | Load events starting from version           |
| `LoadToVersion(ctx, aggType, aggID, maxVersion)`     | Load events up to version (time-travel)     |
| `LoadToTimestamp(ctx, aggType, aggID, maxTime)`      | Load events up to timestamp (time-travel)   |
| `ReadAll(ctx)`                                       | Load all events across aggregates (Journal) |
| `ReadFrom(ctx, afterEventID, limit)`                 | Cursor-based global load (SeekableJournal)  |

## Snapshot Store

Optimize aggregate loading by storing periodic state snapshots.

```go
snapshotStore, _ := storage.NewSQLSnapshotStore(db)
```

Snapshots store raw `[]byte` — serialization is the caller's responsibility.

## Checkpoint Store

Track projection progress for replay.

```go
checkpointStore, _ := storage.NewSQLCheckpointStore(db)
```

## Schema DDL

```go
// PostgreSQL
storage.Schema()          // events table
storage.SnapshotSchema()  // snapshots table
storage.CheckpointSchema()// checkpoints table
storage.SagaSchema()      // sagas table

// SQLite
storage.SQLiteSchema()
storage.SQLiteSnapshotSchema()
storage.SQLiteCheckpointSchema()
storage.SQLiteSagaSchema()
```
