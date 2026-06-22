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

| Item      | Value                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language  | Go 1.26.3                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Modules   | `event`, `command`, `query`, `decider`, `id`, `id/idtest`, `query/querytest`, `dispatcher`, `schema`, `snapshot`, `catalog`, `middleware`, `integration`, `storage`, `storage/memory`, `storage/pebble`, `storage/turso`, `signing`, `encryption`, `otel`, `prometheus`, `watermill`, `transport/http`, `codec`, `kv`, `listing`, `testutil`, `cqrs-gen`, `api-stability`, `stack`, `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`, `stack/bench`, `example/user`, `example/todo`, `example/encryption`, `example/deployer-first`                                                                                                        |
| Build     | `nix run .#build`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Test      | `nix run .#test` or `go test ./event/... ./command/... ./query/... ./decider/... ./id/... ./dispatcher/... ./schema/... ./snapshot/... ./storage/memory/... ./catalog/... ./middleware/... ./integration/... ./signing/... ./encryption/... ./storage/... ./storage/pebble/... ./storage/turso/... ./watermill/... ./transport/http/... ./codec/... ./kv/... ./listing/... ./testutil/... ./cmd/cqrs-gen/... ./prometheus/... ./otel/... ./stack/... ./stack/memory/... ./stack/sqlite/... ./stack/pebble/... ./stack/postgres/... ./stack/bench/... ./example/user/... ./example/todo/... ./example/encryption/... ./example/deployer-first/... -count=1` |
| Lint      | `nix run .#lint`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Format    | `nix fmt`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Dev shell | `nix develop`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| CI        | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |

## Monorepo Structure

Multi-module Go workspace (`go.work`) with 38 modules (26 library + 6 stack presets + 1 integration + 3 examples + 2 cmd):

```
go-cqrs-lite/
├── event/               # EventSink, EventSource, Store, Journal, SeekableJournal, Bus, ImmutableEvent, NewEvent, Clone
│   └── eventtest/       # FakeStore, FakeBus, FakeSnapshotStore, event factories, test assertions
├── command/             # Dispatcher, Handler, Middleware, Command, BasicCommand, PersistedCommand
│                        # Store: CommandSink, CommandSource, Store, CommandJournal, SeekableCommandJournal
│                        # Bus: Publisher, Subscriber, Bus, PublishMiddleware (command pub/sub)
├── query/               # Dispatcher, Handler, Pagination, PaginatedResult[T], RegisterTyped[Q,R], TypedHandler[Q,R]
│                        # Store: PersistedQuery, QuerySink, QuerySource, QueryStore, QueryJournal, SeekableQueryJournal
│   └── querytest/       # New(tb, queryType) test helper — tb.Fatalf on error, no panics
├── decider/             # Decider[State], Repository[State], Execute, Load (pure-function style)
├── id/                  # Branded IDs: id.Of[T] = cbid.ID[T, ulid.ULID], AggregateID, EventID, etc.
│   └── idtest/          # Parse*(tb, s) test helpers — tb.Fatalf on error, no panics
├── dispatcher/          # Generic Dispatcher[H, M] with LifecycleMixin
├── schema/              # Upcaster, VersionedStore, upcasterRegistry (schema evolution); Validator with RegisterType[T]() (ADR-0017)
├── snapshot/            # Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
├── storage/memory/       # MemoryStore, MemorySnapshotStore, MemoryCheckpointStore, MemoryCommandStore, MemoryQueryStore (in-memory test impls)
├── catalog/             # Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters
│   └── schema/          # JSON Schema types, reflection engine, YAML serialization
├── middleware/           # Logging, Retry, Recovery, Validation, Metrics, OTel Tracing+Metrics (command+event+query)
├── signing/             # Event signing/verification: HMAC-SHA256, Ed25519, multisig, middleware
├── encryption/          # Event payload encryption: XChaCha20-Poly1305, AES-256-GCM, codec wrapper, middleware
├── storage/             # SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLCommandStore, SQLQueryStore (PG/SQLite/Turso), SQL schema DDL (embedded)
│   ├── sql/             # Dialect, DBHandle, OwnedDBHandle, QueryEngine, RunInTx, IsDuplicateKeyError (typed codes + string fallback), ScanSlice, CommitTx, MarshalMetadata
│   ├── migrations/      # Embedded .sql DDL files (postgres.sql, sqlite.sql) via //go:embed
│   ├── pebble/          # Embedded KV store (PebbleDB): EventStore, SnapshotStore, CheckpointStore, KVAdapter (kv.Store). CBOR envelope, shared DB
│   └── turso/           # Turso database connector (embedded LibSQL sync), indexing advisor
├── otel/                # Shared OpenTelemetry helpers: Tracer, Meter, Spans, Attributes
├── prometheus/         # OTel→Prometheus metrics bridge: Setup() MeterProvider + /metrics HTTP handler, WithRegistry()
├── listing/             # AggregateListing, AggregateStatus, tombstone detection, StatusMiddleware, InMemoryAggregateReader
├── watermill/           # Watermill adapter: PublisherAdapter, SubscriberAdapter, EventPublisher (cqrs→Watermill), CatchUpSubscriber (replay+live+checkpoint), MessageToEvent
├── transport/http/       # SSE event delivery: SSEBroker, SSEHandler (bridges event.Bus to HTTP clients, ADR-0025)
├── codec/               # Payload encoding: JSON, CBOR (deterministic), Raw passthrough
├── kv/                  # Layer-0 KV store abstraction: Store, MemStore, Iterator, Batch. PLUS TypedStore[T,K] and Cache[T,K] (merged from readmodel, ADR-0032)
├── testutil/            # Shared test helpers: NewCmd(tb, ...) (cross-module test utilities)
├── cmd/cqrs-gen/        # Code generator: typed handler registration from Go structs
├── cmd/api-stability/   # API surface checker: compares exported symbols against golden file
├── integration/         # Cross-module tests (command, event, query, signing, encryption)
└── docs/                # Status reports, ADRs, architecture patterns, storage guide
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
13. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, storage, middleware go.mod files. The `otel/` module re-exports `Int64Counter`, `AddOption`, `AddSpanEvent()`, `ServiceResourceAttributes()`, `CQRSHistogramBoundaries`, `NewCQRSViews()`, and `CounterAddWithAttributes()` for rate metrics, span events, service identification, histogram customization, and metric views.
14. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths by accessing the `*ImmutableEvent` field directly (Event is now a concrete type alias, so no assertion is needed). Used by signing (SHA-256 hashing, CloneEvent), pebble (json.Marshal), storage/sql (ExecContext), transport/http/sse (string conversion). Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
15. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake. The `Event` interface documents this contract for third-party implementors.
16. **Hot-path zero-allocation discipline** — Public API clones stay, but internal hot paths eliminate allocs via: lazy map init (`NewMetadata()` returns zero-value), pre-computed middleware chains (EventBus rebuilds on `Use()`/`UsePublish()` only), cached SQL templates (built once at construction), pre-sized result slices (`make([]T, 0, hint)`), lock-free fast paths (`CircuitBreaker` uses `atomic.Int32`), batch SQL inserts (multi-VALUES with SQLite 999-param chunking). **Lesson learned**: type assertions for fast paths are dead code if users create types via different constructors. Cache at the integration boundary instead.
17. **Load coalescing via singleflight** — `decider.Repository[State]` uses `singleflight.Group` to coalesce concurrent `Load` calls for the same aggregate into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice across callers is safe. Only load is coalesced — Save/Publish still execute independently per caller. Disable via `WithLoadCoalescing[State](false)`.
18. **Go experimental build tags** — Builds use `-tags "goexperiment.arenas goexperiment.jsonv2"` enabling: arena allocations (memory pooling via `event/arena_experiment.go`) and JSON v2 encoding (via `codec/jsonv2_experiment.go`). These are Go experiment flags, NOT standard build tags — they require `GOEXPERIMENT` support in the toolchain. CI and `nix run .#build` apply them automatically. Dead tags (goroutineleakprofile, runtimesecret, simd) were removed — they gated zero files.

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
//   bus := memory.NewMemoryCommandBus()
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
//   the journal instead). See example/deployer-first for the correct pattern.

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

