# Project: go-cqrs-lite

A lightweight CQRS (Command Query Responsibility Segregation) library for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

## Quick Reference

| Item          | Value                                  |
| ------------- | -------------------------------------- |
| Language      | Go 1.26                                |
| Modules       | `core`, `memory`, `catalog`, `middleware`, `xtypes`, `testhelpers` |
| Build         | `make build` or `go build ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...` |
| Test          | `make test` or see "Testing" below     |
| Lint          | `make lint` or `golangci-lint run ./...` |
| Format        | `gofumpt -w .`                         |
| Imports       | `goimports -w .`                       |
| CI            | GitHub Actions: test.yml + lint.yml    |

## Monorepo Structure

Multi-module Go workspace with 6 independent modules:

```
go-cqrs-lite/
├── go.work                          # ties modules together
│
├── core/                            # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                       # deps: cockroachdb/errors, oklog/ulid,
│                                    #       go-json-experiment/json, ginkgo/gomega (test)
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
│   ├── internal/
│   │   └── testhelpers/             # test helpers (internal, not importable)
│   └── event/store_config.go       # vestigial config (returns error, points to memory module)
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
├── xtypes/                          # github.com/larsartmann/go-cqrs-lite/xtypes
│   └── go.mod                       # deps: core
│   ├── command.go                   # TypedCommand with branded ID
│   ├── event.go                     # TypedEvent, EventBuilder
│   ├── aggregate.go                 # TypedAggregate
│   └── id.go                        # Re-exports id types, CommandID
│
├── testhelpers/                     # github.com/larsartmann/go-cqrs-lite/testhelpers
│   └── go.mod                       # deps: core
│   └── helpers.go                   # Shared test utilities (AppendEventsHandler, Noop*, Failing*, etc.)
│
├── example/                         # standalone example modules
│   ├── user/
│   └── catalog/
│
└── docs/
    ├── status/                      # periodic status reports
    └── planning/                    # architectural decisions and migration plans
```

## Testing

From root with go.work:
```bash
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/... ./testhelpers/... -count=1
```

Per-module (isolated, no go.work):
```bash
cd core && GOWORK=off go test ./... -count=1
cd memory && GOWORK=off go test ./... -count=1
cd catalog && GOWORK=off go test ./... -count=1
```

Makefile targets (run from root, use `GOWORK=off`):
```bash
make test          # all tests verbose
make test-race     # race detector
make test-cover    # coverage report
make build         # go build
make lint          # golangci-lint
make check         # fmt + imports + lint + build + test
```

## Architecture

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
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  memory      │ │  catalog     │ │  middleware   │ │  xtypes      │
│  MemoryStore │ │  AsyncAPI    │ │  Logging      │ │  TypedCmd    │
│  MemoryBus   │ │  EventCat    │ │  Retry        │ │  TypedEvt    │
│  Snapshot    │ │  Schema      │ │  Recovery     │ │  TypedAgg    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │  (planned)       │
                              │  storage/        │
                              │  watermill/      │
                              │  projection/     │
                              └─────────────────┘
