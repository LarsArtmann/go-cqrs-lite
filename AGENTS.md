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
| Language  | Go 1.26                                                                                                            |
| Modules   | `core` (incl. `decider`), `memory`, `catalog`, `middleware`, `testhelpers`, `integration`, `storage`, `projection` |
| Build     | `nix run .#build`                                                                                                  |
| Test      | `nix run .#test` or see "Testing" below                                                                            |
| Lint      | `nix run .#lint`                                                                                                   |
| Format    | `nix fmt`                                                                                                          |
| Dev shell | `nix develop`                                                                                                      |
| CI        | GitHub Actions: ci.yml (Nix-based)                                                                                 |

## Monorepo Structure

Multi-module Go workspace with 9 modules (10 including example/user demo):

```
go-cqrs-lite/
├── go.work                          # ties modules together
│
├── core/                            # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                       # deps: oklog/ulid, go-branded-id
│   ├── command/                     # command dispatch, handler, catalog
│   ├── query/                       # query dispatch, pagination, catalog
│   ├── event/                       # event types, Store/Bus/SnapshotStore interfaces
│   │   ├── event.go                # Core struct, NewEvent constructor
│   │   └── options.go              # Option func, With* metadata helpers
│   ├── aggregate/                   # Root, Repository, Core, EventSourcedRepository (OO style)
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
│   ├── types.go                     # Message, Service, Domain, Channel, Schema, MessageID()
│   ├── schema.go                    # SchemaFromType[T]() via reflect
│   ├── registry.go                  # thread-safe Registry, Build() → Catalog
│   ├── adapters/                    # CatalogBuilder, FromDispatcher adapters
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
│  │    Event     │  │  Aggregate   │
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

| Package                | Purpose                                   | Key Types                                                                                                                                                                                    |
| ---------------------- | ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core/command/`        | Command dispatch and handling             | `Dispatcher`, `Handler`, `Middleware`, `Command`, `Core`                                                                                                                                     |
| `core/query/`          | Query dispatch with pagination            | `Dispatcher`, `Handler`, `Pagination`, `PaginatedResult[T]`, `Middleware`, `TypedHandler[T]`, `RegisterTyped[T]`                                                                             |
| `core/event/`          | Event sourcing interfaces and types       | `Store`, `Bus`, `Publisher`, `Subscriber`, `SnapshotStore`, `TransactionalStore`, `GlobalLoader`, `Event`, `Core`, `Metadata`, `Option`, `Version`, `SchemaVersion`, `Type`, `AggregateType` |
| `core/aggregate/`      | Aggregate roots and repository (OO)       | `Root`, `Repository`, `Core`, `EventSourcedRepository`                                                                                                                                       |
| `core/decider/`        | Aggregate via pure functions              | `Decider[State]`, `Repository[State]`, `Execute`, `Load`, `DecideFunc`                                                                                                                       |
| `core/pkg/id/`         | Branded IDs (type alias to go-branded-id) | `id.Of[T]` = `cbid.ID[T, ulid.ULID]`, `AggregateID`, `EventID`, `UserID`, `CorrelationID`, `ClientID`, `CompareIDs`, `FromPtr`                                                               |
| `core/pkg/dispatcher/` | Generic internal dispatcher               | `Dispatcher[H, M]`, `MiddlewareChain[H, M]`, `LifecycleMixin`                                                                                                                                |

### Decider Module (`core/decider/`)

| Package         | Purpose                                    | Key Types                                                |
| --------------- | ------------------------------------------ | -------------------------------------------------------- |
| `core/decider/` | Functional aggregate pattern (recommended) | `Decider[State]`, `Repository[State]`, `Execute`, `Load` |

- **Recommended over `aggregate`** for new consumers: pure functions, no mutable state, no 9-method interface, zero-infrastructure testing.
- `Decider[State]` holds `Initial` state and `Fold func(State, Event) (State, error)`.
- `Repository[State].Execute` does load → fold → decide → save → publish.
- `aggregate` package stays for existing consumers who prefer the OO style.

### Memory Module (`memory/`)

| Package   | Purpose                        | Key Types                                         |
| --------- | ------------------------------ | ------------------------------------------------- |
| `memory/` | In-memory test implementations | `MemoryStore`, `MemoryBus`, `MemorySnapshotStore` |

### Catalog Module (`catalog/`)

| Package                 | Purpose                                | Key Types                                                 |
| ----------------------- | -------------------------------------- | --------------------------------------------------------- |
| `catalog/`              | Registry, schema reflection, MessageID | `Registry`, `Catalog`, `SchemaFromType[T]`, `MessageID()` |
| `catalog/adapters/`     | Builder and dispatcher adapters        | `CatalogBuilder`, `FromCommandDispatcher`                 |
| `catalog/asyncapi/`     | AsyncAPI 3.0 YAML/JSON export          | `Exporter`, `Document`, `MarshalYAML`                     |
| `catalog/d2/`           | D2 diagram text export                 | `Exporter`, `Export()`, `NewExporter()`                   |
| `catalog/eventcatalog/` | EventCatalog MDX generator             | `Exporter`                                                |

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
| `TestMetrics`                                   | Metrics collector for testing                 |

### Projection Module (`projection/`)

| Package       | Purpose                                             | Key Types                                                                      |
| ------------- | --------------------------------------------------- | ------------------------------------------------------------------------------ |
| `projection/` | Projection runner with replay and live subscription | `Runner`, `HandlerRegistry`, `NewRunner`, `Register(Projection)`, `WithLogger` |

- **Runner**: Accepts `event.GlobalLoader` (for replay) + `event.Bus` (for live). Register `event.Projection` instances before `Run()`.
- **Per-projection checkpoint**: Each projection tracked by `Name()`. Events past checkpoint are skipped during replay.
- **Wildcard**: `EventTypes() == nil` subscribes to all event types.
- **HandlerRegistry**: Maps event types to handlers. Useful for building custom projection dispatch.

## Design Principles

1. **Library, not framework** — Consumers import what they need, compose their own stack. No opinionated transport (HTTP/gRPC), message broker (Kafka/NATS), or SQL driver. Integration modules (storage, watermill) are optional.
2. **Every module must be trustworthy on its own** — Quality gate: "Would a consumer trust this enough to import it?" Means: tests, stable API, clear docs. Does NOT mean "another module in this repo uses it."
3. **Minimal core dependencies** — core depends on `oklog/ulid`, `go-branded-id`, `go-error-family`
4. **Composition over inheritance** — Per Go best practices
5. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, `Root`, etc.)
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

### Query Handler (with context)

```go
type Handler = func(context.Context, Query) (any, error)

result, err := dispatcher.Dispatch(ctx, query)
typed, err := query.DispatchTyped[User](ctx, dispatcher, query)
```

