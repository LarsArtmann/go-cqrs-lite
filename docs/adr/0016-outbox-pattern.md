# ADR-0016: Outbox Pattern for Reliable Event Publishing

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-14   |
| Status  | Proposed     |
| Decider | Lars Artmann |

## Context

When using CQRS with Event Sourcing, events must be persisted AND published to
a message bus for projections and external consumers. Doing these in two separate
steps (save to store, then publish to bus) creates a consistency problem:

1. **Save succeeds, publish fails** — events are lost (no downstream consumers see them)
2. **Publish succeeds, save fails** — consumers see events that don't exist in the store
3. **Network partition** — impossible to know which side committed

The Outbox pattern solves this by writing events and an "outbox entry" in the same
transaction. A separate process reads the outbox and publishes to the bus with
at-least-once delivery semantics.

## Decision

**Scope:** Outbox is a consumer concern, not a library concern. The library provides
the building blocks; consumers compose their own outbox.

### Building Blocks Provided

1. **`event.Store.Save`** — atomic event persistence (already transactional in SQL stores)
2. **`event.Publisher`** — the bus interface for publishing
3. **`event.PublishMiddleware`** — intercept publish calls for logging, metrics, etc.

### Consumer Pattern (Not Library Code)

```go
// 1. Save events + outbox entry in the same DB transaction
tx := db.Begin()
store.Save(ctx, ref, events, expectedVersion)  // uses the same tx
outboxRepo.Append(ctx, tx, events)              // writes to outbox table
tx.Commit()

// 2. Background relay reads outbox and publishes
for entry := range outboxRepo.Pending(ctx) {
    err := bus.Publish(ctx, entry.Event)
    if err != nil {
        // retry with backoff
        continue
    }
    outboxRepo.MarkPublished(ctx, entry.ID)
}
```

### Why Not a Library Module?

- **Transaction handling is transport-specific** — SQL, MongoDB, and file-based
  stores have fundamentally different transaction semantics. A library-level
  outbox would need to abstract over all of these, adding complexity for limited
  gain.
- **Consumers already have DB access** — they can add an outbox table with ~20
  lines of code using their existing database driver.
- **`example/taskmanager/` demonstrates the pattern** — a working reference implementation
  exists for consumers to copy (note: `example/todo/` was renamed to `example/taskmanager/`).

## Consequences

- **+** Library stays focused on core CQRS primitives
- **+** Consumers retain full control over transaction boundaries
- **+** No dependency on specific message broker or database features
- **-** Consumers must implement the relay process themselves
- **-** No built-in deduplication (consumers must handle idempotent consumption)

## References

- [Microservices Pattern: Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Eventuate Tram](https://eventuate.io/tractram.html) — reference implementation
- `example/taskmanager/` in this repo — working outbox demonstration
