# Project: go-cqrs-lite

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import modules (`core`, `storage`, `memory`, `catalog`, etc.) into THEIR projects.
> There is no "main app." Every module is independently importable.
>
> | If you catch yourself thinking…              | STOP — this is a LIBRARY, not an app                                                                                                       |
> | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
> | "Nothing in this repo uses it, so delete it" | **DELETING EXTERNAL-FACING API IS BREAKING THE PRODUCT.** Consumers live outside this repo. Zero internal consumers is the EXPECTED state. |
> | "Module needs a service that uses it"        | Module needs tests + stable API, not an internal consumer                                                                                  |
> | "example/ should drive real traffic"         | example/ is a usage demo, not a deployment                                                                                                 |
> | "Unused exports are waste"                   | Public API surface IS the product                                                                                                          |
>
> **The quality gate for every module: "Would a consumer trust this enough to import it?"**

A lightweight CQRS **library/SDK** for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework — no opinionated transport, message broker, or SQL driver.

## Quick Reference

| Item      | Value                                                                                                                                                                                                                                                                                                                                 |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language  | Go 1.26.3                                                                                                                                                                                                                                                                                                                             |
| Modules   | `event`, `command`, `query`, `decider`, `id`, `dispatcher`, `schema`, `snapshot`, `memory`, `catalog`, `middleware`, `integration`, `storage`, `projection`, `signing`, `otel`, `watermill`, `pebble`, `codec`, `turso`, `listing`, `cqrs-gen`, `api-stability`                                                                       |
| Build     | `nix run .#build`                                                                                                                                                                                                                                                                                                                     |
| Test      | `nix run .#test` or `go test ./event/... ./command/... ./query/... ./decider/... ./id/... ./dispatcher/... ./schema/... ./snapshot/... ./memory/... ./catalog/... ./middleware/... ./integration/... ./projection/... ./signing/... ./storage/... ./watermill/... ./pebble/... ./codec/... ./listing/... ./cmd/cqrs-gen/... -count=1` |
| Lint      | `nix run .#lint`                                                                                                                                                                                                                                                                                                                      |
| Format    | `nix fmt`                                                                                                                                                                                                                                                                                                                             |
| Dev shell | `nix develop`                                                                                                                                                                                                                                                                                                                         |
| CI        | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                                                                                                                                         |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 30 modules (22 library + 6 examples + 1 integration + 1 cmd):

```
go-cqrs-lite/
├── event/               # EventSink, EventSource, Store, Journal, Bus, ImmutableEvent, NewEvent, Clone
│                        # Reactive: EventBus, NewReplayEventBus, NewBehaviorEventBus, FilterEventType/Types, ReplayFilter, HandlerToObserver, Map, ScanState, Tap
│   └── eventtest/       # FakeStore, FakeBus, FakeSnapshotStore, event factories, test assertions
├── command/             # Dispatcher, Handler, Middleware, Command, BasicCommand
│                        # Reactive: CommandBus (= ro.Subject[Command]), FilterCommandType
├── query/               # Dispatcher, Handler, Pagination, PaginatedResult[T], RegisterTyped[T]
│                        # Reactive: QueryBus (= ro.Subject[Query]), FilterQueryType
├── decider/             # Decider[State], Repository[State], Execute, Load (pure-function style)
├── id/                  # Branded IDs: id.Of[T] = cbid.ID[T, ulid.ULID], AggregateID, EventID, etc.
├── dispatcher/          # Generic Dispatcher[H, M] with LifecycleMixin
├── schema/              # Upcaster, VersionedStore, upcasterRegistry (schema evolution)
├── snapshot/            # Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
├── memory/              # MemoryStore, MemoryBus, MemorySnapshotStore, MemoryCheckpointStore (in-memory test impls)
├── catalog/             # Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters
│   └── schema/          # JSON Schema types, reflection engine, YAML serialization
├── middleware/           # Logging, Retry, Recovery, Validation, Metrics, OTel Tracing+Metrics (command+event+query)
├── signing/             # Event signing/verification: HMAC-SHA256, Ed25519, multisig, middleware
├── projection/          # Runner (replay+live), HandlerRegistry, Builder with On[T]()
├── storage/             # SQLEventStore, SQLSnapshotStore, SQLCheckpointStore (PG/SQLite/Turso)
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── listing/             # Aggregate listing, tombstone detection, StatusMiddleware, InMemoryAggregateReader
├── watermill/           # Watermill protocol adapter (publisher/subscriber)
├── pebble/              # Embedded key-value event store (PebbleDB)
├── codec/               # Payload encoding: JSON, Raw passthrough
├── turso/               # Turso database connector (embedded LibSQL sync)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── cmd/api-stability/   # API surface checker: compares exported symbols against golden file
├── integration/         # Cross-module tests (command, event, query, signing)
└── docs/                # Status reports, ADRs, architecture patterns, storage guide
```

## Design Principles

1. **Library, not framework** — Consumers import what they need. No opinionated transport, broker, or SQL driver.
2. **Trustworthy modules** — Quality gate: "Would a consumer trust this enough to import it?"
3. **Minimal dependencies** — event depends on `oklog/ulid`, `go-branded-id`, `go-error-family`, `samber/ro`.
4. **Composition over inheritance** — Per Go best practices.
5. **Interface-first design** — All core types are interfaces. Store = EventSink + EventSource (ISP split).
6. **Interface Segregation** — Journal (ReadAll), SeekableJournal (ReadFrom), BackwardsSource.
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

// Reactive event streams (samber/ro)
//   bus := event.NewEventBus()
//   bus.Subscribe(ro.OnNext(func(e event.Event) { ... }))
//   filtered := ro.Pipe1(bus, event.FilterEventType("user.created"))
//   observer := event.HandlerToObserver(myHandler)
//   bus.Next(evt)
//   bus.Complete()
//
//   cmdBus := command.NewCommandBus()
//   cmdBus.Subscribe(ro.OnNext(func(c command.Command) { ... }))
//
//   queryBus := query.NewQueryBus()
//   queryBus.Subscribe(ro.OnNext(func(q query.Query) { ... }))

// Event upcasting (schema migration on load)
//   upcaster := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
//       return event.NewEvent(evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
//           newPayload, event.WithSchemaVersion(2))
//   })
//   versioned := schema.NewVersionedStore(store, upcaster)
//   events, _ := versioned.Load(ctx, event.NewAggregateRef("User", aggregateID))

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
- Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`

## Dependencies

| Category   | Packages                                                                                                                                                                            |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production | oklog/ulid/v2, go-branded-id, go-error-family, samber/ro (event, command, query); go-faster/yaml (catalog); go.opentelemetry.io/otel (otel, event, storage, middleware, projection) |
| Test-only  | onsi/ginkgo/v2, onsi/gomega                                                                                                                                                         |

**Coverage**: 84–100% across 32 packages. See `docs/status/` for latest.

**Module Graph**:

```
Layer 0: id/, dispatcher/, codec/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher, ro), query/ (→dispatcher, ro)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot)
Layer 4: memory/, signing/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability
```

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from projection + command dispatch. See `example/saga-pattern/`.

**v2.0.0 Released**: All 23 modules tagged at v2.0.0 with `/v2` semantic import paths. Replace directives in go.mod files are retained for `GOWORK=off` per-module CI (ignored by consumers). Consumers import via `github.com/larsartmann/go-cqrs-lite/event/v2` etc.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).

> **Schema evolution**: Upcaster and VersionedStore moved to `schema/` module. See `schema/` package.
> **Snapshot persistence**: Snapshot types moved to `snapshot/` module. See `snapshot/` package.
