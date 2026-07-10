# Project: go-cqrs-lite

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import modules (`event`, `command`, `decider`, `storage`, `memory`, `catalog`, etc.) into THEIR projects.
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

> **AI consumer guide:** [`SKILL.md`](SKILL.md) is the canonical reference for AI agents using this library — module decision matrix, composition recipes, conventions, and anti-patterns. This file (AGENTS.md) is for contributors working **inside** the repo.

## Quick Reference

| Item      | Value                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Language  | Go 1.26.3                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Modules   | `event`, `command`, `query`, `decider`, `deriver`, `id`, `id/idtest`, `idempotency`, `metadata`, `query/querytest`, `dispatcher`, `schema`, `snapshot`, `catalog`, `middleware`, `integration`, `storage`, `storage/memory`, `storage/pebble`, `storage/turso`, `signing`, `encryption`, `otel`, `prometheus`, `watermill`, `transport/http`, `transport/grpc`, `codec`, `kv`, `kv/viewstoretest`, `listing`, `graph`, `projection`, `projectionhost`, `scenario`, `scheduling`, `testutil`, `dedup`, `cqrs-gen`, `api-stability`, `doc-check`, `stack`, `stack/memory`, `stack/sqlite`, `stack/turso`, `stack/pebble`, `stack/postgres`, `stack/bench`, `example/taskmanager`, `example/getting-started`                                                                                                                      |
| Build     | `nix run .#build`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Test      | `nix run .#test` or `go test ./event/... ./command/... ./idempotency/... ./query/... ./decider/... ./deriver/... ./id/... ./metadata/... ./dispatcher/... ./schema/... ./snapshot/... ./storage/memory/... ./catalog/... ./middleware/... ./integration/... ./signing/... ./encryption/... ./storage/... ./storage/pebble/... ./storage/turso/... ./watermill/... ./transport/http/... ./transport/grpc/... ./codec/... ./kv/... ./listing/... ./projection/... ./projectionhost/... ./scenario/... ./scheduling/... ./testutil/... ./dedup/... ./cmd/cqrs-gen/... ./cmd/doc-check/... ./prometheus/... ./otel/... ./stack/... ./stack/memory/... ./stack/sqlite/... ./stack/turso/... ./stack/pebble/... ./stack/postgres/... ./stack/bench/... ./example/taskmanager/... ./example/getting-started/... ./graph/... -count=1` |
| Lint      | `nix run .#lint`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Format    | `nix fmt`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Dev shell | `nix develop`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| CI        | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 48 `go.mod` files — all wired into `go.work`. Breakdown: 35 library + 7 stack presets + 2 examples + 3 cmd + 1 integration. Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`:

```
go-cqrs-lite/
├── event/               # EventSink, EventSource, Store, Journal, SeekableJournal, Bus, ImmutableEvent, NewEvent, Clone
│   └── v3/eventtest/    # FakeStore, FakeBus, FakeSnapshotStore, event factories, test assertions
├── command/             # Dispatcher, Handler, Middleware, Command, BasicCommand, PersistedCommand
│                        # Store: CommandSink, CommandSource, Store, CommandJournal, SeekableCommandJournal
│                        # Bus: Publisher, Subscriber, Bus, PublishMiddleware (command pub/sub)
├── query/               # Dispatcher, Handler, Pagination, PaginatedResult[T], RegisterTyped[Q,R], TypedHandler[Q,R]
│                        # Store: PersistedQuery, QuerySink, QuerySource, QueryStore, QueryJournal, SeekableQueryJournal
│   └── querytest/       # New(tb, queryType) test helper — tb.Fatalf on error, no panics
├── decider/             # Decider[State], Repository[State], Execute, Load (pure-function style)
├── deriver/             # Event→command derivation: Deriver, Then, Filter, Idempotent, AsHandler (ADR-0040)
├── id/                  # Branded IDs: id.Of[T] = cbid.ID[T, ulid.ULID], AggregateID, EventID, etc.
│   └── idtest/          # Parse*(tb, s) test helpers — tb.Fatalf on error, no panics
├── metadata/            # Tracing, CustomData[K] (extracted from event/ — shared metadata types for command/query/event)
├── idempotency/         # Dedup store: Store, MemoryStore, KVStore, ErrDuplicate (dedup for at-least-once delivery)
├── dispatcher/          # Generic Dispatcher[H, M] with LifecycleMixin
├── schema/              # Upcaster, VersionedStore, VersionedSeekableJournal, upcasterRegistry (schema evolution); Validator with RegisterType[T]() (ADR-0017)
├── snapshot/            # Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
├── storage/memory/       # MemoryStore, MemorySnapshotStore, MemoryCheckpointStore, MemoryCommandStore, MemoryQueryStore (in-memory test impls)
├── catalog/             # Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters
│   └── schema/          # JSON Schema types, reflection engine, YAML serialization
├── middleware/           # Logging, Retry, Recovery, Validation, Idempotency, Metrics, OTel Tracing+Metrics (command+event+query)
├── signing/             # Event signing/verification: HMAC-SHA256, Ed25519, multisig, middleware
├── encryption/          # Event payload encryption: XChaCha20-Poly1305, AES-256-GCM, codec wrapper, middleware
├── storage/             # SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLCommandStore, SQLQueryStore (PG/SQLite/Turso), SQLKVStore, SQLBackend facade, SQLite/PG helpers
│   ├── sql/             # Dialect, DBHandle, OwnedDBHandle, QueryEngine, RunInTx, IsDuplicateKeyError (typed codes + string fallback), ScanSlice, CommitTx, MarshalMetadata
│   ├── relational/      # RelationalSchema, RelationalProjection, RelationalStore, ProjectionSink (multi-table SQL projections, ADR-0033)
│   ├── view/            # SQLViewStore[V,K] (column-mapped views), ViewMapper, ViewColumn, IndexSpec, AutoMapper
│   ├── migrations/      # Embedded .sql DDL files (postgres.sql, sqlite.sql) via //go:embed
│   ├── pebble/          # Embedded KV store (PebbleDB): EventStore, SnapshotStore, CheckpointStore, KVAdapter (kv.Store). CBOR envelope, shared DB
│   └── turso/           # Turso database connector (embedded Turso Database sync), indexing advisor
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── prometheus/         # OTel→Prometheus metrics bridge: Setup() MeterProvider + /metrics HTTP handler, WithRegistry(), WithViews()
├── listing/             # AggregateListing, AggregateStatus, tombstone detection, StatusMiddleware, InMemoryAggregateReader
├── watermill/           # Watermill adapter: event AND command bridges — EventBus, CommandBus, PublisherAdapter, SubscriberAdapter, EventPublisher, CommandPublisher, CatchUpSubscriber (replay+live+checkpoint), MessageToEvent/MessageToCommand
├── transport/http/       # SSE event delivery: SSEBroker, SSEHandler (bridges event.Bus to HTTP clients, ADR-0025)
├── transport/grpc/       # gRPC transport: RegisterCommandService, RegisterQueryService, CommandClient, QueryClient (ADR-0025)
├── codec/               # Payload encoding: JSON, CBOR (deterministic), Raw passthrough
├── graph/               # Graph projection tier: NodeRef, EdgeRef, GraphSink, GraphDriver, GraphProjection, MemoryDriver, Schema (closed-world validation), ReadableDriver (Query/Traverse/Neighbors/ShortestPath) (ADR-0033, ADR-0039)
├── projection/          # Projection interface (consumer-side): Projection, NewProjection — extracted from event/
├── projectionhost/      # Managed projection host: Host, Worker, DeadLetterStore, Reset/Resettable — composes journal + projection + checkpoint + DLQ into a crash-restart lifecycle with OTel tracing, OnFailed callbacks, bounded dedup ring, WorkerDraining status, configurable shutdown timeout (framework gap A1, ADR-0030)
├── scheduling/          # Durable deadline timers: TimerStore, MemoryTimerStore, Scheduler (idempotent schedule/due/fired; "cancel order after 30 min") (framework gap A6)
├── kv/                  # Layer-0 KV store abstraction: Store, MemStore, Iterator, Batch. PLUS TypedStore[T,K], Cache[T,K], ViewStore[V,K] interface, ViewQuery, ViewQuerier, TombstoneQuerier
├── testutil/            # Shared test helpers: NewCmd(tb, ...) (cross-module test utilities)
├── dedup/               # Bounded dedup ring buffer: Ring, DefaultCapacity (O(1) fixed-capacity ID dedup for stream boundaries)
├── scenario/            # Fluent BDD test DSL: Given/When/Then + ThenError/ThenState for deciders, GivenProjection/ThenNoError for projections (framework gap A5)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── cmd/api-stability/   # API surface checker: compares exported symbols against golden file
├── integration/         # Cross-module tests (command, event, query, signing, encryption)
├── example/taskmanager/    # Flagship full HTTP service: event sourcing, CQRS, projections, middleware, OTel, signing
└── example/getting-started/ # Minimal 80-line example showing the core pipeline
└── docs/                # Status reports, ADRs, architecture patterns, storage guide
```

## AI Skill (Crush)

The repo ships a [Crush](https://github.com/crush) skill at `.agents/skills/go-cqrs-lite/` — the **single source of truth for AI consumers** of this library. It replaces the need to read 28 module READMEs.

- **`SKILL.md`** (root, a symlink to `.agents/skills/go-cqrs-lite/SKILL.md`) — keeps all existing doc links working.
- **Core skill** (`<400 lines`): mental model, module decision matrix, conventions, anti-patterns, cheat sheet, FAQ. Loaded on every trigger.
- **`references/`** — loaded on demand: `recipes.md` (composition), `readmodels.md` (projections/SQL views), `advanced.md` (14 advanced patterns), `modules.md` (per-module table).

**Global availability:** the skill is symlinked into `~/.config/crush/skills/go-cqrs-lite` (reproducibly, via the `flake.nix` devShell `shellHook`) so it triggers from any consumer project, not just inside this repo.

**Contributing:** edit the `.md` files under `.agents/skills/go-cqrs-lite/`, then run `cmd/doc-check` to verify every Go import path + qualified symbol is still valid:

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md
```

