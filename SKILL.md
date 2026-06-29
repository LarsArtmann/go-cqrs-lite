# Skill: go-cqrs-lite — AI Consumer Guide

> **Activate when** a project imports any `github.com/larsartmann/go-cqrs-lite/*/v3` module, or when the user asks how to build CQRS / Event Sourcing systems in Go with this library.
>
> This is the **single source of truth for AI consumers**. It replaces the need to discover and read 28 module READMEs. AGENTS.md (in the library repo) is for contributors; this file is for users.

---

## 0. Mental Model (read this first)

go-cqrs-lite is a **library, not a framework**. You import only the modules you need and compose them. There is no `app.Init()`, no magic wiring, no imposed transport, broker, or SQL driver.

The core loop is:

```
Command → Dispatcher → Handler → Decider (load state → fold → decide → save events → publish)
                                                 ↓
                                      Event Store + Event Bus
                                                 ↓
                                      Projection → Read Model
                                                 ↓
Query   → Dispatcher → Handler → Read Model
```

**Three orthogonal axes you compose independently:**

| Axis              | Question                                    | Modules                                                                              |
| ----------------- | ------------------------------------------- | ------------------------------------------------------------------------------------ |
| **Write model**   | How do I decide + persist changes?          | `event`, `command`, `decider`, `id`                                                  |
| **Read model**    | How do I build queryable state from events? | `stack.Materialize`, `kv`, `listing`, `query`                                        |
| **Storage**       | Where do events/snapshots/checkpoints live? | `storage/memory`, `storage`, `storage/pebble`, `storage/turso`, `kv`, `stack`        |
| **Read models**   | How do I store/query typed projections?     | `kv.TypedStore`, `kv.Cache`                                                          |
| **Cross-cutting** | Security, evolution, observability, docs    | `signing`, `encryption`, `schema`, `middleware`, `otel`, `catalog`, `transport/http` |

You do NOT need all of them. Start with the minimal recipe (§2), then bolt on capabilities.

---

## 1. Module Decision Matrix — "I want to…"

| If you want to…                                       | Use                                                                             | See recipe |
| ----------------------------------------------------- | ------------------------------------------------------------------------------- | ---------- |
| Create/store/load events                              | `event`                                                                         | §2.1       |
| Dispatch type-safe commands                           | `command`                                                                       | §2.1       |
| Run an event-sourced aggregate                        | `decider`                                                                       | §2.1       |
| Generate unique, type-safe IDs                        | `id`                                                                            | §2.1       |
| Encode payloads as JSON/CBOR                          | `codec`                                                                         | §2.1       |
| Build a read model from events                        | `stack.Materialize` + `kv.TypedStore`                                           | §2.3       |
| Dispatch type-safe queries                            | `query`                                                                         | §2.3       |
| List all aggregates + their status                    | `listing`                                                                       | §6.4       |
| Persist to PostgreSQL / SQLite                        | `storage`                                                                       | §2.2       |
| Persist to embedded PebbleDB                          | `storage/pebble`                                                                | §2.2       |
| Offline-first sync via LibSQL                         | `storage/turso`                                                                 | §6.6       |
| Generic key-value abstraction                         | `kv`                                                                            | §6.7       |
| Snapshot aggregates for speed                         | `snapshot`                                                                      | §2.4       |
| Evolve event schemas over time                        | `schema`                                                                        | §2.5       |
| Make event streams tamper-proof                       | `signing`                                                                       | §2.6       |
| Encrypt confidential payloads                         | `encryption`                                                                    | §2.7       |
| Add logging/retry/recovery/circuit-breaker            | `middleware`                                                                    | §2.8       |
| Deduplicate commands on retry (idempotency)           | `idempotency`                                                                   | §2.8       |
| Add OpenTelemetry tracing/metrics                     | `otel` + `middleware`                                                           | §2.8       |
| Auto-generate AsyncAPI/OpenAPI/EventCatalog/D2 docs   | `catalog`                                                                       | §2.9       |
| Soft-delete aggregates without data loss              | `event` (tombstone metadata)                                                    | §6.1       |
| Generate typed handler boilerplate                    | `cmd/cqrs-gen`                                                                  | §6.7       |
| Publish events to Watermill router                    | `watermill`                                                                     | §6.4       |
| Dispatch commands/queries over gRPC                   | `transport/grpc`                                                                | §6.8       |
| Verify doc code references compile                    | `cmd/doc-check`                                                                 | §6.8       |
| In-memory command bus (typed pub/sub)                 | `command` (`NewMemoryBus`)                                                      | §2.1       |
| In-memory implementations for tests/dev               | `memory`                                                                        | §2.1       |
| One-call infrastructure wiring (Bundle presets)       | `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`, `stack/turso` | §2.0       |
| Typed read-model store over KV backend                | `kv.TypedStore`                                                                 | §2.0       |
| Cache decorator for read models                       | `kv.Cache`                                                                      | §2.0       |
| Run projections with crash-restart + checkpoint + DLQ | `projectionhost`                                                                | §6.9       |
| Test deciders/projections with Given/When/Then        | `scenario`                                                                      | §6.10      |
| Schedule delayed commands / durable deadlines         | `scheduling`                                                                    | §6.11      |
| Dead-letter failed dispatches (retry exhaustion)      | `middleware` (DLQ)                                                              | §2.8       |

