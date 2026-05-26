# Project: go-cqrs-lite

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import modules (`core`, `storage`, `memory`, `catalog`, etc.) into THEIR projects.
> There is no "main app." Every module is independently importable.
>
> | Application Lens (WRONG)              | Library/SDK Lens (CORRECT)                                  |
> | ------------------------------------- | ----------------------------------------------------------- |
> | "Zero internal consumers = dead code" | "Zero internal consumers = correct isolation"               |
> | "Module needs a service that uses it" | "Module needs tests + stable API, not an internal consumer" |
> | "example/ should drive real traffic"  | "example/ is a usage demo, not a deployment"                |
> | "Unused exports are waste"            | "Public API surface IS the product"                         |
>
> **The quality gate for every module: "Would a consumer trust this enough to import it?"**

A lightweight CQRS **library/SDK** for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework — no opinionated transport, message broker, or SQL driver.

## Quick Reference

| Item      | Value                                                                                                              |
| --------- | ------------------------------------------------------------------------------------------------------------------ |
| Language  | Go 1.26.3                                                                                                          |
| Modules   | `core` (incl. `decider`), `memory`, `catalog`, `middleware`, `testhelpers`, `integration`, `storage`, `projection` |
| Build     | `nix run .#build`                                                                                                  |
| Test      | `nix run .#test` or see "Testing" below                                                                            |
| Lint      | `nix run .#lint`                                                                                                   |
| Format    | `nix fmt`                                                                                                          |
| Dev shell | `nix develop`                                                                                                      |
| CI        | GitHub Actions: ci.yml (Nix-based)                                                                                 |

## Monorepo Structure

Multi-module Go workspace with 10 modules:

```
go-cqrs-lite/
├── go.work                          # ties modules together
│
├── core/                            # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                       # deps: oklog/ulid, go-branded-id
│   ├── command/                     # command dispatch, handler, catalog
│   ├── query/                       # query dispatch, pagination, catalog
│   ├── event/                       # event types, Store/Bus/SnapshotStore interfaces
│   │   ├── event.go                # ImmutableEvent struct, NewEvent constructor
│   │   └── options.go              # Option func, With* metadata helpers
│   ├── decider/                     # Decider[State], Repository[State], Execute, Load (pure-function style)
│   ├── pkg/
│   │   ├── id/                      # branded IDs: id.Of[T] (type alias for go-branded-id ID[T, ulid.ULID])
│   │   │   └── id.go               # ULID constructors, parsing, CompareIDs, FromPtr
│   │   └── dispatcher/              # generic Dispatcher[H, M] with LifecycleMixin, CheckClosed
│
├── memory/                          # github.com/larsartmann/go-cqrs-lite/memory
│   └── go.mod                       # deps: core
│   ├── store.go                     # MemoryStore (implements event.Store)
│   ├── bus.go                       # MemoryBus (implements event.Bus)
│   └── snapshot.go                  # MemorySnapshotStore (implements event.SnapshotStore)
│
├── catalog/                         # github.com/larsartmann/go-cqrs-lite/catalog
│   └── go.mod                       # deps: core, go-faster/yaml
│   ├── types.go                     # Message, Service, Domain, Channel, Schema, 8 branded ID types (ServiceID, DomainID, MessageID, ChannelID, DataStoreID, FlowID, TeamID, UserID)
│   ├── types_resources.go           # DataStore, Flow, FlowStep, FlowEdge, Team, User
│   ├── schema.go                    # SchemaFromType[T]() via reflect
│   ├── registry.go                  # thread-safe Registry, Build() → Catalog
│   ├── build.go                     # Builder with fluent ServiceOption/DomainOption/ChannelOption APIs (27 option functions)
│   ├── service_config.go            # ServiceOption: Badges, Repository, WritesTo, ReadsTo, Entities, Specifications, Attachments, Owners
│   ├── domain_config.go             # DomainOption: Sends, Receives, Entities, Badges, Owners, Attachments
│   ├── channel_config.go            # ChannelOption: Address, Protocols, Messages, DeliveryGuarantee, Parameters, Routes, Owners, Badges
│   ├── asyncapi/                    # AsyncAPI 3.0 YAML/JSON exporter (Document.ID uses URI branded type)
│   ├── d2/                          # D2 diagram text exporter (uses catalog.MessageID)
│   ├── eventcatalog/                # EventCatalog MDX file generator (auto-derives producers/consumers)
│   └── internal/cattest/            # test helpers (accept branded types)
│
├── middleware/                       # github.com/larsartmann/go-cqrs-lite/middleware
│   └── go.mod                       # deps: core
│   ├── logging.go                   # CommandLogging, EventLogging
│   ├── metrics.go                   # CommandMetrics, EventMetrics
│   ├── recovery.go                  # CommandRecovery, EventRecovery
│   ├── retry.go                     # CommandRetry, EventRetry (exponential backoff)
│   └── validation.go                # CommandValidation, EventValidation, QueryValidation
│
├── testhelpers/                     # github.com/larsartmann/go-cqrs-lite/testhelpers
│   └── go.mod                       # deps: core
│   └── helpers.go                   # Shared test utilities (AppendEventsHandler, Noop*, Failing*, etc.)
│
├── projection/                      # github.com/larsartmann/go-cqrs-lite/projection
│   └── go.mod                       # deps: core, memory (test)
│   ├── runner.go                    # Runner with Register(Projection), replay, live subscription
│   ├── handler.go                   # HandlerRegistry (On, OnAll, Lookup)
│   ├── errors.go                    # Sentinel errors
│   └── options.go                   # RunnerOption functional options
│
└── docs/
    ├── status/                      # periodic status reports
    └── planning/                    # architectural decisions and migration plans
```

