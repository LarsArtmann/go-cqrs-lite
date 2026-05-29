# Storage Backend Guide

go-cqrs-lite supports multiple storage backends for event persistence.

## Backends

### PostgreSQL

Production-ready. Full transactional support with outbox co-participation.

```go
db, _ := sql.Open("pgx", "postgres://...")
storage.PostgresInitSchema(ctx, db)

store, _ := storage.NewSQLEventStore(db)
outbox, _ := storage.NewSQLOutbox(db)
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
outbox, _ := storage.NewSQLiteOutbox(db)
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
| `SaveWithOutbox(ctx, aggType, aggID, events, ver)`   | Atomic save + outbox (TransactionalSink)    |

## Outbox Pattern

Reliable eventual publishing. Events are stored in the same transaction as the event store, then a background poller publishes them.

```go
poller := storage.NewOutboxPoller(outbox, bus,
    storage.WithPollInterval(5*time.Second),
    storage.WithPollLimit(100),
)
go poller.Run(ctx)
```

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
storage.Schema()          // events + outbox tables
storage.SnapshotSchema()  // snapshots table
storage.CheckpointSchema()// checkpoints table
storage.SagaSchema()      // sagas table

// SQLite
storage.SQLiteSchema()
storage.SQLiteSnapshotSchema()
storage.SQLiteCheckpointSchema()
storage.SQLiteSagaSchema()
```