---

## 2. Composition Recipes (copy-paste, verified APIs)

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
| Turso    | `stack/turso`    | LibSQL (turso)   | SQL KV (persistent) |

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
)

type UserState struct{ Name string }
type CreateUser struct{ Name string }
type UserCreated struct{ Name string }

func main() {
    ctx := context.Background()
    store := memory.NewMemoryStore()
    bus := watermill.NewEventBus()

    d := decider.Decider[UserState]{
        Initial: UserState{},
        Fold: func(s UserState, e event.Event) (UserState, error) {
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
import "github.com/larsartmann/go-cqrs-lite/pebble/v3"

backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
defer backend.Close() // closes DB AND all stores

eventStore  := backend.EventStore()
snapStore   := backend.SnapshotStore()
cpStore     := backend.CheckpointStore()
```

> **Rule:** `backend.Close()` closes the stores it owns, NOT an externally-passed `*sql.DB`. For Pebble `Open()`, it closes the DB too.

### 2.3 Read Models (projection + query)

Projections rebuild queryable state from the event stream. There are **two ways to run them** — pick based on your delivery model:

**Option A — `projectionhost` (pull-based, crash-restart, no bus dependency).** Reads directly from an `event.SeekableJournal`. Best for batch replay and systems without a live event bus.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/projection/v3"
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
)

// Define a projection implementing projection.Projection
type TodoProjection struct{ store ReadModel }
func (p *TodoProjection) Name() string { return "todo-read-model" }
func (p *TodoProjection) EventTypes() []event.Type {
    return []event.Type{"todo.created", "todo.updated"} // nil = all types
}
func (p *TodoProjection) Handle(ctx context.Context, evt event.Event) error {
    // mutate your read model
    return nil
}

// journal: any event.SeekableJournal (MemoryStore, SQLEventStore, pebble.EventStore, ...)
host, _ := projectionhost.New(journal, checkpointStore,
    projectionhost.WithBatchSize(100),
    projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 3),
)
_ = host.Register(&TodoProjection{store: readModel})
go host.Start(ctx)   // one goroutine per projection; crash auto-restart + backoff
defer host.Stop()    // graceful drain (30s timeout)
// host drains the journal from each projection's checkpoint, then idles.
// For live push delivery, pair with watermill/CatchUpSubscriber (Option B).
```

**Option B — `CatchUpSubscriber` (push-based, live tail after replay).** Pairs with `stack.Materialize` for ordered, durable projections. See §2.3 "Canonical projection pattern" below and §6.9 for the full `projectionhost` lifecycle.

Query the read model with type-safe dispatch:

```go
qDisp := query.NewDispatcher()
query.RegisterTyped(qDisp, "todo.get",
    func(ctx context.Context, q *GetTodoQuery) (*GetTodoResult, error) {
        return readModel.Get(q.ID)
    })

result, err := query.DispatchTyped[*GetTodoResult](ctx, qDisp, &GetTodoQuery{ID: id})
```

#### SQL-backed views (queryable columns, server-side filtering)

By default, `Materialize` stores views as opaque JSON blobs via `kv.TypedStore`.
For SQL backends (SQLite, Postgres), use `storage.SQLViewStore` to give each
view type its own table with **real, queryable SQL columns** — enabling WHERE,
ORDER BY, and LIMIT/OFFSET at the database level.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/storage/v3"
    "github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// 1. Define how your view maps to SQL columns.
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{
        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
        {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
        {Name: "tombstoned", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Tombstoned }},
    },
    ScanRow: func(scan func(dest ...any) error) (*TodoView, error) {
        var v TodoView
        var completed, tombstoned int
        if err := scan(&v.Title, &completed, &tombstoned); err != nil {
            return nil, err
        }
        v.Completed = completed != 0
        v.Tombstoned = tombstoned != 0
        return &v, nil
    },
    TombstoneColumn: "tombstoned", // enables server-side tombstone filtering in List
}

// 2. Create the SQL-backed view store (auto-creates the table).
store, _ := storage.NewSQLiteViewStore[TodoView, id.AggregateID](db, mapper)

// 3. Use it with Materialize — same API as KV-backed.
mat := stack.Materialize[TodoView, id.AggregateID]{
    Store:        store,               // ← kv.ViewStore interface, SQL-backed
    KeyFromEvent: todoKey,
}
configureMaterialize(&mat)  // OnCreate, OnUpdate, OnTombstone — identical

// 4. Query with SQL power (the killer feature):
//    WHERE, ORDER BY, LIMIT/OFFSET — pushed to the database.
results, _ := store.Query(ctx, kv.ViewQuery{
    Where:   "completed = ?",
    Args:    []any{0},
    OrderBy: "title",
    Limit:   20,
})

// 5. List automatically uses server-side tombstone filtering when
//    TombstoneColumn is configured (no full-table load).
active, _ := mat.List(ctx, stack.ExcludeTombstoned)
```

Shortcut: auto-generate the mapper from struct tags:

```go
type TodoView struct {
    Title      string `view:"title"`
    Completed  bool   `view:"completed"`
    Tombstoned bool   `view:"tombstoned"`
}
mapper := storage.AutoMapperWithTombstone[TodoView]("todos_view", "tombstoned")
// Equivalent to the manual ViewMapper above — zero boilerplate.
```

From a Bundle preset (the one-call path):

```go
b, _ := sqlite.New("app.db")
store, _ := sqlite.SQLViewModel[TodoView, TodoID](b, mapper) // ← uses bundle's DB
mat := stack.Materialize[TodoView, TodoID]{Store: store, ...}
```

Advanced capabilities (all optional, checked at runtime):

```go
// Count without loading records (SELECT COUNT(*))
n, _ := store.Count(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
})

// Batch upsert for projection replay throughput (chunks to respect SQLite 999-param limit)
_ = store.BatchSet(ctx, items)

// Structured (injection-safe) filtering — no raw SQL
results, _ := store.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{
        {Column: "completed", Op: kv.OpEq, Value: false},
        {Column: "title", Op: kv.OpLike, Value: "%urgent%"},
    },
    OrderBy: "title", Limit: 10})