## Testing

From root with go.work:

```bash
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./testhelpers/... ./integration/... ./projection/... ./storage/... -count=1
```

Per-module (isolated, no go.work):

```bash
cd core && GOWORK=off go test ./... -count=1
cd memory && GOWORK=off go test ./... -count=1
cd catalog && GOWORK=off go test ./... -count=1
```

Nix flake apps (run from root):

```bash
nix run .#test          # all tests
nix run .#test-race     # race detector
nix run .#coverage      # coverage report
nix run .#build         # go build
nix run .#vet           # go vet
nix run .#lint          # golangci-lint
nix fmt                 # format all Go files
nix flake check         # formatting check
nix develop             # enter dev shell
```

## Architecture

**Visual diagram**: `docs/web-client-communication.d2` — render with `d2 docs/web-client-communication.d2 docs/web-client-communication.svg` or view `docs/web-client-communication.html`.

```
┌────────────────────────────────────────────────────────────────┐
│                        APPLICATION LAYER                        │
│   HTTP Handlers ──► Command/Query Dispatchers                   │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                         CORE MODULE                             │
│  ┌──────────────┐  ┌──────────────┐
│  │   Command    │  │    Query     │
│  │  Dispatcher  │  │  Dispatcher  │
│  └──────────────┘  └──────────────┘
│  ┌──────────────┐  ┌──────────────┐
│  │    Event     │  │   Decider    │
│  │  Store+Bus   │  │  Repository  │
│  └──────────────┘  └──────────────┘
└────────────────────────────────────────────────────────────────┘
         │           │           │           │
         ▼           ▼           ▼           ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  memory      │ │  catalog     │ │  middleware   │
│  MemoryStore │ │  AsyncAPI    │ │  Logging      │
│  MemoryBus   │ │  EventCat    │ │  Retry        │
│  Snapshot    │ │  Schema      │ │  Recovery     │
└──────────────┘ └──────────────┘ └──────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │  integration/    │
                              │  Cross-module    │
                              │  tests           │
                              └─────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │  storage/        │
                              │  (PostgreSQL)    │
                              │  watermill/      │
                              │  (planned)       │
                              └─────────────────┘
```

## Package Overview

### Core Module (`core/`)

| Package                | Purpose                                   | Key Types                                                                                                                                                                                                                                        |
| ---------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `core/command/`        | Command dispatch and handling             | `Dispatcher`, `Handler`, `Middleware`, `Command`, `BasicCommand`                                                                                                                                                                                 |
| `core/query/`          | Query dispatch with pagination            | `Dispatcher`, `Handler`, `Pagination`, `PaginatedResult[T]`, `Middleware`, `TypedHandler[T]`, `RegisterTyped[T]`                                                                                                                                 |
| `core/event/`          | Event sourcing interfaces and types       | `Store`, `Bus`, `Publisher`, `Subscriber`, `SnapshotStore`, `TransactionalStore`, `GlobalLoader`, `Event`, `ImmutableEvent`, `New`, `Metadata`, `Option`, `Version`, `SchemaVersion`, `Type`, `AggregateType`, `Clock`, `WithReplay`, `IsReplay` |
| `core/aggregate/`      | Aggregate roots and repository (OO)       | `Root`, `Repository`, `ImmutableEvent`, `EventSourcedRepository` _(Deprecated: use decider)_                                                                                                                                                     |
| `core/decider/`        | Aggregate via pure functions              | `Decider[State]`, `Repository[State]`, `Execute`, `ExecuteWithResult`, `Load`, `DecideFunc`, `Result`                                                                                                                                            |
| `core/pkg/id/`         | Branded IDs (type alias to go-branded-id) | `id.Of[T]` = `cbid.ID[T, ulid.ULID]`, `AggregateID`, `EventID`, `UserID`, `CorrelationID`, `ClientID`, `CompareIDs`, `FromPtr`, `DeriveAggregateID`, `AggregateIDFrom`                                                                           |
| `core/pkg/dispatcher/` | Generic internal dispatcher               | `Dispatcher[H, M]`, `MiddlewareChain[H, M]`, `LifecycleMixin`                                                                                                                                                                                    |