### Event Creation

```go
evt, err := event.NewEvent(
    "user.created",
    userID,                     // id.AggregateID (branded, no string conversion)
    "User",
    event.Version(1),           // event.Version (typed, not bare int)
    payload,
    event.WithCorrelationID(correlationID),
)
```

### Branded IDs

```go
type UserID = id.Of[userMarker]
uid := id.New[UserID]()
parsed, err := id.Parse[UserID](uid.String())
```

### Catalog Builder

```go
builder := catalogadapters.NewBuilder("My Service", "1.0.0")
builder.AddService("users", "User Service", "1.0.0", "Manages users")
catalogadapters.AddCommandFromType[CreateUserCmd](builder, "users", "CreateUser", meta)
doc, err := builder.ExportAsyncAPI("User Service", "1.0.0")
```

## Test Patterns

- Table-driven tests preferred
- BDD tests via Ginkgo v2 + Gomega for event, aggregate, query
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

| Package                | Coverage |
| ---------------------- | -------- |
| `core/command`         | 100.0%   |
| `core/query`           | 100.0%   |
| `core/pkg/dispatcher`  | 100.0%   |
| `middleware`           | 100.0%   |
| `catalog/adapters`     | 97.1%    |
| `memory`               | 99.5%    |
| `projection`           | 98.3%    |
| `core/pkg/id`          | 97.8%    |
| `catalog/d2`           | 97.6%    |
| `catalog/openapi`      | 96.6%    |
| `core/aggregate`       | 96.9%    |
| `catalog/eventcatalog` | 95.7%    |
| `catalog`              | 95.3%    |
| `core/event`           | 96.3%    |
| `catalog/asyncapi`     | 93.9%    |
| `catalog/docserver`    | 92.3%    |
| `core/decider`         | 92.7%    |
| `storage`              | 88.1%    |

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

| Package                  | Purpose                                                    | Key Types               |
| ------------------------ | ---------------------------------------------------------- | ----------------------- |
| `integration/aggregate/` | Aggregate integration tests (moved from `core/aggregate/`) | BDD + integration tests |
| `integration/command/`   | Command integration tests (moved from `core/command/`)     | Middleware chain tests  |
| `integration/event/`     | Event integration tests (moved from `core/event/`)         | BDD + benchmark tests   |
| `integration/query/`     | Query integration tests (moved from `core/query/`)         | Middleware chain tests  |

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

7. **Catalog adapters** (`catalog/adapters`) — `CatalogBuilder` provides instance-based methods (`AddCommand`, `AddEvent`, `AddQuery`) and generic zero-instance methods (`AddCommandFromType[T]`, `AddEventFromType[T]`, `AddQueryFromType[T]`). Generic methods use `SchemaFromType[T]()` for compile-time safety. `FromCommandDispatcher` and `FromQueryDispatcher` extract catalog entries from dispatchers.

8. **D2 diagram export** (`catalog/d2`) — Generates D2 text from `*catalog.Catalog`. Services become containers, commands/events/queries become color-coded nodes (command=blue, event=red queue, query=purple), domains become grouping labels. Wire via `CatalogBuilder.ExportD2(title, version)`. Follows same `Exporter` pattern as `asyncapi` and `eventcatalog`.

## Branded Return Types Migration (Session 3)

Interfaces now return branded types instead of primitives:

| Interface | Method          | Old Return | New Return       |
| --------- | --------------- | ---------- | ---------------- |
| `Event`   | `ID()`          | `string`   | `id.EventID`     |
| `Event`   | `AggregateID()` | `string`   | `id.AggregateID` |
| `Event`   | `Version()`     | `int`      | `event.Version`  |
| `Root`    | `ID()`          | `string`   | `id.AggregateID` |
| `Root`    | `Version()`     | `int`      | `event.Version`  |
| `Command` | `AggregateID()` | `string`   | `id.AggregateID` |

**Caller updates**: All `event.NewEvent()` calls pass `id.AggregateID` directly (no re-parse). All `cmd.AggregateID()` and `root.ID()` comparisons use branded types. `repository.go` eliminated redundant `id.ParseAggregateID()` re-parses. `middleware/logging.go` adds `.String()` when formatting IDs for log output. `Version()` callers use `.Int()` when passing to `NewEvent` or `fmt.Printf`. Commit: `cee6c50` (IDs), `de095e5` (Version)

## Bug Fixes (Sessions 1–2)

| Bug                               | Fix                                                                   | Commit    |
| --------------------------------- | --------------------------------------------------------------------- | --------- |
| Retry dead cancellation           | `context.Background().Done()` → `ctx.Done()` in `middleware/retry.go` | `5ad0356` |
| Aggregate version desync          | Removed fallback loop; `Load()` requires `HistoryLoader`              | `1862eae` |
| Wrong error sentinel (dispatcher) | `CheckClosed` used `ErrHandlerNotFound` → `ErrDispatcherClosed`       | `5ad0356` |
| Slice mutation (MemoryStore)      | `Load()`/`LoadFromVersion()` return defensive copies                  | `d5ea811` |
| Wrong error sentinel (snapshot)   | `CheckClosed` used `ErrSnapshotNotFound` → `ErrSnapshotStoreClosed`   | `8e5150c` |

## Code Quality Improvements (Sessions 1–2)

| Improvement                  | Detail                                                                               | Commit     |
| ---------------------------- | ------------------------------------------------------------------------------------ | ---------- |
| Dead code removal            | `evtest.GenerateUUID`, `testutil` package, `query.ErrQueryValidation`                | `1862eae`  |
| Lifecycle unification        | `MemoryBus`/`MemorySnapshotStore` now use `LifecycleMixin`                           | `8e5150c`  |
| EventValidation middleware   | API symmetry: Command/Query/Event all have validation                                | `4fdd447`  |
| MessageID extraction         | Moved from `asyncapi`/`eventcatalog` to `catalog.MessageID()`                        | `c1bc261`  |
| event.go split               | Extracted `Option`/`With*` to `event/options.go` (169 + 90 lines)                    | `699d247`  |
| Dead reflect.Ptr case        | Removed unreachable branch in `goTypeToJSON`                                         | `b23a781`  |
| Dispatcher.Dispatch refactor | Removed unused `handler H` parameter                                                 | `e84e3a1`  |
| Example rewrite              | `example/user/` demonstrates full CQRS + Decider pattern + middleware + EventCatalog | session 37 |

## Known Issues

