# snapshot — Snapshot Persistence

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/snapshot/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/snapshot/v3)

Capture aggregate state at a version to avoid full event replay on each load.

```bash
go get github.com/larsartmann/go-cqrs-lite/snapshot/v3
```

## Quick Start

```go
store.Save(ctx, snapshot.Snapshot{
    AggregateID: aggID, AggregateType: "User",
    Version: 10, State: encodedState, CreatedAt: time.Now(),
})
loaded, _ := store.LoadAtVersion(ctx, ref, 10)
strategy, _ := snapshot.EveryNEvents(100)
```

## Related Modules

- [**event/v2**](../event/README.md) — `event.Version` type used in snapshots
- [**decider/v2**](../decider/README.md) — `WithSnapshotStore` + `WithSnapshotStrategy` for aggregate loading
- [**memory/v2**](../memory/README.md) — `MemorySnapshotStore` for tests
- [**storage/v2**](../storage/README.md) — `SQLSnapshotStore` for PostgreSQL/SQLite
- [**pebble/v2**](../pebble/README.md) — `SnapshotStore` backed by PebbleDB
