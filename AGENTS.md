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

| Item      | Value                                                                                                                                                                                                                  |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language  | Go 1.26.3                                                                                                                                                                                                              |
| Modules   | `core`, `memory`, `catalog`, `middleware`, `testhelpers`, `integration`, `storage`, `projection`, `signing`, `otel`, `watermill`, `pebble`, `codec`, `turso`, `cqrs-gen` |
| Build     | `nix run .#build`                                                                                                                                                                                                      |
| Test      | `nix run .#test` or `go test ./core/... ./memory/... ./catalog/... ./middleware/... ./testhelpers/... ./integration/... ./projection/... ./signing/... ./storage/... ./watermill/... ./pebble/... ./codec/... -count=1` |
| Lint      | `nix run .#lint`                                                                                                                                                                                                       |
| Format    | `nix fmt`                                                                                                                                                                                                              |
| Dev shell | `nix develop`                                                                                                                                                                                                          |
| CI        | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                          |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 21 modules:

```
go-cqrs-lite/
├── core/                # github.com/larsartmann/go-cqrs-lite/core
│   ├── command/         # Dispatcher, Handler, Middleware, Command, BasicCommand
│   ├── query/           # Dispatcher, Handler, Pagination, PaginatedResult[T], RegisterTyped[T]
│   ├── event/           # EventSink, EventSource, Store, Journal, SeekableJournal, TombstoneStatus, Bus, SnapshotStore, ImmutableEvent, NewEvent, Clone, Codec, Upcaster
│   ├── decider/         # Decider[State], Repository[State], Execute, Load (pure-function style)
│   └── pkg/
│       ├── id/          # Branded IDs: id.Of[T] = cbid.ID[T, ulid.ULID], AggregateID, EventID, etc.
│       └── dispatcher/  # Generic Dispatcher[H, M] with LifecycleMixin
├── memory/              # MemoryStore, MemoryBus, MemorySnapshotStore (in-memory test impls)
├── catalog/             # Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters
├── middleware/           # Logging, Retry, Recovery, Validation, Metrics, OTel Tracing+Metrics (command+event+query)
├── signing/             # Event signing/verification: HMAC-SHA256, Ed25519, multisig, middleware
├── testhelpers/         # Noop/Failing/Panic handlers, FakeMetrics, AppendEventsHandler
├── projection/          # Runner (replay+live), HandlerRegistry, Builder with On[T]()
├── storage/             # SQLEventStore, SQLSnapshotStore, SQLOutbox, SQLCheckpointStore (PG/SQLite/Turso)
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── stream/              # Aggregate listing, tombstone detection, StatusMiddleware, SQL/projection readers
├── watermill/           # Watermill protocol adapter (publisher/subscriber)
├── pebble/              # Embedded key-value event store (PebbleDB)
├── codec/               # Payload encoding: JSON, Raw passthrough
├── turso/               # Turso database connector (embedded LibSQL sync)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── integration/         # Cross-module tests (command, event, query, signing)
└── docs/                # Status reports, ADRs, architecture patterns, storage guide
```

## Design Principles

1. **Library, not framework** — Consumers import what they need. No opinionated transport, broker, or SQL driver.
2. **Trustworthy modules** — Quality gate: "Would a consumer trust this enough to import it?"
3. **Minimal core dependencies** — core depends on `oklog/ulid`, `go-branded-id`, `go-error-family`.
4. **Composition over inheritance** — Per Go best practices.
5. **Interface-first design** — All core types are interfaces. Store = EventSink + EventSource (ISP split).
6. **Interface Segregation** — Journal (ReadAll), SeekableJournal (ReadFrom), BackwardsSource, TransactionalSink.
7. **Context-aware** — All handlers accept `context.Context`.
8. **Errors as values** — No panics, explicit error returns, sentinel errors + `%w` wrapping.
9. **Strong types** — No `any` types (except `dialect.go` for database/sql interop). Max 250 lines/file, 30 lines/function.
10. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
11. **Tombstone over delete** — Soft-delete via metadata (TombstoneStatus: Active/Tombstoned/Undetermined). No Delete on Store.

## Error Handling

