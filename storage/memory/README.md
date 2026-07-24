# storage/memory — In-Memory Store Implementations

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/storage/memory/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/storage/memory/v4)

In-memory implementations of all core CQRS persistence interfaces for testing and development. Not for production use.

```bash
go get github.com/larsartmann/go-cqrs-lite/storage/memory/v4
```

## Why?

Every test in go-cqrs-lite needs a fast, deterministic store that implements the same interfaces as the production SQL and Pebble backends. This package provides complete in-memory implementations of `event.Store`, `snapshot.SnapshotStore`, `checkpoint.Store`, `command.Store`, and `query.Store` so tests can run without a database.

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"

// Event store (implements event.Store, event.Journal, event.SeekableJournal)
store := memory.NewMemoryStore()

// Snapshot store
snapStore := memory.NewMemorySnapshotStore()

// Checkpoint store
cpStore := memory.NewMemoryCheckpointStore()

// Command audit store
cmdStore := memory.NewMemoryCommandStore()

// Query audit store
qStore := memory.NewMemoryQueryStore()
```

## API

| Type                    | Implements                                                                         | Constructor                  |
| ----------------------- | ---------------------------------------------------------------------------------- | ---------------------------- |
| `MemoryStore`           | `event.Store`, `event.Journal`, `event.SeekableJournal`                            | `NewMemoryStore()`           |
| `MemorySnapshotStore`   | `snapshot.SnapshotSink`, `SnapshotSource`, `SnapshotStore`                         | `NewMemorySnapshotStore()`   |
| `MemoryCheckpointStore` | `checkpoint.Store`                                                                 | `NewMemoryCheckpointStore()` |
| `MemoryCommandStore`    | `command.CommandSink`, `CommandSource`, `CommandJournal`, `SeekableCommandJournal` | `NewMemoryCommandStore()`    |
| `MemoryQueryStore`      | `query.QuerySink`, `QuerySource`, `QueryJournal`, `SeekableQueryJournal`           | `NewMemoryQueryStore()`      |

## Design

- **Thread-safe**: All stores use `sync.Mutex` for safe concurrent access.
- **Full interface compliance**: Every store implements the complete interface, not just a subset. This makes them drop-in replacements for SQL/Pebble backends in tests.
- **Optimistic concurrency**: `MemoryStore.Save` enforces version-based optimistic concurrency control, same as the SQL backend.
- **No persistence**: Data is lost when the process exits. Use `storage/`, `storage/pebble/`, or `storage/turso/` for production.

## Related Modules

- [**event**](../../event/README.md) — Store, Journal, SeekableJournal interfaces
- [**snapshot**](../../snapshot/README.md) — SnapshotStore interfaces
- [**command**](../../command/README.md) — CommandSink/Source interfaces
- [**query**](../../query/README.md) — QuerySink/Source interfaces
- [**stack/memory**](../../stack/memory/README.md) — All-in-memory stack preset using these stores
