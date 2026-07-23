# stack/memory — All-In-Memory Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/memory/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/memory/v4)

The fastest way to get a working `Bundle` for development, testing, and prototyping. Every capability is wired from in-memory implementations.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/memory/v4
```

## Quick Start

```go
import memory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"

bundle, err := memory.New()
if err != nil { log.Fatal(err) }
defer bundle.Close()

// Full CQRS pipeline ready:
store := bundle.EventStore()
bus   := bundle.EventBus()
repo  := bundle.Repository(decider)
```

## API

| Symbol      | Description                                              |
| ----------- | -------------------------------------------------------- |
| `New()`     | Returns `(*stack.Bundle, error)` with all capabilities. |

## What's Wired

| Capability        | Implementation                          |
| ----------------- | --------------------------------------- |
| Event Store       | `storage/memory.MemoryStore`            |
| Event Bus         | `watermill.EventBus` (GoChannel)        |
| Snapshot Store    | `storage/memory.MemorySnapshotStore`    |
| Checkpoint Store  | `storage/memory.MemoryCheckpointStore`  |
| Command Store     | `storage/memory.MemoryCommandStore`     |
| Query Store       | `storage/memory.MemoryQueryStore`       |
| Read Models       | `kv.MemStore`                           |

## Design

- **No persistence**: Data is lost when the process exits.
- **No new implementation**: Wires existing `memory` and `kv` packages.
- **GoChannel event bus**: In-process pub/sub via `watermill.EventBus`.

## When to Use

- Development and prototyping
- Unit tests that need a full CQRS pipeline
- CI pipelines (no external dependencies)
- Learning the library

Switch to [stack/sqlite](../sqlite/README.md), [stack/pebble](../storage/pebble/README.md), or [stack/postgres](../postgres/README.md) for production by changing one line.

## Related Modules

- [**stack**](../README.md) — The `Bundle` type and `Materialize` builder
- [**storage/memory**](../../storage/memory/README.md) — In-memory store implementations
- [**watermill**](../../watermill/README.md) — In-process event bus