- **Sentinel errors**: `errors.New` in `errors.go` files
- **Contextual errors**: `fmt.Errorf("failed to process %s: %w", name, err)`
- **Classified errors**: `event.NewRejection(...)`, `event.NewConflict(...)` via go-error-family
- **5-family taxonomy**: Rejection / Conflict / Transient / Infrastructure / Corruption

## Key Patterns

```go
// Event creation (typed payload, auto-marshaled)
evt, err := event.NewEvent("user.created", userID, "User", event.Version(1),
    UserCreated{Name: "Alice"}, event.WithCorrelationID(correlationID))

// Decider (pure-function aggregate)
decider := decider.Decider[State]{Initial: State{}, Fold: foldFunc}
result, err := decider.Repository[State].Execute(ctx, repo, aggregateID, decider, command)

// Branded IDs
type UserID = id.Of[userMarker]
uid := id.New[UserID]()

// Query dispatch (type-safe)
result, err := query.DispatchTyped[*GetUserResult](ctx, dispatcher, q)

// Sink/Source split (ISP)
var sink event.EventSink = store   // write side: Save, AppendBatch
var source event.EventSource = store // read side: Load, LoadFromVersion, LoadToVersion, LoadToTimestamp
var journal event.Journal = store   // ReadAll (cross-aggregate)
var seekable event.SeekableJournal = store // ReadFrom (position-based)

// Tombstone soft-delete (no Delete on Store)
status := event.DetectTombstone(events) // Active, Tombstoned, or Undetermined
marked, _ := event.MarkTombstone(evt)   // sets tombstone metadata

// Event upcasting (schema migration on load)
//   upcaster := event.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
//       return event.NewEvent(evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
//           newPayload, event.WithSchemaVersion(2))
//   })
//   versioned := event.NewVersionedStore(store, upcaster)
//   events, _ := versioned.Load(ctx, "User", aggregateID)

// Event signing (tamper-proof streams)
//   signer, _ := signing.NewHMAC(secret)
//   bus.UsePublish(signing.SignMiddleware(signer))
//   bus.Use(signing.VerifyMiddleware(signer))

// OpenTelemetry tracing (opt-in, no-op when no provider configured)
//   tracer := otel.GetTracerProvider().Tracer("my-app")
//   bus.Use(middleware.EventTracing(tracer))
//   bus.UsePublish(middleware.EventPublishTracing(tracer))
//   cmdDispatcher.Use(middleware.CommandTracing(tracer))

// OpenTelemetry metrics (opt-in, replace custom MetricsRecorder)
//   meter := otel.GetMeterProvider().Meter("my-app")
//   recorder, _ := middleware.NewOTelMetricsRecorder(meter)
//   cmdDispatcher.Use(middleware.CommandMetrics(recorder))
```

## Testing

- Table-driven tests preferred; BDD via Ginkgo v2 + Gomega for event/decider/query
- `t.Parallel()` for independent tests; core packages >80% coverage (most >90%)
- Per-module isolation: `cd core && GOWORK=off go test ./... -count=1`

## Dependencies

| Category   | Packages                                                                                                                                                     |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Production | oklog/ulid/v2, go-branded-id, go-error-family (core); go-faster/yaml (catalog); go.opentelemetry.io/otel (otel, core, storage, middleware, projection) |
| Test-only  | onsi/ginkgo/v2, onsi/gomega                                                                                                                                  |

**Coverage**: 84–100% across 27 packages. See `docs/status/` for latest.

**Module Graph**: otel→go.opentelemetry.io/otel; core→otel+codec (prod), memory+testhelpers (test-only); testhelpers→core; memory→core+testhelpers; middleware→core+otel+testhelpers;
catalog→core; storage→core+otel+testhelpers; projection→core+otel+memory+testhelpers; signing→core+testhelpers; listing→core+memory; watermill→core;
pebble→core+codec+otel+testhelpers; codec (leaf); turso→storage; cmd/cqrs-gen→core;
integration→core+memory+testhelpers.

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from projection + command dispatch. See `example/saga-pattern/`.

**Known Blocker**: `replace` directives in `go.mod` files required until v1.0.0 tags pushed to remote. Both `replace` and `go.work` are needed — replace for `GOWORK=off` per-module CI, go.work for developer convenience.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).
