# projection — Consumer-Side Projection Interface

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/projection/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/projection/v4)

The `Projection` interface: the consumer-side contract for turning events into read models. Implemented by `storage.RelationalProjection`, `graph.GraphProjection`, and `stack.Materialize`.

```bash
go get github.com/larsartmann/go-cqrs-lite/projection/v4
```

## Why?

Projections are **consumers** of events, not producers. The `event` package defines what events _are_ (`Event`, `Store`, `Bus`); the `projection` package defines how events are _consumed_ into read models. Keeping them separate follows the dependency-direction principle: `projection` depends on `event`, never the reverse.

## Quick Start

```go
import (
    "context"
    "github.com/larsartmann/go-cqrs-lite/projection/v4"
    cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Create a projection from a handler function and event-type filter:
proj := projection.NewProjection("user-count", func(ctx context.Context, evt cqrsevent.Event) error {
    switch evt.Type() {
    case "user.created":
        // increment counter...
    case "user.deleted":
        // decrement counter...
    }
    return nil
}, []cqrsevent.Type{"user.created", "user.deleted"})

// Register with a projection host:
// host.Register(proj)
```

## API

| Symbol                           | Kind      | Description                                                      |
| -------------------------------- | --------- | ---------------------------------------------------------------- |
| `Projection`                     | Interface | `Name() string`, `Handle(ctx, evt) error`, `EventTypes() []Type` |
| `NewProjection(name, fn, types)` | Func      | Creates a Projection from a handler function and type filter.    |

The `Projection` interface:

```go
type Projection interface {
    Name() string
    Handle(ctx context.Context, evt event.Event) error
    EventTypes() []event.Type
}
```

Events whose type is not in `EventTypes()` are silently skipped by the projection runner.

## Design

- **Type filter is a contract**: `EventTypes()` tells the projection runner which events to deliver. Events outside the filter never reach `Handle`.
- **Clone-safe**: `NewProjection` clones the event-type slice so callers can safely mutate their original after construction.
- **Implementations**: `storage.RelationalProjection` (multi-table SQL), `graph.GraphProjection` (nodes + edges), `stack.Materialize` (KV documents).

## Related Modules

- [**projectionhost**](../projectionhost/README.md) — Managed lifecycle: registers projections, drives them from a journal with checkpoints and dead-letter queues
- [**stack**](../stack/README.md) — `Materialize[V,K]` implements `Projection` for KV-backed read models
- [**graph**](../graph/README.md) — `GraphProjection` implements `Projection` for traversal-heavy read models
- [**storage**](../storage/README.md) — `RelationalProjection` implements `Projection` for multi-table SQL projections
