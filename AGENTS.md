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
│   ├── types.go                     # Message, Service, Domain, Channel, Schema, ServiceID, DomainID, MessageID, ChannelID, GetID()
│   ├── schema.go                    # SchemaFromType[T]() via reflect
│   ├── registry.go                  # thread-safe Registry, Build() → Catalog
│   ├── asyncapi/                    # AsyncAPI 3.0 YAML/JSON exporter (uses catalog.MessageID)
│   ├── d2/                          # D2 diagram text exporter (uses catalog.MessageID)
│   ├── eventcatalog/                # EventCatalog MDX file generator (uses catalog.MessageID)
│   └── internal/cattest/            # test helpers
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

**Raw constructor (for []byte payloads)**:

```go
evt, err := event.NewEvent(
    "user.created",
    userID,                     // id.AggregateID (branded, no string conversion)
    "User",
    event.Version(1),           // event.Version (typed, not bare int)
    payload,                    // []byte
    event.WithCorrelationID(correlationID),
)
```

### Clock Interface (Deterministic Testing)

```go
fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
clock := func() time.Time { return fixedTime }

evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload,
    event.WithClock(clock), // deterministic OccurredAt
)
// evt.OccurredAt() == fixedTime, every time
```

### Branded IDs

```go
type UserID = id.Of[userMarker]
uid := id.New[UserID]()
parsed, err := id.Parse[UserID](uid.String())
```

### Catalog Builder

```go
builder := catalog.NewBuilder("My Service", "1.0.0")
builder.AddService("users", "User Service", "1.0.0", "Manages users")
builder.AddCommandFromType[CreateUserCmd]("users", "CreateUser", meta)
exporter := asyncapi.NewExporter(builder.Build())
doc, err := exporter.Export(context.Background())
```

## Test Patterns

- Table-driven tests preferred
- BDD tests via Ginkgo v2 + Gomega for event, decider, query
- Use `t.Parallel()` for independent tests
- Test error messages contain context
- Core packages >80% coverage (most >90%)

## Dependencies

### Production

| Dependency        | Version | Purpose                           | Module  |
| ----------------- | ------- | --------------------------------- | ------- |
| `oklog/ulid/v2`   | v2.1.0  | ULID generation (binary-sortable) | core    |
| `go-branded-id`   | v0.1.0  | Branded ID type backing           | core    |
| `go-error-family` | v0.1.0  | Error classification taxonomy     | core    |
| `go-faster/yaml`  | v0.4.6  | YAML marshaling                   | catalog |

### Test-only

| Dependency       | Version | Purpose      | Module            |
| ---------------- | ------- | ------------ | ----------------- |
| `onsi/ginkgo/v2` | v2.28.3 | BDD testing  | core, memory, etc |
| `onsi/gomega`    | v1.40.0 | BDD matchers | core, memory, etc |

## Test Coverage Summary

| Package                       | Coverage |
| ----------------------------- | -------- |
| `core/pkg/dispatcher`         | 100.0%   |
| `core/pkg/id`                 | 100.0%   |
| `middleware`                  | 100.0%   |
| `catalog/internal/caseutil`   | 100.0%   |
| `memory`                      | 99.6%    |
| `core/query`                  | 98.4%    |
| `catalog`                     | 96.8%    |
| `catalog/d2`                  | 95.0%    |
| `catalog/openapi`             | 94.4%    |
| `projection`                  | 94.4%    |
| `core/event`                  | 93.8%    |
| `catalog/asyncapi`            | 93.7%    |
| `core/decider`                | 93.6%    |
| `core/command`                | 92.3%    |
| `testhelpers`                 | 91.3%    |
| `catalog/eventcatalog`        | 91.3%    |
| `catalog/docserver`           | 90.1%    |
| `storage`                     | 89.3%    |
| `catalog/internal/schemautil` | 84.2%    |

## Module Dependency Graph

```
testhelpers → core
memory      → core + testhelpers
middleware  → core + testhelpers
catalog    → core (via cattest internal helpers)
storage    → core (go-sqlmock for tests)
projection → core + memory (tests) + testhelpers (tests)
integration → core + memory + testhelpers
example/user → core + memory + catalog + middleware
core        → (no internal deps — independently publishable)
core/decider is a package within core, not a separate module
```

