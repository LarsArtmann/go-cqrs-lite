# decider — Pure-Function Aggregate Pattern

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/decider/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/decider/v2)

The Decider replaces mutable aggregate roots with two pure functions: DecideFunc and Fold.

```bash
go get github.com/larsartmann/go-cqrs-lite/decider/v2
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
