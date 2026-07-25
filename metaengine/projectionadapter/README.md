# metaengine/projectionadapter

> Wraps a [`metaengine.Store`](../README.md) as a [`projection.Projection`](../../projection/README.md),
> so cost-planned stores can be registered with [`projectionhost.Host`](../../projectionhost/README.md)
> and process events through the standard projection lifecycle (checkpoint, retry, DLQ).

## When to Use

You have a `metaengine.Store` (cost-based storage planner) and want to feed it
events from a `projectionhost.Host` — the managed projection runner with crash-restart,
checkpointing, and dead-letter queues.

Without this adapter, you would need to manually wire `Store.Apply` calls into
a projection handler. The adapter does this for you, handling payload decoding
and event-type routing.

## Quick Example

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
    "github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// 1. Plan your store (see metaengine README for full example)
store, _ := metaengine.Plan(engines, query, folds...)

// 2. Wrap as a projection
proj := projectionadapter.New("tasks-projection", store, nil)
// nil decoder → payloads decoded as map[string]any (generic JSON)

// 3. Register with projection host
host, _ := projectionhost.New(journal, checkpointStore)
_ = host.Register(proj)
go host.Start(ctx)
```

## Custom Payload Decoder

By default, the adapter decodes event payloads as `map[string]any`. For typed
decoding, provide a `PayloadDecoder`:

```go
proj := projectionadapter.New("tasks", store, func(eventType string, payload []byte) (any, error) {
    switch eventType {
    case "task.created":
        var e TaskCreated
        err := json.Unmarshal(payload, &e)
        return e, err
    case "task.deleted":
        var e TaskDeleted
        err := json.Unmarshal(payload, &e)
        return e, err
    default:
        return nil, fmt.Errorf("unknown event type: %s", eventType)
    }
})
```

The decoded value is passed to `Store.Apply`, which routes it to all registered
queries that listen for that event type.

## Design Notes

- **Separate package**: This adapter lives in its own module to preserve
  `metaengine`'s zero-dependency boundary. The core `metaengine` package has
  no imports from `event/` or `projection/` — this adapter is the only place
  that bridges them.
- **Event types auto-derived**: `New()` calls `store.EventTypes()` to discover
  which event types the planned queries listen to. No manual registration needed.
- **ADR-0062**: See [`docs/adr/0062-projection-adapter.md`](../../docs/adr/0062-projection-adapter.md)
  for the design rationale.
