# go-cqrs-lite

A lightweight, zero-dependency CQRS (Command Query Responsibility Segregation) library for Go with support for Event Sourcing and strongly-typed domain identifiers.

## What It Does

go-cqrs-lite provides the essential building blocks for implementing CQRS and Event Sourcing patterns:

- **Command Dispatcher** - Type-safe command handling with middleware support
- **Query Dispatcher** - Type-safe query handling with pagination
- **Event Store** - Interface for event persistence with in-memory implementation
- **Event Bus** - Publish/subscribe pattern for domain events
- **Aggregate Roots** - Base implementation for domain-driven design aggregates
- **Strongly-Typed IDs** - Branded identifier types to prevent mixing up IDs (UserId, AggregateId, etc.)
- **Extended Types** - Type-safe wrappers for commands, queries, and events with built-in ID types

## Quick Start

### Installation

```bash
go get github.com/larsartmann/go-cqrs-lite
```

### Requirements

- Go 1.22+

### Dependencies

- `github.com/google/uuid` - UUID generation
- `github.com/cockroachdb/errors` - Error handling with context

## Core Concepts

### Commands (Write Side)

Commands represent intent to change state:

```go
type CreateUser struct {
    command.Base
    Email string
    Name  string
}

type CreateUserHandler struct {
    eventStore event.Store
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd *CreateUserCommand) error {
    user, _ := aggregate.NewUser(cmd.AggregateID(), cmd.Email, cmd.Name)
    for _, evt := range user.UncommittedChanges() {
        if err := h.eventStore.Append(ctx, evt); err != nil {
            return err
        }
    }
    user.MarkChangesAsCommitted()
    return nil
}
```

### Events (Event Sourcing)

Events represent state changes with rich metadata:

```go
event, err := event.NewEvent(
    "user.created",
    userID,
    "User",
    1,
    payload,
    event.WithCorrelationID(requestID),
    event.WithUserID(operatorID),
)
```

### Strongly-Typed IDs

Prevents mixing up different ID types:

```go
// Instead of string IDs
userID := user_id.NewUserID()
aggregateID := aggregate_id.New()

// These won't compile - type mismatch:
// store.GetByID(ctx, userID)  // expects AggregateID, got UserID
```

## Package Structure

| Package       | Purpose                                | Status     |
| ------------- | -------------------------------------- | ---------- |
| `command/`    | Command types, dispatcher, handlers    | ✅ Ready   |
| `query/`      | Query types, dispatcher, handlers      | ✅ Ready   |
| `event/`      | Domain events, store interface, bus    | ✅ Ready   |
| `aggregate/`  | Aggregate root, repository patterns    | ✅ Ready   |
| `pkg/id/`     | Strongly-typed branded identifiers     | ✅ Ready   |
| `xtypes/`     | Type-safe command/query/event wrappers | ✅ Ready   |
| `middleware/` | Logging, metrics, validation (planned) | 🔜 Planned |

## Design Principles

1. **Minimal dependencies** - Only uuid and cockroachdb/errors
2. **Composition over inheritance** - Per Go best practices
3. **Interface-first design** - All core types are interfaces
4. **Context-aware** - All operations accept `context.Context`
5. **Errors as values** - No panics, explicit error returns
6. **File size limits** - Max 250 lines per file

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                        │
│   HTTP Handlers ──► Command/Query Dispatchers               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      CQRS-LITE CORE                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Command    │  │    Query     │  │    Event     │       │
│  │  Dispatcher  │  │  Dispatcher  │  │     Bus      │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     DOMAIN LAYER                             │
│   Aggregates ──► Entities ──► Strongly-Typed IDs           │
└─────────────────────────────────────────────────────────────┘
```

## Usage Example

```go
package main

import (
    "context"
    "github.com/larsartmann/go-cqrs-lite/command"
    "github.com/larsartmann/go-cqrs-lite/event"
    "github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func main() {
    ctx := context.Background()

    // Create in-memory event store
    store := event.NewMemoryStore()

    // Create event bus for publish/subscribe
    bus := event.NewMemoryBus()

    // Register command dispatcher
    cmdDispatcher := command.NewDispatcher()

    // Use strongly-typed IDs
    userID := id.NewUserID()

    // Create and dispatch command
    cmd := command.New("user.create", userID.String())
    cmdDispatcher.Dispatch(ctx, cmd)
}
```

## Comparison

| Feature         | go-cqrs-lite | go-cqrs | cqrs-go |
| --------------- | ------------ | ------- | ------- |
| Minimal deps    | ✅           | ❌      | ❌      |
| Event Sourcing  | ✅           | ✅      | ✅      |
| Event Bus       | ✅           | ✅      | ❌      |
| Strong IDs      | ✅           | ❌      | ❌      |
| Context support | ✅           | ❌      | ✅      |

## Project Status

**Phase:** Production Ready (All core features complete)

| Phase         | Status      | Description                                       |
| ------------- | ----------- | ------------------------------------------------- |
| Foundation    | ✅ Complete | Core types, events, commands, queries, aggregates |
| Event Layer   | ✅ Complete | Event store, event bus, in-memory implementations |
| Command Layer | ✅ Complete | Command dispatcher with middleware support        |
| Query Layer   | ✅ Complete | Query dispatcher with typed results               |
| Middleware    | ✅ Complete | Infrastructure for command/event middleware       |
| Tests         | ✅ Complete | Unit tests for all packages                       |
| CI/CD         | ✅ Complete | GitHub Actions, Makefile, linting                 |
| Documentation | ✅ Complete | README, TODO_LIST, CONTRIBUTING, CODE_OF_CONDUCT  |

See [TODO_LIST.md](TODO_LIST.md) for detailed status.

## License

MIT

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Coding standards
- [CQRS pattern](https://martinfowler.com/bliki/CQRS.html) - Martin Fowler
