# command — CQRS Command Dispatch

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/command/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/command/v2)

Typed command dispatch with middleware chains and lifecycle management.

```bash
go get github.com/larsartmann/go-cqrs-lite/command/v2
```

## Quick Start

```go
cmds := command.NewDispatcher()
cmds.Register("user.create", handler)
err := cmds.Dispatch(ctx, cmd)
```

## Typed Handlers

```go
command.RegisterTyped[CreateUserCmd](cmds, "user.create",
    func(ctx context.Context, cmd *CreateUserCmd) error {
        return handleCreate(cmd)
    },
)
```

## Key Types

| Type                     | Purpose                                                     |
| ------------------------ | ----------------------------------------------------------- |
| `Dispatcher`             | Command dispatcher with handler registry + middleware chain |
| `Command`                | Interface: Type(), AggregateID(), IdempotencyKey()          |
| `BasicCommand`           | Embed in command structs for interface satisfaction         |
| `TypedHandler[T]`        | Type-safe handler receiving T, not Command                  |
| `Middleware`             | func(Handler) Handler — wraps handlers in a chain           |
| `PersistedCommand`       | Stored command record for audit/replay                      |
| `CommandStore`           | `CommandSink + CommandSource` — persist commands            |
| `CommandJournal`         | `ReadAll(ctx)` — full command audit trail                   |
| `SeekableCommandJournal` | `ReadFrom(ctx, afterID, limit)` — batched replay            |

## Command Persistence & Audit

Commands can be persisted for audit trails and replay debugging — the command-side equivalent of event sourcing:

```go
// Create a persisted command record
pc, err := command.NewPersistedCommand("user.create", ref, payload,
    command.WithCorrelationID(corrID))

// Persist via a CommandStore (Sink + Source)
store := memory.NewMemoryCommandStore()
store.Save(ctx, ref, pc)

// Read back
cmds, _ := store.Load(ctx, ref)

// Cross-aggregate audit trail (Journal)
allCmds, _ := store.ReadAll(ctx)

// Position-based replay (SeekableJournal)
batch, _ := store.ReadFrom(ctx, lastCmdID, 100)
```

## Related Modules

- [event/v2](../event/README.md) — Event store/bus with matching Journal/SeekableJournal pattern
- [query/v2](../query/README.md) — Query dispatch with parallel PersistedQuery/QueryStore
- [memory/v2](../memory/README.md) — `MemoryCommandStore` in-memory implementation
- [id/v2](../id/README.md) — Branded `CommandID` and `AggregateID`
