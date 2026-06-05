# memory — In-Memory Test Implementations

In-memory implementations of core CQRS interfaces for testing and development.

**Not intended for production use.**

```bash
go get github.com/larsartmann/go-cqrs-lite/memory/v2
```

## Implementations

| Type                    | Implements                                                                         | Thread-safe | Description                                      |
| ----------------------- | ---------------------------------------------------------------------------------- | ----------- | ------------------------------------------------ |
| `MemoryStore`           | `event.Store` + `Journal` + `SeekableJournal` + `BackwardsSource` + `StreamLoader` | ✅          | Defensive copies on all reads                    |
| `MemoryBus`             | `event.Bus`                                                                        | ✅          | Typed Subscribe + SubscribeAll + middleware      |
| `MemorySnapshotStore`   | `snapshot.SnapshotStore`                                                           | ✅          | Deep-copy snapshots, version-aware LoadAtVersion |
| `MemoryCheckpointStore` | `event.CheckpointStore`                                                            | ✅          | Projection checkpoint persistence                |

## Quick Start

```go
store := memory.NewStore()
bus := memory.NewBus()
snapStore := memory.NewSnapshotStore()
checkpointStore := memory.NewCheckpointStore()

// Use like any event.Store / event.Bus / etc.
store.Save(ctx, ref, events, 0)
bus.Publish(ctx, events...)
```

All implementations support `Close()` lifecycle and return defensive copies.
