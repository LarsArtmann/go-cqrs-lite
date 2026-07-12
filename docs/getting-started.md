# Getting Started with go-cqrs-lite

A lightweight CQRS library for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework — no opinionated transport, message broker, or SQL driver.

## Installation

go-cqrs-lite is a multi-module monorepo. Import only what you need:

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v4
go get github.com/larsartmann/go-cqrs-lite/command/v4
go get github.com/larsartmann/go-cqrs-lite/storage/memory/v4
```

## Quick Start

### 1. Define Events

```go
type UserCreated struct{ Name string }
```

### 2. Create and Store Events

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

store := memory.NewMemoryStore()
bus := event.NewMemoryBus()
aggID := id.NewAggregateID()

// Create a typed event — payload is auto-marshaled to CBOR (the default codec)
evt, err := event.NewEvent(
    "user.created", aggID, "User", event.Version(1),
    UserCreated{Name: "Alice"},
)
if err != nil {
    log.Fatal(err)
}

// Save with optimistic concurrency
ref := id.NewAggregateRef("User", aggID)
if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
    log.Fatal(err)
}

// Publish to subscribers
_ = bus.Publish(ctx, evt)
```

### 3. Dispatch Commands

```go
import "github.com/larsartmann/go-cqrs-lite/command/v4"

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
import "github.com/larsartmann/go-cqrs-lite/decider/v4"

type UserState struct{ Name string }

d := decider.Decider[UserState]{
    Initial: UserState{},
    Fold: func(s UserState, evt event.Event) (UserState, error) {
        p, _ := event.DecodePayloadAuto[UserCreated](evt)
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
import "github.com/larsartmann/go-cqrs-lite/id/v4"

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

## Core Modules

| Module         | Import Path             | Purpose                                                       |
| -------------- | ----------------------- | ------------------------------------------------------------- |
| event          | `.../event/v4`          | Events, Store, Bus, reactive streams                          |
| command        | `.../command/v4`        | Commands, Dispatcher, typed handlers                          |
| query          | `.../query/v4`          | Queries, Dispatcher, typed results                            |
| decider        | `.../decider/v4`        | Pure-function aggregate pattern                               |
| id             | `.../id/v4`             | Branded IDs (AggregateID, EventID, etc.)                      |
| projection     | `.../projection/v4`     | Projection interface (consumer-side)                          |
| projectionhost | `.../projectionhost/v4` | Managed projection lifecycle (goroutines, DLQ, checkpointing) |
| storage/memory | `.../storage/memory/v4` | In-memory implementations (testing)                           |
| storage        | `.../storage/v4`        | SQL stores, Pebble, Turso connectors                          |
| middleware     | `.../middleware/v4`     | Logging, retry, recovery, validation, OTel                    |
| catalog        | `.../catalog/v4`        | Auto-documentation (AsyncAPI, EventCatalog)                   |
| schema         | `.../schema/v4`         | Schema evolution (upcasters, versioned stores)                |
| signing        | `.../signing/v4`        | Event signing/verification (HMAC, Ed25519)                    |

For the full list of 49 modules, see [AGENTS.md](../AGENTS.md).

## Next Steps

- **[SKILL.md](../SKILL.md)** — The AI consumer guide: module decision matrix, composition recipes, conventions, anti-patterns. This is the single best starting point.
- See `example/getting-started/` for a minimal 80-line example showing the core pipeline
- See `example/taskmanager/` for a flagship full HTTP service: event sourcing, CQRS, projections, middleware, OTel, signing
- See `README.md` for the full feature comparison and Quick Start
- Browse `docs/adr/` for 53 architectural decisions
- Check `FEATURES.md` for the full feature inventory
