# decider — Pure-Function Aggregate Pattern

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/decider/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/decider/v3)

The Decider replaces mutable aggregate roots with two pure functions: DecideFunc and Fold.

```bash
go get github.com/larsartmann/go-cqrs-lite/decider/v3
```

## Quick Start

```go
d := decider.Decider[UserState]{
    Initial: UserState{},
    Fold:    foldFunc,
}

repo, _ := decider.NewRepository[UserState](store, bus, d,
    decider.WithSnapshotStore(snapStore),
    decider.WithSnapshotStrategy(snapshot.EveryNEvents(100)),
)

// Execute: load → fold → decide → save → publish
err := repo.Execute(ctx, aggID, "User", decideFunc)

// Time travel
state, ver, _ := repo.LoadAtVersion(ctx, aggID, "User", 3)
```

## Related Modules

- [**event/v2**](../event/README.md) — Event store/bus interfaces consumed by the repository
- [**snapshot/v2**](../snapshot/README.md) — Snapshot strategies (`EveryNEvents`) for performance
- [**id/v2**](../id/README.md) — Branded `AggregateID` for commands
- [**command/v2**](../command/README.md) — Dispatch typed commands into `repo.Execute`
- [**projection/v2**](../projection/README.md) — Build read models from the events the decider emits
- [**schema/v2**](../schema/README.md) — Upcast old events on load
- [**memory/v2**](../memory/README.md) — In-memory store/bus for tests
- [**storage/v2**](../storage/README.md) — SQL persistence (PostgreSQL, SQLite)
