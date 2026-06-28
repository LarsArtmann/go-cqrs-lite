# catalog — API Documentation Generation from Go Types

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/catalog/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/catalog/v3)

Automatically generate [AsyncAPI 3.0](https://www.asyncapi.com/) specs, [EventCatalog](https://www.eventcatalog.dev/) documentation, [OpenAPI/Swagger](https://swagger.io/specification/) specs, and [D2](https://d2lang.com/) architecture diagrams from your Go CQRS types.

```bash
go get github.com/larsartmann/go-cqrs-lite/catalog/v3
```

## Packages

| Package                | Purpose                                         |
| ---------------------- | ----------------------------------------------- |
| `catalog`              | Registry, schema reflection, builder, typed IDs |
| `catalog/asyncapi`     | AsyncAPI 3.0 YAML/JSON exporter                 |
| `catalog/eventcatalog` | EventCatalog MDX file generator                 |
| `catalog/openapi`      | OpenAPI 3.0 YAML/JSON exporter                  |
| `catalog/d2`           | D2 diagram text exporter                        |
| `catalog/docserver`    | HTTP handlers for serving docs (OpenAPI/AsyncAPI UI, D2, health) |
| `catalog/simple`       | Single-service builder facade (streamlined API) |

## Quick Start

```go
package main

import (
    "github.com/larsartmann/go-cqrs-lite/catalog"
    "github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
    "github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
)

type CreateOrder struct {
    ProductID string `json:"product_id" doc:"ID of the product"`
    Quantity  int    `json:"quantity" doc:"Number of items"`
}

type OrderCreated struct {
    OrderID string `json:"order_id" doc:"The new order ID"`
}

func main() {
    reg := catalog.NewRegistry("Order Service", "1.0.0")

    reg.AddService(catalog.Service{
        ID: "order-service", Name: "Order Service",
    })
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

    // AsyncAPI 3.0 YAML
    doc := asyncapi.Exporter{}.Export(c)
    yamlBytes, _ := doc.MarshalYAML()

    // EventCatalog MDX files
    ec := eventcatalog.Exporter{OutputDir: "./eventcatalog"}
    _ = ec.Export(c)
}
```

## Builder API (Fluent)

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
```

### Service Options

```go
builder.ConfigureService("user-svc",
    catalog.Badges(catalog.Badge{Label: "owner", Color: "blue"}),
    catalog.Repository("https://github.com/org/user-svc"),
    catalog.Owners("team-backend"),
)
```

## Schema Reflection

`SchemaFromType[T]()` inspects Go struct fields and produces JSON Schema:

```go
type User struct {
    Email string `json:"email" doc:"User email" format:"email"`
    Age   int    `json:"age" doc:"User age"`
}

schema := catalog.SchemaFromType[User]()
// {"type":"object","properties":{"email":{"type":"string","description":"User email","format":"email"},...}}
```

### Supported Struct Tags

| Tag           | Example                    | Effect                         |
| ------------- | -------------------------- | ------------------------------ |
| `json`        | `json:"name,omitempty"`    | Field name + required/optional |
| `doc`         | `doc:"User email"`         | Description                    |
| `description` | `description:"User email"` | Alias for `doc`                |
| `format`      | `format:"email"`           | JSON Schema format             |
| `enum`        | `enum:"active,inactive"`   | Enum values                    |
| `default`     | `default:"active"`         | Default value                  |

## Registry API

| Method                        | Description                           |
| ----------------------------- | ------------------------------------- |
| `NewRegistry(title, version)` | Create a new registry                 |
| `AddService(svc)`             | Register a service (merges if exists) |
| `AddCommand(serviceID, msg)`  | Add a command to a service            |
| `AddEvent(serviceID, msg)`    | Add an event to a service             |
| `AddQuery(serviceID, msg)`    | Add a query to a service              |
| `AddDomain(domain)`           | Register a domain                     |
| `Build()`                     | Produce immutable `*Catalog`          |

## Exporters

### AsyncAPI 3.0

```go
doc := asyncapi.Exporter{}.Export(catalog)
yamlBytes, _ := doc.MarshalYAML()
jsonBytes, _ := doc.MarshalJSON()
```

Maps CQRS types to AsyncAPI operations:

- Commands → `action: receive`
- Events with `Sends` → `action: send`
- Events with `Receives` → `action: receive`

### EventCatalog

```go
ec := eventcatalog.Exporter{OutputDir: "./eventcatalog"}
ec.Export(catalog)
```

Writes MDX files with YAML frontmatter, auto-deriving producers/consumers from message directions.

### OpenAPI

```go
doc := openapi.Exporter{}.Export(catalog)
yamlBytes, _ := doc.MarshalYAML()
```

### D2 Diagrams

```go
text := d2.NewExporter().Export(catalog)
```

## Branded ID Types

## docserver — HTTP Handlers

Serve auto-generated API documentation from a `*catalog.Catalog` via stdlib `net/http`:

```go
import "github.com/larsartmann/go-cqrs-lite/catalog/v3/docserver"

// Full docs server (OpenAPI/AsyncAPI JSON+YAML+HTML UI, catalog JSON)
ds := docserver.NewDocsServer(func() *catalog.Catalog {
    return builder.Build()
}, docserver.Config{
    ServiceName: "Order Service",
    Version:     "1.0.0",
})

mux := http.NewServeMux()
ds.RegisterRoutes(mux) // registers /docs/openapi, /docs/asyncapi, etc.

// Standalone handlers (no DocsServer needed)
mux.HandleFunc("/diagram.d2", docserver.D2Handler(cat))
mux.HandleFunc("/health", docserver.HealthCheckHandler(cat))
```

### EventCatalog File Generation

```go
// Write MDX files at startup for the EventCatalog CLI to serve.
err := docserver.GenerateEventCatalog(cat, "./eventcatalog")
```

## simple — Single-Service Builder Facade

Most services document a single application. The `simple` package reduces ceremony:

```go
import "github.com/larsartmann/go-cqrs-lite/catalog/v3/simple"

b := simple.New("User Service", "1.0.0")
simple.Command[RegisterUserCmd](b, "register-user",
    simple.WithOperation("POST", "/api/users"))
simple.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)
cat := b.Build()
```

Access the underlying `catalog.Builder` via `b.InnerBuilder()` for multi-service catalogs.

The catalog module provides typed IDs for catalog entries:

```go
type ServiceID string   // catalog.ServiceID
type DomainID string    // catalog.DomainID
type MessageID string   // catalog.MessageID
type ChannelID string   // catalog.ChannelID
type DataStoreID string // catalog.DataStoreID
type FlowID string      // catalog.FlowID
type TeamID string      // catalog.TeamID
type UserID string      // catalog.UserID
```

## Dependencies

| Dependency       | Purpose         |
| ---------------- | --------------- |
| `go-faster/yaml` | YAML marshaling |

## Related Modules

- [**command/v2**](../command/README.md) — Generates docs for command types
- [**event/v2**](../event/README.md) — Generates docs for event types
- [**query/v2**](../query/README.md) — Generates docs for query types
- [**dispatcher/v2**](../dispatcher/README.md) — `CatalogDispatcher` for introspection
