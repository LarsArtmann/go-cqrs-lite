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

| Item      | Value                                                                              |
| --------- | ---------------------------------------------------------------------------------- |
| Language  | Go 1.26                                                                            |
| Modules   | `core`, `memory`, `catalog`, `middleware`, `testhelpers`, `integration`, `storage` |
| Build     | `nix run .#build`                                                                  |
| Test      | `nix run .#test` or see "Testing" below                                            |
| Lint      | `nix run .#lint`                                                                   |
| Format    | `nix fmt`                                                                          |
| Dev shell | `nix develop`                                                                      |
| CI        | GitHub Actions: ci.yml (Nix-based)                                                 |

## Monorepo Structure

Multi-module Go workspace with 8 modules (9 including example/user demo):

```
go-cqrs-lite/
├── go.work                          # ties modules together
│
├── core/                            # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                       # deps: cockroachdb/errors, oklog/ulid,
│                                    #       go-json-experiment/json
│   ├── command/                     # command dispatch, handler, catalog
│   ├── query/                       # query dispatch, pagination, catalog
│   ├── event/                       # event types, Store/Bus/SnapshotStore interfaces
│   │   ├── event.go                # Core struct, NewEvent constructor
│   │   └── options.go              # Option func, With* metadata helpers
│   ├── aggregate/                   # Root, Repository, Core, EventSourcedRepository
│   ├── pkg/
│   │   ├── id/                      # branded IDs: id.Of[T] (AggregateID, EventID, UserID, etc.)
│   │   │   ├── id.go               # Core type, constructors, comparisons
│   │   │   └── id_encoding.go      # JSON/binary/text/SQL marshaling
│   │   └── dispatcher/              # generic Dispatcher[H, M] with LifecycleMixin, CheckClosed
│
├── memory/                          # github.com/larsartmann/go-cqrs-lite/memory
│   └── go.mod                       # deps: core
│   ├── store.go                     # MemoryStore (implements event.Store)
│   ├── bus.go                       # MemoryBus (implements event.Bus)
│   └── snapshot.go                  # MemorySnapshotStore (implements event.SnapshotStore)
│
├── catalog/                         # github.com/larsartmann/go-cqrs-lite/catalog
│   └── go.mod                       # deps: core, go-faster/yaml, go-json-experiment/json
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
│   └── go.mod                       # deps: core, cockroachdb/errors
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
└── docs/
    ├── status/                      # periodic status reports
    └── planning/                    # architectural decisions and migration plans
```

## Testing

From root with go.work:

