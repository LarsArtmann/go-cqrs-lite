# go-cqrs-lite

[![Tests](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/test.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/test.yml)
[![Lint](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/lint.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/LarsArtmann/go-cqrs-lite/core.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-cqrs-lite/core)

A lightweight CQRS (Command Query Responsibility Segregation) library for Go with support for Event Sourcing, strongly-typed domain identifiers, and auto-documentation generation.

**Multi-module monorepo** — import only what you need. Each module has its own `go.mod` with minimal dependencies.

## What It Does

go-cqrs-lite provides the essential building blocks for implementing CQRS and Event Sourcing patterns:

- **Command Dispatcher** - Type-safe command handling with middleware support
- **Query Dispatcher** - Type-safe query handling with pagination
- **Event Store** - Interface for event persistence with in-memory implementation
- **Event Bus** - Publish/subscribe pattern for domain events
- **Decider Pattern** - Functional aggregate approach with pure functions (recommended)
- **Projections** - Build read models from events with replay support
- **Saga / Process Manager** - Coordinate long-running business processes with compensation
- **Stream Loading** - Memory-efficient event iteration for large aggregates
- **Event Versioning** - Upcast legacy events transparently via VersionedStore
- **Strongly-Typed IDs** - Branded identifier types to prevent mixing up IDs
- **Error Classification** - Structured errors with retry semantics
- **Auto-documentation** - Generate AsyncAPI 3.0 specs and EventCatalog from code
- **Watermill Integration** - Adapter for the Watermill message router ecosystem

## Quick Start

### Installation

```bash
# Core CQRS types (commands, queries, events, IDs, decider)
go get github.com/larsartmann/go-cqrs-lite/core

# In-memory implementations (testing)
go get github.com/larsartmann/go-cqrs-lite/memory

# Persistent storage (SQLite, Turso, PostgreSQL, Pebble)
go get github.com/larsartmann/go-cqrs-lite/storage

# API documentation generation (AsyncAPI 3.0 + EventCatalog)
go get github.com/larsartmann/go-cqrs-lite/catalog

# Cross-cutting middleware (logging, retry, validation, recovery, metrics)
go get github.com/larsartmann/go-cqrs-lite/middleware

# Projection runner with replay and live subscription
go get github.com/larsartmann/go-cqrs-lite/projection

# Saga / Process Manager with compensation
go get github.com/larsartmann/go-cqrs-lite/saga

# Watermill message bus adapter
go get github.com/larsartmann/go-cqrs-lite/watermill
```

### Requirements

- Go 1.26+

### Your First CQRS App (5 minutes)

A minimal CQRS + Event Sourcing app using typed handlers and batch event creation:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/memory"
)

type CreateUserCmd struct {
    *command.BasicCommand
    Name string
}

type UserCreated struct{ Name string }

func main() {
    store := memory.NewMemoryStore()
    bus := memory.NewMemoryBus()
    aggID := id.NewAggregateID()

    // Register a type-safe command handler — no manual type assertions
    cmds := command.NewDispatcher()
    command.RegisterTyped(cmds, "user.create",
        func(ctx context.Context, cmd *CreateUserCmd) error {
            // Create events from typed payloads in one call
            events, err := event.NewEvents(aggID, "User", 0,
                []event.Type{"user.created"},
                []any{UserCreated{Name: cmd.Name}},
            )
            if err != nil { return err }
            if err := store.Save(ctx, "User", aggID, events, 0); err != nil { return err }
            for _, e := range events { _ = bus.Publish(ctx, e) }
            return nil
        },
    )

    // Dispatch
    cmd := &CreateUserCmd{BasicCommand: command.MustNew("user.create", aggID), Name: "Alice"}
    if err := cmds.Dispatch(context.Background(), cmd); err != nil { log.Fatal(err) }

    // Load and read
    events, _ := store.Load(context.Background(), "User", aggID)
    for _, e := range events {
        p, _ := event.DecodePayload[UserCreated](e, codec.JSONCodec{})
        fmt.Printf("User created: %s\n", p.Name)
    }
}
```

See `example/user/` for a complete example with the Decider pattern, middleware, and catalog generation.

### Recommended: Decider Pattern

For real applications, use the Decider — pure functions with load→fold→decide→save→publish semantics:

```go
import "github.com/larsartmann/go-cqrs-lite/core/decider"

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

See [`core/README.md`](core/README.md) for the full Decider guide.

### Core Dependencies

| Dependency        | Purpose                                         | Module       |
| ----------------- | ----------------------------------------------- | ------------ |
| `oklog/ulid/v2`   | ULID generation (binary-sortable, time-ordered) | core         |
| `go-branded-id`   | Branded ID type backing                         | core         |
| `go-error-family` | Error classification taxonomy                   | core         |
| `go-faster/yaml`  | YAML marshaling                                 | catalog only |

## Core Concepts

### Commands (Write Side)

Commands embed `command.BasicCommand` for the required interface methods. Register typed handlers
with `command.RegisterTyped` to receive the concrete command type directly — no type assertions needed:

```go
type CreateUserCmd struct {
    *command.BasicCommand
    Email string
    Name  string
}

// Type-safe handler — cmd is *CreateUserCmd, not command.Command
command.RegisterTyped(d, "user.create", func(ctx context.Context, cmd *CreateUserCmd) error {
    return h.repo.Execute(ctx, cmd.AggregateID(), "User", decideCreateUser(cmd.Email, cmd.Name))
})
```

### Events (Event Sourcing)

Events represent state changes with rich metadata:

```go
event, err := event.NewEvent(
    event.Type("user.created"),
    userID,
    event.AggregateType("User"),
    1,
    payload,
    event.WithCorrelationID(requestID),
    event.WithUserID(operatorID),
)
```

### Strongly-Typed IDs

Prevents mixing up different ID types:

```go
import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// Instead of string IDs
userID := id.NewAggregateID()
orderID := id.NewAggregateID()

// These won't compile - type mismatch:
// store.Save(ctx, orderID, events)  // expects AggregateID, got UserID (if you used userID)

// Create domain-specific ID types
type OrderID = id.Of[orderMarker]
```

All IDs are branded types backed by ULID strings:

## Module Structure

| Module           | Import Path                           | Purpose                                          | Dependencies                      | Docs                        |
| ---------------- | ------------------------------------- | ------------------------------------------------ | --------------------------------- | --------------------------- |
| **core**         | `.../core/command`, `.../core/event`  | CQRS types, dispatchers, event sourcing          | ulid, branded-id, go-error-family | [README](core/README.md)    |
| **core/decider** | `.../core/decider`                    | Functional aggregate pattern (recommended)       | core                              |                             |
| **memory**       | `.../memory`                          | In-memory store/bus/snapshot (testing)           | core                              |                             |
| **catalog**      | `.../catalog`, `.../catalog/asyncapi` | AsyncAPI + EventCatalog generation               | core, yaml                        | [README](catalog/README.md) |
| **middleware**   | `.../middleware`                      | Logging, retry, validation, recovery, metrics    | core                              |                             |
| **projection**   | `.../projection`                      | Projection runner with replay and live subscribe | core, memory                      |                             |
| **saga**         | `.../saga`                            | Saga / Process Manager with compensation         | core                              |                             |
| **storage**      | `.../storage`                         | SQLite/Turso/PostgreSQL/Pebble event store       | core                              | [README](storage/README.md) |
| **watermill**    | `.../watermill`                       | Watermill message bus adapter                    | core, watermill                   |                             |
| **testhelpers**  | `.../testhelpers`                     | Shared test utilities (fakes, handlers, mocks)   | core                              |                             |
| **integration**  | `.../integration`                     | Cross-module integration tests                   | core, memory, helpers             |                             |
| **example/user** | `.../example/user`                    | Complete demo: CQRS + Decider + projections      | core, memory, catalog, middleware |                             |

## Design Principles

1. **Pay for what you import** — Each module has its own `go.mod` with only needed dependencies
2. **Composition over inheritance** — Per Go best practices
3. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, `Root`, etc.)
4. **Context-aware** — All operations accept `context.Context`
5. **Errors as values** — No panics in production paths; `Must*` variants provided for test convenience
6. **File size limits** — Max 250 lines per file

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                        │
│   HTTP Handlers ──► Command/Query Dispatchers               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      CORE MODULES                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Command    │  │    Query    │  │    Event    │       │
│  │  Dispatcher  │  │  Dispatcher │  │  Store/Bus   │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │  Aggregate   │  │   Decider   │  │ Projection  │       │
│  │  Repository │  │ Repository  │  │    Runner   │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    INFRASTRUCTURE LAYER                      │
│   MemoryStore ──► Storage (SQLite/Turso/Pebble)             │
│   MemoryBus ─────► Catalog (AsyncAPI/EventCatalog)          │
└─────────────────────────────────────────────────────────────┘
```

## Persistent Storage

### SQLite (recommended for single-process)

```go
package main