| Issue                                                    | Severity | Detail                                                                                                                      |
| -------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW      | Subscribers block publishers (acceptable for test utility)                                                                  |
| `query.Handler` returns `any`                            | LOW      | Violates project "no any" rule; `DispatchTyped[T]` is the workaround. Design doc: `docs/planning/QUERY_HANDLER_GENERICS.md` |
| `CatalogMeta` duplicated across 3 packages               | LOW      | `event.CatalogMeta`, `command.CatalogMeta`, `query.CatalogMeta` — nearly identical                                          |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch     | LOW      | Every aggregate must implement `LoadEvents` and delegate to `LoadFromHistory`                                               |

- **Session 28 (Branching-Flow Context Review)**:
  - **CRITICAL FIX**: `repository.loadEvents` now propagates non-`ErrSnapshotNotFound` snapshot errors instead of silently discarding them. Genuine DB errors are no longer masked.
  - **HIGH FIX**: `SQLEventStore.Load` now returns `event.ErrAggregateNotFound` for empty result sets, consistent with `MemoryStore.Load`.
  - **HIGH FIX**: `SQLSnapshotStore.Load` now translates `sql.ErrNoRows` to `event.ErrSnapshotNotFound`, consistent with `MemorySnapshotStore.Load`.
  - **MEDIUM FIX**: `HandleParallel` now respects context cancellation — returns context error when canceled mid-processing.
  - **DOCUMENTED**: `Save` partial-failure contract (events persisted but unpublished on bus/outbox failure).
  - **DOCUMENTED**: `publishPending` silently swallows errors in background loop; use `PublishNow` for error visibility.
  - **DOCUMENTED**: `MemoryBus` handler ordering (all-handlers before type-specific) and partial-publish semantics.
  - **DOCUMENTED**: `NewRepository` requires non-nil store/bus (undefined behavior if nil).
  - Removed `Load() empty semantics differ` from Known Issues (now fixed).
  - Zero lint, all tests pass across all modules

- **Session 29 (Honest Quality Self-Assessment)**:
  - **BREAKING**: `projection.NewRunner` returns `(*Runner, error)` instead of panicking on nil deps. Added `ErrNilStore`, `ErrNilBus`, `ErrNilCheckpoint` sentinels.
  - **Cleanup**: Removed stale `memory` replace directives from `testhelpers/go.mod` and `middleware/go.mod`.
  - **Interface checks**: Added compile-time `var _ Interface = (*Impl)(nil)` for `JSONCodec`, `slogLogger`, `FakeCheckpointStore`.
  - **Sentinel errors**: Extracted `ErrHandlerNil` to `memory/errors.go`, `ErrAlreadyStarted` to `core/event/errors.go`. Tests updated to `errors.Is`.
  - **Godoc**: Added doc comments to all exported types in `catalog/types.go` and exported functions in `catalog/schema.go`.
  - **Documented**: `samber/ro` forces `[]any` for `Pipe` operator pipeline — acceptable exception to "no any" rule.
  - All tests pass across all modules

- **Session 27 (No-Panic Convention + Code Quality)**:
  - **BREAKING**: `NewInMemoryRunner` returns `(*InMemoryRunner, error)` instead of panicking on nil checkpoint. Added `ErrNilCheckpointStore` sentinel.
  - **BREAKING**: `NewOutboxPublisher` returns `(*OutboxPublisher, error)` instead of panicking on nil outbox/bus. Added `ErrNilOutbox`, `ErrNilBus` sentinels.
  - **BREAKING**: `NewCore` returns `(*Core, error)` with validation for zero ID/empty type. Added `MustNewCore` helper. Added `ErrNilAggregateID`, `ErrEmptyAggregateType` sentinels.
  - **BREAKING**: `Bus` interface now includes `Use(middleware ...Middleware) error`. Updated `FakeBus` and test stubs.
  - **Fix**: `SQLSnapshotStore.LoadAtVersion` now returns snapshot at or before version (was exact match). Matches interface contract and MemorySnapshotStore behavior.
  - **Fix**: `catalog/go.mod` stale replace directives caused 33+ gopls errors.
  - **Fix**: `TestSQLEventStore_Close` was false-positive (`ExpectClose` never fulfilled).
  - **Fix**: `FEATURES.md` stale entries — removed implemented items from "Not Yet Implemented", corrected Bus.Use and Close() claims.
  - **Fix**: D2 exporter field reference `e.Description` → `e.description` after unexporting.
  - **Code quality**: Added doc comments to `event/catalog.go` (CatalogMeta, Catalogable, etc.). Added `MustNewCatalogCore` to event for consistency. Added compile-time interface checks for `ProjectionFunc`, `UpcasterFunc`, `CatalogCore` across all 3 packages. Replaced custom `contains()` helper with `strings.Contains` in outbox publisher tests.
  - Zero lint, all 18 test packages pass

- **Session 20 (Lint Fix + Comprehensive Plan Execution)**:
  - **Lint cleanup**: Fixed 48 lint issues → 0. Added `go-sqlmock` to depguard, `tx`/`ts` to varnamelen ignores, `example/` + `testhelpers/fakes.go` exclusions. Fixed `noinlineerr` (17), `err113` (1), `wrapcheck` (2), `godoclint` (1).
  - **Storage error path tests**: 79.8% → 92.3% (13 → 31 tests). Added BeginTx/query/insert/commit errors, scanEvents parse/scan errors, DDL test, SQL injection safety test.
  - **Code quality**: Extracted `validateEventParams` from `NewEvent` (66→25 lines). Extracted `addChannel`/`addOperation` from `addMessage` (55→25 lines).
  - **toDotAddress**: Fixed digit handling — digits after letters get dot separator, consecutive digits stay grouped.
  - **UpcasterRegistry**: Added cycle detection (visited version tracking).
  - **Documentation**: Documented `InMemoryRunner` fail-fast, `MemoryBus.Publish` RLock. Updated FEATURES.md storage status (BROKEN → PARTIALLY_FUNCTIONAL). Pruned 6 stale TODO_LIST.md items.
  - **Housekeeping**: Deleted orphaned `/user` binary. Added `/user` to `.gitignore`. Documented `storage.Close()` ownership contract.
  - Zero lint, all tests pass across all 9 modules

- **Session 25 (Bug Fixes + Type Safety)**:
  - **CRITICAL FIX**: `SQLSnapshotStore.Save()` double-marshal — `json.Marshal(snap.State)` stored double-encoded JSON (`State` is already `[]byte`). Now stores `snap.State` directly.
  - **CRITICAL FIX**: `storage.Close()` ownership — all 3 SQL stores now return nil (no-op). `*sql.DB` is borrowed, not owned. Previously `Close()` on one store broke others sharing the same DB.
  - **DESIGN FIX**: `CheckpointStore` interface now embeds `io.Closer`, consistent with all other store interfaces (`Store`, `Bus`, `SnapshotStore`, `Outbox`).
  - **BREAKING**: `Event.Version()` and `Root.Version()` return `event.Version` instead of `int`. Consistent with branded ID types. Callers needing `int` use `.Int()`. `SnapshotStrategy.ShouldSnapshot()` parameter also changed.
  - **Tests**: Added `storage/snapshot_test.go` (11 tests) and `storage/checkpoint_test.go` (8 tests) with go-sqlmock.
  - Zero lint, all tests pass across all 9 modules