```

## Package Overview

### Core Module (`core/`)

| Package                    | Purpose                                        | Key Types                                               |
| -------------------------- | ---------------------------------------------- | ------------------------------------------------------- |
| `core/command/`            | Command dispatch and handling                  | `Dispatcher`, `Handler`, `Middleware`, `Command`, `Core` |
| `core/query/`              | Query dispatch with pagination                 | `Dispatcher`, `Handler`, `Pagination`, `PaginatedResult[T]`, `Middleware` |
| `core/event/`              | Event sourcing interfaces and types            | `Store`, `Bus`, `SnapshotStore`, `Event`, `Core`, `Metadata`, `Option` |
| `core/aggregate/`          | Aggregate roots and repository                 | `Root`, `Repository`, `Core`, `EventSourcedRepository`  |
| `core/pkg/id/`             | Branded IDs via generics                       | `id.Of[T]`, `AggregateID`, `EventID`, `UserID`, `CorrelationID` |
| `core/pkg/dispatcher/`     | Generic internal dispatcher                     | `Dispatcher[H, M]`, `MiddlewareChain[H, M]`, `LifecycleMixin` |

### Memory Module (`memory/`)

| Package       | Purpose                      | Key Types             |
| ------------- | ---------------------------- | --------------------- |
| `memory/`     | In-memory test implementations | `MemoryStore`, `MemoryBus`, `MemorySnapshotStore` |

### Catalog Module (`catalog/`)

| Package                 | Purpose                          | Key Types                                    |
| ----------------------- | -------------------------------- | -------------------------------------------- |
| `catalog/`              | Registry, schema reflection, MessageID | `Registry`, `Catalog`, `SchemaFromType[T]`, `MessageID()` |
| `catalog/adapters/`     | Builder and dispatcher adapters  | `CatalogBuilder`, `FromCommandDispatcher`     |
| `catalog/asyncapi/`     | AsyncAPI 3.0 YAML/JSON export   | `Exporter`, `Document`, `MarshalYAML`        |
| `catalog/eventcatalog/` | EventCatalog MDX generator       | `Exporter`                                   |

### Middleware Module (`middleware/`)

| Package        | Purpose                          | Key Types                                    |
| -------------- | -------------------------------- | -------------------------------------------- |
| `middleware/`   | Cross-cutting CQRS middleware     | `CommandLogging`, `CommandRetry`, `CommandRecovery`, `CommandValidation`, `EventValidation`, `QueryValidation`, `CommandMetrics` |

### Xtypes Module (`xtypes/`)

| Package   | Purpose                          | Key Types                                    |
| --------- | -------------------------------- | -------------------------------------------- |
| `xtypes/` | Typed wrappers with branded IDs  | `TypedCommand`, `TypedEvent`, `TypedAggregate`, `EventBuilder`, `CommandID` |

### Testhelpers Module (`testhelpers/`)

| Helper | Purpose |
|--------|----------|
| `AppendEventsHandler` | Bus handler that collects events into a slice |
| `NoopCommandHandler` / `NoopEventHandler` | No-op handlers for middleware tests |
| `FailingCommandHandler` / `FailingEventHandler` | Handlers that always error |
| `PanicCommandHandler` / `PanicEventHandler` | Handlers that panic |
| `CallbackCommandHandler` | Handler that sets a bool flag |
| `CommandMiddleware` / `EventMiddleware` | Call-order tracking middleware |
| `TestMetrics` | Metrics collector for testing |

`core/internal/testhelpers` re-exports from this module for backward compatibility.

## Design Principles

1. **Minimal core dependencies** — core depends on `cockroachdb/errors`, `oklog/ulid`, `go-composable-business-types`, `go-json-experiment/json`
2. **Composition over inheritance** — Per Go best practices
3. **Interface-first design** — All core types are interfaces (`Store`, `Bus`, `Root`, etc.)
4. **Context-aware** — All handlers accept `context.Context`
5. **Errors as values** — No panics, explicit error returns, sentinel errors + wrapping
6. **File size limits** — Max 250 lines per file
7. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps

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

| Dependency              | Version  | Purpose              | Module   |
| ----------------------- | -------- | -------------------- | -------- |
| `cockroachdb/errors`   | v1.12.0  | Error wrapping       | core, middleware |
| `oklog/ulid/v2`             | v2.1.0   | ULID generation (binary-sortable) | core     |
| `go-composable-business-types` | v0.1.0 | Branded ID type backing | core |
| `go-faster/yaml`        | v0.4.6   | YAML marshaling      | catalog  |
| `go-json-experiment/json` | v0.0.0 | JSON v2             | core, catalog |

### Test-only

| Dependency          | Version  | Purpose          | Module |
| ------------------- | -------- | ---------------- | ------ |
| `onsi/ginkgo/v2`    | v2.28.1  | BDD testing      | core   |
| `onsi/gomega`       | v1.39.1  | BDD matchers     | core   |

## Test Coverage Summary

| Package                  | Coverage |
| ------------------------ | -------- |
| `catalog/adapters`       | 98.8%    |
| `memory`                 | 99.2%    |
| `catalog/asyncapi`       | 96.3%    |
| `xtypes`                 | 95.7%    |
| `catalog`                | 91.2%    |
| `query`                  | 91.5%    |
| `catalog/eventcatalog`   | 89.7%    |
| `event`                  | 89.0%    |
| `pkg/id`                 | 73.1%    |
| `middleware`              | 84.6%    |
| `command`                | 84.4%    |
| `internal/dispatcher`    | 77.4%    |
| `aggregate`              | 77.3%    |

## Module Dependency Graph

```
testhelpers → core
memory      → core + testhelpers
middleware  → core + testhelpers
catalog    → core (via cattest internal helpers)
core       → memory + testhelpers
xtypes     → standalone
```

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
│ catalog/asyncapi/   │  │ catalog/eventcatalog/   │
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

## Branded Return Types Migration (Session 3)

Interfaces now return branded ID types instead of `string`:

| Interface | Method | Old Return | New Return |
| --------- | ------ | ---------- | ---------- |
| `Event` | `ID()` | `string` | `id.EventID` |
| `Event` | `AggregateID()` | `string` | `id.AggregateID` |
| `Root` | `ID()` | `string` | `id.AggregateID` |
| `Command` | `AggregateID()` | `string` | `id.AggregateID` |

**Caller updates**: All `event.NewEvent()` calls pass `id.AggregateID` directly (no re-parse). All `cmd.AggregateID()` and `root.ID()` comparisons use branded types. `repository.go` eliminated redundant `id.ParseAggregateID()` re-parses. `middleware/logging.go` adds `.String()` when formatting IDs for log output. Commit: `cee6c50`

## Bug Fixes (Sessions 1–2)

| Bug | Fix | Commit |
|-----|-----|-------|
| Retry dead cancellation | `context.Background().Done()` → `ctx.Done()` in `middleware/retry.go` | `5ad0356` |
| Aggregate version desync | Removed fallback loop; `Load()` requires `HistoryLoader` | `1862eae` |
| Wrong error sentinel (dispatcher) | `CheckClosed` used `ErrHandlerNotFound` → `ErrDispatcherClosed` | `5ad0356` |
| Slice mutation (MemoryStore) | `Load()`/`LoadFromVersion()` return defensive copies | `d5ea811` |
| Wrong error sentinel (snapshot) | `CheckClosed` used `ErrSnapshotNotFound` → `ErrSnapshotStoreClosed` | `8e5150c` |

## Code Quality Improvements (Sessions 1–2)

| Improvement | Detail | Commit |
|-------------|--------|-------|
| Dead code removal | `evtest.GenerateUUID`, `testutil` package, `query.ErrQueryValidation` | `1862eae` |
| Lifecycle unification | `MemoryBus`/`MemorySnapshotStore` now use `LifecycleMixin` | `8e5150c` |
| EventValidation middleware | API symmetry: Command/Query/Event all have validation | `4fdd447` |
| MessageID extraction | Moved from `asyncapi`/`eventcatalog` to `catalog.MessageID()` | `c1bc261` |
| event.go split | Extracted `Option`/`With*` to `event/options.go` (169 + 90 lines) | `699d247` |
| Dead reflect.Ptr case | Removed unreachable branch in `goTypeToJSON` | `b23a781` |
| Dispatcher.Dispatch refactor | Removed unused `handler H` parameter | `e84e3a1` |
| Example simplification | `example/user/` uses `aggregate.EventSourcedRepository` | `6815ef3` |

## Known Issues

| Issue | Severity | Detail |
|-------|----------|--------|
| `MemoryBus.Publish` holds RLock during handler execution | LOW | Subscribers block publishers (acceptable for test utility) |
| `xtypes.TypedCommand.Command()` allocates on every call | LOW | Creates new `command.Core` each time |
| `go.work` version mismatch | LOW | FIXED — go.work now tracked in VCS |
| `toDotAddress` number handling | LOW | "Get3DView" → "get.3.d.view" instead of "get.3d.view" |
| No `EventRetry` tests | LOW | `EventValidation` tested, `EventRetry` still needs test coverage |
| `pkg/id` coverage | LOW | 73.1% — missing tests for `ULID()`, `Get()`, `Parse`/`MustParse` on `CausationID`, `CorrelationID`, `EventID`, `RequestID` |

## Cleanup Done (Post-Migration)

- Removed `query.Result[T]` (dead code, zero callers)
- Removed unused error sentinels: `ErrEventNotFound`, `ErrInvalidEventType`, `ErrCommandValidation`, `ErrQueryValidation`
- Removed unused `Streamer` interface
- Removed vestigial `store_config.go`
- Removed `internal/testutil` package (unused)
- Removed `evtest.GenerateUUID` (unused)
- Fixed `.golangci.yml` v2 schema errors (removed stale `wrapcheck`, `formatters`, migrated `exclude-rules` to `exclusions.rules`)
- Removed redundant `//nolint:err113` from test files (now excluded via config)
- Added CONTRIBUTING.md for multi-module workflow
- Added CI badges to README.md
- Unified lifecycle: `MemoryBus` and `MemorySnapshotStore` use `LifecycleMixin` (no more manual `closed bool`)
- Extracted `MessageID()` from `asyncapi`/`eventcatalog` to `catalog` package (removes eventcatalog→asyncapi coupling)
- Split `event/event.go` under 250 lines (extracted `options.go`)
- Split `id/id.go` under 250 lines (extracted `id_encoding.go` — JSON/binary/text/SQL marshaling)
- Added `EventValidation` middleware for API symmetry
- Removed dead `reflect.Ptr` case in `catalog/schema.go`
- Removed unused `handler` parameter from `dispatcher.Dispatch()`
- Simplified `example/user/` to use `aggregate.EventSourcedRepository`
- **Branded return types**: `Event.ID()` → `id.EventID`, `Event.AggregateID()` → `id.AggregateID`, `Root.ID()` → `id.AggregateID`, `Command.AggregateID()` → `id.AggregateID`. All callers updated, redundant re-parses eliminated.
- **ULID migration**: `id.Of[T]` now wraps `cbid.ID[T, ulid.ULID]` instead of `cbid.ID[T, string]`. IDs are binary-sortable, time-ordered, 16-byte binary form. All serialization reimplemented locally. ~120 test fixtures migrated to valid ULIDs.
- Removed `NewWithPrefix` and `PrefixString` — prefix incompatible with ULID format, function silently discarded the prefix parameter
- **Deduplication campaign**: Resolved ALL code duplication. 16 → 0 clone groups (art-dupl -t 27). Created shared `testhelpers/` module at repo root to break `internal` package boundary. `core/internal/testhelpers` now re-exports from shared module.
- **go.work tracked in VCS**: Removed from .gitignore — multi-module workspace needs reproducible structure.
- **Lint-clean**: All 22 lint issues resolved across core, catalog. Added `gochecknoglobals` exclusion for testhelpers re-export shim.
- **query.Handler type alias**: Middleware uses `query.Handler` type alias for consistency.

