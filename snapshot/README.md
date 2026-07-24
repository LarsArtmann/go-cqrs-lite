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
    AggregateID:   aggID,
    AggregateType: "User",
    Version:       10,
    State:         encodedState,
    CreatedAt:     time.Now(),
})

// Load the latest snapshot at or before a version
loaded, err := store.LoadAtVersion(ctx, ref, 10)

// Configure the decider to use snapshots
repo, _ := decider.NewRepository(store, bus, d,
    decider.WithSnapshotStore(snapStore),
    decider.WithSnapshotStrategy(snapshot.EveryNEvents(100)),
)
```

## API

### Core Types

| Type             | Description                                                                      |
| ---------------- | -------------------------------------------------------------------------------- |
| `Snapshot`       | Captured state: `AggregateID`, `AggregateType`, `Version`, `State`, `CreatedAt`. |
| `SnapshotSink`   | Write side: `Save(ctx, Snapshot)`.                                               |
| `SnapshotSource` | Read side: `Load(ctx, ref)`, `LoadAtVersion(ctx, ref, v)`.                       |
| `SnapshotStore`  | `SnapshotSink` + `SnapshotSource` + `Delete` + `Close`.                          |

### Strategies

| Strategy                 | Description                                                               |
| ------------------------ | ------------------------------------------------------------------------- |
| `EveryNEvents(n)`        | Snapshot after every N events appended. Simple, predictable.              |
| `NewReadPressure(loads)` | Snapshot after N loads + next write. Optimizes for read-heavy aggregates. |
| `WithInnerStrategy(s)`   | Combines with `NewReadPressure` — either strategy triggers.               |

### Typed Snapshot Store

```go
typedStore := snapshot.NewTypedStore[UserState](store, codec.JSONCodec{})
typedStore.SaveTyped(ctx, ref, UserState{Name: "Alice"}, 10)
state, version, _ := typedStore.LoadTyped(ctx, ref)
```

## Design

- **One snapshot per aggregate**: Stores keep only the latest snapshot. Older versions are silently ignored (no state regression).
- **ReadPressure**: Tracks aggregate load frequency. After `loads` loads, the next write triggers a snapshot. This optimizes for hot aggregates that are read frequently but rarely written.
- **AggregateAwareStrategy**: Optional interface that allows strategies to inspect the aggregate type and state before deciding.
- **Codec**: Snapshot state is encoded via a `codec.Codec`. Default in `decider.NewRepository` is CBOR.

## Implementations

| Implementation           | Module                                        |
| ------------------------ | --------------------------------------------- |
| `MemorySnapshotStore`    | [storage/memory](../storage/memory/README.md) |
| `SQLSnapshotStore`       | [storage](../storage/README.md)               |
| PebbleDB `SnapshotStore` | [storage/pebble](../storage/pebble/README.md) |
| `FakeSnapshotStore`      | [eventtest](../event/v4/eventtest/README.md)  |

## Related Modules

- [**decider**](../decider/README.md) — `WithSnapshotStore` + `WithSnapshotStrategy` for aggregate loading
- [**event**](../event/README.md) — `event.Version` type used in snapshots
- [**codec**](../codec/README.md) — Serialization of snapshot state