// Projection reset (DELETE FROM table)
_ = store.DeleteAll(ctx)

// Secondary indexes for fast lookups (set on mapper)
mapper.Indexes = []storage.IndexSpec{
    {Name: "idx_title", Columns: []string{"title"}},
}
```

#### Canonical projection pattern: CatchUpSubscriber + Materialize

For ordered, durable projections, use `CatchUpSubscriber` — NOT the Watermill
Router directly. The Router processes messages in parallel (one goroutine per
message), which breaks ordering for projections that need FIFO guarantees.

The canonical pattern (see `example/deployer-first`):

```go
b, _ := sqlite.New("app.db")
defer b.Close()

// 1. Create the CatchUpSubscriber from the Bundle.
catchUp, _ := b.CatchUpSubscriber()
defer catchUp.Close()

// 2. Create a SQL-backed or KV-backed Materialize.
store, _ := sqlite.SQLViewModel[TodoView, TodoID](b, mapper)
mat := stack.Materialize[TodoView, TodoID]{Store: store, ...}

// 3. Subscribe to a topic — CatchUpSubscriber replays from the journal
//    (Phase 1, ModeReplay), then hands off to live (Phase 2).
msgs, _ := catchUp.Subscribe(ctx, "todo.created")

// 4. Consume from a SINGLE goroutine — FIFO ordering guaranteed.
go func() {
    for msg := range msgs {
        _ = mat.HandlerFunc()(msg)
        msg.Ack()
    }
}()
```

**Why not the Router?** The Router spawns one goroutine per message
(`message/router.go:30`). For ordered projections, consume the
CatchUpSubscriber's output channel from a single goroutine instead. The
EventBus default uses `BlockPublishUntilSubscriberAck=true` for ordered live
delivery and `Persistent=false` to avoid GoChannel's unordered persistent
replay (the CatchUpSubscriber handles replay from the journal instead).

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
// Multisig: signing/v2/multisig
```

### 2.7 Encrypted Payloads (encryption)

Confidential event payloads encrypted at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/encryption/v3"

enc, _ := encryption.NewXChaCha20Poly1305(key)   // or NewAES256GCM(key)
bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("key-v1")))
bus.Use(encryption.DecryptMiddleware(enc))

// Composable codec wrapper (JSON envelope, encrypted payload)
codec := encryption.NewCodec(codec.JSONCodec{}, enc)

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
)

store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()

// Rejects duplicate commands within the TTL. Default key: cmd.ID().
// Pass a custom KeyExtractor for client-supplied idempotency keys.
cmdDispatcher.Use(idempotency.CommandIdempotency(store, 10*time.Minute, nil))
```

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

---

## 3. Critical Conventions (AI gets these wrong)

These are **non-negotiable** rules. Violating them breaks the library's contracts.

### 3.1 Tombstone over delete — NEVER call Delete

There is **no `Delete` on Store**. Soft-delete via metadata:

```go
// Correct: mark tombstone
marked, _ := event.MarkTombstone(evt)
status := event.DetectTombstone(events) // Active | Tombstoned | Undetermined
```

Use `listing/` for tombstone-aware aggregate status read models.

### 3.2 Sink/Source split — use the right interface

`Store` is split for ISP. Don't take `Store` when you only need one side:

```go
var sink event.EventSink = store        // write: Save, AppendBatch
var source event.EventSource = store    // read: Load, LoadFromVersion, LoadToVersion, LoadToTimestamp
var journal event.Journal = store       // cross-aggregate: ReadAll
var seekable event.SeekableJournal = store // position-based: ReadFrom(afterID, limit)
```

### 3.3 Decode payloads with a codec — never type-assert

```go
// Correct
payload, err := event.DecodePayload[UserCreated](evt, codec.JSONCodec{})

