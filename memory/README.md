# memory — In-Memory Test Implementations

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/memory/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/memory/v2)

In-memory implementations of all core CQRS interfaces for testing and development. Thread-safe, defensive-copy semantics on all reads.

**Not intended for production use.**

```bash
go get github.com/larsartmann/go-cqrs-lite/memory/v2
```

## Implementations

| Type                    | Implements                                                                         | Description                                      |
| ----------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------ |
| `MemoryStore`           | `event.Store` + `Journal` + `SeekableJournal` + `BackwardsSource` + `StreamLoader` | Defensive copies on all reads                    |
| `MemoryBus`             | `event.Bus`                                                                        | Typed Subscribe + SubscribeAll + middleware      |
| `MemorySnapshotStore`   | `snapshot.SnapshotStore`                                                           | Deep-copy snapshots, version-aware LoadAtVersion |
| `MemoryCheckpointStore` | `event.CheckpointStore`                                                            | Projection checkpoint persistence                |
| `MemoryCommandStore`    | `command.Store` + `CommandJournal` + `SeekableCommandJournal`                      | Command audit trail persistence                  |
| `MemoryCommandBus`      | `command.Bus` (`ro.Subject[Command]`)                                              | Reactive command stream                          |
| `MemoryQueryStore`      | `query.QueryStore` + `QueryJournal` + `SeekableQueryJournal`                       | Query audit trail persistence                    |

## Quick Start

```go
store := memory.NewMemoryStore()
bus := memory.NewMemoryBus()

// Events
store.Save(ctx, ref, events, 0)
bus.Publish(ctx, events...)

// Command & query audit trails
cmdStore := memory.NewMemoryCommandStore()
cmdStore.Save(ctx, ref, persistedCmd)
cmds, _ := cmdStore.ReadAll(ctx)          // full audit trail

queryStore := memory.NewMemoryQueryStore()
queryStore.SaveQuery(ctx, persistedQuery)
queries, _ := queryStore.ReadAllQueries(ctx)
```

All implementations support `Close()` lifecycle and return defensive copies.

## Related Modules

- [event/v2](../event/README.md) — Event store/bus interfaces
- [command/v2](../command/README.md) — Command store/journal interfaces
- [query/v2](../query/README.md) — Query store/journal interfaces
- [snapshot/v2](../snapshot/README.md) — Snapshot store interfaces
- [projection/v2](../projection/README.md) — `MemoryCheckpointStore` for projection tests
- [storage/v2](../storage/README.md) — Production SQL implementations
- [pebble/v2](../pebble/README.md) — Production embedded (PebbleDB) implementations
