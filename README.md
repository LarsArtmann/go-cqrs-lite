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
- **Strongly-Typed IDs** - Branded identifier types to prevent mixing up IDs (UserId, AggregateId, etc.)
- **Extended Types** - Type-safe wrappers for commands, queries, and events with built-in ID types

## Quick Start

### Installation

```bash
# Core CQRS types (commands, queries, events, aggregates, IDs)
go get github.com/larsartmann/go-cqrs-lite/core

# In-memory implementations (testing)
go get github.com/larsartmann/go-cqrs-lite/memory

# API documentation generation (AsyncAPI 3.0 + EventCatalog)
go get github.com/larsartmann/go-cqrs-lite/catalog

# Cross-cutting middleware (logging, retry, validation, recovery, metrics)
go get github.com/larsartmann/go-cqrs-lite/middleware

# Type-safe wrappers with branded IDs
go get github.com/larsartmann/go-cqrs-lite/xtypes
```

### Requirements

- Go 1.26+

### Core Dependencies

| Dependency | Purpose | Module |
|---|---|---|
| `cockroachdb/errors` | Error wrapping | core |
| `oklog/ulid/v2` | ULID generation (binary-sortable, time-ordered) | core |
| `go-composable-business-types` | Branded ID type backing | core |
| `go-json-experiment/json` | JSON v2 | core |
| `go-faster/yaml` | YAML marshaling | catalog only |

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

## Module Structure

| Module | Import Path | Purpose | Dependencies |
|---|---|---|---|
| **core** | `.../core/...` | CQRS types, dispatchers, event sourcing | errors, ulid, json |
| **memory** | `.../memory` | In-memory store/bus/snapshot (testing) | core |
| **catalog** | `.../catalog/...` | AsyncAPI + EventCatalog generation | core, yaml |
| **middleware** | `.../middleware` | Logging, retry, validation, recovery, metrics | core |
| **xtypes** | `.../xtypes` | Typed wrappers with branded IDs | core |

## Design Principles

1. **Pay for what you import** — Each module has its own `go.mod` with only needed dependencies
2. **Composition over inheritance** — Per Go best practices
3. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, `Root`, etc.)
4. **Context-aware** — All operations accept `context.Context`
5. **Errors as values** — No panics, explicit error returns
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

    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/memory"
)

func main() {
    ctx := context.Background()

    // Create in-memory event store and bus (testing)
    store := memory.NewMemoryStore()
    bus := memory.NewMemoryBus()

    // Register command dispatcher
    cmdDispatcher := command.NewDispatcher()

    // Use strongly-typed IDs
    userID := id.NewAggregateID()

    // Create and dispatch command
    cmd := command.New("user.create", userID.String())
    cmdDispatcher.Dispatch(ctx, cmd)
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

## xtypes (Extended Types)

The `xtypes` package provides type-safe wrappers around core CQRS types, eliminating stringly-typed aggregate IDs and reducing boilerplate.

### EventBuilder

Fluent builder for events with compile-time type safety:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
    "github.com/larsartmann/go-cqrs-lite/xtypes"
)

aggregateID := id.NewAggregateID()

evt, err := xtypes.NewEventBuilder(
    "order.created",
    aggregateID,
    "Order",
    1,
).
    WithPayload(jsonPayload).
    WithCorrelationID(correlationID).
    WithUserID(operatorID).
    Build()
```

### TypedCommand

Wrap commands with strongly-typed aggregate IDs:

```go
cmd := xtypes.NewTypedCommand("create.order", aggregateID)
fmt.Println(cmd.Type())        // "create.order"
fmt.Println(cmd.AggregateID()) // id.AggregateID
```

### TypedAggregate

Aggregate roots with branded IDs and event replay:

```go
agg := xtypes.NewTypedAggregate(aggregateID, "Order")
evt, _ := xtypes.NewEventBuilder("order.created", aggregateID, "Order", 1).Build()
agg.RecordEvent(ctx, evt)

events := agg.UncommittedChanges()
agg.MarkChangesAsCommitted()
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

**Phase:** Production Ready (All core features complete)

| Phase         | Status      | Description                                       |
| ------------- | ----------- | ------------------------------------------------- |
| Foundation    | ✅ Complete | Core types, events, commands, queries, aggregates |
| Event Layer   | ✅ Complete | Event store, event bus, in-memory implementations |
| Command Layer | ✅ Complete | Command dispatcher with middleware support        |
| Query Layer   | ✅ Complete | Query dispatcher with typed results               |
| Middleware    | ✅ Complete | Logging, metrics, retry, validation, recovery     |
| Tests         | ✅ Complete | Unit + integration + benchmarks + fuzzing         |
| CI/CD         | ✅ Complete | GitHub Actions, Makefile, linting                 |
| Documentation | ✅ Complete | README, TODO_LIST, CONTRIBUTING, CODE_OF_CONDUCT  |

See [TODO_LIST.md](TODO_LIST.md) for detailed status.

## License

MIT

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Coding standards
- [CQRS pattern](https://martinfowler.com/bliki/CQRS.html) - Martin Fowler