import (
    "context"
    "log"

    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/memory"
    "github.com/larsartmann/go-cqrs-lite/storage"
)

func main() {
    ctx := context.Background()

    // Open SQLite database
    db, err := storage.OpenSQLite("myapp.db")
    if err != nil { log.Fatal(err) }

    // Production safety
    storage.SQLiteEnableWAL(ctx, db)
    storage.ConfigureSQLitePool(db)
    storage.SQLiteInitSchema(ctx, db)

    // Create event store + in-memory bus
    store, _ := storage.NewSQLiteEventStore(db)
    bus := memory.NewMemoryBus()

    // Use with decider, aggregate, or directly
    aggID := id.NewAggregateID()
    evt, _ := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"name":"Alice"}`))
    store.Save(ctx, "User", aggID, []event.Event{evt}, 0)
    bus.Publish(ctx, evt)
}
```

### Turso (offline-first with sync)

```go
// Local Turso database
-db, _ := storage.OpenTurso("myapp.db")
storage.TursoInitSchema(ctx, db)
store, _ := storage.NewTursoEventStore(db)

// Synced Turso database (push/pull with remote)
syncDB, _ := storage.OpenTursoSync(ctx, "myapp.db", "libsql://db.turso.io", "token")
syncDB.Push(ctx)       // send local writes to remote
-syncDB.Pull(ctx)       // fetch remote changes
syncDB.Checkpoint(ctx) // compact WAL
store, _ := storage.NewTursoEventStore(syncDB.DB())
```

