# catalog — API Documentation Generation from Go Types

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/catalog/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/catalog/v3)

Automatically generate [AsyncAPI 3.0](https://www.asyncapi.com/) specs, [EventCatalog](https://www.eventcatalog.dev/) documentation, [OpenAPI/Swagger](https://swagger.io/specification/) specs, and [D2](https://d2lang.com/) architecture diagrams from your Go CQRS types.

```bash
go get github.com/larsartmann/go-cqrs-lite/catalog/v3
```

## Packages

| Package                | Purpose                                                          |
| ---------------------- | ---------------------------------------------------------------- |
| `catalog`              | Registry, schema reflection, builder, typed IDs                  |
| `catalog/asyncapi`     | AsyncAPI 3.0 YAML/JSON exporter                                  |
| `catalog/eventcatalog` | EventCatalog MDX file generator                                  |
| `catalog/openapi`      | OpenAPI 3.0 YAML/JSON exporter                                   |
| `catalog/d2`           | D2 diagram text exporter                                         |
| `catalog/docserver`    | HTTP handlers for serving docs (OpenAPI/AsyncAPI UI, D2, health) |
| `catalog/simple`       | Single-service builder facade (streamlined API)                  |

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
Generates `llms.txt` and `schemas.txt` for AI consumption.

#### Full Resource Coverage

The exporter supports all EventCatalog resource types:

| Resource      | Exported To                    | Notes                                                                    |
| ------------- | ------------------------------ | ------------------------------------------------------------------------ |
| Services      | `services/<id>/index.mdx`      | With messages, specs, data stores, external flag, base config (sidebar, styles, editUrl, draft, visualiser) |
| Domains       | `domains/<id>/index.mdx`       | With ubiquitous language, sub-domains, data products, base config        |
| Entities      | `entities/<id>/index.mdx`      | DDD entities: aggregateRoot, identifier, properties with references/relationTypes |
| Data Products | `data-products/<id>/index.mdx` | Data mesh products with inputs/outputs, output contracts, hidden flag    |
| Agents        | `agents/<id>/index.mdx`        | AI agents with sends/receives, model, tools, data stores                 |
| Channels      | `channels/<id>/index.mdx`      | With protocols, parameters, routes, delivery guarantees                  |
| Data Stores   | `data/<id>/index.mdx`          | Databases/caches with authoritative, accessMode, classification          |
| Flows         | `flows/<id>/index.mdx`         | All step types: service, message, agent, dataStore, dataProduct, subFlow |
| Teams         | `teams/<id>.mdx`               | With external source sync, hidden, readOnly                              |
| Users         | `users/<id>.mdx`               | With external source sync, hidden, readOnly                              |
| Custom Docs   | `docs/<slug>/index.mdx`        | Global documentation pages (ADRs, architecture docs)                     |

#### DDD Ubiquitous Language

```go
builder.AddDomain("orders", "Orders", "1.0.0", "Order management", "order-svc")
builder.ConfigureDomain("orders",
    catalog.DomainUbiquitousLanguage(
        catalog.UbiquitousLanguageTerm{Name: "Order", Description: "A customer purchase request"},
        catalog.UbiquitousLanguageTerm{Name: "Fulfillment", Description: "Completing an order"},
    ),
    catalog.DomainSubDomains("checkout", "shipping"),
    catalog.DomainDataProducts("order-analytics"),
)
```

#### External Services

```go
builder.AddService("stripe", "Stripe", "1.0.0", "")
builder.ConfigureService("stripe", catalog.ServiceExternalSystem())
```

#### Entities, Data Products, Agents

```go
builder.AddEntity(catalog.Entity{
    ID: "order", Name: "Order", Version: "1.0.0",
    AggregateRoot: true,
    Identifier:    "orderId",
    Properties: []catalog.EntityProperty{
        {Name: "orderId", Type: "string", Required: true},
        {Name: "customerId", Type: "string", References: "Customer", RelationType: "many-to-one"},
    },
    Schema: catalog.SchemaFromType[Order]()})

builder.AddDataProduct(catalog.DataProduct{ID: "metrics", Name: "Metrics", Version: "1.0.0",
    Inputs:  []catalog.Ref{{ID: "OrderCreated"}},
    Outputs: []catalog.Ref{{ID: "MetricsReady"}}})

builder.AddAgent(catalog.Agent{ID: "fraud-bot", Name: "Fraud Bot", Version: "1.0.0",
    Summary: "Reviews risky payments",
    Receives: []catalog.Ref{{ID: "PaymentInitiated", Version: "1.0.0"}},
    Sends:    []catalog.Ref{{ID: "FraudReviewCompleted", Version: "1.0.0"}},
    ReadsFrom: []catalog.DataStoreID{"fraud-db"},
    Model: &catalog.AgentModel{Provider: "OpenAI", Name: "gpt-4.1", Version: "2025-04-14"},
    Tools: []catalog.AgentTool{
        {Name: "Risk lookup", Type: "mcp", URL: "https://mcp.example.com/risk"},
    },
})
```

#### Flow Step Types

Flows support all EventCatalog step types — service, message, channel, actor,
external system, custom, **agent**, **data store**, **data product**, and **sub-flow**:

```go
builder.AddFlow(catalog.Flow{
    ID: "checkout", Name: "Checkout Flow", Version: "1.0.0",
    Steps: []catalog.FlowStep{
        {ID: "1", Title: "Agent validates", Agent: &catalog.FlowStepRef{ID: "checkout-bot"}},
        {ID: "2", Title: "Read inventory", DataStore: &catalog.FlowStepRef{ID: "inv-db"},
            NextStep: &catalog.FlowEdge{ID: "3"}},
        {ID: "3", Title: "Publish metrics", DataProduct: &catalog.FlowStepRef{ID: "sales-data"}},
    },
})
```

#### Custom Documentation Pages

```go
builder.AddCustomDoc(catalog.CustomDoc{
    ID:      "adr-001",
    Title:   "ADR-001: Event Sourcing",
    Summary: "Why we chose event sourcing",
    Slug:    "adrs/adr-001",
    Content: "## Decision\nWe adopted event sourcing for auditability.",
})
```

#### Message Deprecation

```go
// Simple boolean deprecation
reg.AddEvent("svc", catalog.Message{
    ID: "old-event", Name: "OldEvent", Version: "1.0.0",
    Deprecated: true,
})

// Structured deprecation with date and message
reg.AddEvent("svc", catalog.Message{
    ID: "old-event", Name: "OldEvent", Version: "1.0.0",
    Deprecation: &catalog.DeprecationInfo{
        Date:    &time.Now(),
        Message: "Use new-event instead",
    },
})
```

#### External Team/User Sync

```go
builder.AddUser(catalog.User{
    ID: "alice", Name: "Alice",
    Source: &catalog.Source{Provider: "github", ID: "alice@org"},
    ReadOnly: true,
})
```

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
type ServiceID string     // catalog.ServiceID
type DomainID string      // catalog.DomainID
type MessageID string     // catalog.MessageID
type ChannelID string     // catalog.ChannelID
type DataStoreID string   // catalog.DataStoreID
type FlowID string        // catalog.FlowID
type TeamID string        // catalog.TeamID
type UserID string        // catalog.UserID
type EntityID string      // catalog.EntityID
type DataProductID string // catalog.DataProductID
type AgentID string       // catalog.AgentID
type CustomDocID string   // catalog.CustomDocID
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
