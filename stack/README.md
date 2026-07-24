# stack — Composition Layer and Bundle

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/v4)

The composition root for go-cqrs-lite. A `Bundle` is a bag of peer capability fields (event sink/source, journals, publishers, snapshot store, checkpoint store, read-model backend) that a deployment wires together. Plus `Materialize[V,K]` — the tombstone-aware projection builder.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/v4
```

## Why?

go-cqrs-lite is a library, not a framework. The stack layer provides the glue: a `Bundle` that holds references to all infrastructure components, and a set of presets (memory, SQLite, Pebble, Postgres, Turso) that wire them together. Consumers swap infrastructure by changing one line — the domain code doesn't change.

## Quick Start

### Use a Preset

```go
import sqlite "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

bundle, err := sqlite.New("app.db")
if err != nil { log.Fatal(err) }
defer bundle.Close()

// Access capabilities:
store := bundle.EventStore()       // event.Store
bus   := bundle.EventBus()          // event.Bus
repo  := bundle.Repository(decider) // decider.Repository[State]
```

### Build a Read Model

```go
mat := stack.Materialize[TodoView, TodoID]{
    Store:        kvStore,
    KeyFromEvent: func(evt event.Event) (TodoID, error) { ... },
    OnCreate:     func(ctx, evt) (*TodoView, error) { ... },
    OnUpdate:     func(ctx, evt, existing *TodoView) (*TodoView, error) { ... },
    OnTombstone:  func(ctx, evt, existing *TodoView) (*TodoView, error) { ... },
}

// Register as a projection:
host.Register(mat.AsProjection())
```

## API

### Bundle

| Symbol                    | Kind   | Description                                                        |
| ------------------------- | ------ | ------------------------------------------------------------------ |
| `Bundle`                  | Struct | Peer capability fields: events, commands, queries, snapshots, etc. |
| `Bundle.EventStore()`     | Method | Returns `event.Store` (or error if not configured).                |
| `Bundle.EventBus()`       | Method | Returns `event.Bus` (or error if not configured).                  |
| `Bundle.Repository(d)`    | Method | Returns `decider.Repository[State]` wired to store + bus.          |
| `Bundle.ReadModel(...)`   | Method | Returns a `kv.ViewStore[V,K]` for read models.                     |
| `Bundle.HealthCheck(ctx)` | Method | Pings the DB + calls HealthCheck on registered resources.          |
| `Bundle.Close()`          | Method | Closes all registered closers (deduplicated by pointer).           |

### Materialize[V, K]

| Symbol              | Kind   | Description                                                             |
| ------------------- | ------ | ----------------------------------------------------------------------- |
| `Materialize[V, K]` | Struct | Tombstone-aware projection builder. Implements `projection.Projection`. |
| `TombstonePolicy`   | Type   | `IncludeTombstoned`, `ExcludeTombstoned` (default), `OnlyTombstoned`.   |
| `Store`             | Field  | Any `kv.ViewStore[V, K]` implementation.                                |
| `KeyFromEvent`      | Field  | Extracts the view key from an event.                                    |
| `OnCreate`          | Field  | Callback for new view creation.                                         |
| `OnUpdate`          | Field  | Callback for view updates.                                              |
| `OnTombstone`       | Field  | Callback for tombstone events.                                          |
| `OnRebirth`         | Field  | Callback when a tombstoned view is re-created.                          |

### Options

| Option                         | Description                                            |
| ------------------------------ | ------------------------------------------------------ |
| `WithEventSink(s)`             | Sets the write-side event store.                       |
| `WithEventSource(s)`           | Sets the read-side event store.                        |
| `WithSeekableJournal(j)`       | Sets the position-based journal for projection hosts.  |
| `WithPublisher(p)`             | Sets the event publisher.                              |
| `WithSnapshotStore(s)`         | Sets the snapshot store.                               |
| `WithCheckpointStore(c)`       | Sets the checkpoint store.                             |
| `WithShutdownDependency(a, b)` | Declares that `a` closes AFTER `b` (topological sort). |

## Design

- **"Bundle" not "Container"**: Fields are peers (no LIFO ordering). Ownership is explicit and deduplicated by pointer.
- **Interface Segregation**: Stores are segregated interfaces (`EventSink`, `EventSource`, `Journal`, `SeekableJournal`) rather than a fat `event.Store`, so a write-only consumer stays oblivious to read paths.
- **No Provider interface**: Go can't do partial interface implementation, so presets are ordinary functions returning `(*Bundle, error)`.
- **Resource lifecycle**: `Bundle.Close` deduplicates closers by pointer and closes each once. Presets roll back on partial construction failure.
- **Shutdown ordering**: `WithShutdownDependency("eventstore", "projectionhost")` ensures the event store closes AFTER projections drain.

## Codec Defaults

```
LAYER                      | DEFAULT CODEC     | HOW TO OVERRIDE
---------------------------|-------------------|----------------------------------
stack.ReadModel/Materialize| CBORCodec         | stack.WithDefaultCodec(json)
event.New()                | CBORCodec         | event.DefaultCodec = codec.JSONCodec{}
                             or event.WithCodec(c) per-event
```

Events are self-describing: `evt.Encoding()` is stamped on every event, so mixed JSON+CBOR streams decode correctly.

## Related Modules

- [**stack/memory**](memory/README.md) — All-in-memory preset
- [**stack/sqlite**](sqlite/README.md) — SQLite preset (recommended for single-node)
- [**stack/pebble**](pebble/README.md) — PebbleDB preset
- [**stack/postgres**](postgres/README.md) — PostgreSQL preset
- [**stack/turso**](turso/README.md) — Turso preset with remote sync
- [**kv**](../kv/README.md) — `ViewStore[V,K]` interface for read models
