# Getting Started with go-cqrs-lite

A lightweight CQRS library for Go with Event Sourcing, branded IDs, and auto-documentation.

## Installation

go-cqrs-lite is a multi-module monorepo. Import only what you need:

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v3
go get github.com/larsartmann/go-cqrs-lite/command/v3
go get github.com/larsartmann/go-cqrs-lite/memory/v3
```

## Quick Start

### 1. Define Events

```go
type UserCreated struct{ Name string }
```

### 2. Create and Store Events

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/id/v3"
    "github.com/larsartmann/go-cqrs-lite/memory/v3"
)

store := memory.NewMemoryStore()
bus := memory.NewMemoryBus()
aggID := id.NewAggregateID()

// Create a typed event — payload is auto-marshaled to JSON
evt, err := event.NewEvent(
    "user.created", aggID, "User", event.Version(1),
    UserCreated{Name: "Alice"},
)
if err != nil {
    log.Fatal(err)
}

// Save with optimistic concurrency
ref := event.NewAggregateRef("User", aggID)
if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
    log.Fatal(err)
}

// Publish to subscribers
_ = bus.Publish(ctx, evt)
```

### 3. Dispatch Commands

```go
import "github.com/larsartmann/go-cqrs-lite/command/v3"

// Define a command embedding BasicCommand
type CreateUserCmd struct {
    *command.BasicCommand
    Name string
}

// Type-safe handler registration
cmds := command.NewDispatcher()
command.RegisterTyped(cmds, "user.create",
    func(ctx context.Context, cmd *CreateUserCmd) error {
        // handle command
        return nil
    },
)

// Dispatch
cmd := &CreateUserCmd{
    BasicCommand: command.MustNew("user.create", aggID),
    Name: "Alice",
}
cmds.Dispatch(ctx, cmd)
```

### 4. Event Sourcing with Decider

The Decider pattern uses pure functions for load → fold → decide → save → publish:

```go
import "github.com/larsartmann/go-cqrs-lite/decider/v3"

type UserState struct{ Name string }

d := decider.Decider[UserState]{
    Initial: UserState{},
    Fold: func(s UserState, evt event.Event) (UserState, error) {
        p, _ := event.DecodePayload[UserCreated](evt, codec.JSONCodec{})
        s.Name = p.Name
        return s, nil
    },
}

repo, _ := decider.NewRepository[UserState](store, bus, d)

// Execute a command (load state → fold → decide → save → publish)
repo.Execute(ctx, aggID, "User", func(s UserState, v event.Version) ([]event.Event, error) {
    return event.NewEvents(aggID, "User", v,
        []event.Type{"user.created"}, []any{UserCreated{Name: "Alice"}})
})

// Load current state
state, version, _ := repo.Load(ctx, aggID, "User")
```

### 5. Branded IDs

```go
import "github.com/larsartmann/go-cqrs-lite/id/v3"

// Use built-in types
aggID := id.NewAggregateID()
eventID := id.NewEventID()

// Create custom branded types
type OrderID = id.Of[struct{}]
orderID := id.New[OrderID]()
```

## Architecture

```
Command → Dispatcher → Handler → Decider → Store + Bus
                                            ↓
Query   → Dispatcher → Handler            Projection
```

## Modules

| Module     | Import Path       | Purpose                                     |
| ---------- | ----------------- | ------------------------------------------- |
| event      | `…/event/v2`      | Events, Store, Bus, reactive streams        |
| command    | `…/command/v2`    | Commands, Dispatcher, typed handlers        |
| query      | `…/query/v2`      | Queries, Dispatcher, typed results          |
| decider    | `…/decider/v2`    | Pure-function aggregate pattern             |
| id         | `…/id/v2`         | Branded IDs (AggregateID, EventID, etc.)    |
| projection | `…/projection/v2` | Catch-up projections with replay            |
| memory     | `…/memory/v2`     | In-memory implementations (testing)         |
| storage    | `…/storage/v2`    | PostgreSQL, SQLite, Turso, Pebble stores    |
| middleware | `…/middleware/v2` | Logging, retry, recovery, validation, etc.  |
| catalog    | `…/catalog/v2`    | Auto-documentation (AsyncAPI, EventCatalog) |
| schema     | `…/schema/v2`     | Schema evolution (upcasters)                |
| signing    | `…/signing/v2`    | Event signing/verification (HMAC, Ed25519)  |

## Next Steps

- **[SKILL.md](../SKILL.md)** — The AI consumer guide: module decision matrix, composition recipes, conventions, anti-patterns
- See `example/todo/` for a complete application with HTTP API, projections, and Pebble storage
- See `example/user/` for advanced patterns (Decider, signing, middleware, catalog generation)
- See `example/encryption/` for event encryption patterns (bus, store, key rotation)
- See `README.md` for the full Quick Start and feature comparison
- Browse `docs/adr/` for architectural decisions
- Check `FEATURES.md` for full feature inventory
