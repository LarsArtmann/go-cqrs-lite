# Getting Started with go-cqrs-lite

A lightweight CQRS library for Go with Event Sourcing, branded IDs, and auto-documentation.

## Installation

```bash
go get github.com/larsartmann/go-cqrs-lite/core
```

## Quick Start

### 1. Define Commands

```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type CreateUser struct {
    command.Core
    Email string
}

cmd, _ := command.New("CreateUser", id.NewAggregateID())
```

### 2. Dispatch Commands

```go
d := command.NewDispatcher()

d.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
    // handle command
    return nil
})

d.Dispatch(ctx, cmd)
```

### 3. Create Events

```go
import "github.com/larsartmann/go-cqrs-lite/core/event"

evt, _ := event.NewEvent(
    "UserCreated",
    aggregateID,
    "User",
    1,
    []byte(`{"email":"alice@example.com"}`),
    event.WithCorrelationID(correlationID),
)
```

### 4. Event Sourcing with Decider

```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/decider"
    "github.com/larsartmann/go-cqrs-lite/memory"
)

store := memory.NewMemoryStore()
bus := memory.NewMemoryBus()
repo := decider.NewRepository(store, bus, initialState, foldFunc)

// Execute: load → fold → decide → save → publish
newState, err := repo.Execute(ctx, aggregateID, decideFunc)
```

### 5. Branded IDs

```go
import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// Use built-in types
aggID := id.NewAggregateID()
eventID := id.NewEventID()

// Create custom branded types
type OrderID = id.Of[struct{}]
orderID := id.New[OrderID]()
```

## Architecture

```
Command → Dispatcher → Handler → Aggregate → Store + Bus
                                                  ↓
Query   → Dispatcher → Handler              Projection
```

For a detailed visual walkthrough of how a web client communicates with go-cqrs-lite, see the [Web Client Communication Diagram](web-client-communication.d2) (render with `d2` CLI or view [web-client-communication.html](web-client-communication.html)).

## Modules

| Module      | Import          | Purpose                                                |
| ----------- | --------------- | ------------------------------------------------------ |
| core        | `…/core/…`      | CQRS primitives (command, event, query, decider, id) |
| memory      | `…/memory`      | In-memory implementations (testing)                    |
| catalog     | `…/catalog/…`   | Auto-documentation (AsyncAPI 3.0, EventCatalog)        |
| middleware  | `…/middleware`  | Logging, retry, recovery, validation, metrics          |
| storage     | `…/storage`     | PostgreSQL, SQLite, Turso, Pebble stores               |
| testhelpers | `…/testhelpers` | Test utilities (fakes, helpers)                        |

## Next Steps

- See `example/user/` for a complete working demo
- Browse `docs/planning/` for architectural decisions
- Check `FEATURES.md` for full feature inventory
