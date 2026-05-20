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
- **Aggregate Roots** - Base implementation for domain-driven design aggregates
- **Decider Pattern** - Functional aggregate approach with pure functions (recommended)
- **Projections** - Build read models from events with replay support
- **Strongly-Typed IDs** - Branded identifier types to prevent mixing up IDs (UserId, AggregateId, etc.)
- **Error Classification** - Structured errors with retry semantics (Rejection, Conflict, Transient, etc.)
- **Auto-documentation** - Generate AsyncAPI 3.0 specs and EventCatalog from code

## Quick Start

### Installation

```bash
# Core CQRS types (commands, queries, events, IDs, decider)
go get github.com/larsartmann/go-cqrs-lite/core

# In-memory implementations (testing)
go get github.com/larsartmann/go-cqrs-lite/memory

# API documentation generation (AsyncAPI 3.0 + EventCatalog)
go get github.com/larsartmann/go-cqrs-lite/catalog

# Cross-cutting middleware (logging, retry, validation, recovery, metrics)
go get github.com/larsartmann/go-cqrs-lite/middleware

# Projection runner with replay and live subscription
go get github.com/larsartmann/go-cqrs-lite/projection
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
    *command.Core
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
    cmd := &CreateUserCmd{Core: command.MustNew("user.create", aggID), Name: "Alice"}
    if err := cmds.Dispatch(context.Background(), cmd); err != nil { log.Fatal(err) }

    // Load and read
    events, _ := store.Load(context.Background(), "User", aggID)
    for _, e := range events {
        p, _ := event.DecodePayload[UserCreated](e, event.JSONCodec{})
        fmt.Printf("User created: %s\n", p.Name)
    }
}
```

See `example/user/` for a complete example with the Decider pattern, middleware, and catalog generation.

### Core Dependencies

| Dependency        | Purpose                                         | Module       |
| ----------------- | ----------------------------------------------- | ------------ |
| `oklog/ulid/v2`   | ULID generation (binary-sortable, time-ordered) | core         |
| `go-branded-id`   | Branded ID type backing                         | core         |
| `go-error-family` | Error classification taxonomy                   | core         |
| `go-faster/yaml`  | YAML marshaling                                 | catalog only |

## Core Concepts

### Commands (Write Side)

Commands embed `command.Core` for the required interface methods. Register typed handlers
with `command.RegisterTyped` to receive the concrete command type directly — no type assertions needed:

```go
type CreateUserCmd struct {
    *command.Core
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

| Module           | Import Path                           | Purpose                                          | Dependencies                      |
| ---------------- | ------------------------------------- | ------------------------------------------------ | --------------------------------- |
| **core**         | `.../core/command`, `.../core/event`  | CQRS types, dispatchers, event sourcing          | ulid, branded-id, go-error-family |
| **core/decider** | `.../core/decider`                    | Functional aggregate pattern (recommended)       | core                              |
| **memory**       | `.../memory`                          | In-memory store/bus/snapshot (testing)           | core                              |
| **catalog**      | `.../catalog`, `.../catalog/asyncapi` | AsyncAPI + EventCatalog generation               | core, yaml                        |
| **middleware**   | `.../middleware`                      | Logging, retry, validation, recovery, metrics    | core                              |
| **projection**   | `.../projection`                      | Projection runner with replay and live subscribe | core, memory                      |
| **storage**      | `.../storage`                         | PostgreSQL/SQLite/Pebble event store             | core                              |
| **testhelpers**  | `.../testhelpers`                     | Shared test utilities (fakes, handlers, mocks)   | core                              |
| **integration**  | `.../integration`                     | Cross-module integration tests                   | core, memory, helpers             |
| **example/user** | `.../example/user`                    | Complete demo: CQRS + Decider + projections      | core, memory, catalog, middleware |

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
│   MemoryStore ──► Storage (PostgreSQL/SQLite/Pebble)        │
│   MemoryBus ─────► Catalog (AsyncAPI/EventCatalog)          │
└─────────────────────────────────────────────────────────────┘
```

## Usage Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/decider"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/memory"
)

// UserState represents the aggregate state
type UserState struct {
    Email string
    Name  string
}

// Decide handles commands and returns events (pure function)
func decide(state UserState, cmd *command.Core) ([]event.Event, error) {
    return nil, nil // simplified
}

// Fold reconstructs state from events (pure function)
func fold(state UserState, evt event.Event) (UserState, error) {
    return state, nil // simplified
}