## Design Principles

1. **Library, not framework** — Consumers import what they need. No opinionated transport, broker, or SQL driver.
2. **Trustworthy modules** — Quality gate: "Would a consumer trust this enough to import it?"
3. **Minimal dependencies** — event depends on `oklog/ulid`, `go-branded-id`, `go-error-family`.
4. **Composition over inheritance** — Per Go best practices.
5. **Interface-first design** — Core store/bus types are interfaces (Store = EventSink + EventSource, ISP split). `event.Event` is a concrete type alias (`= *ImmutableEvent`) — the single implementation, so the interface was removed to eliminate type assertions on every internal hot path.
6. **Interface Segregation** — Journal (ReadAll), SeekableJournal (ReadFrom), BackwardsSource.
7. **Context-aware** — All handlers accept `context.Context`.
8. **Errors as values** — No panics, explicit error returns, sentinel errors + `%w` wrapping.
9. **Strong types** — No `any` as a value type in domain/business logic. Legitimate exceptions: JSON schema serialization (`catalog/`: `map[string]any`, `Payload any`), `recover()` return value (`middleware/recovery.go`), and `database/sql` interop (`storage/sql/dialect.go`). Generic type constraints (`[T any]`) are standard Go and always allowed. Max 350 lines/file (CI-enforced), 30 lines/function.
10. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
11. **Tombstone over delete** — Soft-delete via metadata (TombstoneStatus: Active/Tombstoned/Undetermined). No Delete on Store.
12. **Dependency budgets** — Per-module direct PRODUCTION dep limits enforced by `nix run .#check-layers`. Test-only packages (gomega, ginkgo, rapid) are excluded from the count. Adding production deps requires explicit budget review.
13. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, storage, middleware go.mod files. The `otel/` module re-exports `Int64Counter`, `AddOption`, `AddSpanEvent()`, `ServiceResourceAttributes()`, `CQRSHistogramBoundaries`, `NewCQRSViews()`, `CounterAddWithAttributes()`, `Setup()`, `WithStdoutExporter()`, `TextMapPropagator()`, and `Version()` for provider setup, stdout exporters, W3C propagation, and instrumentation scope versioning. Span names follow the `{component}.{action}` convention — see `docs/SPAN_NAMING.md`.
14. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths by accessing the `*ImmutableEvent` field directly (Event is now a concrete type alias, so no assertion is needed). Used by signing (SHA-256 hashing, CloneEvent), pebble (json.Marshal), storage/sql (ExecContext), transport/http/sse (string conversion). Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
15. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake. The `Event` interface documents this contract for third-party implementors.
16. **Hot-path zero-allocation discipline** — Public API clones stay, but internal hot paths eliminate allocs via: lazy map init (`NewMetadata()` returns zero-value), pre-computed middleware chains (EventBus rebuilds on `Use()`/`UsePublish()` only), cached SQL templates (built once at construction), pre-sized result slices (`make([]T, 0, hint)`), lock-free fast paths (`CircuitBreaker` uses `atomic.Int32`), batch SQL inserts (multi-VALUES with SQLite 999-param chunking). **Lesson learned**: type assertions for fast paths are dead code if users create types via different constructors. Cache at the integration boundary instead.
17. **Load coalescing via singleflight** — `decider.Repository[State]` uses `singleflight.Group` to coalesce concurrent `Load` calls for the same aggregate into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice across callers is safe. Only load is coalesced — Save/Publish still execute independently per caller. Disable via `WithLoadCoalescing[State](false)`.
18. **Go experimental build tags** — Builds use `-tags "goexperiment.arenas goexperiment.jsonv2"` enabling: arena allocations (memory pooling via `event/arena_experiment.go`) and JSON v2 encoding (via `codec/jsonv2_experiment.go`). These are Go experiment flags, NOT standard build tags — they require `GOEXPERIMENT` support in the toolchain. CI and `nix run .#build` apply them automatically. Dead tags (goroutineleakprofile, runtimesecret, simd) were removed — they gated zero files.

