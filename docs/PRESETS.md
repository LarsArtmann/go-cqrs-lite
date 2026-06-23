# Bundle Presets

> **Consumers should NOT decide on infrastructure.** Presets assemble the full
> CQRS stack from one call. The deployer picks a preset; the app developer
> never imports a backend. For **why** each engine fits each concern, see
> [Infrastructure Recommendations](INFRASTRUCTURE_RECOMMENDATIONS.md).

## Quick Start

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v3"

b, err := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()

// Typed read model
store, _ := stack.ReadModel[TodoView, TodoID](b, codec.JSONCodec{},
    kv.WithTypedKeyPrefix[TodoView, TodoID]("todos:"))
```

## Available Presets

| Preset   | Module           | Backend          | Persistent | Bus                    | Read Models      |
| -------- | ---------------- | ---------------- | ---------- | ---------------------- | ---------------- |
| Memory   | `stack/memory`   | In-memory        | No         | Memory                 | Memory KV        |
| SQLite   | `stack/sqlite`   | SQLite (modernc) | Yes        | Memory                 | SQL KV (cqrs_kv) |
| Pebble   | `stack/pebble`   | PebbleDB (LSM)   | Yes        | Memory                 | Pebble KV        |
| Postgres | `stack/postgres` | PostgreSQL (pgx) | Yes        | Memory / LISTEN-NOTIFY | SQL KV (cqrs_kv) |
| Turso    | `stack/turso`    | LibSQL           | Yes        | Memory                 | SQL KV (cqrs_kv) |

All presets wire every capability: event store + bus, command store, query
store, snapshot store, checkpoint store, and read-model backend.

## Usage

### Memory (development, testing)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/memory/v3"

b, _ := memory.New()
defer b.Close()
```

### SQLite (single-node persistence)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"

b, _ := sqlite.New("app.db")          // or ":memory:"
defer b.Close()
```

Options: `WithoutWAL()`, `WithoutAutoMigrate()`.

#### Multi-DB split (SQLite flagship)

Split concerns across separate database files to eliminate reader/writer
contention in production:

```go
b, _ := sqlite.New("primary.db",
    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
    sqlite.WithQueryDB("queries.db"),  // command + query audit logs
    sqlite.WithViewDB("views.db"),     // materialized views (KV)
)
defer b.Close()
```

| Database     | Contains                       | Rationale                                        |
| ------------ | ------------------------------ | ------------------------------------------------ |
| **Event DB** | events, snapshots, checkpoints | Event-sourcing write model, isolated from reads  |
| **Query DB** | commands, queries              | Operational audit log, isolated from write model |
| **View DB**  | materialized views (`cqrs_kv`) | Read scans don't contend with event appends      |

Default (single database) is fine for development and low-traffic apps. See
[Infrastructure Recommendations](INFRASTRUCTURE_RECOMMENDATIONS.md) for the
full rationale.

### Pebble (embedded high-throughput)

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v3"

b, _ := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()
```

Options: `WithPebbleOptions(*pebble.Options)`, `WithLogger(*slog.Logger)`.

Read models are **persisted to the same PebbleDB** via a shared KV adapter.

### Postgres (managed database)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/postgres/v3"

b, _ := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
defer b.Close()
```

Option: `WithoutAutoMigrate()`.

Distributed bus (cross-process pub/sub): `WithDistributedBus(listener)` wires
Postgres LISTEN/NOTIFY instead of the in-process GoChannel bus.

Tests require `POSTGRES_TEST_DSN` env var; they skip when unset.

### Turso (edge / offline-first)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/turso/v3"

// Local-only (like SQLite)
b, _ := turso.New("app.db")

// With remote sync (offline-first)
b, _ := turso.NewSync(ctx, "app.db", remoteURL, authToken)
defer b.Close()

// Sync control (Push / Pull / Checkpoint)
b.Sync().Push(ctx)
```

Options: `WithoutAutoMigrate()`, `WithEventDB`, `WithQueryDB`, `WithViewDB`
(local mode only — multi-DB split is not supported with sync).

## Bundle Fields

The Bundle uses Interface Segregation — each capability is a separate field:

```go
type Bundle struct {
    EventSink   event.EventSink       // Save, AppendBatch
    EventSource event.EventSource     // Load, LoadFromVersion, ...
    Journal     event.Journal         // ReadAll (cross-aggregate)
    SeekableJournal event.SeekableJournal // ReadFrom (position-based)
    Publisher   event.Publisher       // Publish
    Subscriber  event.Subscriber      // Subscribe
    CommandSink   command.CommandSink
    CommandSource command.CommandSource
    QuerySink   query.QuerySink
    QuerySource query.QuerySource
    SnapshotStore   snapshot.SnapshotStore
    CheckpointStore event.CheckpointStore
    ReadModels      kv.Store // read-model KV store
}
```

Fields may be nil. Accessors return an error when a required capability is absent.

## Read Models

Read models are **persisted to the same database** as events for SQLite, Pebble,
and Postgres presets (via a SQL `kv.Store` over the `cqrs_kv` table, or a shared
Pebble KV adapter). Only the Memory preset holds read models in process memory.
This means a read model survives a process restart for every persistent preset.

```go
store, _ := stack.ReadModel[TodoView, TodoID](b, codec.JSONCodec{},
    kv.WithTypedKeyPrefix[TodoView, TodoID]("todos:"))

store.Set(ctx, id, &TodoView{Title: "buy milk"})
got, _ := store.Get(ctx, id)
```

### With Cache

```go
import "github.com/larsartmann/go-cqrs-lite/kv/v3"

cached, _ := kv.NewCache(store,
    kv.WithCacheCapacity[TodoView, TodoID](10_000),
    kv.WithCacheTTL[TodoView, TodoID](5*time.Minute),
)
defer cached.Close()
```

## Custom Bundle

```go
b, _ := stack.New(
    stack.WithEventStore(myEventStore),
    stack.WithBus(myBus),
    stack.WithReadModels(kv.NewMemStore()),
    stack.WithCloser(myDB),
)
```

## Contract Tests

Every preset satisfies the same behavioral contract. Run the suite:

```go
contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
    return sqlite.New(filepath.Join(t.TempDir(), "test.db"))
})
```

Tests: BundleFields, EventRoundtrip, CommandRoundtrip, ReadModelRoundtrip,
CloseIdempotent.

## Zero Overhead

Bundle field access is a struct field load (0.20 ns/op), identical to a local
variable. Method calls through Bundle fields match direct store calls in ns/op
and allocs/op. See `stack/bench/`.