func main() {
    ctx := context.Background()

    // Create infrastructure
    store := memory.NewMemoryStore()
    bus := memory.NewMemoryBus()

    // Create decider (recommended pattern - pure functions)
    userDecider := decider.Decider[UserState]{
        Initial: UserState{},
        Fold:    fold,
    }

    // Create repository
    repo, err := decider.NewRepository(store, bus, userDecider)
    if err != nil {
        log.Fatal(err)
    }

    // Create dispatcher and register handler
    cmdDispatcher := command.NewDispatcher()
    cmdDispatcher.Register(command.Type("user.create"), func(ctx context.Context, cmd command.Command) error {
        // Use repository to execute command
        payload, _ := json.Marshal(map[string]any{"email": "test@example.com"})
        evt, _ := event.NewEvent(event.Type("user.created"), id.NewAggregateID(), event.AggregateType("User"), 1, payload)
        return bus.Publish(ctx, evt)
    })

    // Use strongly-typed IDs
    userID := id.NewAggregateID()

    // Create and dispatch command
    cmd, err := command.New(command.Type("user.create"), userID)
    if err != nil {
        log.Fatal(err)
    }

    err = cmdDispatcher.Dispatch(ctx, cmd)
    if err != nil {
        log.Fatal(err)
    }

    _ = repo // repository ready for use
}
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

### Catalog Adapters

The `catalog/adapters` package provides a fluent builder API:

```go
builder := adapters.NewBuilder("My API", "1.0.0")
builder.AddService("user-svc", "User Service", "1.0.0", "Manages users")

// Instance-based: requires a constructed command with CatalogCore
builder.AddCommand("user-svc", myCmd)

// Generic zero-instance: no construction needed, uses reflection on type T
adapters.AddCommandFromType[CreateUser](
    builder, "user-svc", "user.create",
    command.CatalogMeta{Name: "CreateUser", Version: "1.0.0", Summary: "Create a user"},
)

// Auto-discover from dispatcher entries
cmdDispatcher.RegisterCatalogEntry("user.create", command.CatalogMeta{...})
adapters.FromCommandDispatcher(builder, "user-svc", cmdDispatcher)

// Export
cat := builder.Build()
builder.ExportEventCatalog("./eventcatalog")
doc, _ := builder.ExportAsyncAPI("My API", "1.0.0")
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

## Comparison

| Feature         | go-cqrs-lite | go-cqrs | cqrs-go |
| --------------- | ------------ | ------- | ------- |
| Minimal deps    | ✅           | ❌      | ❌      |
| Event Sourcing  | ✅           | ✅      | ✅      |
| Event Bus       | ✅           | ✅      | ❌      |
| Strong IDs      | ✅           | ❌      | ❌      |
| Context support | ✅           | ❌      | ✅      |
| Auto-docs       | ✅           | ❌      | ❌      |
| Middleware      | ✅           | ❌      | ❌      |
| Benchmarks      | ✅           | ❌      | ❌      |

## Project Status

**Phase:** Active Development (core stable, storage module partially functional)

| Phase         | Status      | Description                                       |
| ------------- | ----------- | ------------------------------------------------- |
| Foundation    | ✅ Complete | Core types, events, commands, queries, aggregates |
| Event Layer   | ✅ Complete | Event store, event bus, in-memory implementations |
| Command Layer | ✅ Complete | Command dispatcher with middleware support        |
| Query Layer   | ✅ Complete | Query dispatcher with typed results               |
| Middleware    | ✅ Complete | Logging, metrics, retry, validation, recovery     |
| Decider       | ✅ Complete | Functional aggregate pattern (recommended)        |
| Projections   | ✅ Complete | Projection runner with replay and live subscribe  |
| Storage       | ⚠️ Partial  | PostgreSQL/SQLite/Pebble (partially functional)   |
| Tests         | ✅ Complete | Unit + integration + benchmarks + fuzzing         |
| CI/CD         | ✅ Complete | GitHub Actions, Nix flake, linting                |
| Documentation | ✅ Complete | README, TODO_LIST, CONTRIBUTING, CODE_OF_CONDUCT  |

See [FEATURES.md](FEATURES.md) for detailed feature inventory and maturity ratings.

## License

MIT

## References

- [DOMAIN_GLOSSARY.md](DOMAIN_GLOSSARY.md) - Domain context and project understanding
- [CQRS pattern](https://martinfowler.com/bliki/CQRS.html) - Martin Fowler