### Decider Module (`core/decider/`)

| Package         | Purpose                                    | Key Types                                                                     |
| --------------- | ------------------------------------------ | ----------------------------------------------------------------------------- |
| `core/decider/` | Functional aggregate pattern (recommended) | `Decider[State]`, `Repository[State]`, `Execute`, `ExecuteWithResult`, `Load` |

- Pure functions, no mutable state, no 9-method interface, zero-infrastructure testing.
- `Decider[State]` holds `Initial` state and `Fold func(State, Event) (State, error)`.
- `Repository[State].Execute` does load → fold → decide → save → publish.
- Replaced the deprecated `core/aggregate` package (deleted Session 99).

### Memory Module (`memory/`)

| Package   | Purpose                        | Key Types                                         |
| --------- | ------------------------------ | ------------------------------------------------- |
| `memory/` | In-memory test implementations | `MemoryStore`, `MemoryBus`, `MemorySnapshotStore` |

### Catalog Module (`catalog/`)

| Package                 | Purpose                                | Key Types                                                                                                                                     |
| ----------------------- | -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `catalog/`              | Registry, schema reflection, typed IDs | `Registry`, `Catalog`, `SchemaFromType[T]`, `GetID()`, `Validate()`, `ServiceID`, `DomainID`, `MessageID`, `ChannelID`, `Change`, `Violation` |
| `catalog/asyncapi/`     | AsyncAPI 3.0 YAML/JSON export          | `Exporter`, `Document`, `MarshalYAML`                                                                                                         |
| `catalog/d2/`           | D2 diagram text export                 | `Exporter`, `Export()`, `NewExporter()`                                                                                                       |
| `catalog/eventcatalog/` | EventCatalog MDX generator             | `Exporter`                                                                                                                                    |

### Middleware Module (`middleware/`)

| Package       | Purpose                       | Key Types                                                                                                                                                                                                             |
| ------------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `middleware/` | Cross-cutting CQRS middleware | `CommandLogging`, `CommandRetry`, `CommandRecovery`, `CommandValidation`, `EventValidation`, `QueryValidation`, `CommandMetrics`, `ErrValidationFailed`, `ErrRetryExhausted`, `ErrRetryCanceled`, `ErrPanicRecovered` |

### Testhelpers Module (`testhelpers/`)

| Helper                                          | Purpose                                       |
| ----------------------------------------------- | --------------------------------------------- |
| `AppendEventsHandler`                           | Bus handler that collects events into a slice |
| `NoopCommandHandler` / `NoopEventHandler`       | No-op handlers for middleware tests           |
| `FailingCommandHandler` / `FailingEventHandler` | Handlers that always error                    |
| `PanicCommandHandler` / `PanicEventHandler`     | Handlers that panic                           |
| `CallbackCommandHandler`                        | Handler that sets a bool flag                 |
| `CommandMiddleware` / `EventMiddleware`         | Call-order tracking middleware                |
| `FakeMetrics`                                   | Metrics collector for testing                 |

### Projection Module (`projection/`)

| Package       | Purpose                                             | Key Types                                                                                                        |
| ------------- | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `projection/` | Projection runner with replay and live subscription | `Runner`, `HandlerRegistry`, `Builder`, `NewBuilder`, `On[T]`, `NewRunner`, `Register(Projection)`, `WithLogger` |