## Migration State

The monorepo is mid-migration from a single module to multi-module. Current state:

| Phase | Description | Status |
|-------|-------------|--------|
| 0 | Fix query handler ctx, delete pkg/errors, replace custom YAML | Done |
| 1 | go.work + move into `core/` subdirectory | Done |
| 2 | Extract `memory/` module | Done |
| 3 | Extract `catalog/` module | Done |
| 4 | Extract middleware + xtypes | Done |
| 5 | Storage module (sqlc event store) | Planned |
| 6 | Watermill module (pub/sub) | Planned |
| 7 | Projection module (samber/ro internally) | Planned |
| 8 | Snapshot module (SQL-backed) | Planned |
| 9 | Test utilities module | Done — `testhelpers/` at repo root |
| 10 | Tag releases | Planned |

See `docs/planning/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md` for the full migration plan.

## Deferred Changes

These were identified but explicitly deferred because they affect all consumers or require external coordination:

| Change | Reason |
|--------|--------|
| ~~`Root.ID()` → return `id.AggregateID` instead of `string`~~ | ✅ Done (commit `7cc3e20`) |
| ~~`Event.AggregateID()` → return `id.AggregateID`~~ | ✅ Done (commit `7cc3e20`) |
| ~~`Event.ID()` → return `id.EventID`~~ | ✅ Done (commit `7cc3e20`) |
| ~~`Command.AggregateID()` → return `id.AggregateID`~~ | ✅ Done (commit `7cc3e20`) |
| ~~`event.go:129` `aggregateID.IsZero`~~ | ✅ Done (now uses branded `id.AggregateID.IsZero()`) |
| Stale `replace` directives in `middleware/go.mod`, `xtypes/go.mod` | Requires sibling `go-composable-business-types` repo |

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Coding standards
- `docs/planning/2026-04-23_PROJECT_REVIEW.md` - In-depth code review
- `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` - Watermill decision analysis
- `docs/planning/2026-04-24_SAMBER_RO_PRO_CONTRA.md` - samber/ro for projections
- CQRS patterns from: ChastityAPI, Cyberdom, Domination, GmbHG