- **Session 30 (Architecture Roadmap Planning)**:
  - **Research synthesis**: Reviewed 11 planning/research documents across go-cqrs-lite and go-localfirst projects, including error handling brainstorm (9 families, 74 issues), offline-first dimensions (15 topics), hybrid architecture proposal, and 13 innovative CQRS/ES projects.
  - **Roadmap created**: `docs/planning/2026-05-01_ARCHITECTURE_ROADMAP.md` — 5-phase plan with 3 initiatives (Error Taxonomy in library, Offline-First Primitives, Error Handling in go-localfirst).
  - **Key decisions**:
    - Error taxonomy: 5 families in go-cqrs-lite (Rejection, Conflict, Transient, Corruption, Infrastructure), 4 more in go-localfirst (Staleness, Divergence, Pipeline, Transport)
    - `core/pkg/errors/` package for library-level error classification
    - `IdempotencyKey()` added to Command interface (breaking, with BaseCommand migration helper)
    - Client metadata options: `WithClientID`, `WithClientOccurredAt`, `WithClientTimezone`, `WithCausationID`
    - Single write path in go-localfirst (sync operations become commands)
    - Dual bus pattern (event bus + notification bus) for error broadcasting
  - **Explicit non-goals documented**: No sync protocol, no client-side store, no event signing, no WASM/mobile SDK
  - No code changes this session — planning only

- **Session 31 (Error Taxonomy + Offline-First Primitives)**:
  - **NEW**: `event.Family` enum (Rejection, Conflict, Transient, Corruption, Infrastructure) in `core/event/errors.go`
  - **NEW**: `event.Error` struct with Code, Message, Family, cause — extractable via `errors.As`
  - **NEW**: `event.Classify(err) Family` — maps sentinels to families, defaults to Transient
  - **NEW**: `event.IsRetryable(err) bool` — returns true for Transient family
  - **NEW**: Constructors: `NewRejection`, `NewConflict`, `NewTransient`, `NewCorruption`, `NewInfrastructure`
  - **NEW**: `id.ClientID` branded type in `core/pkg/id/client_id.go`
  - **NEW**: `event.WithClientID(id.ClientID)` and `event.WithClientOccurredAt(time.Time)` options
  - **BREAKING**: `command.Command` interface now requires `IdempotencyKey() string`
  - **FIX**: `projection/runner.go` — stopped silently discarding handler errors (`_ =` → proper error handling with retry)
  - **ENHANCEMENT**: `projection/runner.go` — wired `WithRetry` option with exponential backoff using `event.IsRetryable()`
  - **ENHANCEMENT**: `middleware.DefaultRetryConfig()` — `IsRetryable` now defaults to `event.IsRetryable` (was: always false)
  - **DOCS**: `docs/OFFLINE_FIRST_METADATA.md` — convention-based metadata keys for offline-first
  - **DOCS**: `docs/planning/2026-05-01_EXECUTION_PLAN.md` — comprehensive task list with effort/impact estimates
  - Zero lint, all tests pass across all modules

- **Session 32 (Test Coverage + Type Quality + Cleanup)**:
  - **FIX**: `Classify(nil)` returns `Rejection` (was `Transient`), `IsRetryable(nil)` returns `false`
  - **FIX**: `ErrDuplicateProjection` classified as `Conflict` in `Classify()`
  - **NEW**: `event.Error` implements `fmt.Formatter` — `%+v` shows `family:code: message` with cause chain
  - **NEW**: `event.Version.String()` — returns decimal representation
  - **TEST**: Projection retry: `TestRunner_RetryOnTransientError`, `TestRunner_NoRetryOnNonRetryableError`
  - **TEST**: `TestWithClientID`, `TestWithClientOccurredAt`, `TestClientID`, `MustParseClientID` panic test
  - **DOCS**: `RetryConfig.IsRetryable` field documented
  - **CLEANUP**: Removed stale "FakeStore/MemoryStore key separator mismatch" Known Issue
  - **KNOWN ISSUE**: Cross-package sentinels (aggregate, projection, storage) not classified — circular dependency. Documented in `Classify()` doc comment.
  - **KNOWN ISSUE**: `WithBatchSize`/`WithBatchWindow`/`WithConcurrency` options set fields but runner never reads them — dead API surface.
  - Zero lint, all 20 test packages pass

- **Session 34 (World-Class Library Cleanup)**:
  - **BREAKING**: Removed dead public API — `WithBatchSize`, `WithBatchWindow`, `WithConcurrency` (3 options that silently did nothing)
  - **BREAKING**: Removed 5 unused error sentinels from `projection/errors.go` (`ErrRunnerStopped`, `ErrDuplicateHandler`, `ErrCheckpointLoad`, `ErrStoreLoad`, `ErrNilStore`)
  - **BREAKING**: Removed unused `FakeCheckpointStore` from `testhelpers/`
  - **DOCS**: Added godoc to 57 exported symbols across 8 files: `projection/runner.go`, `projection/errors.go`, `memory/bus.go`, `memory/store.go`, `memory/snapshot.go`, `core/aggregate/errors.go`, `catalog/eventcatalog/exporter.go`, `catalog/asyncapi/types.go`
  - **NEW**: `String()` on `event.Type`, `event.AggregateType`, `command.Type`, `query.Type`
  - **NEW**: `*event.Error.Is(error) bool` — matches by Code+Family for `errors.Is` support
  - **NEW**: Compile-time `var _ io.Closer` for `*projection.Runner`, `*event.OutboxPublisher`, `*command.Dispatcher`, `*query.Dispatcher`
  - **RESOLVED**: "WithBatchSize/WithBatchWindow/WithConcurrency dead API surface" Known Issue — removed
  - Zero lint, all 21 test packages pass

- **Session 36 (Continuation — Cleanup + Audit)**:
  - **TEST**: `command.Core.IdempotencyKey()` — default returns `""`, embed override test
  - **REFACTOR**: Split `testhelpers/helpers.go` (293 lines) into 3 files: `handlers.go` (131), `event_helpers.go` (58), `assertions.go` (113)
  - **REFACTOR**: Trimmed `aggregate/repository.go` from 254 → 244 lines (extracted `publishChanges` helper)
  - **QUALITY**: Added compile-time `var _` interface checks for `event.Core→Event`, `command.Core→Command`, `query.Core→Query`
  - **QUALITY**: Added godoc to 4 `*event.Error` methods (`Error`, `Unwrap`, `Is`, `Format`)
  - **AUDIT**: Full codebase audit — all files ≤250 lines, zero TODO/FIXME, all exported errors use correct patterns
  - Zero lint, all 21 test packages pass

