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

| Item      | Value                                                                                                                                                                                                                                                                                                                                                                 |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language  | Go 1.26.3                                                                                                                                                                                                                                                                                                                                                             |
| Modules   | `event`, `command`, `query`, `decider`, `id`, `dispatcher`, `schema`, `snapshot`, `memory`, `catalog`, `middleware`, `integration`, `storage`, `projection`, `signing`, `encryption`, `otel`, `watermill`, `pebble`, `codec`, `turso`, `listing`, `testutil`, `cqrs-gen`, `api-stability`                                                                             |
| Build     | `nix run .#build`                                                                                                                                                                                                                                                                                                                                                     |
| Test      | `nix run .#test` or `go test ./event/... ./command/... ./query/... ./decider/... ./id/... ./dispatcher/... ./schema/... ./snapshot/... ./memory/... ./catalog/... ./middleware/... ./integration/... ./projection/... ./signing/... ./encryption/... ./storage/... ./watermill/... ./pebble/... ./codec/... ./listing/... ./testutil/... ./cmd/cqrs-gen/... -count=1` |
| Lint      | `nix run .#lint`                                                                                                                                                                                                                                                                                                                                                      |
| Format    | `nix fmt`                                                                                                                                                                                                                                                                                                                                                             |
| Dev shell | `nix develop`                                                                                                                                                                                                                                                                                                                                                         |
| CI        | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                                                                                                                                                                         |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 28 modules (22 library + 1 integration + 3 examples + 2 cmd):

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
├── encryption/          # Event payload encryption: XChaCha20-Poly1305, AES-256-GCM, codec wrapper, middleware
├── projection/          # Runner (replay+live), HandlerRegistry, Builder with On[T]()
├── storage/             # SQLEventStore, SQLSnapshotStore, SQLCheckpointStore (PG/SQLite/Turso)
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── listing/             # AggregateListing, AggregateStatus, tombstone detection, StatusMiddleware, InMemoryAggregateReader
├── watermill/           # Watermill protocol adapter (publisher/subscriber)
├── pebble/              # Embedded key-value event store (PebbleDB), CBOR envelope with JSON backward compat
├── codec/               # Payload encoding: JSON, CBOR (deterministic), Raw passthrough
├── turso/               # Turso database connector (embedded LibSQL sync)
├── testutil/            # Shared test helpers: MustNewCmd (cross-module test utilities)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── cmd/api-stability/   # API surface checker: compares exported symbols against golden file
├── integration/         # Cross-module tests (command, event, query, signing, encryption)
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
12. **Dependency budgets** — Per-module direct dep limits enforced by `nix run .#check-layers`. Adding deps requires explicit budget review.
13. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, projection, storage, middleware go.mod files.
14. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths via `*ImmutableEvent` type assertion. Used by signing (SHA-256 hashing, CloneEvent), pebble (json.Marshal), storage/sql (ExecContext), middleware/sse (string conversion). Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
15. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake. The `Event` interface documents this contract for third-party implementors.

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

// Processing mode (replay vs live context)
replayCtx := event.WithProcessingMode(ctx, event.ModeReplay)
mode := event.ProcessingModeFrom(ctx)    // ModeLive or ModeReplay

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

// Event encryption (confidential payloads)
//   enc, _ := encryption.NewXChaCha20Poly1305(key)
//   bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v1")))
//   bus.Use(encryption.DecryptMiddleware(enc))
//
//   // Composable codec wrapper
//   codec := encryption.NewCodec(codec.JSONCodec{}, enc)
//   alg := encryption.ExtractAlgorithm(evt)  // "xchacha20-poly1305"
//   keyID := encryption.ExtractKeyID(evt)    // "key-v1"

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
- Golden tests use shared `eventtest.AssertGolden(t, path, got, *update)` from `event/v2/eventtest`
- Modules without event dependency (otel, codec) keep their own local golden helper

### Lint Conventions

- **Always `nix fmt` BEFORE placing `//nolint` directives** — golines (max-len: 120) reformats long lines and moves nolint comments to wrong positions
- For `gosec` G115 (integer overflow) conversions, extract a helper function that isolates the `uint64()`/`uint32()` call on a short single line
- Keep `//nolint` comments under ~40 chars to survive formatting
- When adding new dependencies, add them to `.golangci.yml` depguard allow list at the same time

## Dependencies

| Category   | Packages                                                                                                                                                                                                                                         |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Production | oklog/ulid/v2, go-branded-id, go-error-family, samber/ro (event, command, query); go-faster/yaml (catalog); go.opentelemetry.io/otel (otel, event, storage, middleware, projection); golang.org/x/crypto (encryption); fxamacker/cbor/v2 (codec) |
| Test-only  | onsi/ginkgo/v2, onsi/gomega, pgregory.net/rapid (event, encryption)                                                                                                                                                                              |

**Coverage**: 84–100% across 32 packages. See `docs/status/` for latest.

**Module Graph**:

```
Layer 0: id/, dispatcher/, codec/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher, ro), query/ (→dispatcher, ro)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot)
Layer 4: memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability
```

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from projection + command dispatch. See `example/todo/` for a real projection-based architecture.

**v2.1.0 Released**: Performance-focused release with 62 commits since v2.0.0. Major perf improvements (alloc reductions across event/signing/listing/catalog/memory), production bug fixes (HealthCheck OOM, race conditions, closed state tracking), new `query.TypedHandler[Q, R]`, and comprehensive benchmarking infrastructure. All 22 library + 2 cmd modules tagged at v2.1.0 with `/v2` semantic import paths. Replace directives in go.mod files are retained for `GOWORK=off` per-module CI (ignored by consumers). Consumers import via `github.com/larsartmann/go-cqrs-lite/event/v2` etc.

**v2.2.0 Released**: Operational readiness, testing rigor, and developer experience release with 81 commits since v2.1.0. Health check/metrics/SSE middleware, config loader, graceful shutdown, Docker packaging, property-based tests (rapid), snapshot tests, simulation framework, benchmark baseline regression detection in CI, gosec security scanning, module layer architecture checks, module READMEs, and doc.go with pkg.go.dev examples across 12 modules.

**v2.3.0 Released**: Lint hygiene, coverage, and release readiness with 231 commits since v2.2.0. Zero lint issues across all 27 modules, CBOR codec + Pebble CBOR envelope, encryption module (XChaCha20-Poly1305 + AES-256-GCM), phantom types across library, command store interfaces, OTel abstraction via `otel/` re-exports, ADR-0008–0015, comprehensive fuzz/property/snapshot testing, storage/sql coverage 37.4%→89.2%, otel 73.0%→97.3%.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).

> **Schema evolution**: Upcaster and VersionedStore moved to `schema/` module. See `schema/` package.
> **Snapshot persistence**: Snapshot types moved to `snapshot/` module. See `snapshot/` package.