### Deterministic Testing with Clock

```go
import "github.com/larsartmann/go-cqrs-lite/core/event"

fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
clock := func() time.Time { return fixedTime }

evt, _ := event.NewEvent(
    "user.created", aggID, "User", 1, payload,
    event.WithClock(clock), // deterministic OccurredAt
)
// evt.OccurredAt() == fixedTime, every time
```

## Catalog Integration

Automatically generate [AsyncAPI 3.0](https://www.asyncapi.com/) specs and [EventCatalog](https://www.eventcatalog.dev/) documentation from your Go CQRS types — zero manual documentation effort.

### How It Works

```
Go Structs ──► SchemaFromType[T]() ──► Registry ──► Catalog
                                                      │
                                          ┌───────────┴───────────┐
                                          ▼                       ▼
                                   AsyncAPI 3.0 YAML      EventCatalog MDX
```

1. Define your commands, events, and queries as Go structs
2. `SchemaFromType[T]()` auto-generates JSON Schema via reflection
3. Register services and messages in a `Registry`
4. Export to AsyncAPI YAML or EventCatalog MDX files

### Usage

```go
package main

import (
    "github.com/larsartmann/go-cqrs-lite/catalog"
    "github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
    "github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

type CreateOrder struct {
    ProductID string `json:"product_id" doc:"ID of the product to order"`
    Quantity  int    `json:"quantity" doc:"Number of items"`
}

type OrderCreated struct {
    OrderID string `json:"order_id" doc:"The new order ID"`
}

func main() {
    reg := catalog.NewRegistry("Order Service", "1.0.0")

    reg.AddService(catalog.Service{ID: "order-service", Name: "Order Service"})
    reg.AddCommand("order-service", catalog.Message{
        ID: "create-order", Name: "CreateOrder", Version: "1.0.0",
        Summary:   "Place a new order",
        Schema:    catalog.SchemaFromType[CreateOrder](),
        Direction: catalog.Receives,
    })
    reg.AddEvent("order-service", catalog.Message{
        ID: "order-created", Name: "OrderCreated", Version: "1.0.0",
        Summary:   "An order was placed",
        Schema:    catalog.SchemaFromType[OrderCreated](),
        Direction: catalog.Sends,
    })

    c := reg.Build()

    // Export AsyncAPI 3.0 YAML
    doc := asyncapi.Exporter{}.Export(c)
    yamlBytes, _ := doc.MarshalYAML()

    // Export EventCatalog MDX files
    ec := eventcatalog.Exporter{OutputDir: "./eventcatalog"}
    _ = ec.Export(c)
    // Creates: services/order-service/index.mdx
    //          services/order-service/commands/create-order/index.mdx
    //          services/order-service/events/order-created/index.mdx
}
```

### Schema Reflection

`SchemaFromType[T]()` inspects Go struct fields and produces JSON Schema. It reads `json`, `doc`/`description`, and `format` struct tags:

```go
type User struct {
    Email string `json:"email" doc:"User email" format:"email"`
    Age   int    `json:"age" doc:"User age"`
}

schema := catalog.SchemaFromType[User]()
// {"type":"object","properties":{"email":{"type":"string","description":"User email","format":"email"},...}}
```

Supported struct tags:

| Tag           | Example                    | Effect                         |
| ------------- | -------------------------- | ------------------------------ |
| `json`        | `json:"name,omitempty"`    | Field name + required/optional |
| `doc`         | `doc:"User email"`         | Description                    |
| `description` | `description:"User email"` | Alias for `doc`                |
| `format`      | `format:"email"`           | JSON Schema format             |
| `enum`        | `enum:"active,inactive"`   | Enum values (comma-separated)  |
| `default`     | `default:"active"`         | Default value                  |
| `nullable`    | `nullable`                 | Marks field as nullable        |
| `deprecated`  | `deprecated`               | Marks field as deprecated      |
| `pattern`     | `pattern:"^[a-z]+$"`       | Regex pattern                  |

### Catalog Builder

The `catalog` package provides a fluent builder API:

```go
builder := catalog.NewBuilder("User Service", "1.0.0")
builder.AddService("user-svc", "User Service", "1.0.0", "Manages users",
    catalog.Command[CreateUserCmd]("user.create"),
    catalog.Event[UserCreatedEvent]("user.created", catalog.Sends),
    catalog.Query[GetUserQuery]("user.get"),
)
builder.AddDomain("identity", "Identity", "1.0.0",
    "User identity management", "user-svc")
cat := builder.Build()

// Export to AsyncAPI, EventCatalog, D2, or OpenAPI
asyncapi.NewExporter().Export(cat)
eventcatalog.NewExporter(cat).Export("./eventcatalog")
d2.NewExporter().Export(cat)
```

### Registry API

| Method                                    | Description                           |
| ----------------------------------------- | ------------------------------------- |
| `NewRegistry(title, version)`             | Create a new registry                 |
| `AddService(svc)`                         | Register a service (merges if exists) |
| `AddCommand(serviceID, msg)`              | Add a command to a service            |
| `AddEvent(serviceID, msg)`                | Add an event to a service             |
| `AddQuery(serviceID, msg)`                | Add a query to a service              |
| `AddDomain(domain)`                       | Register a domain                     |
| `AddServiceToDomain(serviceID, domainID)` | Link service to domain                |
| `AddChannel(ch)`                          | Register a channel                    |
| `Build()`                                 | Produce immutable `*Catalog`          |

### AsyncAPI 3.0 Output

The AsyncAPI exporter maps CQRS types to AsyncAPI 3.0 operations:

- Commands → `action: receive` (service receives commands)
- Events with `Sends` → `action: send`
- Events with `Receives` → `action: receive`
- Queries → `action: receive`

Default server: `kafka` at `localhost:9092`. Channel addresses use dot-separated lowercase (e.g., `CreateOrder` → `create.order`).

### EventCatalog Output

Writes an EventCatalog project directory structure with MDX files containing YAML frontmatter:

```
eventcatalog/
├── eventcatalog.config.js
├── services/
│   └── order-service/
│       ├── index.mdx
│       ├── commands/
│       │   └── create-order/
│       │       ├── index.mdx
│       │       └── schema.json
│       └── events/
│           └── order-created/
│               ├── index.mdx
│               └── schema.json
```

## Event Builder

Fluent builder for events with compile-time type safety:

```go
import (
    "encoding/json"

    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

aggregateID := id.NewAggregateID()

payload, _ := json.Marshal(map[string]any{"order_id": aggregateID.String()})

evt, err := event.NewBuilder(
    event.Type("order.created"),
    aggregateID,
    event.AggregateType("Order"),
    event.Version(1),
).
    WithPayload(payload).
    WithCorrelationID(id.NewCorrelationID()).
    WithUserID(id.NewUserID()).
    Build()
```

## Saga / Process Manager

Coordinate long-running business processes across multiple aggregates with automatic compensation (rollback) on failure:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/saga"
)