```bash
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./testhelpers/... ./integration/... -count=1
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

| Package                | Purpose                             | Key Types                                                                                              |
| ---------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `core/command/`        | Command dispatch and handling       | `Dispatcher`, `Handler`, `Middleware`, `Command`, `Core`                                               |
| `core/query/`          | Query dispatch with pagination      | `Dispatcher`, `Handler`, `Pagination`, `PaginatedResult[T]`, `Middleware`                              |
| `core/event/`          | Event sourcing interfaces and types | `Store`, `Bus`, `SnapshotStore`, `Event`, `Core`, `Metadata`, `Option`                                 |
| `core/aggregate/`      | Aggregate roots and repository      | `Root`, `Repository`, `Core`, `EventSourcedRepository`                                                 |
| `core/pkg/id/`         | Branded IDs via generics            | `id.Of[T]`, `AggregateID`, `EventID`, `UserID`, `CorrelationID`, `Ptr()`, `FromPtr()`, `fmt.Formatter` |
| `core/pkg/dispatcher/` | Generic internal dispatcher         | `Dispatcher[H, M]`, `MiddlewareChain[H, M]`, `LifecycleMixin`                                          |

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

| Package       | Purpose                       | Key Types                                                                                                                        |
| ------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `middleware/` | Cross-cutting CQRS middleware | `CommandLogging`, `CommandRetry`, `CommandRecovery`, `CommandValidation`, `EventValidation`, `QueryValidation`, `CommandMetrics` |

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

## Design Principles

1. **Library, not framework** — Consumers import what they need, compose their own stack. No opinionated transport (HTTP/gRPC), message broker (Kafka/NATS), or SQL driver. Integration modules (storage, watermill) are optional.
2. **Every module must be trustworthy on its own** — Quality gate: "Would a consumer trust this enough to import it?" Means: tests, stable API, clear docs. Does NOT mean "another module in this repo uses it."
3. **Minimal core dependencies** — core depends on `cockroachdb/errors`, `oklog/ulid`, `go-branded-id`, `go-json-experiment/json`
4. **Composition over inheritance** — Per Go best practices
5. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, `Root`, etc.)
6. **Context-aware** — All handlers accept `context.Context`
7. **Errors as values** — No panics, explicit error returns, sentinel errors + wrapping
8. **File size limits** — Max 250 lines per file
9. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps

## Code Conventions

- Use `fmt.Errorf` for error messages with context
- Use `errors.New` (cockroachdb/errors) for sentinel errors
- Wrap errors with context using `errors.Wrapf` or `errors.Wrap`
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
    return errors.Wrapf(err, "failed to process %s", name)
}
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
    1,
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

| Dependency                | Version | Purpose                           | Module           |
| ------------------------- | ------- | --------------------------------- | ---------------- |
| `cockroachdb/errors`      | v1.12.0 | Error wrapping                    | core, middleware |
| `oklog/ulid/v2`           | v2.1.0  | ULID generation (binary-sortable) | core             |
| `go-branded-id`           | v0.1.0  | Branded ID type backing           | core             |
| `go-faster/yaml`          | v0.4.6  | YAML marshaling                   | catalog          |
| `go-json-experiment/json` | v0.0.0  | JSON v2                           | core, catalog    |

### Test-only

| Dependency       | Version | Purpose      | Module |
| ---------------- | ------- | ------------ | ------ |
| `onsi/ginkgo/v2` | v2.28.1 | BDD testing  | core   |
| `onsi/gomega`    | v1.39.1 | BDD matchers | core   |

## Test Coverage Summary

| Package                | Coverage |
| ---------------------- | -------- |
| `core/command`         | 100.0%   |
| `core/query`           | 100.0%   |
| `core/pkg/dispatcher`  | 100.0%   |
| `core/pkg/id`          | 100.0%   |
| `middleware`           | 99.4%    |
| `memory`               | 99.0%    |
| `catalog/adapters`     | 98.8%    |
| `catalog/asyncapi`     | 97.9%    |
| `core/event`           | 96.3%    |
| `catalog/eventcatalog` | 95.5%    |
| `core/aggregate`       | 95.6%    |
| `catalog`              | 94.4%    |
| `storage`              | 79.6%    |

## Module Dependency Graph

```
testhelpers → core
memory      → core + testhelpers
middleware  → core + testhelpers
catalog    → core (via cattest internal helpers)
storage    → core (go-sqlmock for tests)
integration → core + memory + testhelpers
core        → (no internal deps — independently publishable)
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

| Improvement                  | Detail                                                                 | Commit    |
| ---------------------------- | ---------------------------------------------------------------------- | --------- |
| Dead code removal            | `evtest.GenerateUUID`, `testutil` package, `query.ErrQueryValidation`  | `1862eae` |
| Lifecycle unification        | `MemoryBus`/`MemorySnapshotStore` now use `LifecycleMixin`             | `8e5150c` |
| EventValidation middleware   | API symmetry: Command/Query/Event all have validation                  | `4fdd447` |
| MessageID extraction         | Moved from `asyncapi`/`eventcatalog` to `catalog.MessageID()`          | `c1bc261` |
| event.go split               | Extracted `Option`/`With*` to `event/options.go` (169 + 90 lines)      | `699d247` |
| Dead reflect.Ptr case        | Removed unreachable branch in `goTypeToJSON`                           | `b23a781` |
| Dispatcher.Dispatch refactor | Removed unused `handler H` parameter                                   | `e84e3a1` |
| Example simplification       | `example/user/` demonstrates full CQRS lifecycle + EventCatalog export | `6815ef3` |

## Known Issues

| Issue                                                    | Severity   | Detail                                                                                           |
| -------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------ |
| **FakeStore/MemoryStore key separator mismatch**         | **HIGH**   | `FakeStore` uses `"/"`, `MemoryStore` uses `":"`. Different behavior for same interface.         |
| **`Load()` empty semantics differ**                      | **MEDIUM** | `MemoryStore.Load()` returns `ErrAggregateNotFound`; `SQLEventStore.Load()` returns empty slice. |
| `MemoryBus.Publish` holds RLock during handler execution | LOW        | Subscribers block publishers (acceptable for test utility)                                       |
| `query.Handler` returns `any`                            | LOW        | Violates project "no any" rule; `DispatchTyped[T]` is the workaround                             |
| `CatalogMeta` duplicated across 3 packages               | LOW        | `event.CatalogMeta`, `command.CatalogMeta`, `query.CatalogMeta` — nearly identical               |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch     | LOW        | Every aggregate must implement `LoadEvents` and delegate to `LoadFromHistory`                    |

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
