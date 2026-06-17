# projection — Replay+Live Projection Runner

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/projection/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/projection/v2)

Build read models from event streams with automatic checkpoint management.

```bash
go get github.com/larsartmann/go-cqrs-lite/projection/v2
```

## Quick Start

**Option A: Builder + typed handlers (recommended for simple projections)**

```go
b := projection.NewBuilder("user-projection")
projection.On[UserCreated](b, "user.created", codec.JSONCodec{}, func(ctx context.Context, e UserCreated) error {
    // update read model
    return nil
})
proj := b.Build()

runner, _ := projection.NewRunner(store, bus, checkpointStore)
_ = runner.Register(proj)
go runner.Run(ctx)
```

**Option B: Manual event.Projection implementation (full control)**

```go
type TodoProjection struct{ store ReadModel }
func (p *TodoProjection) Name() string { return "todo-read-model" }
func (p *TodoProjection) EventTypes() []event.Type { return []event.Type{"todo.created"} }
func (p *TodoProjection) Handle(ctx context.Context, evt event.Event) error { /* ... */ return nil }

runner, _ := projection.NewRunner(store, bus, checkpointStore)
_ = runner.Register(&TodoProjection{store: rm})
go runner.Run(ctx) // replays history, then tails live events
```

## Related Modules

- [**event/v2**](../event/README.md) — Event store/bus and `CheckpointStore` interfaces the runner consumes
- [**query/v2**](../query/README.md) — Dispatch typed queries against the read model you build
- [**listing/v2**](../listing/README.md) — Tombstone-aware aggregate listing read model
- [**memory/v2**](../memory/README.md) — In-memory `CheckpointStore` for tests
- [**storage/v2**](../storage/README.md) — SQL `CheckpointStore` for production
- [**pebble/v2**](../pebble/README.md) — Embedded `CheckpointStore` (PebbleDB)