// Wrong — Payload() returns []any, not your type
payload := evt.Payload().(UserCreated) // DON'T
```

### 3.4 OTel via otel/ — never go.opentelemetry.io directly

Modules must import `github.com/larsartmann/go-cqrs-lite/otel/v3`, not `go.opentelemetry.io/otel`. The otel module re-exports the needed types and keeps the SDK indirect in go.mod.

### 3.5 Strong types — no `any` in public APIs

The only exception is DB interop (`dialect.go`). Branded IDs prevent mixing ID types:

```go
type UserID = id.Of[struct{}]   // cannot be passed where OrderID is expected
uid := id.New[UserID]()
```

### 3.6 Defensive clone on accessors

`Payload()`, `Metadata()`, `EventTypes()` return **clones**, not internal references. For hot internal read-only paths, use `event.PayloadReadOnly(evt)` via `*ImmutableEvent` type assertion (zero-copy). This is internal-only.

### 3.7 Errors as values — 5 families, no panics

```go
event.NewRejection("user.create.empty_email", "...")    // client error, not retryable
event.NewConflict("user.create.duplicate", "...")        // optimistic concurrency
event.NewTransient("store.timeout", "...")               // retryable
event.NewInfrastructure("store.connection", "...")       // system-level
event.NewCorruption("store.invalid_event", "...")        // data integrity
```

### 3.8 Event causality for traceability

Link commands to the events they produced:

```go
ctx = event.WithCommandCausality(ctx, "user.create", cmdID)
// decider.Repository applies CommandCausalityEnricher(ctx) automatically
cmdType, cmdID, ok := event.CommandCausalityFromContext(ctx)
```

---

## 4. Anti-Patterns to Avoid

| Anti-pattern                               | Correct approach                                             |
| ------------------------------------------ | ------------------------------------------------------------ |
| Adding a `Delete()` method to Store        | Use tombstone metadata (`event.MarkTombstone`)               |
| Taking `Store` param when you only read    | Take `EventSource` or `Journal`                              |
| Type-asserting `evt.Payload()`             | Use `event.DecodePayload[T](evt, codec)`                     |
| Importing `go.opentelemetry.io` directly   | Import `otel/v2` re-exports                                  |
| Manually setting event version in `Decide` | Let `event.NewEvents` auto-increment from the passed version |
| Creating a saga/process-manager module     | Use projection + command dispatch (see `example/todo/`)      |
| Editing dependency go.mod files by hand    | Use `go get` commands                                        |
| Using `any` types in public APIs           | Use generics / branded types                                 |
| Storing the \*sql.DB lifetime in backend   | `backend.Close()` closes stores, NOT your `*sql.DB`          |

---

## 5. Module Reference (quick lookup)

### Core (Layer 0–3)

| Module       | Import          | One-liner                                                                                                                                                                                                                        |
| ------------ | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`         | `id/v3`         | Branded IDs: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. All 8 markers exported (`AggregateMarker`, `EventMarker`, `CommandMarker`, …) for `BrandNamer` integration. Custom via `id.Of[struct{}]`.                                     |
| `dispatcher` | `dispatcher/v3` | Generic `Dispatcher[H, M]` with `LifecycleMixin`. Base for command/query dispatchers.                                                                                                                                            |
| `codec`      | `codec/v3`      | Payload encoding: `JSONCodec{}`, `CBORCodec{}` (deterministic), `RawCodec{}`.                                                                                                                                                    |
| `event`      | `event/v3`      | `Event`, `Store` (=`EventSink`+`EventSource`), `Bus`, `Journal`, `SeekableJournal`, `NewEvent`, `NewEvents`, `DecodePayload[T]`, 5-family errors, tombstone (`TombstoneMark`), causality (`Causation`), `Tracing`, `Checkpoint`. |
| `command`    | `command/v3`    | `Dispatcher`, `Handler`, `RegisterTyped`, `BasicCommand`, `PersistedCommand`, `CommandSink`/`Source`, `CommandBus` (pub/sub).                                                                                                    |
| `query`      | `query/v3`      | `Dispatcher`, `TypedHandler[Q,R]`, `RegisterTyped`, `PaginatedResult[T]`, `PersistedQuery`, `QuerySink`/`Source`.                                                                                                                |
| `decider`    | `decider/v3`    | `Decider[State]{Initial, Fold}`, `Repository[State]` (`Execute`, `Load`, `LoadAtVersion`), snapshot integration.                                                                                                                 |

### Read models (Layer 4–5)

| Module    | Import       | One-liner                                                                                                              |
| --------- | ------------ | ---------------------------------------------------------------------------------------------------------------------- |
| `kv`      | `kv/v3`      | `ViewStore[V,K]` interface, `TypedStore[V,K]`, `Cache[V,K]`, `MemStore`. Foundation for all read models.               |
| `stack`   | `stack/v3`   | `Materialize[V,K]` (deployer-first projection builder), `Bundle`, presets. Accepts any `kv.ViewStore`.                 |
| `listing` | `listing/v3` | `AggregateListing`, `AggregateStatus` (Active/Tombstoned/Undetermined), `StatusMiddleware`, `InMemoryAggregateReader`. |
| `query`   | `query/v3`   | (see Core) — query the read model.                                                                                     |

### Storage (Layer 5)