// OrderSaga defines a 3-step order fulfillment process.
type OrderSaga struct{}

func (OrderSaga) SagaType() string { return "order-fulfillment" }

func (OrderSaga) Steps() []saga.Step {
    return []saga.Step{
        {
            Name: "reserve-inventory",
            Action: func(ctx context.Context, instanceID id.AggregateID) command.Command {
                return &ReserveInventoryCmd{/* ... */}
            },
            Compensate: func(ctx context.Context, instanceID id.AggregateID) command.Command {
                return &ReleaseInventoryCmd{/* ... */}
            },
            Timeout: 5 * time.Second,
        },
        {
            Name: "charge-payment",
            Action: func(ctx context.Context, instanceID id.AggregateID) command.Command {
                return &ChargePaymentCmd{/* ... */}
            },
            Compensate: func(ctx context.Context, instanceID id.AggregateID) command.Command {
                return &RefundPaymentCmd{/* ... */}
            },
            Timeout: 10 * time.Second,
        },
        {
            Name: "ship-order",
            Action: func(ctx context.Context, instanceID id.AggregateID) command.Command {
                return &ShipOrderCmd{/* ... */}
            },
        },
    }
}

func main() {
    store := saga.NewMemoryStore()
    cmds := command.NewDispatcher()
    runner := saga.NewRunner(store, cmds)

    // Register the saga definition
    _ = runner.Register(OrderSaga{})

    // Start a new saga instance
    ctx := context.Background()
    instance, _ := runner.Start(ctx, "order-fulfillment", &CreateOrderCmd{})
    fmt.Printf("Saga started: %s\n", instance.ID)

    // Execute steps sequentially
    _ = runner.ExecuteStep(ctx, instance.ID) // reserve inventory
    _ = runner.ExecuteStep(ctx, instance.ID) // charge payment
    _ = runner.ExecuteStep(ctx, instance.ID) // ship order

    // If any step fails, completed steps are automatically compensated
    // (inventory released, payment refunded) in reverse order.
}
```

## Stream Loading

Memory-efficient event iteration for large aggregates or projection replay:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/storage"
)

func main() {
    ctx := context.Background()
    db, _ := storage.OpenSQLite("myapp.db")
    store, _ := storage.NewSQLiteEventStore(db)

    aggID := id.NewAggregateID()

    // Load events as a stream instead of a slice — constant memory usage
    stream, err := store.LoadStream(ctx, "Order", aggID)
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    var version event.Version
    for {
        evt, ok := stream.Next()
        if !ok {
            break
        }
        version = evt.Version()
        fmt.Printf("Event %s at version %d\n", evt.Type(), version)
    }

    if err := stream.Err(); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Aggregate at version %d\n", version)
}
```