## Error Handling

- **Sentinel errors**: `errors.New` in `errors.go` files
- **Contextual errors**: `fmt.Errorf("failed to process %s: %w", name, err)`
- **Classified errors**: `errorfamily.NewRejection(...)`, `errorfamily.WrapConflict(...)` via [go-error-family](https://github.com/larsartmann/go-error-family) — imported directly, no facade
- **5-family taxonomy**: Rejection / Conflict / Transient / Infrastructure / Corruption
- **Direct import**: All modules import `errorfamily "github.com/larsartmann/go-error-family"` directly. The `event/` package retains type aliases (`event.Family`, `event.Error`) and family constants for backward compatibility, but error construction/classification/wrapping functions were removed. Use `go-error-family` directly.

## Key Patterns

```go
// Event creation (typed payload, auto-marshaled)
evt, err := event.NewEvent("user.created", userID, "User", event.Version(1),
    UserCreated{Name: "Alice"}, event.WithCorrelationID(correlationID))

// Decider (pure-function aggregate)
decider := decider.Decider[State]{Initial: State{}, Fold: foldFunc}
result, err := decider.Repository[State].Execute(ctx, repo, aggregateID, decider, command)

// Branded IDs (markers are exported: UserMarker, CorrelationMarker, RequestMarker, AggregateMarker)
type UserID = id.Of[id.UserMarker]
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

// Command & query persistence (audit trail)
//   cmd, _ := command.NewPersistedCommand("user.create", ref, payload)
//   cmdStore.Save(ctx, ref, cmd)         // CommandSink
//   cmds, _ := cmdStore.Load(ctx, ref)   // CommandSource
//   var journal command.CommandJournal = cmdStore        // ReadAll (global)
//   var seekable command.SeekableCommandJournal = cmdStore // ReadFrom(afterCmdID, limit)
//
//   pq, _ := query.NewPersistedQuery("user.get", payload)
//   qStore.SaveQuery(ctx, pq)            // QuerySink
//   queries, _ := qStore.LoadQueries(ctx, after) // QuerySource
//   var qJournal query.QueryJournal = qStore             // ReadAllQueries
//   var qSeekable query.SeekableQueryJournal = qStore     // ReadQueriesFrom(afterReqID, limit)

// Command bus (pub/sub) — typed subscription dispatch
//   bus := command.NewMemoryBus()
//   bus.Subscribe("user.create", handlerFunc)  // typed subscription
//   bus.SubscribeAll(auditHandler)             // catch-all (audit log)
//   bus.Use(middleware.CommandTracing(tracer)) // middleware chain
//   bus.Publish(ctx, cmd1, cmd2)               // variadic publish

// Event causality (command → event traceability)
//   ctx = event.WithCommandCausality(ctx, "user.create", cmdID)
//   // decider.Repository applies CommandCausalityEnricher(ctx) automatically;
//   // resulting events carry metadata.command.type and metadata.command.id
//   cmdType, cmdID, ok := event.CommandCausalityFromContext(ctx)

// Pebble single-DB full stack — PebbleBackend facade (preferred)
//   backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
//   defer backend.Close() // closes DB AND all stores
//   eventStore := backend.EventStore()
//   snapStore  := backend.SnapshotStore()
//   cpStore    := backend.CheckpointStore()
//   // All three share db via disjoint key prefixes (cqrs_event:, cqrs_snapshot:, cqrs_checkpoint:)
//
// Pebble single-DB manual wiring (advanced)
//   db, _ := pebble.Open(dir, &pebble.Options{})
//   eventStore, _ := pebble.NewStore(db, logger)
//   snapStore, _  := pebble.NewSnapshotStore(db, logger)
//   cpStore, _    := pebble.NewCheckpointStore(db, logger)

// Pebble as kv.Store (generic KV interface, ADR-0023)
//   kvStore, _ := pebble.NewKVStore(db, pebble.WithSyncWrites())
//   defer kvStore.Close()
//   kvStore.Set([]byte("k"), []byte("v"))    // → nil
//   val, _ := kvStore.Get([]byte("k"))        // → "v"
//   batch, _ := kvStore.Batch()               // atomic writes
//   // WithBorrowedDB() = adapter doesn't close the DB (shared via Backend)

// Event upcasting (schema migration on load)
//   upcaster := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
//       return event.NewEvent(evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
//           newPayload, event.WithSchemaVersion(2))
//   })
//   versioned := schema.NewVersionedStore(store, upcaster)
//
//   // VersionedSeekableJournal — upcasters for projectionhost (ADR-feedback-gap1)
//   // VersionedStore implements EventSource, but projectionhost.New() takes
//   // SeekableJournal. VersionedSeekableJournal bridges this gap.
//   vjournal, _ := schema.NewVersionedSeekableJournal(store, upcaster)
//   host, _ := projectionhost.New(vjournal, cpStore)
//   events, _ := versioned.Load(ctx, id.NewAggregateRef("User", aggregateID))

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

// OTel one-call setup + bundle (preferred for new code)
//   provider, _ := cqrsotel.Setup(
//       cqrsotel.WithService("my-app", "1.0.0", "instance-1"),
//       cqrsotel.WithStdoutExporter(os.Stdout), // dev; swap for OTLP in prod
//   )
//   defer provider.Shutdown(ctx)
//   bundle, _ := middleware.NewOTelBundle(
//       cqrsotel.NewTracer("my-app"), cqrsotel.NewMeter("my-app"),
//   )
//   cmdDisp.Use(bundle.Command()...)
//   bus.Use(bundle.Event()...)
//   bus.UsePublish(bundle.Publish()...)
//   qryDisp.Use(bundle.Query()...)
//   // Tracing-only: pass nil meter + middleware.WithMetricsDisabled()
//
//   // otel.Setup + prometheus.Setup composition (tracing + Prometheus metrics):
//   // Call otel.Setup for tracing, then override the meter provider with
//   // prometheus.Setup, passing CQRS views so histogram boundaries are applied.
//   tracingProvider, _ := cqrsotel.Setup(cqrsotel.WithService("app", "1.0", "i1"))
//   defer tracingProvider.Shutdown(ctx)
//   metricsProvider, _ := cqrsprometheus.Setup(
//       cqrsprometheus.WithViews(cqrsotel.NewCQRSViews()...),
//   )
//   defer metricsProvider.Shutdown(ctx)
//   otel.SetMeterProvider(metricsProvider.AsMeterProvider())

// OpenTelemetry tracing (opt-in, no-op when no provider configured)
//   tracer := otel.GetTracerProvider().Tracer("my-app")
//   bus.Use(middleware.EventTracing(tracer))
//   bus.UsePublish(middleware.EventPublishTracing(tracer))
//   cmdDispatcher.Use(middleware.CommandTracing(tracer))

// OpenTelemetry metrics (opt-in, replace custom MetricsRecorder)
//   meter := otel.GetMeterProvider().Meter("my-app")
//   recorder, _ := middleware.NewOTelMetricsRecorder(meter)
//   cmdDispatcher.Use(middleware.CommandMetrics(recorder))
//
//   // Rate metrics (Int64Counter + Float64Histogram)
//   counter, _ := meter.Int64Counter("cqrs.operation.count")
//   cmdDispatcher.Use(middleware.CommandOTelMetricsWithCounter(histogram, counter))
//
//   // Span events for projection retry observability
//   cqrsotel.AddSpanEvent(span, "retry_attempt", cqrsotel.AttrInt("attempt", 2))
//
//   // Service identification in traces
//   attrs := cqrsotel.ServiceResourceAttributes("my-app", "1.0.0", "instance-1")
//
//   // Custom histogram boundaries for CQRS latency ranges
//   _ = cqrsotel.CQRSHistogramBoundaries // [0.05, 0.1, ..., 10000] ms

// CBOR compact codec (opt-in, ~35% smaller payloads via toarray)
//   codec := codec.CBORCompactCodec{}  // NOT compatible with CBORCodec data
//   data, _ := codec.Encode(event)     // struct fields encoded as positional array
//
//   // Human-readable CBOR for debugging
//   diag, _ := codec.Diagnose(cborData)
//   log.Printf("CBOR: %s", diag)
//
//   // toarray struct tag — positional CBOR arrays (30-40% payload reduction)
//   type UserCreated struct {
//       _     struct{} `cbor:",toarray"`
//       Name  string
//       Email string
//   }
//
//   // Streaming CBOR encoder/decoder (large batches without materializing)
//   enc := codec.NewCBOREncoder(w)
//   _ = enc.Encode(event)
//   dec := codec.NewCBORDecoder(r)
//   _ = dec.Decode(&event)
//
//   // Zero-allocation encoding via BufferEncoder interface
//   buf := &bytes.Buffer{}
//   if be, ok := codec.(codec.BufferEncoder); ok { be.EncodeToBuffer(payload, buf) }

// Stack-level CBOR default (one-call adoption for all read models)
//   bundle, _ := sqlite.New(dsn, stack.WithDefaultCodec(codec.CBORCodec{}))
//   store, _ := stack.ReadModel[Todo, TodoID](bundle, nil) // nil → uses CBOR
//   mat, _ := stack.NewMaterialize[Todo, TodoID](bundle, nil, keyFunc)   // nil → uses CBOR
//
// ⚠️ CODEC DEFAULTS (read this if you're debugging encoding issues):
//   The default codec differs by layer:
//
//   LAYER                     | DEFAULT CODEC     | HOW TO OVERRIDE
//   --------------------------|-------------------|----------------------------------
//   stack.ReadModel/Materialize| CBORCodec         | stack.WithDefaultCodec(json)
//   event.New()               | JSONCodec         | event.DefaultCodec = codec.CBORCodec{}
//                             |                   |   or event.WithCodec(c) per-event
//   kv.NewTypedStore()        | CBORCodec (v4)    | kv.WithTypedCodec(c)
//   snapshot.NewTypedStore()  | CBORCodec (v4)    | positional arg: NewTypedStore(store, c)
//   command typed store       | CBORCodec (v4)    | positional arg: NewTypedCommandStore(store, c)
//   query typed store         | CBORCodec (v4)    | positional arg: NewTypedQueryStore(store, c)
//
//   Events are SELF-DESCRIBING: evt.Encoding() is stamped on every event,
//   so mixed JSON+CBOR event streams decode correctly via DecodePayloadAuto.
//   Blind stores (kv/snapshot/command/query) are now self-describing too via
//   the ADR-0044 envelope: WrapEncode/UnwrapDecode stamp the codec on write and
//   auto-detect it on read. The UnwrapDecode fallback uses JSONCodec for
//   backward compat with pre-envelope data.
//
//   One-call CBOR for both events AND read models:
//   bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))
//   // Then in decide functions: event.WithCodec(bundle.EventCodec())

// Mixed-stream decode — JSON and CBOR events in the same store (ADR-CODEC)
//   // DecodePayloadAuto dispatches based on each event's encoding stamp.
//   // Use this in fold/apply functions when migrating from JSON to CBOR:
//   p, err := event.DecodePayloadAuto[UserCreated](evt)
//   // Internally: codec.ForEncoding(evt.Encoding()) picks JSON or CBOR.
//   // For unknown encodings ("raw", "encrypted"), returns an error.

// Pebble recommended defaults (bloom filter, concurrent compactions, logging)
//   backend, _ := pebble.Open(dir, pebble.DefaultOptions(), logger)
//   // Or with operational logging:
//   backend, _ := pebble.Open(dir, pebble.DefaultOptionsWithLogging(logger), logger)
//
//   // LSM metrics for health checks
//   metrics := backend.Metrics()
//   hitRate := float64(metrics.BlockCacheHits) /
//       float64(metrics.BlockCacheHits + metrics.BlockCacheMisses)

// SQLite busy_timeout (eliminates "database is locked" errors)
//   _ = storage.SQLiteEnableWAL(ctx, db)  // now includes busy_timeout=5000

// SQL backend facade — all stores share one *sql.DB connection
//   backend, _ := storage.NewSQLiteBackend(db)  // or NewSQLBackend for Postgres
//   eventStore  := backend.EventStore()                   // *SQLEventStore (eager)
//   cmdStore, _ := backend.CommandStore()                 // *SQLCommandStore (lazy, goroutine-safe)
//   qStore, _   := backend.QueryStore()                   // *SQLQueryStore (lazy, goroutine-safe)
//   snapStore,_ := backend.SnapshotStore()                // *SQLSnapshotStore (lazy, goroutine-safe)
//   cpStore, _  := backend.CheckpointStore()              // *SQLCheckpointStore (lazy, goroutine-safe)
//   defer backend.Close()                                 // closes all stores (NOT the *sql.DB)
//   // Each store embeds *sqlpkg.OwnedDBHandle for Close/checkClosed lifecycle

// SQLite foreign keys (opt-in referential integrity)
//   _ = storage.SQLiteEnableForeignKeys(ctx, db)  // PRAGMA foreign_keys=ON

// HKDF key derivation (multi-tenant encryption)
//   key, _ := encryption.DeriveKey(masterKey, "tenant:acme", 32)  // HKDF-SHA256
//   enc, _ := encryption.NewXChaCha20Poly1305(key)

// Decider singleflight (concurrent load coalescing)
//   // Repository[State] uses singleflight.Group internally — concurrent Load
//   // calls for the same aggregate coalesce into one store.Load query.
//   // No API change needed; it's transparent.

// OTel distributed correlation (baggage propagation)
//   ctx = cqrsotel.WithCorrelationID(ctx, "abc-123") // store in baggage
//   corrID := cqrsotel.CorrelationIDFromContext(ctx)  // retrieve from baggage
//   otel.SetTextMapPropagator(cqrsotel.NewTextMapPropagator()) // W3C trace + baggage

// Watermill middleware wrappers
//   router.AddMiddleware(watermill.CorrelationIDMiddleware())
//   router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultRetryConfig()))
//   router.AddMiddleware(watermill.ProcessingModeMiddleware()) // reconstruct replay/live flag

// Pebble backup and consistent reads
//   backend.Checkpoint("backups/2026-06-17")         // point-in-time DB snapshot
//   snap := backend.NewSnapshot(); defer snap.Close() // consistent read view

// Codec zero-allocation encoding (BufferEncoder)
//   buf := &bytes.Buffer{}
//   if be, ok := codec.(codec.BufferEncoder); ok { be.EncodeToBuffer(payload, buf) }

// Typed Metadata fields (ADR-0031)
//   // Tracing is embedded in event.Metadata — field promotion keeps JSON shape
//   md := evt.Metadata()
//   fmt.Println(md.CorrelationID)  // promoted from Tracing
//   if md.Tombstone != nil { ... } // typed tombstone mark
//   if md.Causation != nil { ... } // typed command causation

// TypedDecider[State, Cmd] — command type bound at compile time (ADR-0001)
//   d := decider.TypedDecider[CounterState, IncrementCmd]{
//       Initial: CounterState{},
//       Decide:  decideIncrement,
//       Fold:    foldCounter,
//   }
//   repo, _ := decider.NewTypedRepository(store, bus, d)
//   err := repo.ExecuteCommand(ctx, aggID, "Counter", IncrementCmd{Amount: 5})

// kv.TypedStore and kv.Cache (ADR-0032 — moved from readmodel)
//   store := kv.NewTypedStore[UserView, UserID](kvBackend)
//   cache, _ := kv.NewCache[UserView, UserID](store, kv.WithCacheCapacity(500))

// SQL-backed views with queryable columns (storage.SQLViewStore)
//   mapper := storage.ViewMapper[TodoView]{
//       Table: "todos_view",
//       Columns: []storage.ViewColumn[TodoView]{
//           {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
//           {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
//       },
//       ScanRow: func(scan func(dest ...any) error) (*TodoView, error) { ... },
//       TombstoneColumn: "tombstoned", // optional: server-side tombstone filtering
//   }
//   store, _ := storage.NewSQLiteViewStore[TodoView, id.AggregateID](db, mapper)
//   mat := stack.Materialize[TodoView, id.AggregateID]{Store: store, ...}
//   // Query with SQL power: WHERE, ORDER BY, LIMIT/OFFSET
//   results, _ := store.Query(ctx, kv.ViewQuery{
//       Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
//   })
//
//   // Advanced capabilities (optional interfaces — checked at runtime):
//   // Count:        store.Count(ctx, kv.ViewQuery{Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}}})
//   // BatchSet:     store.BatchSet(ctx, items) // chunked upsert (SQLite 999-param aware)
//   // DeleteAll:    store.DeleteAll(ctx)       // DELETE FROM table (projection reset)
//   // Query:        store.Query(ctx, kv.ViewQuery{Conditions: []kv.Condition{
//   //                   {Column: "completed", Op: kv.OpEq, Value: false}}, OrderBy: "title", Limit: 10})
//   // AutoMapper:   storage.AutoMapperWithTombstone[TodoView]("todos", "tombstoned") // from struct tags
//   // Indexes:      ViewMapper.Indexes = []storage.IndexSpec{{Name: "idx_title", Columns: []string{"title"}}}
//
//   // From a Bundle preset (one-call path):
//   //   store, _ := sqlite.SQLViewModel[TodoView, TodoID](bundle, mapper)

// Relational projections — multi-table, SQL-dialect-independent (storage.RelationalProjection)
//   // NOTE: This tier is SQL-ONLY (SQLite/Postgres/MySQL), portable at deployment
//   // via the dialect — NOT portable to KV or Graph. Row/column/table/set-predicate
//   // semantics are relational by design. For KV/document backends use
//   // stack.Materialize + kv.ViewStore[V,K] (one document per key). A graph tier
//   // would need a distinct sink (MergeNode/MergeEdge) — see RelationalSink docs.
//   // SQLViewStore/Materialize write ONE record to ONE table per event. When an
//   // event must update several related tables atomically (message + guild +
//   // channel + user + attachments[], a member_roles junction, an append-only
//   // message_edits history table), use RelationalProjection instead.
//   schema := storage.RelationalSchema{Tables: []storage.RelationalTable{
//       {Name: "messages", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "TEXT"}, {Name: "channel_id", Type: "TEXT"},
//           {Name: "content", Type: "TEXT"}, {Name: "created_at", Type: "TEXT"},
//       }},
//       {Name: "attachments", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "TEXT"}, {Name: "message_id", Type: "TEXT"}, {Name: "filename", Type: "TEXT"},
//       }},
//       // Junction table: composite primary key
//       {Name: "member_roles", PrimaryKey: []string{"guild_id", "user_id", "role_id"}, Columns: ...},
//       // Append-only history: autoincrement PK declared in Type, no PrimaryKey
//       {Name: "message_edits", Columns: []storage.RelationalColumn{
//           {Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT", Nullable: true},
//           {Name: "message_id", Type: "TEXT"}, {Name: "before_content", Type: "TEXT"},
//       }},
//   }}
//
//   // Handler is dialect-agnostic — never touches *sql.DB. Backend chosen at
//   // deployment via the dialect passed to NewRelationalProjection.
//   proj, _ := storage.NewRelationalProjection("discord-messages", schema, db, sqlpkg.SQLiteDialect{},
//       func(ctx context.Context, evt event.Event, sink storage.ProjectionSink) error {
//           var p MessageCreated
//           _ = json.Unmarshal(evt.Payload(), &p)
//           sink.Ensure(ctx, "channels", storage.Row{"id": p.ChannelID, "name": "", "created_at": p.CreatedAt})
//           sink.Upsert(ctx, "messages", storage.Row{  // conflict on PK "id"
//               "id": p.ID, "channel_id": p.ChannelID, "content": p.Content, "created_at": p.CreatedAt,
//           })
//           for _, a := range p.Attachments {
//               sink.Ensure(ctx, "attachments", storage.Row{"id": a.ID, "message_id": p.ID, "filename": a.Name})
//           }
//           return nil  // all writes commit atomically; error → full rollback
//       }, []event.Type{"MESSAGE_CREATED"})
//   // proj implements projection.Projection → register with any projection runner.
//
//   // Read side: dialect-agnostic queries (replaces hand-written SQL).
//   reader, _ := storage.NewRelationalStore(schema, db, sqlpkg.SQLiteDialect{})
//   counts, _ := reader.CountMany(ctx, []string{"messages", "channels", "users"}) // stats endpoint
//   _ = reader.Query(ctx, "messages", []string{"id", "content"}, kv.ViewQuery{
//       Conditions: []kv.Condition{{Column: "channel_id", Op: kv.OpEq, Value: chID},
//                                   {Column: "created_at", Op: kv.OpLt, Value: cursor}},
//       OrderBy: "created_at", Desc: true, Limit: 50,
//   }, func(scan func(...any) error) error { var r Row; return scan(&r.ID, &r.Content) })

// Graph projections — nodes + edges for traversal-heavy read models (graph.GraphProjection)
//   // The third projection tier. Where Materialize writes ONE document per key
//   // and RelationalProjection writes across SQL tables, GraphProjection merges
//   // events into nodes and edges — the right shape for variable-depth traversal,
//   // adjacency, path-finding, causation DAGs, reply chains, role memberships,
//   // reaction networks. Use when N-hop queries (recursive CTEs in SQL) dominate.
//   //
//   // Writes ARE portable across backends (openCypher MERGE semantics shared by
//   // Neo4j, Memgraph, Apache Age, RedisGraph). Reads are NOT abstracted — run
//   // native Cypher/Gremlin via the driver. This asymmetry is documented.
//   driver := graph.NewMemoryDriver() // or graph/neo4j.NewDriver(...) in sibling module
//   proj, _ := graph.NewGraphProjection("discord-graph", driver,
//       func(ctx context.Context, evt event.Event, sink graph.GraphSink) error {
//           var p MessageCreated
//           _ = json.Unmarshal(evt.Payload(), &p)
//           msgRef := graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ID}
//           sink.MergeNode(msgRef, map[string]any{"created_at": p.CreatedAt})
//           // Auto-creates endpoint nodes — handlers need not pre-merge.
//           sink.MergeEdge(graph.EdgeRef{Type: "AUTHORED_BY", From: msgRef,
//               To: graph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.AuthorID}}, nil)
//           // The recursive edge — relational tier needs WITH RECURSIVE CTE.
//           if p.ReplyToMessageID != "" {
//               sink.MergeEdge(graph.EdgeRef{Type: "REPLY_TO", From: msgRef,
//                   To: graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ReplyToMessageID}},
//                   map[string]any{"at": p.CreatedAt})
//           }
//           return nil // atomic: all merges commit or all roll back
//       }, []event.Type{"MESSAGE_CREATED"})
//   // proj implements projection.Projection → register with any projection runner.
//   // Reads: driver.Query/Traverse/Neighbors/ShortestPath (memory) or native Cypher (Neo4j).
//
//   // Graph Schema validation (opt-in, ADR-0039) — catch typos at the sink boundary
//   schema := &graph.Schema{
//       Nodes: []graph.NodeType{
//           {Label: "User", KeyProp: "id", Properties: []graph.PropertyType{{Name: "name"}}},
//           {Label: "Message", KeyProp: "id", Properties: []graph.PropertyType{{Name: "content"}}},
//       },
//       Edges: []graph.EdgeType{
//           {Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"},
//       },
//   }
//   proj, _ := graph.NewGraphProjection("graph", driver, handler, types, graph.WithSchema(schema))
//   // → MergeNode with Label "Bogus" now returns errSinkUnknownNodeLabel
//
//   // Graph Read API (MemoryDriver only — Go-native predicates, NOT a query language)
//   nodes := driver.Query(graph.Pattern{Label: "User", Where: func(p map[string]any) bool { return p["active"] == true }})
//   ancestors := driver.Traverse(msgRef, "REPLY_TO", -1) // BFS unlimited depth
//   neighbors, edges := driver.Neighbors(centerRef)      // 1-hop adjacency
//   path, _ := driver.ShortestPath(userA, userB)          // BFS shortest path

// gRPC transport (remote command/query dispatch, ADR-0025)
//   srv := grpc.NewServer()
//   cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)
//   cqrsgrpc.RegisterQueryService(srv, qDispatcher)
//   // Client:
//   conn, _ := grpc.NewClient("host:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
//   cmdClient := cqrsgrpc.NewCommandClient(conn)
//   err := cmdClient.Dispatch(ctx, cmd) // transparent remote dispatch

// Idempotency middleware (dedup for at-least-once delivery, all 3 CQRS message types)
//   store := idempotency.NewMemoryStore(5 * time.Minute)
//   defer store.Close()
//   cmds.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
//   bus.Use(middleware.EventIdempotency(store, 10*time.Minute, nil))
//   qry.Use(middleware.QueryIdempotency(store, 10*time.Minute, keyFn))
//   // nil keyExtractor defaults to cmd.ID().String() / evt.ID().String()
//   // For queries, a keyExtractor must be provided (no default identity)

// In-memory command bus (typed pub/sub, first command.Bus impl)
//   bus := command.NewMemoryBus()
//   bus.Subscribe("user.create", handlerFunc)
//   bus.Publish(ctx, cmd1, cmd2)

// Watermill CatchUpSubscriber (replay from journal + live handoff, ADR-0030)
//   catchUp, _ := watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)
//   defer catchUp.Close()
//   msgs, _ := catchUp.Subscribe(ctx, "user.created")
//   // Phase 1: replay historical events with ProcessingMode=ModeReplay
//   // Phase 2: live handoff with EventID-based deduplication
//   // Checkpoint saved after every forwarded event
//
//   // The synchronous replay path is ALWAYS ordered. The LIVE phase uses
//   // BlockPublishUntilSubscriberAck=true for ordered delivery.

// ⚠️ ORDERING — Watermill Router processes messages in parallel (one goroutine
//   per message, message/router.go:30). Do NOT route ordered projections
//   through the Router. Instead, consume the CatchUpSubscriber's output channel
//   from a single goroutine (FIFO guarantees ordering). The EventBus default
//   GoChannel uses BlockPublishUntilSubscriberAck=true + Persistent=false:
//   the former ensures ordered live delivery, the latter avoids GoChannel's
//   unordered Persistent-mode replay (CatchUpSubscriber handles replay from
//   the journal instead). See example/taskmanager for the correct pattern.

// stack.Materialize[V,K] — tombstone-aware projection builder (ADR-0030)
//   mat := stack.Materialize[UserView, UserID]{
//       Store:       kvStore,
//       KeyFromEvent: func(evt event.Event) (UserID, error) { ... },
//       OnCreate:    func(ctx, evt) (*UserView, error) { ... },
//       OnUpdate:    func(ctx, evt, existing *UserView) (*UserView, error) { ... },
//       OnTombstone: func(ctx, evt, existing *UserView) (*UserView, error) { ... },
//   }
//   router.AddNoPublisherHandler("users", topic, catchUpSub, mat.HandlerFunc())

// Watermill EventPublisher — cqrs events → Watermill topic (ADR-0028)
//   pub := watermill.NewEventPublisher(wmPublisher, "events")
//   repo, _ := decider.NewRepository(store, pub, decider)
//
//   // W3C trace context is injected into message metadata on publish.
//   // Extract on consume to link producer → consumer spans:
//   //   ctx := watermill.ExtractContext(msg.Context(), msg)
//   // Or add as router middleware: router.AddMiddleware(watermill.TraceContextMiddleware())

// Watermill CommandBus — command distribution over any broker (ADR-0025)
//   bus := watermill.NewCommandBus()  // GoChannel (single-process)
//   defer bus.Close()
//   bus.Subscribe("user.create", handlerFunc)
//   bus.Publish(ctx, cmd)
//
//   // Multi-process: inject a NATS/Redis/Kafka backend
//   bus := watermill.NewCommandBus(
//       watermill.WithCommandBackend(natsPub, natsSub, closer))
//
//   // Or wrap an existing message.Publisher as command.Publisher
//   pub := watermill.NewCommandPublisher(wmPublisher, "commands")

// Multi-DB SQLite preset (deployer chooses database isolation)
//   bundle, _ := sqlite.New(":memory:",
//       sqlite.WithEventDB("events.db"),   // events+snapshots+checkpoints
//       sqlite.WithQueryDB("queries.db"),  // command+query audit
//       sqlite.WithViewDB("views.db"),     // read-model KV store
//   )
//
// Multi-DB Postgres preset (same API, separate databases on same server)
//   bundle, _ := postgres.New(primaryDSN,
//       postgres.WithEventDB("postgres://host/events_db"),
//       postgres.WithQueryDB("postgres://host/queries_db"),
//       postgres.WithViewDB("postgres://host/views_db"))

// Pure event-sourcing mode (no publisher needed)
//   repo, _ := decider.NewRepository(store, nil, decider)
//   // Events are persisted but NOT published — for pure ES without a bus

// Pebble backup + graceful shutdown (production operations)
//   b, _ := pebble.New("/var/lib/myapp/pebble")
//   defer b.GracefulClose(ctx) // Bundle.GracefulClose bounds Close with a timeout
//   _ = b.Checkpoint("/backups/2026-06-21") // point-in-time physical snapshot
//   m := b.Metrics()                         // LSM health (block cache hit rate, etc.)

// SSE with Last-Event-ID reconnection (resilient event delivery)
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 1000)) // replay cap
//   // Clients sending "Last-Event-ID" header get missed events replayed
//   // from the journal before live streaming begins. Dedup prevents
//   // duplicate delivery (same strategy as CatchUpSubscriber).
//
//   // Unlimited replay + browser timeout safety:
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 0), // 0 = unlimited streaming
//       http.WithReplayTimeout(30*time.Second))     // stop replay after 30s
//   // replayLimit <= 0 streams ALL events in batches of 500 (memory-bounded).
//   // WithReplayTimeout caps the duration — on timeout, an
//   // SSEReplayIncompleteEvent advisory is sent before live starts.
//   // DefaultSSEReplayLimit (1000) is the suggested bounded cap.
//
//   // Byte-budgeted replay (prevent memory exhaustion from large payloads):
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithReconnectJournal(journalStore, 0),
//       http.WithReplayByteBudget(8*1024*1024))           // 8MB default
//   // WithReplayByteBudget(0) also defaults to 8MB (safety).
//   // http.SSEReplayBudgetDisabled = -1 to explicitly disable budgeting.
//
//   // Payload transform for non-JSON codecs (CBOR→JSON for browsers):
//   broker, _ := http.NewSSEBroker(bus,
//       http.WithPayloadTransform(func(evt event.Event) []byte {
//           p, err := event.DecodePayloadAuto[YourPayload](evt)
//           if err != nil { return event.PayloadReadOnly(evt) }
//           jsonBytes, _ := json.Marshal(p)
//           return jsonBytes
//       }))
//   // Without this, CBOR-encoded events go out as raw CBOR bytes that browsers
//   // cannot parse. Applied uniformly across live + replay paths.

// Managed projection host — the "last loop every consumer rewrites" (projectionhost)
//   host, _ := projectionhost.New(journal, checkpointStore,
//       projectionhost.WithBatchSize(100),
//       projectionhost.WithDeadLetterStore(dlqStore, 3), // poison after 3 retries
//       projectionhost.WithOnFailed(func(name, err string) { // terminal failure alert
//           alerting.Notify(ctx, fmt.Sprintf("projection %q failed: %s", name, err))
//       }),
//       projectionhost.WithShutdownTimeout(60*time.Second), // default 30s
//   )
//   _ = host.Register(&UserProjection{})  // Register returns error (name must be unique)
//   _ = host.Register(&OrderProjection{})
//   go host.Start(ctx)          // one goroutine per projection, crash auto-restart + backoff
//   defer host.Stop()           // graceful drain (WorkerDraining → WorkerStopped)
//   for _, s := range host.Status() {  // health: idle/running/live/backoff/draining/stopped/failed
//       fmt.Printf("%s: %s processed=%d errors=%d\n", s.Name, s.Status, s.Processed, s.Errors)
//   }
//   // Rebuild a projection from scratch after fixing a handler bug:
//   host.Stop()
//   host.Reset(ctx, "users") // drops checkpoint + calls Resettable.Reset if implemented
//   host.Start(ctx)           // replays from zero
//   // OTel tracing: automatic spans (projectionhost.handle_event) when provider configured
//   // Reads directly from event.SeekableJournal — no Watermill dependency.
//   // For live push delivery, pair with watermill/CatchUpSubscriber.
//
//   // SQLite dead-letter store (persists poison events across restarts):
//   dlqStore, _ := projectionhost.NewSQLiteDeadLetterStore(db)
//   host, _ := projectionhost.New(journal, cpStore,
//       projectionhost.WithDeadLetterStore(dlqStore, 3))

// Scenario-testing DSL — fluent Given/When/Then for deciders + projections (scenario)
//   scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{},
//       mustEvent(evtIncremented)).           // pre-existing events folded into state
//       When(incrementCmd{}, decideIncrement). // pure decide function
//       Then(evtIncremented)                   // asserts emitted event TYPES
//   // Also: .ThenError(target) / .ThenState(fold, initial, expected)
//   scenario.GivenProjection(t, proj, evt1, evt2).ThenNoError()  // projection handles all OK

// Scheduled commands / durable deadlines — "cancel order after 30 min unpaid" (scheduling)
//   store := scheduling.NewMemoryTimerStore()
//   sched := scheduling.New(store, dispatchFunc,
//       scheduling.WithPollInterval(500*time.Millisecond),
//       scheduling.WithMaxRetries(5),
//   )
//   _ = store.Schedule(ctx, scheduling.Timer{
//       ID: "order-123-timeout", FireAt: time.Now().Add(30 * time.Minute), Payload: cancelCmd,
//   })  // idempotent: re-scheduling the same ID is a no-op
//   _ = store.Cancel(ctx, "order-123-timeout") // order paid → cancel timeout
//   go sched.Start(ctx) // polls Due(), dispatches, MarkFired(); retries failed dispatches

// Health checks — Kubernetes liveness/readiness probes (stack.Bundle)
//   bundle, _ := sqlite.New(dsn)
//   if err := bundle.HealthCheck(ctx); err != nil {
//       // database unreachable or a registered resource is unhealthy
//   }
//   // SQLEventStore implements stack.HealthChecker via db.PingContext.
//   // Bundle.HealthCheck pings the DB + calls HealthCheck on every registered
//   // closer that implements the interface.

// Shutdown ordering — declare close-time dependencies (stack.Bundle)
//   bundle, _ := sqlite.New(dsn,
//       stack.WithShutdownDependency("eventstore", "projectionhost"),
//   )
//   defer bundle.Close() // topological sort: eventstore closes AFTER projectionhost
//   // Cycles fall back to registration order. Use for stores that projections
//   // read from — projections must drain before the event store closes.
```

## Testing

- Table-driven tests preferred; BDD via Ginkgo v2 + Gomega for event/decider/query; fluent Given/When/Then via `scenario/v3` (framework gap A5)
- `t.Parallel()` for independent tests; core packages >80% coverage (most >90%)
- Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`
- Golden tests use shared `eventtest.AssertGolden(t, path, got, *update)` from `event/v3/eventtest`
- Modules without event dependency (otel, codec) keep their own local golden helper
- **eventtest nested module**: `event/v3/eventtest` lives at `event/v3/eventtest/` (directory MUST match module path per Go spec — a path without a trailing `/vN` suffix requires `go.mod` at the exact subdirectory). `go mod tidy` in `event/` (or any consumer) emits warnings about the nested `go.mod`; run `go mod tidy -e` to suppress (warnings-only, not a build failure). Tagged as `event/v3/eventtest/v0.1.0` (v0 because the path's last element is `eventtest`, not `/vN`). See `docs/adr/0045-eventtest-module-path-fix.md` for the full rationale.

### Lint Conventions

- **Always `nix fmt` BEFORE placing `//nolint` directives** — golines (max-len: 120) reformats long lines and moves nolint comments to wrong positions
- For `gosec` G115 (integer overflow) conversions, extract a helper function that isolates the `uint64()`/`uint32()` call on a short single line
- Keep `//nolint` comments under ~40 chars to survive formatting
- When adding new dependencies, add them to `.golangci.yml` depguard allow list at the same time
- **SQL store helpers live in `storage/sql/`** — `RunInTx`, `IsDuplicateKeyError`, `CommitTx`, `ScanSlice`, `MarshalMetadata`. Don't duplicate transaction/duplicate-key logic in domain-specific store files. The `sql` package already imports `otel` for span recording.
- **`scanCommand` and `scanQuery` must unmarshal metadata** — both scan a `metadataJSON []byte` column. Use `json.Unmarshal` into `command.Metadata` / `query.Metadata` (aliases for `event.Metadata`), then pass via `WithCommandMetadata` / `WithQueryMetadata`. Forgetting this causes silent metadata loss on SQL load.

## Dependencies

| Category   | Packages                                                                                                                                                                                                                                             |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production | oklog/ulid/v2, go-branded-id, go-error-family, go-faster/yaml (catalog); go.opentelemetry.io/otel (otel, event, storage, middleware, prometheus); prometheus/client_golang (prometheus); golang.org/x/crypto (encryption); fxamacker/cbor/v2 (codec) |
| Test-only  | onsi/ginkgo/v2, onsi/gomega, pgregory.net/rapid (event, encryption)                                                                                                                                                                                  |

**Coverage**: core modules 84–98% (event 91.3%, decider 98.3%, id 97.6%, dispatcher 98.0%, schema 93.5%, storage/memory 94.1%, command 89.4%); mid-tier 81–84% (snapshot 81.1%, query 83.9%); newer modules 70–76% (kv 70.2%, codec 76.0%). Workspace total: 78.7%. Bundle layer (stack presets, cache) 0–87% — presets emphasise the shared contract suite + happy paths, so constructor error branches are lighter (stack/postgres shows 0% locally because its tests skip without `POSTGRES_TEST_DSN`). See `docs/status/` for latest.

**Module Graph** (four-tier model, see [ADR-0046](docs/adr/0046-four-tier-model.md) and [FOUR-TIER-MODEL.md](docs/architecture-understanding/FOUR-TIER-MODEL.md)):

```
Tier 0 — Primitives: id/, dispatcher/, codec/, kv/, dedup/
Tier 1 — Core Domain: event/, command/, query/, scheduling/, metadata/
Tier 2 — Domain Utilities: schema/, snapshot/, projection/, idempotency/, deriver/
Tier 3 — Aggregation: decider/, graph/, scenario/, projectionhost/, listing/
Tier 4 — Infrastructure: storage/memory/, storage/, middleware/, signing/, encryption/, otel/, watermill/, transport/http/, transport/grpc/, storage/pebble/, storage/turso/, prometheus/
Tier 5 — Composition: stack/, stack/memory/, stack/sqlite/, stack/pebble/, stack/postgres/, stack/turso/
Tier 6 — Tooling & Examples: catalog/, integration/, stack/bench/, examples/, cmd/*
```

> Note: the old 7-layer system (pre-ADR-0046) was inaccurate — kv/ depends on codec/, command/
> depends on event/, and 38 of 48 modules depend on codec/. The four-tier model reflects reality.

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from bus.SubscribeAll + command dispatch. See `example/taskmanager/` for a real architecture.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).

> **Schema evolution**: Upcaster and VersionedStore moved to `schema/` module. See `schema/` package.
> **Snapshot persistence**: Snapshot types moved to `snapshot/` module. See `snapshot/` package.