| Module     | Import              | One-liner                                                                                                                                                                                                                                                                       |
| ---------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `memory`   | `storage/memory/v3` | `MemoryStore`, `MemorySnapshotStore`, `MemoryCheckpointStore`, `MemoryCommandStore`, `MemoryQueryStore`. Tests & dev. (`MemoryBus`/`MemoryCommandBus` deprecated — use `watermill.EventBus`)                                                                                    |
| `storage`  | `storage/v3`        | `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`, `SQLCommandStore`, `SQLQueryStore`, `SQLKVStore`, **`SQLViewStore`** (column-mapped views). PG/SQLite. `NewSQLiteBackend`/`NewSQLBackend` facade. `sql/` sub-package: `RunInTx`, `IsDuplicateKeyError`, `ScanSlice`. |
| `pebble`   | `storage/pebble/v3` | `EventStore`, `SnapshotStore`, `CheckpointStore`, `NewKVStore`. CBOR envelope. Shared DB via disjoint key prefixes. `Open()` facade.                                                                                                                                            |
| `turso`    | `storage/turso/v3`  | Turso/LibSQL connector, embedded sync, `indexing/` sub-package for index management. Delegates to `storage`.                                                                                                                                                                    |
| `kv`       | `kv/v3`             | `Store` (Reader+Writer+Closer), `MemStore`, `Iterator`, `Batch`, `TypedStore[T,K]`, `Cache[T,K]` (Otter LRU).                                                                                                                                                                   |
| `snapshot` | `snapshot/v3`       | `Snapshot`, `SnapshotSink`/`Source`/`Store`, `SnapshotStrategy`, `EveryNEvents(n)`.                                                                                                                                                                                             |
| `schema`   | `schema/v3`         | `Upcaster`, `VersionedStore`, `upcasterRegistry`. Schema evolution on read.                                                                                                                                                                                                     |

### Cross-cutting (Layer 4–5)

| Module           | Import              | One-liner                                                                                                                                      |
| ---------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `signing`        | `signing/v3`        | `NewHMAC`, `NewEd25519`, `multisig`, `SignMiddleware`/`VerifyMiddleware`. Tamper-proof streams.                                                |
| `encryption`     | `encryption/v3`     | `NewXChaCha20Poly1305`, `NewAES256GCM`, `Codec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`, `StaticKeyResolver`.                         |
| `middleware`     | `middleware/v3`     | `Logging`, `Retry`, `Recovery`, `Validation`, `Metrics`, `CircuitBreaker`, `EventTracing`, `CommandMetrics`, etc. For command + event + query. |
| `transport/http` | `transport/http/v3` | `NewSSEBroker`, `SSEHandler`. Bridges `event.Bus` to Server-Sent Events HTTP clients.                                                          |
| `otel`           | `otel/v3`           | `Tracer`, `Meter`, `Spans`, `Attributes`. Re-exports — import this, not go.opentelemetry.io.                                                   |
| `catalog`        | `catalog/v3`        | `Registry`, `SchemaFromType[T]()`, exporters: `asyncapi`, `d2`, `eventcatalog`, `openapi`.                                                     |
| `watermill`      | `watermill/v3`      | `EventBus` (GoChannel-backed, replaces `memory.MemoryBus`), `CatchUpSubscriber`, `EventPublisher`, `MessageToEvent`. ADR-0028.                 |

### Reliability & Testing (Layer 1–3)

| Module           | Import              | One-liner                                                                                                                    |
| ---------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `idempotency`    | `idempotency/v3`    | `Store`, `MemoryStore`, `KVStore` (any `kv.Store`+`ConditionalWriter`), `ErrDuplicate`, middleware + `KeyExtractor`. Dedup.  |
| `scheduling`     | `scheduling/v3`     | `TimerStore`, `MemoryTimerStore`, `Scheduler` (poll + retry). Idempotent durable deadlines ("cancel order after 30 min").    |
| `projection`     | `projection/v3`     | `Projection`, `NewProjection`. Consumer-side projection interface extracted from `event/`.                                   |
| `projectionhost` | `projectionhost/v3` | `Host`, `WorkerState`, `DeadLetterStore`, `MemoryDeadLetterStore`. Managed lifecycle: crash-restart, checkpoint, poison DLQ. |
| `scenario`       | `scenario/v3`       | Fluent BDD: `Given/When/Then`, `ThenError`, `ThenState`, `GivenProjection/ThenNoError`. Test deciders + projections.         |

### Tooling (Layer 6)

| Module              | Import               | One-liner                                                                                                    |
| ------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------ |
| `testutil`          | `testutil/v2`        | `MustNewCmd(tb, ...)`, `NoopCommandHandler`. Shared test helpers (zero panics).                              |
| `id/idtest`         | `id/v2/idtest`       | `ParseAggregateID(tb, s)`, `ParseEventID(tb, s)`. Branded-ID test helpers — `tb.Fatalf` on error, no panics. |
| `query/querytest`   | `query/v2/querytest` | `New(tb, queryType)`. Construct valid test queries — `tb.Fatalf` on error.                                   |
| `event/eventtest`   | `event/v2/eventtest` | `FakeStore`, `FakeBus`, `AssertGolden`. Event test doubles and golden test helpers.                          |
| `cmd/cqrs-gen`      | (go install)         | Code generator: typed handler registration from `//cqrs:command` / `//cqrs:query` markers.                   |
| `cmd/doc-check`     | (go run)             | Doc verifier: scans Markdown for Go code references, checks symbols exist.                                   |
| `cmd/api-stability` | (go install)         | API surface checker: compares exports against `docs/api_surface.txt` golden file.                            |
| `transport/grpc`    | `transport/grpc/v3`  | `RegisterCommandService`, `RegisterQueryService`, `NewCommandClient`, `NewQueryClient`. gRPC transport.      |

