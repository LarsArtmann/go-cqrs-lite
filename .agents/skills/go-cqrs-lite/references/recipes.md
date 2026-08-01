## 2. Composition Recipes (copy-paste, verified APIs)

> **Contents** — jump to the recipe you need:
>
> - [§2.0 Bundle Presets](#20-bundle-presets--one-call-infrastructure-wiring) — one-call infrastructure wiring
> - [§2.1 Minimal Event Sourcing](#21-minimal-event-sourcing-event--command--decider--id--memory)
> - [§2.2 Production Persistence](#22-production-persistence-storage-or-pebble)
> - §2.3 Read Models → moved to [`readmodels.md`](readmodels.md) (projections, SQL views, CatchUpSubscriber, projection-tier selection)
> - [§2.4 Snapshots for Performance](#24-snapshots-for-performance-snapshot)
> - [§2.5 Schema Evolution](#25-schema-evolution-schema)
> - [§2.6 Tamper-Proof Event Streams](#26-tamper-proof-event-streams-signing)
> - [§2.7 Encrypted Payloads](#27-encrypted-payloads-encryption)
> - [§2.8 Observability & Middleware](#28-observability--middleware-otel--middleware)
> - [§2.9 Auto-Documentation](#29-auto-documentation-catalog)

### 2.0 Bundle Presets — one-call infrastructure wiring

> **New in v2.7.** Consumers should NOT decide on infrastructure manually.
> The deployer picks a preset; the app developer never imports a backend.

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"

// One call wires: event store + bus, command store, query store,
// snapshot store, checkpoint store, read-model backend.
b, err := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()

// Diagnostics: verify wiring (prints ✓/✗ for each capability)
fmt.Println(b.Debug())

// Typed read model over the Bundle's shared KV backend
store := kv.NewTypedStore[TodoView, TodoID](b.ReadModels)

// Command handlers use b.EventSink (asserts to event.Store)
// Queries use the read model store
// Projections use b.Journal + b.Subscriber + b.CheckpointStore
```

#### Production options (SQLite / Turso)

```go
// SQLite with production optimizations (WAL + synchronous=NORMAL are default)
b, _ := sqlite.New("app.db",
    sqlite.WithPragmas(
        sqlopt.WithOptimizations(),  // cache_size, temp_store, mmap_size PRAGMAs
        sqlopt.WithForeignKeys(),    // referential integrity (opt-in)
    ),
)

// Turso with remote sync + optimizations
b, _ := turso.NewSync(ctx, "local.db", "libsql://my-db.turso.io", "token",
    turso.WithPragmas(sqlopt.WithOptimizations()),
    turso.WithSyncOptions(turso.WithClientName("edge-node-1")),
)

// Disable WAL if running on a network filesystem
b, _ := sqlite.New("app.db", sqlite.WithPragmas(sqlopt.WithoutWAL()))
b, _ := turso.New("app.db", turso.WithPragmas(sqlopt.WithoutWAL()))
```

#### Postgres distributed bus (cross-process pub/sub)

```go
listener, _ := postgres.NewPgxListenerFromDSN(ctx, dsn)
b, _ := postgres.New(dsn, postgres.WithDistributedBus(listener))
// Events now propagate via LISTEN/NOTIFY to other processes sharing the DB
```

Available presets:

| Preset   | Module           | Backend          | Read Models         |
| -------- | ---------------- | ---------------- | ------------------- |
| Memory   | `stack/memory`   | In-memory        | Memory KV           |
| SQLite   | `stack/sqlite`   | SQLite (modernc) | SQL KV (persistent) |
| Pebble   | `stack/pebble`   | PebbleDB (LSM)   | Pebble KV           |
| Postgres | `stack/postgres` | PostgreSQL (pgx) | SQL KV (persistent) |
| MySQL    | `stack/mysql`    | MySQL/MariaDB    | SQL KV (persistent) |
| Turso    | `stack/turso`    | Turso Database   | SQL KV (persistent) |

Multi-DB split (SQLite, Turso, Postgres, MySQL) — isolates event writes from
read-model scans by routing each concern to a separate database:

```go
b, _ := sqlite.New("primary.db",
    sqlite.WithDSN(
        sqlopt.WithEventDB("events.db"),   // events + snapshots + checkpoints
        sqlopt.WithQueryDB("queries.db"),  // command + query audit
        sqlopt.WithViewDB("views.db"),     // read models (cqrs_kv)
    ),
)
```

See [`docs/MIGRATION_TO_STACK.md`](docs/MIGRATION_TO_STACK.md) for how to
replace hand-wired infrastructure with presets.

Read-model cache decorator:

```go
cached, _ := kv.NewCache(store,
    kv.WithCacheCapacity[TodoView, TodoID](10_000),
    kv.WithCacheTTL[TodoView, TodoID](5*time.Minute))
```

#### Shared database (events + reads in one \*sql.DB)

When your events and read models share a single SQLite/Postgres database, skip the
`stack/*` presets (they create separate connections). Wire manually instead:

```go
import (
    "database/sql"
    "github.com/larsartmann/go-cqrs-lite/storage/v4"
    cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

// One shared *sql.DB for everything
db, _ := sql.Open("sqlite3", "app.db")

// Event store + journal from the same DB
backend, _ := storage.NewSQLiteBackend(db)
eventStore := backend.EventStore()

// Read model views from the same DB
viewStore, _ := storage.NewSQLiteViewStore[TodoView, TodoID](db, mapper)

// Projections read from the journal, write to viewStore — same DB, same tx if needed
// This architecture is common for single-process apps (DiscordSync, SEC, etc.)
```

**When to use this:** single-process apps, embedded systems, personal projects,
CI test harnesses. **When NOT to use:** multi-process or distributed setups that
need separate connection pools or cross-process pub/sub.

See [`docs/PRESETS.md`](docs/PRESETS.md) and [`docs/INFRASTRUCTURE_RECOMMENDATIONS.md`](docs/INFRASTRUCTURE_RECOMMENDATIONS.md) for full documentation.

#### Bundle.Debug() — verify your wiring

After constructing a Bundle (from a preset or manual assembly), call `Debug()`
to see which capabilities are set. This is the fastest way to catch missing
wiring — each field shows ✓ (set) or ✗ (nil):

```go
b, _ := sqlite.New("app.db")
fmt.Println(b.Debug())
// Output:
// Bundle capabilities:
//   EventSink:           ✓
//   EventSource:         ✓
//   Journal:             ✓
//   SeekableJournal:     ✓
//   Publisher:           ✓
//   Subscriber:          ✓
//   CommandSink:         ✓
//   CommandSource:       ✓
//   QuerySink:           ✓
//   QuerySource:         ✓
//   SnapshotStore:       ✓
//   CheckpointStore:     ✓
//   ReadModels:          ✓
```

A ✗ on `Journal` or `SeekableJournal` means `CatchUpSubscriber` will fail.
A ✗ on `ReadModels` means `stack.ReadModel` and `stack.NewMaterialize` will fail.
Use this in tests to verify your preset configuration before deployment.

### 2.1 Minimal Event Sourcing (event + command + decider + id + memory)

The foundation. Every app starts here.

```go
package main

import (
    "context"
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/codec/v4"
    "github.com/larsartmann/go-cqrs-lite/command/v4"
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
    cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

type UserState struct{ Name string }
type CreateUser struct{ Name string }
type UserCreated struct{ Name string }

func main() {
    ctx := context.Background()
    store := memory.NewMemoryStore()
    bus := cqrswatermill.NewEventBus()

    d := decider.Decider[UserState]{
        Initial: UserState{},
        Apply: func(s UserState, e event.Event) (UserState, error) {
            p, _ := event.DecodePayload[UserCreated](e, codec.JSONCodec{})
            s.Name = p.Name
            return s, nil
        },
    }
    repo, _ := decider.NewRepository[UserState](store, bus, d)

    cmds := command.NewDispatcher()
    aggID := id.NewStreamID()
    command.RegisterTyped(cmds, "user.create",
        func(ctx context.Context, cmd *CreateUser) error {
            return repo.Execute(ctx, aggID, "User", func(s UserState, v event.Version) ([]event.Event, error) {
                return event.NewEvents(aggID, "User", v,
                    []event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
            })
        })

    _ = cmds.Dispatch(ctx, &CreateUser{Name: "Alice"})

    state, _, _ := repo.Load(ctx, aggID, "User")
    fmt.Printf("User: %s\n", state.Name) // User: Alice
}
```

### 2.2 Production Persistence (storage or pebble)

Replace `memory` with a real backend. Two choices:

**SQL (PostgreSQL / SQLite):**

```go
import "github.com/larsartmann/go-cqrs-lite/storage/v4"

// db is a *sql.DB (Postgres or SQLite)
backend, _ := storage.NewSQLiteBackend(db)   // or NewSQLBackend(db) for Postgres (dialect auto-detected from driver)
defer backend.Close()                        // closes all stores (NOT the *sql.DB)

eventStore := backend.EventStore()           // *SQLEventStore
cmdStore, _ := backend.CommandStore()        // *SQLCommandStore (lazy)
qStore, _   := backend.QueryStore()          // *SQLQueryStore (lazy)
snapStore,_ := backend.SnapshotStore()       // *SQLSnapshotStore (lazy)
cpStore, _  := backend.CheckpointStore()     // *SQLCheckpointStore (lazy)
```

**Embedded PebbleDB (single binary, one DB for the full stack):**

```go
import "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"

backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
defer backend.Close() // closes DB AND all stores

eventStore  := backend.EventStore()
snapStore   := backend.SnapshotStore()
cpStore     := backend.CheckpointStore()
```

> **Rule:** `backend.Close()` closes the stores it owns, NOT an externally-passed `*sql.DB`. For Pebble `Open()`, it closes the DB too.

### 2.4 Snapshots for Performance (snapshot)

Avoid replaying long event streams. Snapshots cache stream state at a version.

```go
import "github.com/larsartmann/go-cqrs-lite/snapshot/v4"

strategy, _ := snapshot.EveryNEvents(100)                                 // returns (SnapshotStrategy, error)
repo, _ := decider.NewRepository[UserState](store, bus, d,
    decider.WithSnapshotStore(snapStore),                               // SQL/Pebble/memory
    decider.WithSnapshotStrategy(strategy),                             // snapshot every 100 events
)
// repo.Load now reads the latest snapshot + replays only post-snapshot events
```

### 2.5 Schema Evolution (schema)

Migrate old event payloads on read without rewriting history.

```go
import "github.com/larsartmann/go-cqrs-lite/schema/v4"

// Upcast UserCreated v1 → v2 (adds a default field)
upcaster := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
    old, _ := event.DecodePayload[UserCreatedV1](evt, codec.JSONCodec{})
    newPayload, _ := codec.JSONCodec{}.Encode(UserCreatedV2{Name: old.Name, Email: ""})
    return event.NewEvent(evt.Type(), evt.StreamID(), evt.StreamType(), evt.Version(),
        newPayload, event.WithSchemaVersion(2))
})

versioned := schema.NewVersionedStore(eventStore, upcaster)
// versioned.Load transparently applies upcasters
```

### 2.6 Tamper-Proof Event Streams (signing)

Cryptographic signatures detect tampering in transit and at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/signing/v4"

signer, _ := signing.NewHMAC(secret)
bus.UsePublish(signing.SignMiddleware(signer))   // sign on publish
bus.Use(signing.VerifyMiddleware(signer))        // verify on receive
// Ed25519: signing.NewEd25519(privateKey, publicKey)
// Multisig: signing/v4/multisig
```

### 2.7 Encrypted Payloads (encryption)

Confidential event payloads encrypted at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/encryption/v4"

enc, _ := encryption.NewXChaCha20Poly1305(key)   // or NewAES256GCM(key)
bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v1")))
bus.Use(encryption.DecryptMiddleware(enc))

// Composable codec wrapper (JSON envelope, encrypted payload)
encryptedCodec := encryption.NewCodec(codec.JSONCodec{}, enc)

// Key rotation via resolver (map of KeyID → Decrypter)
resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
    "key-v1": oldDecrypter,
    "key-v2": newDecrypter,
})
```

### 2.8 Observability & Middleware (otel + middleware)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/middleware/v4"
    "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

tracer := otel.GetTracerProvider().Tracer("my-app")
bus.Use(middleware.EventTracing(tracer))
bus.UsePublish(middleware.EventPublishTracing(tracer))

meter := otel.GetMeterProvider().Meter("my-app")
recorder, _ := middleware.NewOTelMetricsRecorder(meter)
cmdDispatcher.Use(middleware.CommandTypedMetrics(recorder))

// Other middleware: Logging, Retry, Recovery, Validation, CircuitBreaker
cmdDispatcher.Use(middleware.CommandRecovery())
cmdDispatcher.Use(middleware.CommandRetry(3, time.Second))
```

> **Rule:** Import OTel via `otel/` re-exports, NOT `go.opentelemetry.io` directly.

#### Tracing + Prometheus metrics (otel.Setup + prometheus.Setup)

`otel.Setup` configures tracing. `prometheus.Setup` configures a Prometheus
meter provider with a `/metrics` endpoint. Pass `cqrsotel.NewCQRSViews()` so
histogram boundaries match CQRS latency ranges:

```go
import (
    cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
    cqrsprometheus "github.com/larsartmann/go-cqrs-lite/prometheus/v4"
    "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// 1. Tracing provider (spans)
tracingProvider, _ := cqrsotel.Setup(
    cqrsotel.WithService("my-app", "1.0.0", "instance-1"),
)
defer tracingProvider.Shutdown(ctx)

// 2. Prometheus meter provider with CQRS histogram boundaries
metricsProvider, _ := cqrsprometheus.Setup(
    cqrsprometheus.WithViews(cqrsotel.NewCQRSViews()...),
)
defer metricsProvider.Shutdown(ctx)

// 3. Wire Prometheus as the global meter provider
otel.SetMeterProvider(metricsProvider.AsMeterProvider())

// 4. Create the OTel middleware bundle (uses global tracer + meter providers)
bundle, _ := middleware.NewOTelBundle(
    cqrsotel.NewTracer("my-app"), cqrsotel.NewMeter("my-app"),
)
cmdDispatcher.Use(bundle.Command()...)
bus.Use(bundle.Event()...)
bus.UsePublish(bundle.Publish()...)
qDispatcher.Use(bundle.Query()...)
```

`WithViews` is optional — without it, Prometheus uses SDK-default histogram
boundaries. With it, latency histograms use CQRS-optimized buckets
(`[0.05, 0.1, ..., 10000]` ms).

#### Command Idempotency (dedup on retry)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/idempotency/v4"
    "github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()

// Rejects duplicate commands within the TTL. Default key: cmd.ID().String().
// Pass a custom key extractor for client-supplied idempotency keys.
// Also available: middleware.EventIdempotency, middleware.QueryIdempotency.
cmdDispatcher.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
```

#### Query Middleware (symmetric with command middleware)

Query middleware provides the same recovery, logging, metrics, and retry capabilities
as command middleware. Apply it to your query dispatcher for production-grade resilience:

```go
// Query middleware chain (same pattern as command middleware)
qDisp.Use(middleware.QueryRecovery())           // panic → error, don't crash the process
qDisp.Use(middleware.QueryLogging(slog.Default())) // structured query logging
qDisp.Use(middleware.QueryRetry(3, time.Second))   // retry transient failures

// OTel metrics for queries
meter := otel.GetMeterProvider().Meter("my-app")
recorder, _ := middleware.NewOTelMetricsRecorder(meter)
qDisp.Use(middleware.QueryTypedMetrics(recorder))

// OTel tracing for queries
tracer := otel.GetTracerProvider().Tracer("my-app")
qDisp.Use(middleware.QueryTracing(tracer))
```

The full middleware matrix (all symmetric across command/event/query):

|         | Recovery | Logging | Retry | CircuitBreaker | Metrics | Tracing |
| ------- | -------- | ------- | ----- | -------------- | ------- | ------- |
| Command | ✅       | ✅      | ✅    | ✅             | ✅      | ✅      |
| Event   | ✅       | ✅      | ✅    | ✅             | ✅      | ✅      |
| Query   | ✅       | ✅      | ✅    | ✅             | ✅      | ✅      |

### 2.9 Auto-Documentation (catalog)

Generate AsyncAPI 3.0, EventCatalog, OpenAPI, and D2 diagrams from your Go types.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/catalog/v4"
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/d2"
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"
)

reg := catalog.NewRegistry("My API", "1.0.0")
reg.RegisterEvent("user.created", catalog.SchemaFromType[UserCreated]())
reg.RegisterCommand("user.create", catalog.SchemaFromType[CreateUser]())

asyncAPIDoc, _ := asyncapi.Generate(reg)
openAPIDoc, _  := openapi.Generate(reg)
catalogDir, _  := eventcatalog.Generate(reg)
d2Diagram, _   := d2.Generate(reg)
```

### 2.10 Cost-Based Storage Planning (metaengine)

The metaengine picks the cheapest backend per query — memory for small collections, SQLite for large ones.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Define a query with filter pushdown
q := metaengine.Query[FindTaskInput, FindTaskResult]("find-task",
    metaengine.FilterOn(func(r FindTaskResult) TaskID { return r.ID }),
    metaengine.Volume(500_000),
)

// Register folds (how events update the materialized state)
store, _ := metaengine.Plan(
    []metaengine.Engine{
        metaengine.NewMemoryEngine(),
        // sqliteEngine, // metaengine.NewSQLiteEngine(db)
    },
    q,
    metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, FindTaskResult) {
        return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
    }),
    metaengine.On(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
)

// Execute: the engine routes to the cheapest backend
result, _ := store.Execute(FindTaskInput{ID: "task-1"})
```

### 2.11 SQL-Backed Idempotency (idempotency/sqlstore)

Durable dedup for at-least-once delivery, surviving process restarts.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

// SQLite-backed (caller owns the *sql.DB)
store, _ := sqlstore.NewSQLiteStore(ctx, db)

// Postgres-backed
// store, _ := sqlstore.NewPostgresStore(ctx, pgDB)

// Use with idempotency middleware
cmds.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
```

### 2.12 Retry with Backoff (retry)

Zero-dependency retry with exponential backoff and jitter.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/retry/v4"
)

config := retry.Config{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    Jitter:       0.1,
}

err := retry.Do(ctx, config, func(ctx context.Context) error {
    return flakyOperation(ctx)
})
if errors.Is(err, retry.ErrExhausted) {
    // all attempts failed
}
```

### 2.13 Scaling Out — NATS Transport & Parquet Journal (design docs)

The library ships no opinionated broker or columnar store (principle #1: "library,
not framework"), but design studies exist for two common scale-out backends. Read
these before wiring NATS JetStream or a Parquet-backed event journal — they cover
the integration shape, ordering guarantees, and materialization strategies.

- **NATS JetStream transport** — [`docs/design/transport-nats.md`](../../../../docs/design/transport-nats.md)
  (design) and [`docs/planning/nats-transport-design.md`](../../../../docs/planning/nats-transport-design.md)
  (implementation plan). Use when you need cross-process command/event distribution
  beyond the in-process bus and the Postgres LISTEN/NOTIFY bridge.
- **Parquet event journal + DuckDB materializations** —
  [`docs/planning/parquet-journal-design.md`](../../../../docs/planning/parquet-journal-design.md)
  and [`docs/research/archive/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md`](../../../../docs/research/archive/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md).
  Use for analytical read models over large immutable event logs (columnar scans
  instead of row-by-row replay).

> These are design-stage documents, not released modules. They describe how a
> consumer would compose the existing store/bus/projection interfaces against
> these backends.

### 2.14 CBOR→JSON for Browser SSE Clients (codec + transport/http)

Store events in compact CBOR but serve JSON over SSE to browsers. The
`codec.TranscodeToJSON` primitive decodes CBOR generically and re-encodes as
JSON; `http.CBORToJSONTransform` wraps it as a ready-made payload transform
with graceful fallback (on decode failure the raw payload is sent, so clients
never see a gap).

```go
import cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"

// One-liner: every SSE/backfill payload is transcoded CBOR→JSON.
broker, err := cqrshttp.NewSSEBroker(bus,
    cqrshttp.WithPayloadTransform(cqrshttp.CBORToJSONTransform))
```

Non-CBOR events (JSON/Raw) pass through with zero overhead (1.9 ns, 0 allocs).
The transform is applied uniformly across live, replay, and backfill paths.

When you need schema-aware JSON (reconstructing field names from `toarray`
structs) or custom logging, call `codec.TranscodeToJSON` directly inside your
own `func(event.Event) []byte` and use `event.DecodePayloadAuto[T]` for typed
decoding. See `codec/README.md` → "CBOR → JSON Transcoding".

### 2.15 Metaengine SSE Streaming with Reconnection (metaengine)

Stream metaengine value changes as Server-Sent Events with automatic
Last-Event-ID reconnection. Clients that disconnect receive missed events
from a bounded replay journal on reconnect.

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

// 1. Create a watcher with replay journal (buffer 1000 recent changes).
watcher := metaengine.NewWatcher[TaskView](store, "tasks")
replay := watcher.WithReplay(1000)
defer watcher.Close()

// 2. Serve SSE — writes id: <seq> on every event.
http.HandleFunc("/events/tasks", func(w http.ResponseWriter, r *http.Request) {
    _ = metaengine.ServeSSE(w, r, watcher,
        metaengine.WithSSETimeout(30*time.Minute),
        metaengine.WithSSEHeartbeat(30*time.Second),
        metaengine.WithSSEReplayLimit(500), // cap replay backlog
    )
})

// Browser clients connect with EventSource — Last-Event-ID is automatic:
//   const es = new EventSource("/events/tasks");
//   es.onmessage = (e) => console.log(JSON.parse(e.data));
```

Without `WithReplay`, ServeSSE works as a plain live stream (no id field,
no reconnection). The replay journal uses a ring buffer with `dedup.Ring`
for the replay-to-live overlap.

### 2.16 Metaengine Cursor Pagination with PrefetchCache (metaengine)

Paginate metaengine scans with HTTP-safe encoded cursors and a prefetch
cache that eliminates the limit+1 round-trip pattern.

```go
reader := metaengine.NewReader[TaskView](store, "tasks").
    WithPrefetch(metaengine.NewPrefetchCache())

// Page 1: no cursor.
page1, cursor1, _ := reader.ScanPage(ctx,
    metaengine.WithSort("Title", false),
    metaengine.WithLimit(20),
)

// Encode cursor for HTTP transport (base64+JSON, URL-safe).
encoded, _ := cursor1.Encode()
// → pass `encoded` as ?cursor=... query param

// Page 2: parse encoded cursor string (cache hit from prefetch).
page2, cursor2, _ := reader.ScanPage(ctx,
    metaengine.WithSort("Title", false),
    metaengine.WithLimit(20),
    metaengine.WithCursorString(encoded), // HTTP-safe path
)

// Or use the raw value directly:
// page2, _, _ := reader.ScanPage(ctx, ..., metaengine.WithCursor(cursor1.Value))
```

Both `WithCursor(raw)` and `WithCursorString(encoded)` produce identical
PrefetchCache keys because the cache normalizes through `Cursor.Encode()`.
The `PrefetchCache` is thread-safe (RWMutex).

### 2.17 Flight Recorder — Capture Trace on Slow/Error (flightrecorder + middleware)

Buffer the last few seconds of execution trace in memory. When an operation
crosses a latency threshold or errors, snapshot the trace for offline analysis
with `go tool trace`.

```go
import (
    flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
    "github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// 1. Create and start the recorder (only 1 active per process).
recorder, _ := flightrecorder.New(
    flightrecorder.WithMinAge(10*time.Second),  // buffer window
    flightrecorder.WithMaxBytes(10<<20),        // 10 MiB
    flightrecorder.WithFile("debug.trace"),     // snapshot destination
)
recorder.Start()
defer recorder.Close() // stops recording + closes file

// 2. Wire triggers into middleware (snapshot when cmd >100ms OR errors).
cmdDisp.Use(middleware.CommandFlightRecorder(recorder,
    flightrecorder.OnErrorOrLatency(100*time.Millisecond)))
bus.Use(middleware.EventFlightRecorder(recorder,
    flightrecorder.OnError()))
qryDisp.Use(middleware.QueryFlightRecorder(recorder,
    flightrecorder.OnLatency(500*time.Millisecond)))

// 3. Also wire into decider and projection host:
// repo, _ := decider.NewRepository(store, bus, d,
//     decider.WithFlightRecorder[MyState](recorder,
//         flightrecorder.OnErrorOrLatency(200*time.Millisecond)))
// host, _ := projectionhost.New(journal, cpStore,
//     projectionhost.WithFlightRecorder(recorder, flightrecorder.OnAlways()))

// 4. Analyze: go tool trace debug.trace
// Once-semantics: first trigger captures, rest are no-ops (call Reset() for multiple)
```

**Trigger options:**

| Trigger | Fires when |
|---------|-----------|
| `OnLatency(threshold)` | Duration exceeds threshold |
| `OnError()` | Operation returned non-nil error |
| `OnErrorOrLatency(threshold)` | Either of the above (recommended default) |
| `OnAlways()` | Every operation (testing/baseline) |
| `OnAny(triggers...)` | Any trigger fires (OR) |
| `OnAll(triggers...)` | All triggers fire (AND) |

**Stack bundle integration** (lifecycle management + discovery):

```go
bundle, _ := sqlite.New(dsn, stack.WithFlightRecorder(recorder))
defer bundle.Close() // stops recorder automatically
// Access for trigger wiring: bundle.FlightRecorder()
```

## Metaengine + Stack Bundle Integration

Wire a cost-based query planner into the Bundle lifecycle with one option.

```go
// 1. Declare your query (Counter ADT for O(1) status counts)
type StatusCounts struct{}

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[StatusCounts, map[string]int64](
        "task_counts",
        metaengine.On(TaskCreated{}, func(e TaskCreated) metaengine.Delta {
            return metaengine.Delta{e.Status: +1}
        }),
        metaengine.On(TaskCompleted{}, func(e TaskCompleted) metaengine.Delta {
            return metaengine.Delta{"active": -1, "completed": +1}
        }),
    ),
)

// 2. Register with the Bundle (lifecycle managed automatically)
bundle, _ := sqlite.New(dsn,
    sqlite.WithStack(stack.WithMetaEngine(store)),
)
defer bundle.Close() // closes the metaengine Store too

// 3. Query at runtime
counts, _ := metaengine.ExecuteTyped[StatusCounts, map[string]int64](
    ctx, bundle.MetaEngine(), StatusCounts{},
)

// 4. For projection lifecycle (checkpoint, retry, DLQ), wrap in adapter
adapter := projectionadapter.New("task_counts", store, payloadDecoder)
host.Register(adapter)
```

Key points:
- `WithMetaEngine(store)` registers the Store for `Bundle.Close()` — no manual cleanup
- `bundle.MetaEngine()` returns the Store for runtime queries
- benchkit auto-discovers via `MetaEngine() != nil` (unless `Config.SkipMetaEngine`)
- The consumer calls `metaengine.Plan()` themselves (typed generics can't flow through `any`)
- `sqlite.WithStack()` is the passthrough for additional `stack.Option`s on any SQL preset

### Filtered Scan with Metaengine (Map + FilterOnField + SQLite Pushdown)

For read models that need filtered/sorted scans, declare a Map query with
`FilterOnField` and `SortOnField`. The planner pushes these to SQLite's
`json_extract()` WHERE/ORDER BY for O(logN) scans instead of O(N) Go-side
filtering (50x faster at 10K rows).

```go
type listInput struct{ Status string }
type itemView struct {
    ID       string `json:"id"`
    Status   string `json:"status"`
    Priority int    `json:"priority"`
}

query := metaengine.Query[listInput, itemView]("items",
    metaengine.On(createdEvt{}, func(e createdEvt) (string, itemView) {
        return e.ID, itemView{ID: e.ID, Status: "open", Priority: e.Priority}
    }),
    metaengine.On(deletedEvt{}, metaengine.Remove[itemView]()),
    metaengine.FilterOnField[itemView]("status", metaengine.FilterEq),
    metaengine.SortOnField[itemView]("priority", true),
)

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    query,
)
reader := metaengine.NewReader[itemView](store, "items")

// Filtered scan (SQLite pushdown):
active, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "active"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(50))

// Point lookup:
item, found, _ := reader.Get(ctx, "item-123")
```

### Bridging Stream IDs to Map Keys (eventWithID Wrapper)

Map fold handlers need the entity ID as the key, but the event payload doesn't
contain it — it lives in the event's StreamID. Use `EventDecoder` to wrap the
typed payload with the stream ID:

```go
type eventWithID[P any] struct {
    ID      string
    Payload P
}

func myDecoder(evt event.Event) (any, error) {
    id := evt.StreamID().String()
    switch evt.Type() {
    case "created":
        var p createdPayload
        _ = json.Unmarshal(evt.Payload(), &p)
        return eventWithID[createdPayload]{ID: id, Payload: p}, nil
    default:
        return nil, fmt.Errorf("no fold for %q", evt.Type())
    }
}

adapter := projectionadapter.New("items", store, nil,
    projectionadapter.WithEventDecoder(myDecoder))
```

### Multi-Engine Distribution (Counter to Memory, Map to SQLite)

The planner inspects query shapes and assigns each to the cheapest supporting
engine. Counters (O(1) reads) go to Memory; filtered Maps go to SQLite
(pushdown scans):

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    counterQuery, // Counter → Memory
    mapQuery,     // Map + FilterOnField → SQLite
)
```

### Vector ADT — Semantic Search (k-NN)

Fold events carrying embeddings into a vector index, then query by similarity:

```go
type DocEmbedded struct {
    ID     string
    Values []float32
}
type SemanticSearchInput struct {
    Query []float32
    K     int
}

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[SemanticSearchInput, []metaengine.VectorResult]("embeddings",
        metaengine.On(DocEmbedded{}, func(e DocEmbedded) metaengine.Embedding {
            return metaengine.Embedding{ID: e.ID, Values: e.Values}
        }),
    ),
)

store.Apply(ctx, "DocEmbedded", DocEmbedded{ID: "d1", Values: []float32{0.1, 0.9, 0.3}})

results, _ := metaengine.VectorExecuteTyped[SemanticSearchInput](ctx, store, SemanticSearchInput{
    Query: []float32{0.1, 0.9, 0.3},
    K:     5,
})
// results[0].ID = "d1", results[0].Distance = 0.0
```

### Search ADT — Full-Text Search

Fold events carrying text into an inverted index, then query by relevance:

```go
type DocIndexed struct {
    ID      string
    Content string
}
type FullTextSearchInput struct {
    Query string
    Limit int
}

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[FullTextSearchInput, []metaengine.SearchResult]("docs",
        metaengine.On(DocIndexed{}, func(e DocIndexed) metaengine.IndexedText {
            return metaengine.IndexedText{ID: e.ID, Content: e.Content}
        }),
    ),
)

store.Apply(ctx, "DocIndexed", DocIndexed{ID: "d1", Content: "the quick brown fox"})

results, _ := metaengine.SearchExecuteTyped[FullTextSearchInput](ctx, store, FullTextSearchInput{
    Query: "quick fox",
    Limit: 10,
})
// results[0].ID = "d1", results[0].Score = TF-IDF relevance
```

### Spatial ADT — Geo Proximity Search

Fold events carrying coordinates into a spatial index, then query by radius:

```go
type PlaceLocated struct {
    ID string
    X  float64 // longitude
    Y  float64 // latitude
}
type NearbySearchInput struct {
    X      float64
    Y      float64
    Radius float64 // meters
    Limit  int
}

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[NearbySearchInput, []metaengine.SpatialResult]("places",
        metaengine.On(PlaceLocated{}, func(e PlaceLocated) metaengine.Point {
            return metaengine.Point{ID: e.ID, X: e.X, Y: e.Y}
        }),
    ),
)

store.Apply(ctx, "PlaceLocated", PlaceLocated{ID: "brandenburg-gate", X: 13.3777, Y: 52.5163})

results, _ := metaengine.SpatialExecuteTyped[NearbySearchInput](ctx, store, NearbySearchInput{
    X: 13.4050, Y: 52.5200, // Berlin center
    Radius: 5000,             // 5km
    Limit:  20,
})
// Uses haversine great-circle distance (meters)
```

### Temporal Queries — Point-in-Time Reads

The Memory engine implements `VersionedStorage` for as-of queries:

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    // ... your queries ...
)

// Later, query the value as it existed at a past timestamp:
val, err := store.ExecuteAsOf(ctx, "users", "u1", someTimestamp)
if err == metaengine.ErrNotFound {
    // key did not exist at that time
}
```

### DuckDB Engine — Columnar Analytics

For analytical workloads (GROUP BY, aggregations on large datasets):

```go
import duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"

eng, _ := duckdbengine.New("analytics.db") // or "" for in-memory
defer eng.Close()

store, _ := metaengine.Plan(
    []metaengine.Engine{eng},
    // Counter queries benefit from DuckDB's vectorized GROUP BY
    metaengine.Query[CountInput, map[string]int64]("category_counts",
        metaengine.On(ItemCreated{}, func(e ItemCreated) metaengine.Delta {
            return metaengine.Delta{e.Category: e.Count}
        }),
    ),
)
// Requires CGo (DuckDB C++ runtime, statically linked)
```

### Postgres Engine — Production Durability

For production workloads needing ACID durability and concurrent readers:

```go
import pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"

eng, _ := pgengine.New("postgres://user:pass@host:5432/db?sslmode=disable")
defer eng.Close()

// JSONB columns for efficient JSON storage, GIN indexes for future query pushdown
store, _ := metaengine.Plan(
    []metaengine.Engine{eng},
    // Map and Counter queries with ON CONFLICT upserts
    metaengine.Query[GetInput, TodoView]("todos",
        metaengine.On(TodoCreated{}, func(e TodoCreated) (string, TodoView) {
            return e.ID, TodoView{ID: e.ID, Title: e.Title}
        }),
    ),
)
// Pure Go (pgx/v5 driver, no CGo)
```