### Integration Module (`integration/`)

| Package                | Purpose                                                | Key Types              |
| ---------------------- | ------------------------------------------------------ | ---------------------- |
| `integration/command/` | Command integration tests (moved from `core/command/`) | Middleware chain tests |
| `integration/event/`   | Event integration tests (moved from `core/event/`)     | BDD + benchmark tests  |
| `integration/query/`   | Query integration tests (moved from `core/query/`)     | Middleware chain tests |

## Catalog System Architecture

The `catalog` module provides automatic documentation generation from Go CQRS types to AsyncAPI 3.0 and EventCatalog formats.

### Three-Layer Design

```
┌──────────────────────────────────────────────────────┐
│                   catalog (core)                      │
│  types.go — Message, Service, Domain, Channel, Schema │
│  schema.go — SchemaFromType[T]() via reflect          │
│  registry.go — Thread-safe Registry, Build() → Catalog│
└──────────────────────┬───────────────────────────────┘
                       │ Catalog (immutable IR)
           ┌───────────┴───────────┐
           ▼                       ▼
┌─────────────────────┐  ┌─────────────────────────┐
│ catalog/asyncapi/   │  │ catalog/d2/            │  │ catalog/eventcatalog/   │
│ AsyncAPI 3.0 YAML   │  │ MDX files on disk       │
│ Document.MarshalYAML│  │ services/{id}/index.mdx │
│ Document.MarshalJSON│  │ schemas/schema.json     │
└─────────────────────┘  └─────────────────────────┘
```

### Key Design Decisions

1. **go-faster/yaml** — Replaced custom YAML marshaler (`catalog/yaml/`, deleted). Well-maintained, zero-transitive-dep YAML library.

2. **Reflection-based schema generation** — `SchemaFromType[T any]() *Schema` uses `reflect.TypeOf` to inspect struct fields. Reads `json` (name + omitempty), `doc`/`description` (description), and `format` (format) struct tags. **Anonymous (embedded) fields are automatically skipped**.

3. **Type alias for MarshalJSON** — AsyncAPI `Document.MarshalJSON()` uses `type alias Document` to break infinite recursion when calling `json.MarshalIndent`.

4. **Registry pattern** — Thread-safe with `sync.RWMutex`. `AddService` merges messages into existing services. `Build()` produces an immutable `*Catalog`.