- **Session 37 (Decider Package + Example Rewrite)**:
  - **NEW**: `core/decider` package — functional aggregate pattern using pure functions
  - **NEW**: `decider.Decider[State]` — holds Initial state + Fold function (pure)
  - **NEW**: `decider.Repository[State]` — wraps event.Store + event.Bus, provides Execute (load→fold→decide→save→publish) and Load (read-only)
  - **NEW**: `decider.DecideFunc[State]` — `func(state State, version event.Version) ([]event.Event, error)`
  - **NEW**: Sentinel errors: `ErrNilStore`, `ErrNilBus`, `ErrNilFold`, `ErrLoadFailed`, `ErrFoldFailed`, `ErrSaveFailed`
  - **DESIGN**: `aggregate` package stays for existing consumers; `decider` is recommended for new consumers
  - **DESIGN**: `Execute` handles `ErrAggregateNotFound` as empty stream (new aggregate, version 0)
  - **TEST**: 11 tests covering create, update, decide error, fold error, save error, publish error, no events, load, nil checks
  - **EXAMPLE**: Complete rewrite of `example/user/` — 10 files, full CQRS stack using Decider pattern
  - **EXAMPLE**: Demonstrates commands, events, projections, queries, middleware chain, error classification, EventCatalog
  - **EXAMPLE**: No split-brain types, no `log.Fatalf` in helpers, uses library's `SlogAdapter`
  - **EXAMPLE**: README with architecture diagram, file structure, and pattern explanations
  - Zero lint, all 22 test packages pass

- **Session 38 (Deep Cleanup + Deduplication + Type Safety)**:
  - **FIX**: `EveryNEvents` now validates `n > 0` (prevents division-by-zero)
  - **FIX**: `FakeOutbox.Ack` now respects IDs parameter (was clearing all entries)
  - **FIX**: Golden tests use `strings.TrimSpace` for comparison (prevents trailing-newline drift)
  - **REFACTOR**: Extracted `insertEvents` helper in `storage/event_store.go` (removes duplicated loop from Save/AppendBatch)
  - **REFACTOR**: Extracted `pollPublishAck` in `core/event/outbox_publisher.go` (publishPending/PublishNow now share one cycle)
  - **REFACTOR**: Exported `event.SubscribesTo` and removed duplicated `subscribesTo` from `projection/runner.go`
  - **REFACTOR**: Removed dead `dispatcher.Typed` interface (no production references)
  - **REFACTOR**: Replaced 2 dynamic errors with sentinels (`query.ErrEmptyQueryType`, `catalog.ErrNilSchema`)
  - **API**: `projection/HandlerRegistry.On` now accepts `event.Type` instead of `string`
  - Zero lint, all 21 test packages pass

- **Session 39 (Deep Cleanup + Dead Code Removal)**:
  - **CHORE**: Removed stale `.golangci-lint.yml` (minimal 12-linter config replaced by comprehensive 60+ linter `.golangci.yml`)
  - **CHORE**: Removed 6 dead `//nolint:ireturn` directives (ireturn linter is not enabled)
  - **REFACTOR(middleware)**: Replaced `crypto/rand` with `math/rand/v2` for jitter — simpler, faster, no modulo bias
  - **REFACTOR(testhelpers)**: Extracted `fakeStreamKey` helper — deduplicated 5× inline `string(aggregateType) + ":" + aggregateID.String()`
  - **REFACTOR(catalog)**: Added `catalog.ErrDomainNotFound` sentinel, replaced 2× `fmt.Errorf("domain %q not found")` with `errors.Is`-compatible sentinel
  - **FIX(aggregate)**: `MustNewCore` panic now includes context prefix (`"aggregate.MustNewCore: ..."`), consistent with all other `Must*` helpers
  - **REFACTOR(catalog/d2)**: Simplified `sanitizeID` to single-pass `strings.Map`; removed redundant `Sends` case in action switch
  - **FIX(testhelpers)**: `FakeOutbox` uses monotonic counter for IDs (was `len(Entries)`, produced duplicates after `Ack`)
  - Net -69 lines across 11 files, zero lint, all 21 test packages pass

- **Session 43 (Bug Fixes + Test Quality + Coverage)**:
  - **FIX(event)**: `HandleParallel` goroutine now has panic recovery — prevents deadlock from panicking projection handlers
  - **FIX(event)**: `OutboxPublisher.run()` goroutine now has panic recovery — prevents process crash from panicked publish cycle
  - **FIX(aggregate)**: `TestCoreDoesNotImplementRootDirectly` now has proper assertions (was zero-assertion always-passing test)
  - **FIX(event)**: `TestOutboxPublisher_PublishNow_ContextCanceled` now asserts error (stub also made context-aware)
  - **FIX(testhelpers)**: `FakeOutbox` uses `sync.RWMutex` + `RLock()` for `PollPending` (was exclusive `Lock()` for read-only)
  - **FIX(testhelpers)**: `FakeStore.Save` reads `saveFn` under `RLock` (was data race with `SaveFn()`)
  - **FIX(testhelpers)**: All `FakeStore`/`FakeOutbox` methods use `defer` unlock (was manual unlock without defer)
  - **REFACTOR(decider)**: Split `decider.go` (292→243 lines) by extracting `loadFromSnapshot` to `options.go`
  - **TEST(decider)**: Coverage 77.4%→94.3% — added 8 tests: snapshot decode error, store load error, fold error during replay, save snapshot error, Delete error, EveryNEvents validation, snapshot+events after, nil snapshot fallback
  - **TEST(projection)**: Replaced all 9× `time.Sleep` with channel-based sync via `subscribeSignalBus` wrapper — no flaky CI timing
  - **TEST(memory)**: Added concurrent access tests for `MemoryStore` (10 goroutines × 50 saves + 50 loads) and `MemoryBus` (5 publishers × 20 events) — passes with `-race`
  - Zero lint, all 20 test packages pass