- **Runner**: Accepts `event.GlobalLoader` (for replay) + `event.Bus` (for live). Register `event.Projection` instances before `Run()`.
- **Per-projection checkpoint**: Each projection tracked by `Name()`. Events past checkpoint are skipped during replay.
- **Wildcard**: `EventTypes() == nil` subscribes to all event types.
- **HandlerRegistry**: Maps event types to handlers. Useful for building custom projection dispatch.
- **Builder**: Fluent API for defining projections with `On[T]()` type-safe handlers. JSON-decodes payloads into typed structs.
- **Replay context**: `event.WithReplay(ctx, true)` / `event.IsReplay(ctx)` — handlers can distinguish replay from live events.

## Design Principles

1. **Library, not framework** — Consumers import what they need, compose their own stack. No opinionated transport (HTTP/gRPC), message broker (Kafka/NATS), or SQL driver. Integration modules (storage, watermill) are optional.
2. **Every module must be trustworthy on its own** — Quality gate: "Would a consumer trust this enough to import it?" Means: tests, stable API, clear docs. Does NOT mean "another module in this repo uses it."
3. **Minimal core dependencies** — core depends on `oklog/ulid`, `go-branded-id`, `go-error-family`
4. **Composition over inheritance** — Per Go best practices
5. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, etc.)
6. **Context-aware** — All handlers accept `context.Context`
7. **Errors as values** — No panics, explicit error returns, sentinel errors + wrapping
8. **File size limits** — Max 250 lines per file
9. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps

## Code Conventions

- Use `fmt.Errorf` for error messages with context
- Use `errors.New` (stdlib) for sentinel errors
- Wrap errors with context using `fmt.Errorf` with `%w` (cockroachdb/errors removed in Session 54)
- Context as first parameter in all public functions
- Max 30 lines per function
- No `any` types

## Error Handling Pattern

```go
// Sentinel errors (in errors.go)
var ErrNotFound = errors.New("not found")

// Contextual errors (in functions)
if id == "" {
    return fmt.Errorf("id is required for operation %q", operation)
}

// Error wrapping
if err != nil {
    return fmt.Errorf("failed to process %s: %w", name, err)
}

// Classified errors (via go-error-family)
return event.NewRejection("user.create.empty_email", "email is required")
return event.NewConflict("user.create.duplicate", "user already exists")
```

## Key Patterns

### Command Handler

```go
func (h *Handler) Handle(ctx context.Context, cmd *CreateUser) error {
    if cmd.Email == "" {
        return fmt.Errorf("email is required for user creation")
    }
    // ... implementation
}
```

### Query Handler (typed bookend pattern)

```go
// Handler returns `any` at the boundary (required for heterogeneous dispatch).
// Use RegisterTyped[T] + DispatchTyped[T] for type-safe results.
type Handler = func(context.Context, Query) (any, error)

// Registration (type-safe result)
_ = query.RegisterTyped[*GetUserResult](dispatcher, "GetUser", func(ctx context.Context, q query.Query) (*GetUserResult, error) {
    return &GetUserResult{Name: "Alice"}, nil
})

// Dispatch (type-safe result)
result, err := query.DispatchTyped[*GetUserResult](ctx, dispatcher, query)
```

### Event Creation

**Typed constructor (recommended)**:

```go
evt, err := event.New(
    "user.created",        // event.Type
    userID,                  // id.AggregateID
    "User",                  // event.AggregateType
    event.Version(1),        // event.Version
    UserCreated{Name: "Alice"}, // struct payload (auto-marshaled)
    event.WithCorrelationID(correlationID),
)
```

### Branded IDs

```go
type UserID = id.Of[userMarker]
uid := id.New[UserID]()
parsed, err := id.Parse[UserID](uid.String())
```

## Test Patterns

- Table-driven tests preferred
- BDD tests via Ginkgo v2 + Gomega for event, decider, query
- Use `t.Parallel()` for independent tests
- Test error messages contain context
- Core packages >80% coverage (most >90%)

## Dependencies

### Dependencies

**Production**: oklog/ulid/v2, go-branded-id, go-error-family (core); go-faster/yaml (catalog).
**Test-only**: onsi/ginkgo/v2, onsi/gomega.

**Coverage**: 84–100% across 18 packages. See `docs/status/` for latest.

**Module Graph**: testhelpers→core; memory→core+testhelpers; middleware→core+testhelpers;
catalog→core; storage→core; projection→core; integration→core+memory+testhelpers;
example/user→core+memory+catalog+middleware.

**Integration Tests**: `integration/command/`, `integration/event/`, `integration/query/`.

> **Historical details**: Session milestones, bug fixes, code quality improvements,
> catalog architecture, and known issues extracted to
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).
