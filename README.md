# cqrs-lite

A lightweight, zero-dependency CQRS (Command Query Responsibility Segregation) library for Go, extracted from battle-tested patterns across multiple production projects.

## Overview

`cqrs-lite` provides the essential building blocks for implementing CQRS and Event Sourcing patterns without the complexity of enterprise frameworks. It combines the best patterns from:

| Source Project  | Patterns Extracted                                         |
| --------------- | ---------------------------------------------------------- |
| **ChastityAPI** | Event Store, Domain Events with metadata, Snapshot support |
| **Cyberdom**    | Event Bus, concurrent handlers, sync/async publishing      |
| **Domination**  | Type-safe commands, event builders, aggregate patterns     |
| **GmbHG**       | Handler separation, middleware patterns                    |

## Design Principles

1. **Zero external dependencies** - Only stdlib + `google/uuid`
2. **Composition over inheritance** - Per Go best practices
3. **Interface-first design** - All core types are interfaces
4. **Context-aware** - All operations accept `context.Context`
5. **Errors as values** - No panics, explicit error returns
6. **File size limits** - Max 250 lines per file (per HOW_TO_GOLANG.md)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                        │
│   HTTP Handlers ──► Command/Query Dispatchers                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      CQRS-LITE CORE                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Command    │  │    Query     │  │    Event     │       │
│  │  Dispatcher  │  │  Dispatcher  │  │     Bus      │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│         │                 │                 │                │
│         ▼                 ▼                 ▼                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Handlers   │  │   Handlers   │  │ Event Store  │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│                              │                 │              │
│                              ▼                 ▼              │
│                       ┌──────────────┐  ┌──────────────┐     │
│                       │  Middleware  │  │  Snapshots   │     │
│                       └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     DOMAIN LAYER                             │
│   Aggregates ──► Entities ──► Value Objects                  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Installation

```bash
go get github.com/larsartmann/cqrs-lite
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/larsartmann/cqrs-lite/command"
    "github.com/larsartmann/cqrs-lite/query"
    "github.com/larsartmann/cqrs-lite/event"
)

// 1. Define your command
type CreateUserCommand struct {
    command.Base
    Email string
    Name  string
}

// 2. Define your handler
type CreateUserHandler struct {
    store event.Store
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd *CreateUserCommand) error {
    // Create user, emit events
    return h.store.Append(ctx, event.New("user.created", cmd.AggregateID, payload))
}

// 3. Register and dispatch
func main() {
    cmdDispatcher := command.NewDispatcher()
    cmdDispatcher.Register(&CreateUserHandler{}, &CreateUserCommand{})

    err := cmdDispatcher.Dispatch(ctx, &CreateUserCommand{
        Email: "user@example.com",
        Name:  "John Doe",
    })
}
```

## Package Structure

| Package       | Purpose                             | Lines     |
| ------------- | ----------------------------------- | --------- |
| `command/`    | Command types, dispatcher, handlers | ~200      |
| `query/`      | Query types, dispatcher, handlers   | ~200      |
| `event/`      | Domain events, store interface, bus | ~300      |
| `aggregate/`  | Aggregate root, repository patterns | ~150      |
| `middleware/` | Logging, metrics, retry, validation | ~250 each |

## Features

### Commands (Write Side)

- Type-safe command definitions
- Command dispatcher with handler registration
- Middleware support (validation, logging, retry)
- Idempotency support via command IDs

### Queries (Read Side)

- Type-safe query definitions
- Query dispatcher with handler registration
- Pagination and filtering support
- Caching middleware

### Events

- Domain events with rich metadata
- Event store interface (SQLite, PostgreSQL adapters)
- Event bus for pub/sub
- Snapshot support for aggregate optimization
- Correlation/causation ID tracking

### Middleware

- **Logging** - Structured logging of all operations
- **Metrics** - OpenTelemetry integration
- **Retry** - Exponential backoff for transient failures
- **Validation** - Pre-execution validation

## Comparison with Alternatives

| Feature           | cqrs-lite | go-cqrs | cqrs-go |
| ----------------- | --------- | ------- | ------- |
| Zero dependencies | ✅        | ❌      | ❌      |
| Event Sourcing    | ✅        | ✅      | ✅      |
| Middleware        | ✅        | ❌      | Partial |
| Event Bus         | ✅        | ✅      | ❌      |
| Snapshot support  | ✅        | ❌      | ❌      |
| Context support   | ✅        | ❌      | ✅      |
| Complexity        | Low       | Medium  | High    |

## Requirements

- Go 1.21+
- For event store: SQLite 3.x or PostgreSQL 14+

## License

MIT

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Project coding standards
- [PARTS_ANALYSIS.md](https://github.com/larsartmann/index) - Source project analysis
