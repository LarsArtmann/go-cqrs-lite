# storage — SQL Event Store Backends

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/storage/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/storage/v4)

Persistent event store implementations for PostgreSQL, SQLite, and SQLite-compatible backends. Implements the `event.Store`, `snapshot.SnapshotStore`, and `event.CheckpointStore` interfaces.

> **Pebble and Turso are now separate modules.** See `github.com/larsartmann/go-cqrs-lite/pebble` and `github.com/larsartmann/go-cqrs-lite/turso` for those backends.

```bash
go get github.com/larsartmann/go-cqrs-lite/storage/v4
```

## Quick Start: SQLite

```go
db, _ := storage.OpenSQLite("myapp.db")
storage.SQLiteEnableWAL(ctx, db)
storage.ConfigureSQLitePool(db)
storage.SQLiteInitSchema(ctx, db)

store, _ := storage.NewSQLiteEventStore(db)
bus := memory.NewMemoryBus()

// Use with decider
repo, _ := decider.NewRepository[UserState](store, bus, myDecider)
```

## Quick Start: PostgreSQL

```go
db, _ := sql.Open("pgx", "postgres://user:pass@localhost/mydb")
storage.PostgresInitSchema(ctx, db)

store, _ := storage.NewSQLEventStore(db)
```

## Components

### SQLEventStore

Implements `event.Store` with optimistic concurrency control:

```go
store, _ := storage.NewSQLiteEventStore(db)

// Save events (checks expected version)
store.Save(ctx, "User", aggID, events, expectedVersion)

// Load all events for an aggregate
events, _ := store.Load(ctx, "User", aggID)

// Load from specific version
events, _ := store.LoadFromVersion(ctx, "User", aggID, 5)

// Load up to a timestamp
events, _ := store.LoadToTimestamp(ctx, "User", aggID, someTime)

// Stream events (cursor-based, memory-efficient)
stream, _ := store.LoadStream(ctx, "User", aggID)
defer stream.Close()
for {
    evt, ok := stream.Next()
    if !ok { break }
    // process evt
}
```

### SQLSnapshotStore

Implements `snapshot.SnapshotStore`:

```go
snapStore, _ := storage.NewSQLiteSnapshotStore(db)
snapStore.Save(ctx, snapshot.Snapshot{
    AggregateID:   aggID,
    AggregateType: "User",
    Version:       10,
    State:         encodedState,
})
snap, _ := snapStore.Load(ctx, "User", aggID)
```

### SQLCheckpointStore

Tracks projection positions:

```go
cpStore, _ := storage.NewSQLiteCheckpointStore(db)
cpStore.Save(ctx, "user-projection", lastEventID)
checkpoint, _ := cpStore.Load(ctx, "user-projection")
```

### PebbleEventStore

Moved to separate module: `github.com/larsartmann/go-cqrs-lite/pebble`

```go
import "github.com/larsartmann/go-cqrs-lite/pebble"

db, _ := pebble.Open("data", &pebble.Options{})
store := pebble.NewPebbleStore(db, slog.Default())
```

## Schema Management

```go
// SQLite
storage.SQLiteInitSchema(ctx, db)  // creates all tables
storage.SQLiteEnableWAL(ctx, db)   // write-ahead logging for performance

// PostgreSQL
storage.PostgresInitSchema(ctx, db) // creates all tables

// Turso is a separate module: github.com/larsartmann/go-cqrs-lite/turso
// See the Turso section below
```

### DDL Functions

Get DDL strings for custom schema management:

```go
storage.EventSchema()          // PostgreSQL events DDL
storage.SnapshotSchema()       // PostgreSQL snapshots DDL
storage.CheckpointSchema()     // PostgreSQL checkpoints DDL

storage.SQLiteEventSchema()    // SQLite variants...
```

## Dialect

The `Dialect` interface abstracts SQL differences between PostgreSQL and SQLite (placeholder style, timestamp format, DDL):

```go
type Dialect interface {
    Placeholder(index int) string
    FormatTime(t time.Time) any
    ScanTimeDest() any
    ParseTime(src any) (time.Time, error)
    EventSchema() string
    SnapshotSchema() string
    CheckpointSchema() string
}
```

Provided implementations: `PostgresDialect{}`, `SQLiteDialect{}`.

## sql/ Subpackage

The `storage/sql` subpackage contains shared SQL infrastructure used by all SQL-based stores:

| Component              | Description                                |
| ---------------------- | ------------------------------------------ |
| `Base`                 | Shared `*sql.DB` + `Dialect` holder        |
| `Dialect`              | PostgreSQL/SQLite abstraction interface    |
| `Placeholders`         | Generate comma-separated placeholder lists |
| `ParseSQLiteTimestamp` | Multi-format SQLite timestamp parser       |
| `SharedInsertEvents`   | Shared event insertion logic               |
| `SharedCheckpointLoad` | Shared checkpoint read logic               |
| `SharedEventLoad`      | Shared event scanning logic                |
| `DeleteByAggregate`    | Shared DELETE implementation               |

The subpackage also defines all SQL-level sentinel errors: `ErrNilDB`, `ErrAggregateTypeMismatch`, `ErrAggregateIDMismatch`, `ErrVersionMismatch`, `ErrConcurrencyConflict`, `ErrUnsupportedTimestamp`, `ErrUnexpectedTimeType`.

## Turso

Turso is a separate module. See `github.com/larsartmann/go-cqrs-lite/turso`.

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v4"

db, _ := turso.OpenInMemory()
turso.InitSchema(ctx, db)
store, _ := turso.NewEventStore(db)
```

## Dependencies

| Dependency | Purpose              |
| ---------- | -------------------- |
| `event`    | Event/ID interfaces  |
| `snapshot` | Snapshot persistence |
| `otel`     | OTel helpers         |
| `listing`  | Aggregate listing    |

## Related Modules

- [**event**](../event/README.md) — Event store interfaces implemented here
- [**snapshot**](../snapshot/README.md) — Snapshot store interfaces implemented here
- [**decider**](../decider/README.md) — Wires SQL stores into the aggregate repository
- [**pebble**](../storage/pebble/README.md) — Embedded alternative backend (PebbleDB, CBOR)
- [**turso**](../turso/README.md) — Turso connector that delegates to this module
- [**memory**](../storage/memory/README.md) — In-memory implementations for tests
- [**otel**](../otel/README.md) — Span recording via `otel/` re-exports
- [**listing**](../listing/README.md) — SQL-backed aggregate reader