---

## 6. Advanced Patterns

### 6.1 Tombstone Soft-Delete & Rebirth

```go
// Delete: emit a tombstone event
marked, _ := event.MarkTombstone(evt)
store.Save(ctx, ref, []event.Event{marked}, expectedVersion)

// Detect: check aggregate status
status := event.DetectTombstone(events) // Active | Tombstoned | Undetermined

// Rebirth: emit a new event after tombstone (tombstone is just metadata)
// See example/user/ for the full tombstone + rebirth cycle
```

### 6.2 Command & Query Persistence (audit trail)

```go
// Persist commands for audit/replay
cmd, _ := command.NewPersistedCommand("user.create", ref, payload)
cmdStore.Save(ctx, ref, cmd)              // CommandSink
cmds, _ := cmdStore.Load(ctx, ref)        // CommandSource
var journal command.CommandJournal = cmdStore        // ReadAll (global)
var seekable command.SeekableCommandJournal = cmdStore // ReadFrom(afterCmdID, limit)

// Persist queries
pq, _ := query.NewPersistedQuery("user.get", payload)
qStore.SaveQuery(ctx, pq)                 // QuerySink
queries, _ := qStore.LoadQueries(ctx, after) // QuerySource
```

### 6.3 Aggregate Listing (read model for all aggregates)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/listing/v3"
    "github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// In-memory reader (consumes a Journal to track aggregate statuses)
reader := listing.NewInMemoryAggregateReader(journal)
builder := listing.NewListBuilder(reader)

// For SQL-backed listing, register the projection with the runner:
// proj, _ := storage.NewAggregateProjection(ctx, db, "aggregate_listing", dialect)
// runner.Register(proj)

// reader.List() → []AggregateListing with Status: Active | Tombstoned
```

### 6.4 Watermill Integration

```go
// Bridge go-cqrs-lite events to a Watermill router
publisher := watermill.NewPublisherAdapter(bus)      // wraps event.Publisher
subscriber := watermill.NewSubscriberAdapter(bus)     // wraps event.Bus
messages, _ := subscriber.Subscribe(ctx, "user.created")
// Use with standard Watermill handler funcs
```

### 6.5 Turso Offline-First

```go
import "github.com/larsartmann/go-cqrs-lite/turso/v3"

// Offline-first: local embedded LibSQL with background sync to Turso cloud
db, _ := turso.OpenSync(ctx, "file:local.db", "libsql://my-db.turso.io", authToken)
backend, _ := turso.NewBackend(db)
// Or without sync: db, _ := turso.Open("file:local.db")
```

### 6.6 Pebble as KV Store

```go
import (
    "github.com/cockroachdb/pebble"
    cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v3"
)

db, _ := pebble.Open(dir, &pebble.Options{})               // raw cockroachdb/pebble
kvStore, _ := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
defer kvStore.Close()
kvStore.Set([]byte("k"), []byte("v"))
val, _ := kvStore.Get([]byte("k"))
```

### 6.7 Code Generation (cqrs-gen)

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen/v3@latest
```

Add markers to your types:

```go
//cqrs:command
type CreateUser struct { Name string }

//cqrs:query
type GetUser struct { ID string }
```

Run:

```bash
cqrs-gen -type . -output handlers_gen.go -pkg myapp
```

Generates typed `Register*` boilerplate.

### 6.8 gRPC Transport (remote command/query dispatch)

Expose local dispatchers over gRPC, or dispatch to a remote CQRS server.

```go
import (
    "google.golang.org/grpc"
    cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

// --- Server side: expose your dispatchers over gRPC ---

srv := grpc.NewServer()
cqrsgrpc.RegisterCommandService(srv, cmdDispatcher) // cmdDispatcher: *command.Dispatcher
cqrsgrpc.RegisterQueryService(srv, qDispatcher)     // qDispatcher: *query.Dispatcher

lis, _ := net.Listen("tcp", ":50051")
go srv.Serve(lis)

// --- Client side: dispatch to a remote server ---

conn, _ := grpc.NewClient(
    "localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()), // or TLS
)
defer conn.Close()

// Command dispatch — transparent remote call
cmdClient := cqrsgrpc.NewCommandClient(conn)
err := cmdClient.Dispatch(ctx, myCommand) // same interface as local dispatcher

// Query dispatch — JSON result unmarshaled into your struct
qClient := cqrsgrpc.NewQueryClient(conn)
var result GetUserResult
err := qClient.Ask(ctx, "user.get", &result) // queryType + out pointer
```

Command payloads are carried in metadata (`metadata.Custom["payload"]`); handlers
extract them via `cmd.Metadata().Custom["payload"]`. Query results are JSON-encoded
on the wire. The `CommandClient` implements the same `Dispatch` interface as a local
dispatcher — swap them freely.