5. **AsyncAPI mapping** — Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive`. Channel addresses via `toDotAddress` (CamelCase → dot.separated).

6. **EventCatalog structure** — MDX files with YAML frontmatter (`---` delimited). `schema.json` only created when schema is non-nil. Service frontmatter includes `sends`, `receives`, `commands`, and `queries` lists.

7. **D2 diagram export** (`catalog/d2`) — Generates D2 text from `*catalog.Catalog`. Services become containers, commands/events/queries become color-coded nodes (command=blue, event=red queue, query=purple), domains become grouping labels. Wire via `catalog.Builder` directly. Follows same `Exporter` pattern as `asyncapi` and `eventcatalog`.

## Branded Return Types Migration (Session 3)

Interfaces now return branded types instead of primitives:

| Interface | Method          | Old Return | New Return       |
| --------- | --------------- | ---------- | ---------------- |
| `Event`   | `ID()`          | `string`   | `id.EventID`     |
| `Event`   | `AggregateID()` | `string`   | `id.AggregateID` |
| `Event`   | `Version()`     | `int`      | `event.Version`  |
| `Command` | `AggregateID()` | `string`   | `id.AggregateID` |

**Note**: The `Root` rows were removed when `core/aggregate` was deleted (Session 99).

**Caller updates**: All `event.NewEvent()` calls pass `id.AggregateID` directly (no re-parse). `middleware/logging.go` adds `.String()` when formatting IDs for log output. `Version()` callers use `.Int()` when passing to `NewEvent` or `fmt.Printf`. Commit: `cee6c50` (IDs), `de095e5` (Version)

## Bug Fixes (Sessions 1–2)

| Bug                               | Fix                                                                   | Commit    |
| --------------------------------- | --------------------------------------------------------------------- | --------- |
| Retry dead cancellation           | `context.Background().Done()` → `ctx.Done()` in `middleware/retry.go` | `5ad0356` |
| Aggregate version desync          | Removed fallback loop; `Load()` requires `HistoryLoader`              | `1862eae` |
| Wrong error sentinel (dispatcher) | `CheckClosed` used `ErrHandlerNotFound` → `ErrDispatcherClosed`       | `5ad0356` |
| Slice mutation (MemoryStore)      | `Load()`/`LoadFromVersion()` return defensive copies                  | `d5ea811` |
| Wrong error sentinel (snapshot)   | `CheckClosed` used `ErrSnapshotNotFound` → `ErrSnapshotStoreClosed`   | `8e5150c` |

## Code Quality Improvements (Sessions 1–2)

| Improvement                  | Detail                                                                                                 | Commit     |
| ---------------------------- | ------------------------------------------------------------------------------------------------------ | ---------- |
| Dead code removal            | `evtest.GenerateUUID`, `testutil` package, `query.ErrQueryValidation`                                  | `1862eae`  |
| Lifecycle unification        | `MemoryBus`/`MemorySnapshotStore` now use `LifecycleMixin`                                             | `8e5150c`  |
| EventValidation middleware   | API symmetry: Command/Query/Event all have validation                                                  | `4fdd447`  |
| MessageID extraction         | Moved from `asyncapi`/`eventcatalog` to `catalog.GetID()` (was `MessageID()`, renamed in Session 76)   | `c1bc261`  |
| Type naming overhaul         | `Core`→`ImmutableEvent`/`BasicCommand`/`BasicQuery`, `CatalogEntry`→`HandlerMeta` across all modules   | Session 95 |
| Constructor consistency      | `NewCheckpointStore`→`NewMemoryCheckpointStore`, `NewWithDialect` constructors for all 4 storage types | Session 95 |
| Command/Query decoupling     | command/ and query/ import go-error-family directly instead of event/ for error constructors           | Session 95 |
| Logger interface removal     | Custom `middleware.Logger` + `SlogAdapter` replaced with `*slog.Logger` (Go standard since 1.21)       | Session 95 |
| Dispatch closed-state fix    | command/query Dispatch() now pre-checks closed state, returning domain sentinels consistently          | Session 96 |
| cqrs-htmx removal            | example/todo: removed broken external dep, inlined `chainMiddleware`                                   | Session 96 |
| decider file organization    | Moved `loadFromSnapshot`/`shouldSnapshot` from options.go → load.go                                    | Session 96 |
| FakeMetrics rename           | `TestMetrics` → `FakeMetrics` (consistent with FakeBus/FakeStore/etc.)                                 | Session 96 |
| event_new.go rename          | `codec_typed.go` → `event_new.go` (file contains event constructor, not codec)                         | Session 96 |
| FakeStore completeness       | Added `AppendBatchFn`, `LoadToVersionFn`, `LoadToTimestampFn` setters                                  | Session 96 |
| event.go split               | Extracted `Option`/`With*` to `event/options.go` (169 + 90 lines)                                      | `699d247`  |
| Dead reflect.Ptr case        | Removed unreachable branch in `goTypeToJSON`                                                           | `b23a781`  |
| Dispatcher.Dispatch refactor | Removed unused `handler H` parameter                                                                   | `e84e3a1`  |
| Example rewrite              | `example/user/` demonstrates full CQRS + Decider pattern + middleware + EventCatalog                   | session 37 |

## Known Issues

| Issue                                                    | Severity  | Detail                                                                                                                      |
| -------------------------------------------------------- | --------- | --------------------------------------------------------------------------------------------------------------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW       | Subscribers block publishers (acceptable for test utility)                                                                  |
| `query.Handler` returns `any`                            | LOW       | Violates project "no any" rule; `DispatchTyped[T]` is the workaround. Design doc: `docs/planning/QUERY_HANDLER_GENERICS.md` |
| `CatalogMeta` duplicated across 2 packages               | **FIXED** | Consolidated to `dispatcher.HandlerMeta`; `command.CatalogMeta`/`query.CatalogMeta` deleted                                 |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch     | **FIXED** | `core/aggregate` package deleted (Session 99); decider has no such split                                                    |

## Session History

> Detailed per-session change logs have been extracted to [`docs/sessions/SESSION_HISTORY.md`](docs/sessions/SESSION_HISTORY.md) for brevity. This section previously covered Sessions 20–86.

Key milestones:

| Session | Milestone                                                                                                                                                                                                                                                                |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 20      | Zero lint, storage error path tests 79.8→92.3%                                                                                                                                                                                                                           |
| 25      | Bug fixes: double-marshal, Close() ownership, Version branded type                                                                                                                                                                                                       |
| 27      | No-panic convention: `New*` returns `(*T, error)`                                                                                                                                                                                                                        |
| 31      | Error taxonomy (5 families), ClientID, IdempotencyKey on Command                                                                                                                                                                                                         |
| 37      | `core/decider` package + example/user rewrite                                                                                                                                                                                                                            |
| 44      | ISP (Publisher/Subscriber), extensible error classification                                                                                                                                                                                                              |
| 45      | `id.Of[T]` as type alias for `go-branded-id`                                                                                                                                                                                                                             |
| 48      | Shared SnapshotStrategy, PublishChanges, SaveSnapshot                                                                                                                                                                                                                    |
| 54      | Removed cockroachdb/errors, added TypedHandler[T]                                                                                                                                                                                                                        |
| 55–56   | TransactionalStore, GlobalLoader on SQL, `errors.AsType`                                                                                                                                                                                                                 |
| 65      | Architectural type safety: Version, SchemaVersion, OutboxStatus                                                                                                                                                                                                          |
| 68      | Module hygiene, file splits, 0 production files >250 lines                                                                                                                                                                                                               |
| 73      | File splits, golden test refresh, coverage improvements                                                                                                                                                                                                                  |
| 80      | Time-travel: LoadToVersion, LoadToTimestamp, decider API                                                                                                                                                                                                                 |
| 81      | Position-based replay, SQL composite index                                                                                                                                                                                                                               |
| 83      | Version arithmetic (Add/Sub/Cmp), deprecated API removal                                                                                                                                                                                                                 |
| 86      | Catalog quality sweep, MemorySnapshotStore deep copy                                                                                                                                                                                                                     |
| 89      | API surface reduction: ~60 exports removed, 89.3→92.1% coverage                                                                                                                                                                                                          |
| 90      | Projection builder On[T](), IsReplay, event.New, ExecuteWithResult, DeriveAggregateID, aggregate deprecation                                                                                                                                                             |
| 92      | Query quality: typed bookend docs, example/todo typed handlers + Pagination, design doc closed                                                                                                                                                                           |
| 93      | Zero lint across 10 modules, decider dual-wrap fix, registry deterministic Build, testhelpers 10→64.6%                                                                                                                                                                   |
| 94      | gci v2 fix, buildflow config, orphaned go.mod replace, testhelpers 64.6→80.3%, caseutil 76.5→100%                                                                                                                                                                        |
| 95      | Naming overhaul: Core→ImmutableEvent/BasicCommand/BasicQuery, CatalogEntry→HandlerMeta, NewCheckpointStore→NewMemoryCheckpointStore, command/query decoupled from event, NewWithDialect constructors for all storage types, Go 1.26.3 aligned, InMemoryRunner deprecated |
| 96      | Dispatch() closed-state fix, cqrs-htmx removal, decider file organization, FakeMetrics/FakeStore renames, event_new.go rename                                                                                                                                            |
| 99      | Deleted `core/aggregate` + `integration/aggregate` (~3700 lines), deleted `catalog/adapters` (616 lines), migrated example/user to `catalog.Builder`, added Dispatch() + NewWithDialect tests, storage coverage 88.7→89.3%                                               |
