# storage — SQL and Pebble Event Store Backends

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/storage.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-cqrs-lite/storage)

Persistent event store implementations for PostgreSQL, SQLite, Turso, and Pebble. Implements the `event.Store`, `event.SnapshotStore`, `event.Bus` (outbox), and `saga.Store` interfaces from core.

```bash
go get github.com/larsartmann/go-cqrs-lite/storage
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

## SQLBackend (Recommended)

`SQLBackend` creates all SQL-backed stores at once — event store, outbox, transactional store, and saga store — sharing a single `*sql.DB`:

```go
backend, _ := storage.NewSQLiteBackend(db)

// Access individual stores
backend.EventStore()        // *SQLEventStore — implements event.Store
backend.Outbox()            // *SQLOutbox — append/poll/ack
backend.TransactionalStore() // event.TransactionalStore — atomic save+outbox
backend.SagaStore()         // saga.Store — persistent saga state
```

### Constructors

| Function                          | Dialect    |
| --------------------------------- | ---------- |
| `NewSQLBackend(db)`               | PostgreSQL |
| `NewSQLiteBackend(db)`            | SQLite     |
| `NewSQLBackendWithDialect(db, d)` | Custom     |

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

### SQLOutbox + OutboxPoller

Transactional outbox pattern — save events and outbox entries atomically, then poll and publish:

```go
outbox, _ := storage.NewSQLiteOutbox(db)

// Append events to outbox (call after store.Save)
outbox.Append(ctx, events)

// Poll for pending entries
entries, _ := outbox.PollPending(ctx, 100)

// Ack processed entries
outbox.Ack(ctx, entryIDs)

// Background poller
poller := storage.NewOutboxPoller(outbox, bus,
    storage.WithPollInterval(5*time.Second),
    storage.WithBatchSize(50),
)
poller.Start(ctx) // runs until ctx cancelled
```

### SQLSnapshotStore

Implements `event.SnapshotStore`:

```go
snapStore, _ := storage.NewSQLiteSnapshotStore(db)
snapStore.Save(ctx, event.Snapshot{
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

### SQLSagaStore

Persistent saga state (implements `saga.Store`):

```go
sagaStore, _ := storage.NewSQLiteSagaStore(db)
sagaStore.Save(ctx, &saga.State{...})
state, _ := sagaStore.Load(ctx, sagaInstanceID)
running, _ := sagaStore.LoadAllRunning(ctx)
```

### PebbleEventStore

Embedded key-value store for single-process apps. No SQL dependency:

```go
db, _ := pebble.Open("data", &pebble.Options{})
store := storage.NewPebbleStore(db, slog.Default())

// Same event.Store interface
store.Save(ctx, "User", aggID, events, 0)
events, _ := store.Load(ctx, "User", aggID)
```

## Schema Management

```go
// SQLite
storage.SQLiteInitSchema(ctx, db)  // creates all tables
storage.SQLiteEnableWAL(ctx, db)   // write-ahead logging for performance

// PostgreSQL
storage.PostgresInitSchema(ctx, db) // creates all tables

// Turso
db, _ := storage.OpenTurso("myapp.db")
storage.TursoInitSchema(ctx, db)
```

### DDL Functions

Get DDL strings for custom schema management:

```go
storage.EventSchema()          // PostgreSQL events DDL
storage.SnapshotSchema()       // PostgreSQL snapshots DDL
storage.CheckpointSchema()     // PostgreSQL checkpoints DDL
storage.OutboxSchema()         // PostgreSQL outbox DDL
storage.SagaSchema()           // PostgreSQL sagas DDL

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
    OutboxSchema() string
    SagaSchema() string
}
```

Provided implementations: `PostgresDialect{}`, `SQLiteDialect{}`.

## Dependencies

| Dependency           | Purpose                           |
| -------------------- | --------------------------------- |
| `core`               | Event/ID/saga interfaces          |
| `saga`               | Saga state types for SQLSagaStore |
| `cockroachdb/pebble` | PebbleEventStore (optional)       |