### 6.9 Managed Projection Host (crash-restart + checkpoint + DLQ)

The "last loop every consumer rewrites", now a library module. Composes any
`event.SeekableJournal` + `event.CheckpointStore` + your `projection.Projection`s
into a managed lifecycle with per-projection goroutines, exponential-backoff
restarts, persisted checkpoints, and a poison-message dead-letter queue.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/projection/v3"
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
)

// journal: any event.SeekableJournal (MemoryStore, SQLEventStore, pebble.EventStore, ...)
// cpStore: event.CheckpointStore (memory.MemoryCheckpointStore, SQLCheckpointStore, ...)
host, _ := projectionhost.New(journal, cpStore,
    projectionhost.WithBatchSize(100),
    projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 3), // poison after 3 retries
)
_ = host.Register(&UserProjection{})   // Register returns error; Name() must be unique
_ = host.Register(&OrderProjection{})

go host.Start(ctx)    // one goroutine per projection; crash auto-restart + exponential backoff
defer host.Stop()     // graceful drain (30s timeout)

for _, s := range host.Status() {     // health snapshot per worker
    fmt.Printf("%s: %s processed=%d errors=%d restarts=%d\n",
        s.Name, s.Status, s.Processed, s.Errors, s.Restarts)
}
// Worker states: idle, running, backoff, draining, stopped, failed.
// Reads directly from event.SeekableJournal — NO message-bus dependency.
// For live (push) delivery alongside replay, pair with watermill/CatchUpSubscriber.
```

### 6.10 Scenario-Testing DSL (Given/When/Then)

Fluent BDD for deciders and projections — no store or bus needed, just pure functions.

```go
import "github.com/larsartmann/go-cqrs-lite/scenario/v3"

// Decider: pure fold + pure decide
scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{},
    mustEvent(evtIncremented)).            // pre-existing events folded into state
    When(incrementCmd{}, decideIncrement).  // pure decide function
    Then(evtIncremented)                    // asserts emitted event TYPES
// Variants:
//   .ThenError(target)                 // asserts decide returns an error wrapping target
//   .ThenState(fold, initial, expected)// folds produced events, asserts final state

// Projection: feed events, assert no error
scenario.GivenProjection(t, &UserProjection{}, evt1, evt2).ThenNoError()
scenario.GivenProjection(t, &BrokenProj{}, badEvt).ThenError() // expects >= 1 error
```

### 6.11 Scheduled Commands / Durable Deadlines

Classic ES need — "cancel the order 30 minutes after creation if still unpaid" — as a
library primitive. `TimerStore` persists timers across restarts; `Scheduler` polls and
dispatches. Scheduling is idempotent (same `TimerID` is a no-op), so it is safe to
re-schedule on command retry.

```go
import "github.com/larsartmann/go-cqrs-lite/scheduling/v3"

store := scheduling.NewMemoryTimerStore()
sched := scheduling.New(store, func(ctx context.Context, t scheduling.Timer) error {
    return cmdDispatcher.Dispatch(ctx, t.Payload.(CancelOrderCmd))
},
    scheduling.WithPollInterval(500*time.Millisecond),
    scheduling.WithMaxRetries(5),
)

_ = store.Schedule(ctx, scheduling.Timer{
    ID:      "order-123-timeout",
    FireAt:  time.Now().Add(30 * time.Minute),
    Payload: CancelOrderCmd{OrderID: "123"},
})
_ = store.Cancel(ctx, "order-123-timeout") // order paid → cancel the timeout

go sched.Start(ctx) // polls Due(), dispatches via callback, MarkFired(); retries failures
```

---

## 7. Testing Patterns

```go
// In-memory test implementations
store := memory.NewMemoryStore()
bus := watermill.NewEventBus()
snapStore := memory.NewMemorySnapshotStore()
cpStore := memory.NewMemoryCheckpointStore()

// Event test helpers (for golden tests, assertions)
import "github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"

eventtest.AssertGolden(t, "testdata/event.golden", got, update)

// Shared test utilities
import "github.com/larsartmann/go-cqrs-lite/testutil/v3"

cmd := testutil.NewCmd(t, "user.create", aggID) // t = *testing.T
```

---

## 8. Dependency Layering (module graph)

```
Layer 0: id/, dispatcher/, codec/, kv/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher, ro), query/ (→dispatcher, ro)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot)
Layer 4: memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability
```

**Saga pattern:** No dedicated module. Multi-step orchestration = projection + command dispatch. See `example/todo/`.

---

## 9. Examples in the Repo

| Example        | Path                  | Demonstrates                                                         |
| -------------- | --------------------- | -------------------------------------------------------------------- |
| **todo**       | `example/todo/`       | Full app: HTTP API, decider, projections, queries, Pebble storage    |
| **user**       | `example/user/`       | Advanced: signing, middleware chains, catalog gen, tombstone/rebirth |
| **encryption** | `example/encryption/` | Bus-level + store-level encryption, key rotation                     |

---

## 10. Where to Find More

| Need                    | Source                                                         |
| ----------------------- | -------------------------------------------------------------- |
| Per-module API details  | Each module's `README.md` and `doc.go` (renders on pkg.go.dev) |
| Architectural decisions | `docs/adr/` (23 ADRs)                                          |
| Storage deep-dive       | `docs/STORAGE_GUIDE.md`                                        |
| Error system            | `docs/error-taxonomy.md`                                       |
| Signing internals       | `docs/signing-architecture.md`                                 |
| Domain glossary         | `docs/DOMAIN_LANGUAGE.md`                                      |
| Migration guides        | `docs/MIGRATION.md`, `docs/MIGRATION_v1.md`                    |
| Feature inventory       | `FEATURES.md`                                                  |
| Contributor guide       | `AGENTS.md` (in repo)                                          |

---

## 11. Quick API Cheat Sheet

```go
// Events
evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1), payload, opts...)
events, _ := event.NewEvents(aggID, "User", baseVersion, []event.Type{...}, []any{...})
p, _ := event.DecodePayload[T](evt, codec.JSONCodec{})
ref := event.NewAggregateRef("User", aggID)

