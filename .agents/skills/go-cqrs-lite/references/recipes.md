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
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v3"

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
    sqlite.WithOptimizations(),  // cache_size, temp_store, mmap_size PRAGMAs
    sqlite.WithForeignKeys(),    // referential integrity (opt-in)
)

// Turso with remote sync + optimizations
b, _ := turso.NewSync(ctx, "local.db", "libsql://my-db.turso.io", "token",
    turso.WithOptimizations(),
    turso.WithSyncOptions(turso.WithClientName("edge-node-1")),
)

// Disable WAL if running on a network filesystem
b, _ := sqlite.New("app.db", sqlite.WithoutWAL())
b, _ := turso.New("app.db", turso.WithoutWAL())
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
| Turso    | `stack/turso`    | Turso Database   | SQL KV (persistent) |

Multi-DB split (SQLite, Turso, Postgres only) — isolates event writes from
read-model scans by routing each concern to a separate database:

```go
b, _ := sqlite.New("primary.db",
    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
    sqlite.WithQueryDB("queries.db"),  // command + query audit
    sqlite.WithViewDB("views.db"),     // read models (cqrs_kv)
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
    "github.com/larsartmann/go-cqrs-lite/storage/v3"
    cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v3"
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

    "github.com/larsartmann/go-cqrs-lite/codec/v3"
    "github.com/larsartmann/go-cqrs-lite/command/v3"
    "github.com/larsartmann/go-cqrs-lite/decider/v3"
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/id/v3"
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
    cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
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
    aggID := id.NewAggregateID()
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
import "github.com/larsartmann/go-cqrs-lite/storage/v3"

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
import "github.com/larsartmann/go-cqrs-lite/storage/pebble/v3"

backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
defer backend.Close() // closes DB AND all stores

eventStore  := backend.EventStore()
snapStore   := backend.SnapshotStore()
cpStore     := backend.CheckpointStore()
```

> **Rule:** `backend.Close()` closes the stores it owns, NOT an externally-passed `*sql.DB`. For Pebble `Open()`, it closes the DB too.

### 2.4 Snapshots for Performance (snapshot)

Avoid replaying long event streams. Snapshots cache aggregate state at a version.

```go
import "github.com/larsartmann/go-cqrs-lite/snapshot/v3"

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
import "github.com/larsartmann/go-cqrs-lite/schema/v3"

// Upcast UserCreated v1 → v2 (adds a default field)
upcaster := schema.NewUpcaster("UserCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
    old, _ := event.DecodePayload[UserCreatedV1](evt, codec.JSONCodec{})
    newPayload, _ := codec.JSONCodec{}.Encode(UserCreatedV2{Name: old.Name, Email: ""})
    return event.NewEvent(evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
        newPayload, event.WithSchemaVersion(2))
})

versioned := schema.NewVersionedStore(eventStore, upcaster)
// versioned.Load transparently applies upcasters
```

### 2.6 Tamper-Proof Event Streams (signing)

Cryptographic signatures detect tampering in transit and at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/signing/v3"

signer, _ := signing.NewHMAC(secret)
bus.UsePublish(signing.SignMiddleware(signer))   // sign on publish
bus.Use(signing.VerifyMiddleware(signer))        // verify on receive
// Ed25519: signing.NewEd25519(privateKey, publicKey)
// Multisig: signing/v3/multisig
```

### 2.7 Encrypted Payloads (encryption)

Confidential event payloads encrypted at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/encryption/v3"

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
    "github.com/larsartmann/go-cqrs-lite/middleware/v3"
    "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

tracer := otel.GetTracerProvider().Tracer("my-app")
bus.Use(middleware.EventTracing(tracer))
bus.UsePublish(middleware.EventPublishTracing(tracer))

meter := otel.GetMeterProvider().Meter("my-app")
recorder, _ := middleware.NewOTelMetricsRecorder(meter)
cmdDispatcher.Use(middleware.CommandMetrics(recorder))

// Other middleware: Logging, Retry, Recovery, Validation, CircuitBreaker
cmdDispatcher.Use(middleware.CommandRecovery())
cmdDispatcher.Use(middleware.CommandRetry(3, time.Second))
```

> **Rule:** Import OTel via `otel/` re-exports, NOT `go.opentelemetry.io` directly.

#### Command Idempotency (dedup on retry)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/idempotency/v3"
    "github.com/larsartmann/go-cqrs-lite/middleware/v3"
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
qDisp.Use(middleware.QueryMetrics(recorder))

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
    "github.com/larsartmann/go-cqrs-lite/catalog/v3"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3/asyncapi"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3/d2"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3/eventcatalog"
    "github.com/larsartmann/go-cqrs-lite/catalog/v3/openapi"
)

reg := catalog.NewRegistry("My API", "1.0.0")
reg.RegisterEvent("user.created", catalog.SchemaFromType[UserCreated]())
reg.RegisterCommand("user.create", catalog.SchemaFromType[CreateUser]())

asyncAPIDoc, _ := asyncapi.Generate(reg)
openAPIDoc, _  := openapi.Generate(reg)
catalogDir, _  := eventcatalog.Generate(reg)
d2Diagram, _   := d2.Generate(reg)
```
