# projection — Replay+Live Projection Runner

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/projection/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/projection/v2)

Build read models from event streams with automatic checkpoint management.

```bash
go get github.com/larsartmann/go-cqrs-lite/projection/v2
```

## Quick Start

```go
b := projection.NewBuilder("user-projection")
b.On("user.created", handler)
runner := b.Runner(store, bus)
go runner.Run(ctx)
```

## Related Modules

- [**event/v2**](../event/README.md) — Event store/bus and `CheckpointStore` interfaces the runner consumes
- [**query/v2**](../query/README.md) — Dispatch typed queries against the read model you build
- [**listing/v2**](../listing/README.md) — Tombstone-aware aggregate listing read model
- [**memory/v2**](../memory/README.md) — In-memory `CheckpointStore` for tests
- [**storage/v2**](../storage/README.md) — SQL `CheckpointStore` for production
- [**pebble/v2**](../pebble/README.md) — Embedded `CheckpointStore` (PebbleDB)