`LoadStream` is available on `SQLEventStore` (cursor-based, memory-bounded) and can be adapted to any `Store` via `event.NewStoreStreamAdapter`.

## Watermill Integration

Adapter for the [Watermill](https://watermill.io/) message router ecosystem. Publish and subscribe to CQRS events via Watermill's `message.Message`:

```go
package main

import (
    "context"
    "log"

    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/watermill"
)

func main() {
    ctx := context.Background()

    // Create a Watermill publisher (e.g., NATS, Kafka, GoChannel)
    pub := /* your watermill publisher */

    // Wrap it with the CQRS adapter
    adapter := watermill.NewPublisher(pub)

    // Create a CQRS event
    evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload)

    // Publish — all 15 event fields are preserved in message metadata
    if err := adapter.Publish(ctx, evt); err != nil {
        log.Fatal(err)
    }

    // Subscribe: reconstruct events from Watermill messages
    sub := /* your watermill subscriber */
    messages, _ := sub.Subscribe(ctx, "events")
    for msg := range messages {
        evt, err := watermill.ToEvent(msg)
        if err != nil {
            msg.Nack()
            continue
        }
        // Process the reconstructed event
        log.Printf("Received: %s (version %d)", evt.Type(), evt.Version())
        msg.Ack()
    }
}
```

## Comparison

| Feature            | go-cqrs-lite | go-cqrs | cqrs-go |
| ------------------ | ------------ | ------- | ------- |
| Minimal deps       | ✅           | ❌      | ❌      |
| Event Sourcing     | ✅           | ✅      | ✅      |
| Event Bus          | ✅           | ✅      | ❌      |
| Strong IDs         | ✅           | ❌      | ❌      |
| Context support    | ✅           | ❌      | ✅      |
| Auto-docs          | ✅           | ❌      | ❌      |
| Middleware         | ✅           | ❌      | ❌      |
| Benchmarks         | ✅           | ❌      | ❌      |
| Saga / Process Mgr | ✅           | ❌      | ❌      |
| Stream Loading     | ✅           | ❌      | ❌      |
| Watermill Adapter  | ✅           | ❌      | ❌      |

## Project Status

**Phase:** Active Development (core stable, storage module complete for SQLite/Turso)

| Phase          | Status      | Description                                       |
| -------------- | ----------- | ------------------------------------------------- |
| Foundation     | ✅ Complete | Core types, events, commands, queries, aggregates |
| Event Layer    | ✅ Complete | Event store, event bus, in-memory implementations |
| Command Layer  | ✅ Complete | Command dispatcher with middleware support        |
| Query Layer    | ✅ Complete | Query dispatcher with typed results               |
| Middleware     | ✅ Complete | Logging, metrics, retry, validation, recovery     |
| Decider        | ✅ Complete | Functional aggregate pattern (recommended)        |
| Projections    | ✅ Complete | Projection runner with replay and live subscribe  |
| Storage        | ✅ Complete | SQLite, Turso, PostgreSQL, Pebble, In-Memory      |
| Tests          | ✅ Complete | Unit + integration + benchmarks + fuzzing         |
| CI/CD          | ✅ Complete | GitHub Actions, Nix flake, linting                |
| Saga           | ✅ Complete | Saga / Process Manager with compensation          |
| Watermill      | ✅ Complete | Watermill message bus adapter                     |
| Stream Loading | ✅ Complete | Memory-efficient event stream iteration           |
| Documentation  | ✅ Complete | README, TODO_LIST, CONTRIBUTING, CODE_OF_CONDUCT  |

See [FEATURES.md](FEATURES.md) for detailed feature inventory and maturity ratings.

## License

MIT

## References

- [DOMAIN_GLOSSARY.md](DOMAIN_GLOSSARY.md) - Domain context and project understanding
- [CQRS pattern](https://martinfowler.com/bliki/CQRS.html) - Martin Fowler