// Store (Sink/Source split)
store.Save(ctx, ref, events, expectedVersion)    // optimistic concurrency
events, _ := store.Load(ctx, ref)
events, _ := store.LoadFromVersion(ctx, ref, v)
allEvents, _ := journal.ReadAll(ctx)              // cross-aggregate

// Bus
bus.Publish(ctx, evt1, evt2)
bus.Subscribe("user.created", handler)
bus.Use(middleware...)
bus.UsePublish(middleware...)

// Decider
d := decider.Decider[State]{Initial: initState, Fold: foldFunc}
repo, _ := decider.NewRepository[State](store, bus, d)
repo.Execute(ctx, aggID, "User", decideFunc)      // load → fold → decide → save → publish
state, ver, _ := repo.Load(ctx, aggID, "User")

// Commands
cmds := command.NewDispatcher()
command.RegisterTyped(cmds, "user.create", handlerFunc)
cmds.Use(middleware.CommandRecovery())
cmds.Dispatch(ctx, cmd)

// Queries
qDisp := query.NewDispatcher()
query.RegisterTyped(qDisp, "user.get", handlerFunc)
result, _ := query.DispatchTyped[*Result](ctx, qDisp, q)

// IDs
aggID := id.NewAggregateID()
eventID := id.NewEventID()
type OrderID = id.Of[struct{}]
orderID := id.New[OrderID]()

// Codec
data, _ := codec.JSONCodec{}.Encode(payload)
payload, _ := codec.JSONCodec{}.Decode(data)
```

---

## 12. Common Pitfalls (FAQ)

### "My event payload won't decode"

**Cause:** `event.NewEvent` takes `[]byte`, not a struct. You must encode the payload before passing it.

```go
// Wrong — won't compile (struct where []byte expected)
evt, _ := event.NewEvent("user.created", aggID, "User", 1, UserCreated{Name: "Alice"})

// Correct — encode first
payload, _ := codec.JSONCodec{}.Encode(UserCreated{Name: "Alice"})
evt, _ := event.NewEvent("user.created", aggID, "User", 1, payload)

// Or use NewEvents (accepts []any, encodes internally)
events, _ := event.NewEvents(aggID, "User", 0,
    []event.Type{"user.created"}, []any{UserCreated{Name: "Alice"}})
```

### "My decider Repository won't load — type parameter error"

**Cause:** Go infers the type parameter from the `Decider[State]` argument, so you rarely need to specify it explicitly.

```go
// Both work — the second is more idiomatic
repo, _ := decider.NewRepository[UserState](store, bus, d)
repo, _ := decider.NewRepository(store, bus, d) // type inferred from d (Decider[UserState])
```

### "snapshot.EveryNEvents returns an error"

**Cause:** It returns `(SnapshotStrategy, error)`, not just the strategy. Handle the error:

```go
strategy, _ := snapshot.EveryNEvents(100) // ← returns two values
repo, _ := decider.NewRepository(store, bus, d, decider.WithSnapshotStrategy(strategy))
```

### "Projection Builder — `.On()` doesn't exist as a method"

**Cause:** `On` is a **free function** with a type parameter, not a method on `*Builder`:

```go
// Wrong
b.On("user.created", handler)

// Correct — free function with type parameter
projection.On[UserCreated](b, "user.created", codec.JSONCodec{}, handler)
```

### "Pebble KV — `NewKVAdapter` not found"

**Cause:** The constructor is `NewKVStore`, not `NewKVAdapter`. The option is `WithSyncWrites()`, not `WithKVSyncWrites()`:

```go
// Wrong
kvStore := pebble.NewKVAdapter(db, pebble.WithKVSyncWrites())

// Correct
kvStore, _ := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
```

### "My SQL backend needs a dialect parameter"

**Cause:** `NewSQLBackend` infers the dialect from the `*sql.DB` driver name — no explicit dialect needed:

```go
// Wrong
backend, _ := storage.NewSQLBackend(db, sql.PostgresDialect{})

// Correct — dialect auto-detected
backend, _ := storage.NewSQLBackend(db)
```

### "catalog.NewRegistry needs arguments"

**Cause:** It requires a title and version:

```go
// Wrong
reg := catalog.NewRegistry()

// Correct
reg := catalog.NewRegistry("My API", "1.0.0")
```