- **Session 44 (Architecture: ISP + Error Classification + DI + Lint)**:
  - **NEW**: `event.Publisher` and `event.Subscriber` sub-interfaces — `event.Bus` composes both. Non-breaking ISP improvement. Repositories accept `Publisher`, projections accept `Subscriber`.
  - **NEW**: `RegisterClassification(sentinel, family)` — extensible error classification. External packages register sentinels via `init()` without circular deps. `Classify()` checks registered map.
  - **NEW**: `command` and `query` packages register their sentinels (`ErrHandlerNotFound` → Rejection, `ErrDispatcherClosed` → Infrastructure, etc.)
  - **NEW**: `WithLogger(*slog.Logger)` option for `projection.Runner` — replaces global `slog.Default()`. DI over globals.
  - **STYLE**: `errors.As` → `errors.AsType` in `example/user/main.go` (Go 1.26 API)
  - **LINT**: Resolved `gochecknoglobals` (inline struct initialization for classifier), `gochecknoinits` (nolint directives for registration pattern), `wsl_v5` in test file
  - **SKIPPED**: `CatalogMeta` consolidation — `event.CatalogMeta` has extra `AggregateType` field; no clean shared location
  - Zero lint (all 46 remaining issues are pre-existing), all 21 test packages pass

- **Session 45 (go-branded-id Integration: Type Alias)**:
  - **REFACTOR**: `id.Of[T]` changed from wrapper struct to type alias `cbid.ID[T, ulid.ULID]` — eliminates all delegation boilerplate
  - **DELETED**: `core/pkg/id/id_encoding.go` (32 lines) — all encoding (JSON, SQL, Text, Binary, Gob) now inherited from `go-branded-id`
  - **REMOVED**: Delegated methods from `id.go` — `IsZero`, `Equal`, `Or`, `Reset`, `Get`, `String`, `GoString`, `Format`, `Ptr` all inherited from `cbid.ID`
  - **NEW**: `CompareIDs[T](a, b Of[T]) int` — replaces `Compare()` method (cbid.ID.Compare returns ErrNotOrdered for ulid.ULID)
  - **RE-EXPORT**: `FromPtr[T]` — delegates to `cbid.FromPtr` (package-level functions not promoted by type alias)
  - Net -89 lines (141→84 in id.go, id_encoding.go deleted), all 21 test packages pass

- **Session 46 (Deep Duplication Audit + Status Report)**:
  - **AUDIT**: Identified 5 duplication targets: SnapshotStrategy (aggregate↔decider), publishChanges (aggregate↔decider), saveSnapshot (aggregate↔decider), ISP (Bus used everywhere), error classification gaps
  - **STATUS**: 21 test packages pass, 90.9% total coverage, 46 pre-existing lint issues, 0 TODOs
  - **PLANNING**: Prioritized deduplication and ISP as highest-impact, lowest-effort improvements

- **Session 47 (Execution Plan + Status Report)**:
  - **PLAN**: 66-task, 9-phase execution plan — Phases 1-7 (implementation), Phase 8 (docs), Phase 9 (future-looking)
  - **STATUS**: 21 packages pass, 31,509 LOC (10,067 production + 21,442 test), 154 commits since May 1
  - **PHASES**: (1) SnapshotStrategy extraction, (2) ISP activation, (3) error classification completion, (4) zero lint, (5) test coverage gaps, (6) code quality cleanup, (7) deduplicate publishChanges/saveSnapshot, (8) documentation, (9) future-looking

- **Session 48 (Execution: Phases 1–7)**:
  - **NEW**: `core/event/snapshot_strategy.go` — canonical `SnapshotStrategy` interface, `EveryNEvents(n)`, shared `ShouldSnapshot()` helper
  - **NEW**: `event.Publisher` and `event.Subscriber` sub-interfaces — `event.Bus` composes both. Backward-compatible ISP.
  - **NEW**: `event.RegisterClassification(sentinel, family)` — extensible error classification; `aggregate`, `projection`, `storage` register via `init()`
  - **NEW**: `event.ErrProjectionPanicked` sentinel for panic recovery in `HandleParallel` and `OutboxPublisher`
  - **NEW**: `event.PublishChanges()` and `event.SaveSnapshot()` — shared helpers eliminating duplication in aggregate/decider repositories
  - **NEW**: Cross-module classification tests in `integration/event/classify_test.go`
  - **NEW**: Coverage tests for memory (99.1%), projection (92.5%), storage (93.6%), aggregate (95.8%)
  - **FIX**: Root `go.mod` module path `LarsArtmann` → `larsartmann` (consistent with sub-modules)
  - **FIX**: All 50+ lint issues → 0 across 8 linted modules (exhaustruct, wrapcheck, noinlineerr, gosec, goconst, prealloc, tagliatelle, fatcontext)
  - **FIX**: `storage/snapshot.go` scanSignature fills all struct fields, wraps `sql.Row.Scan` error
  - **FIX**: `storage/outbox.go` extracted `pollPendingQuery` constant, nolint for G201/tagliatelle
  - **REFACTOR**: `aggregate.Repository` accepts `event.Publisher`, `decider.Repository` accepts `event.Publisher`, `projection.Runner` accepts `event.Subscriber`
  - **REFACTOR**: Deleted duplicate `publishChanges` and `saveSnapshot` from aggregate/decider repositories; replaced with shared `event.PublishChanges()` / `event.SaveSnapshot()`
  - **REFACTOR**: Type aliases for `SnapshotStrategy` in aggregate and decider (backward-compatible)
  - 6 commits (7437986, d28d03d, 09bbbba, 57b3939, 6f8d8f6, 7bc841b), zero lint, all 22 test packages pass

- **Session 50 (Documentation Fixes + Benchmarks + Design Docs)**:
  - **FIX**: TODO_LIST.md — corrected false "zero benchmarks exist" claim (26 benchmarks existed in 7 files)
  - **FIX**: FEATURES.md — corrected 9 stale coverage numbers, added `core/decider` to Module Maturity Matrix, added ISP Publisher row to Aggregate features, updated "Last audited" date to 2026-05-03
  - **FIX**: CHANGELOG.md — merged duplicate `### Changed` sections under `[Unreleased]`
  - **NEW**: `core/decider/benchmark_test.go` — 4 benchmarks (Execute, Execute_Update, Load, Fold)
  - **NEW**: `projection/benchmark_test.go` — 3 benchmarks (Register, NewRunner, CurrentCheckpoint)
  - **FIX**: Replaced deadlocking `BenchmarkRunner_Replay` (from Session 49) with non-blocking benchmarks
  - **FIX**: Removed unused `benchEvent` helper from projection benchmark
  - **DOCS**: `docs/planning/OUTBOX_TRANSACTION_API.md` — TransactionalStore interface design for atomic save+outbox
  - **DOCS**: `docs/planning/QUERY_HANDLER_GENERICS.md` — TypedHandler[T] migration plan
  - **DOCS**: `docs/planning/SAGA_DESIGN.md` — added answers to open questions, integration with existing types, 4-phase implementation plan (18h estimate)
  - **NEW**: `core/event/benchmark_test.go` — 6 benchmarks (NewEvent, NewEvent_WithOptions, Classify, IsRetryable, PublishChanges, DecodePayload)
  - **NEW**: `middleware/benchmark_test.go` — 4 benchmarks (CommandLogging, CommandRecovery, CommandValidation, CommandRetry)
  - **DOCS**: `docs/planning/2026-05-04_05-54-SESSION_50_EXECUTION_PLAN.md` — comprehensive Pareto-based execution plan with mermaid graph
  - **INVESTIGATED**: `memory/go.mod` and `projection/go.mod` ginkgo/gomega warnings — already direct deps, gopls workspace false positive
  - Total 43 benchmarks across 12 files, zero lint, all 22 test packages pass

