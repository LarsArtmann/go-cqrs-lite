# Bundle Presets

> **Consumers should NOT decide on infrastructure.** Presets assemble the full
> CQRS stack from one call. The deployer picks a preset; the app developer
> never imports a backend.

## Quick Start

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"

b, err := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()

// Typed read model
store, _ := stack.ReadModel[TodoView, TodoID](b, codec.JSONCodec{},
    readmodel.WithKeyPrefix[TodoView, TodoID]("todos:"))
```

## Available Presets

| Preset | Module | Backend | Persistent | Bus | Read Models |
|--------|--------|---------|------------|-----|-------------|
| Memory | `stack/memory` | In-memory | No | Memory | Memory KV |
| SQLite | `stack/sqlite` | SQLite (modernc) | Yes | Memory | Memory KV |
| Pebble | `stack/pebble` | PebbleDB (LSM) | Yes | Memory | Pebble KV |
| Postgres | `stack/postgres` | PostgreSQL (pgx) | Yes | Memory | Memory KV |

All presets wire every capability: event store + bus, command store, query
store, snapshot store, checkpoint store, and read-model backend.

## Usage

### Memory (development, testing)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/memory/v2"

b, _ := memory.New()
defer b.Close()
```

### SQLite (single-node persistence)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v2"

b, _ := sqlite.New("app.db")          // or ":memory:"
defer b.Close()
```

Options: `WithoutWAL()`, `WithoutAutoMigrate()`.

### Pebble (embedded high-throughput)

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"

b, _ := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()
```

Options: `WithPebbleOptions(*pebble.Options)`, `WithLogger(*slog.Logger)`.

Read models are **persisted to the same PebbleDB** via a shared KV adapter.

### Postgres (managed database)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/postgres/v2"

b, _ := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
defer b.Close()
```

Option: `WithoutAutoMigrate()`.

Tests require `POSTGRES_TEST_DSN` env var; they skip when unset.

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
    ReadModels      readmodel.Backend // kv.Store
}
```

Fields may be nil. Accessors return an error when a required capability is absent.

## Read Models

```go
store, _ := stack.ReadModel[TodoView, TodoID](b, codec.JSONCodec{},
    readmodel.WithKeyPrefix[TodoView, TodoID]("todos:"))

store.Set(ctx, id, &TodoView{Title: "buy milk"})
got, _ := store.Get(ctx, id)
```

### With Cache

```go
import "github.com/larsartmann/go-cqrs-lite/readmodel/cache/v2"

cached, _ := cache.New(store,
    cache.WithCapacity[TodoView, TodoID](10_000),
    cache.WithTTL[TodoView, TodoID](5*time.Minute),
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
