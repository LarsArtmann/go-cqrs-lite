# snapshot — Snapshot Persistence

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/snapshot/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/snapshot/v2)

Capture aggregate state at a version to avoid full event replay on each load.

```bash
go get github.com/larsartmann/go-cqrs-lite/snapshot/v2
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