- **Session 51 (Error Sentinel Audit + EveryNEvents + Status Report)**:
  - **BREAKING**: `event.EveryNEvents` returns `(SnapshotStrategy, error)` instead of `SnapshotStrategy`. Added `MustEveryNEvents` for panic behavior.
  - **NEW**: `ErrInvalidSnapshotInterval` sentinel — classified as `Rejection`
  - **NEW**: `ErrEmptyEventType`, `ErrNilAggregateID`, `ErrEmptyAggregateType` sentinels in `event` — classified as `Rejection`
  - **NEW**: `ErrEmptyCommandType`, `ErrNilAggregateID` sentinels in `command` — classified as `Rejection`
  - **NEW**: `core/decider/errors.go` — extracted 6 sentinels from `decider.go`, registered all via `init()` with `RegisterClassification`
  - **NEW**: `ErrProjectionPanicked` classified as `Corruption` in `Classify()`
  - **REFACTOR**: All validation errors in `event.NewEvent` and `command.New` now wrap sentinels with `%w` — callers can use `errors.Is()`
  - **REFACTOR**: Split `event/errors.go` (259→51 lines) into `errors.go` (sentinels) + `errors_taxonomy.go` (Family, Error, Classify, RegisterClassification, 211 lines)
  - **REFACTOR**: Updated aggregate/decider `EveryNEvents` aliases to `MustEveryNEvents` for backward compat
  - **TEST**: Added `core/event/snapshot_strategy_test.go` (5 tests) for EveryNEvents error/MustEveryNEvents
  - **TEST**: Updated `TestNewEvent_InvalidInputErrors` and command tests to assert `errors.Is()` for sentinels
  - **DOCS**: `docs/status/2026-05-02_SESSION_51_COMPREHENSIVE_STATUS.md`
  - 38 sentinel errors across 7 modules, all classified. Zero lint, all 22 test packages pass

- **Session 52 (Code Quality: No-Panic Convention + Interface Checks + Outbox Safety)**:
  - **FIX**: Renamed `newCatalogEvent` → `mustNewCatalogEvent` in `example/user/catalog.go` (no-panic convention)
  - **NEW**: Compile-time `var _ SnapshotStrategy = (*everyN)(nil)` interface check
  - **PERF**: Replaced `fmt.Sprintf` with `strconv.Itoa` in `event.Version.String()` and `storage/outbox.go`
  - **FIX**: Added batch chunking to `storage/outbox.Ack()` (max 500 IDs per DELETE) to prevent PostgreSQL parameter overflow
  - **REFACTOR**: Extracted `trySnapshot` from `aggregate.Save`, `saveSnapshotAfterEvents` from `decider.Execute`
  - **DOCS**: Added godoc to 14 exported symbols in `catalog/asyncapi` (9) and `catalog/types` (5)
  - Zero lint, all 22 test packages pass

- **Session 53 (Godoc Completion + Deduplication + Coverage Accuracy)**:
  - **DOCS**: Added godoc to 14 exported symbols in `catalog/d2` (5) and `catalog/adapters` (8)
  - **REFACTOR**: Extracted `reconstructEvent` from `storage/helpers.go:scanEvent` (58→28 lines). Reused in `storage/outbox.go:reconstructOutboxEvent` (33→8 lines). Removed unused `id` import from outbox.
  - **FIX**: Updated stale benchmark count in `TODO_LIST.md` (33→43, middleware+event now covered)
  - **FIX**: Updated coverage numbers in `FEATURES.md` to match actual (event 93.6→94.4%, aggregate 95.3→95.5%, decider 95.6→95.0%, storage 93.6→94.8%)
  - Total 91.6% coverage, zero lint, all 22 test packages pass

- **Session 54 (Sentinel Errors + Dependency Elimination + TypedHandler)**:
  - **NEW**: `middleware/errors.go` — 4 sentinel errors: `ErrValidationFailed`, `ErrRetryExhausted`, `ErrRetryCanceled`, `ErrPanicRecovered`
  - **NEW**: `query.TypedHandler[T any]` type and `RegisterTyped[T]()` function — compile-time type-safe query handler registration
  - **BREAKING**: Removed `cockroachdb/errors` dependency from all modules. `errors.Wrap`/`errors.Wrapf` → `fmt.Errorf` with `%w`. No API changes — all sentinel errors are stdlib `errors.New`. Removed 6 transitive deps (cockroachdb/errors, logtags, redact, getsentry/sentry-go, gogo/protobuf, pkg/errors).
  - **BREAKING**: Removed `go-json-experiment/json` dependency from core and storage. Replaced with `encoding/json`. No API changes — only plain `Marshal`/`Unmarshal` was used.
  - **FIX**: All middleware validation/retry/recovery functions now wrap errors with sentinels — callers can use `errors.Is(err, middleware.ErrValidationFailed)`
  - **TEST**: 5 new middleware sentinel tests, 4 new TypedHandler tests
  - **REFACTOR**: `core/event/event.go` doc comment updated — no longer claims cockroachdb/errors dependency
  - Net -169 lines from cockroachdb removal, 4 commits, zero lint, all 22 test packages pass

- **Session 55–56 (Comprehensive Codebase Improvement Sweep)**:
  - **FIX**: Golden test files updated to match go-faster/yaml indentation
  - **REFACTOR**: `errors.As` → `errors.AsType[*Error]` (Go 1.22+ API) across all modules
  - **FIX**: `OutboxPublisher.run()` logs panics instead of silently swallowing
  - **FIX**: `NewEvent` copies payload bytes to prevent caller mutation
  - **REFACTOR**: Inlined `Deleter` interface into `Store` and `SnapshotStore` — removed unnecessary sub-interface
  - **NEW**: `event.TransactionalStore` interface — `SaveWithOutbox(ctx, aggregateID, expectedVersion, events, outbox)` for atomic save+outbox append
  - **NEW**: `storage.SQLTransactionalStore` — single `*sql.Tx` wraps event insert + outbox append + commit
  - **NEW**: `event.GlobalLoader` implemented on `SQLEventStore` — `LoadAll()` returns all events ordered by `occurred_at ASC`
  - **EVALUATED**: Shared repository core — rejected (aggregate vs decider have fundamentally different API shapes)
  - **EVALUATED**: `io.Closer` removal from interfaces — deferred (breaking change, needs focused design session)
  - **EVALUATED**: `IdempotencyKey` auto-generation — rejected (correct by design, `""` means no dedup key)
  - Zero lint, all 22 test packages pass

