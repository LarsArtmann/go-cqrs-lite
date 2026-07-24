# snapshot — Aggregate State Snapshots

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/snapshot/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/snapshot/v4)

Capture aggregate state at a version to avoid full event replay on each load. Snapshot strategies decide when to capture; snapshot stores handle persistence.

```bash
go get github.com/larsartmann/go-cqrs-lite/snapshot/v4
```

## Quick Start

```go
// Save a snapshot
store.Save(ctx, snapshot.Snapshot{
    StreamID:   aggID,
    StreamType: "User",
    Version:    10,
    State:      encodedState,
    CreatedAt:  time.Now(),
})

// Load the latest snapshot
loaded, err := store.Load(ctx, ref)

// Load at or before a specific version
loaded, err := store.LoadAtVersion(ctx, ref, 10)

// Configure the decider to use snapshots
strategy, _ := snapshot.EveryNEvents(100)
repo, _ := decider.NewRepository(store, bus, d,
    decider.WithSnapshotStore(snapStore),
    decider.WithSnapshotStrategy(strategy),
)
```

## API

### Core Types

| Type             | Description                                             |
| ---------------- | ------------------------------------------------------- |
| `Snapshot`       | Captured state: `StreamID`, `StreamType`, `Version`, `State []byte`, `CreatedAt`. |
| `SnapshotSink`   | Write side: `Save(ctx, Snapshot)`.                      |
| `SnapshotSource` | Read side: `Load(ctx, ref)`, `LoadAtVersion(ctx, ref, v)`. |
| `SnapshotStore`  | `SnapshotSink` + `SnapshotSource`.                      |

### Strategies

| Strategy                 | Description                                                          |
| ------------------------ | -------------------------------------------------------------------- |
| `EveryNEvents(n)`        | Snapshot after every N events. Returns `(SnapshotStrategy, error)`.  |
| `NewReadPressure(loads)` | Snapshot after N loads + next write. Optimizes for read-heavy aggregates. |
| `WithInnerStrategy(s)`   | Combines with `NewReadPressure` — either strategy triggers.          |

### TypedSnapshotStore

```go
typedStore := snapshot.NewTypedStore[UserState](store, codec.JSONCodec{})

// Save typed snapshot
typedStore.Save(ctx, snapshot.TypedSnapshot[UserState]{
    StreamID:   aggID,
    StreamType: "User",
    Version:    10,
    State:      UserState{Name: "Alice"},
    CreatedAt:  time.Now(),
})

// Load typed snapshot
ts, _ := typedStore.Load(ctx, ref)
// ts.State is UserState, ts.Version is event.Version
```

## Design

- **One snapshot per aggregate**: Stores keep only the latest snapshot. Older versions are silently ignored (no state regression).
- **ReadPressure**: Tracks aggregate load frequency. After `loads` loads, the next write triggers a snapshot. This optimizes for hot aggregates that are read frequently but rarely written.
- **TypedSnapshot**: Generic wrapper that carries the decoded `State` alongside `StreamID`, `StreamType`, `Version`, and `CreatedAt`.
- **Codec**: Snapshot state is encoded via a `codec.Codec`. Default in `decider.NewRepository` is CBOR.

## Implementations

| Implementation                    | Module                                            |
| --------------------------------- | ------------------------------------------------- |
| `MemorySnapshotStore`             | [storage/memory](../storage/memory/README.md)     |
| `SQLSnapshotStore`                | [storage](../storage/README.md)                   |
| PebbleDB `SnapshotStore`          | [storage/pebble](../storage/pebble/README.md)     |
| `FakeSnapshotStore`               | [eventtest](../event/v4/eventtest/README.md)      |

## Related Modules

- [**decider**](../decider/README.md) — `WithSnapshotStore` + `WithSnapshotStrategy` for aggregate loading
- [**event**](../event/README.md) — `event.Version` type used in snapshots
- [**codec**](../codec/README.md) — Serialization of snapshot state
