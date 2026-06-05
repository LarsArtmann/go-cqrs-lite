# ADR 0010: Remove io.Closer from Core Interfaces

## Status

Proposed

## Context

Nine core interfaces embed `io.Closer`:

- `event.Bus`
- `event.EventSink` (via `event.Store`)
- `event.EventSource` (via `event.Store`)
- `event.CheckpointSink`
- `event.CheckpointSource`
- `snapshot.SnapshotSink`
- `snapshot.SnapshotSource`
- `command.CommandStore` (sink + source)

This forces every consumer to implement `Close() error` even when:

- The consumer is a stateless adapter (e.g., a read-only proxy)
- The underlying resource has no cleanup (e.g., in-memory store)
- The consumer wants to delegate lifecycle management elsewhere

It also conflates two concerns: "what this type does" (store events, dispatch commands) and "how its lifecycle is managed" (open/close).

## Decision

For v3, remove `io.Closer` from all interfaces. Introduce a standalone `Lifecycle` interface:

```go
// Lifecycle manages resource lifetime. Optional — not required by any core interface.
type Lifecycle interface {
    Close() error
}
```

Implementations that need cleanup (SQL stores, bus subscriptions) will implement `Lifecycle` separately. Consumers check with a type assertion:

```go
if lc, ok := store.(Lifecycle); ok {
    lc.Close()
}
```

## Consequences

- **Breaking change** — existing implementations must remove `Close()` from interface satisfaction
- Implementations that currently embed `io.Closer` can still implement it as a concrete method
- Memory/test implementations can drop the empty `Close()` stub
- Cleaner ISP: interfaces describe behavior, not lifecycle