- **Session 57–58 (Code Quality Sweep: Deduplication + Function Decomposition)**:
  - **REFACTOR**: Extracted typed constants in `example/user/` — 18 bare string literals replaced with `event.Type`, `event.AggregateType`, `command.Type`, `query.Type` constants across 7 files
  - **REFACTOR**: Extracted `foldEvents` method in `core/decider` — deduplicated identical fold loop between `loadFromStore` and `loadFromSnapshot`
  - **REFACTOR**: Unified `Classify()` to use registered map — event package sentinels now registered via `init()` + `RegisterClassification()`, eliminated 30-line hardcoded switch. Single code path for all classification
  - **REFACTOR**: Extracted `kindToTagName` helper in `catalog/asyncapi` — maps `MessageKind` to tag name string, replacing inline switch in `addMessageSchema`
  - **REFACTOR**: Extracted `collectMessageIDs` helper in `catalog/eventcatalog` — collects sends/receives/commands/queries from service messages, reducing `writeService` from 47 to ~35 lines
  - **REFACTOR**: Extracted `persistChanges` helper in `core/aggregate` — separated persistence routing (outbox/transactional/direct) from aggregate lifecycle. `Save()` from 54 to 21 lines
  - **REFACTOR**: Simplified `SQLEventStore.LoadAll` — `return scanEvents(rows)` instead of assign+check+return. File from 253 to 248 lines (under 250 limit)
  - Zero files over 250 lines, all functions under 30 lines (except `Export` in asyncapi at 55 lines — already well-decomposed with helpers)
  - Zero lint, all 22 test packages pass

- **Session 65 (Architectural Type Safety Sweep)**:
  - **BREAKING**: `NewEvent` signature: `version int` → `version Version`. All callers updated across 40+ files to pass `event.Version(n)` or `version.Increment()`.
  - **NEW**: `event.SchemaVersion` type — distinct from `Version` (stream position) to prevent mixing schema version with event version. `ParseSchemaVersion`, `Int()`, `String()`, `IsZero()`.
  - **BREAKING**: `Event.SchemaVersion()` returns `SchemaVersion` instead of `int`. `Core.schemaVersion` field typed.
  - **BREAKING**: `Upcaster.SourceVersion()` returns `SchemaVersion`. `NewUpcaster` takes `SchemaVersion`.
  - **BREAKING**: `WithSchemaVersion` takes `SchemaVersion` instead of `int`.
  - **BREAKING**: `NewCatalogCore`/`MustNewCatalogCore` take `Version` instead of `int`.
  - **NEW**: `ErrVersionNotPositive` sentinel — classified as `Rejection`. `validateEventParams` now checks `version.IsZero()` (no longer accepts `int`).
  - **NEW**: `storage.OutboxStatus` type with `OutboxStatusPending` constant — documents outbox status values.
  - **NEW**: `middleware.RetryConfig.Validate()` — validates `MaxAttempts >= 1`, `InitialDelay > 0`, `Multiplier > 1`. Returns `ErrValidationFailed`.
  - **NEW**: Middleware error classification — `ErrValidationFailed → Rejection`, `ErrRetryExhausted → Infrastructure`, `ErrRetryCanceled → Infrastructure`, `ErrPanicRecovered → Corruption`.
  - **NEW**: `memory.ErrHandlerNil → Rejection`, `catalog.ErrDomainNotFound → Rejection`, `catalog.ErrNilSchema → Rejection` classifications.
  - **FIX**: `storage/helpers.go:342` — `fmt.Sprintf` replaced with string concatenation (perfsprint lint).
  - Zero lint, all 22 test packages pass

- **Session 68 (Module Hygiene + File Size Compliance)**:
  - **FIX**: Removed `storage` production dependency on `memory` module — `PebbleBackendMemory` now returns `ErrPebbleProviderRequired` instead of calling `memory.NewMemoryStore()`. Restores ADR-0003 DAG rule.
  - **REFACTOR**: Split `storage/helpers.go` (433→239) → `storage/sql_helpers.go` (205) — SQL-agnostic shared helpers extracted.
  - **REFACTOR**: Split `catalog/asyncapi/exporter.go` (258→79) → `catalog/asyncapi/builder.go` (182) — Export logic extracted.
  - **REFACTOR**: Split `storage/pebble_event_store.go` (321→156) → `storage/pebble_helpers.go` (176) — Delete, AppendBatch, Close, helpers extracted.
  - **REFACTOR**: Split `catalog/registry.go` (254→208) → `catalog/registry_helpers.go` (47) — Copy helpers extracted.
  - **RESULT**: Zero production files exceed 250 lines. All test packages pass.

- **Session 73 (File Splits + Coverage + Type Safety + Golden Tests)**:
  - **REFACTOR**: Split `catalog/openapi/exporter.go` (318→254) → `catalog/openapi/convert.go` (68) — extracted `toKebab`/`toPascal`.
  - **REFACTOR**: Split `storage/event_store.go` (305→233) → `storage/event_store_scan.go` — extracted `scanEvents`/`scanEvent`/`insertEvents`.
  - **REFACTOR**: Split `storage/outbox.go` (255→152) → `storage/outbox_helpers.go` — extracted outbox serialization helpers.
  - **REFACTOR**: Split `catalog/docserver/docserver.go` (265→229) → `catalog/docserver/builders.go` — extracted `buildOpenAPI`/`buildAsyncAPI`/`buildCatalog`.
  - **REFACTOR**: Split `catalog/adapters/adapters_test.go` (543→250) → `catalog/adapters/dispatcher_test.go` + `catalog/adapters/export_test.go`.
  - **TEST**: `storage/dialect_test.go` — 15 tests for PostgresDialect, SQLiteDialect, `placeholders()`. Storage coverage: 86.9% → 88.1%.
  - **TEST**: `catalog/openapi/exporter_test.go` — 5 new tests (WithBasePath, nil schema, schemaToAny(nil), empty catalog, toKebab edge cases). Coverage: 83.9% → 96.6%.
  - **FIX**: `outboxEvent.Version` and `outboxEvent.SchemaVersion` changed from bare `int` to `event.Version`/`event.SchemaVersion` — type safety.
  - **FIX**: Refreshed 3 stale golden test files (asyncapi.yaml, eventcatalog-config.js, package.json).
  - Zero catalog lint, all 22 test packages pass