// Multi-DB SQLite preset (deployer chooses database isolation)
//   bundle, _ := sqlite.New(":memory:",
//       sqlite.WithEventDB("events.db"),   // events+snapshots+checkpoints
//       sqlite.WithQueryDB("queries.db"),  // command+query audit
//       sqlite.WithViewDB("views.db"),     // read-model KV store
//   )

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
```

## Testing

- Table-driven tests preferred; BDD via Ginkgo v2 + Gomega for event/decider/query
- `t.Parallel()` for independent tests; core packages >80% coverage (most >90%)
- Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`
- Golden tests use shared `eventtest.AssertGolden(t, path, got, *update)` from `event/v3/eventtest`
- Modules without event dependency (otel, codec) keep their own local golden helper

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

**Coverage**: core modules 84–100%; Bundle layer (stack presets, cache) 0–87% — presets emphasise the shared contract suite + happy paths, so constructor error branches are lighter (stack/postgres shows 0% locally because its tests skip without `POSTGRES_TEST_DSN`). See `docs/status/` for latest.

**Module Graph**:

```
Layer 0: id/, dispatcher/, codec/, kv/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher, ro), query/ (→dispatcher, ro)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot)
Layer 4: storage/memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, listing/, watermill/, transport/http/, storage/pebble/, storage/turso/, prometheus/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability
```

> **Saga pattern**: No dedicated saga module. Multi-step orchestration emerges from bus.SubscribeAll + command dispatch. See `example/todo/` for a real architecture.

> **Historical details**: Session milestones, catalog architecture, and known issues in
> [`docs/sessions/SESSION_MILESTONES.md`](docs/sessions/SESSION_MILESTONES.md)
> and [`docs/planning/CATALOG_ARCHITECTURE.md`](docs/planning/CATALOG_ARCHITECTURE.md).

> **Schema evolution**: Upcaster and VersionedStore moved to `schema/` module. See `schema/` package.
> **Snapshot persistence**: Snapshot types moved to `snapshot/` module. See `snapshot/` package.
