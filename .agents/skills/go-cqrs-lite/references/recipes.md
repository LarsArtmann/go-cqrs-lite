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

**Option B — `CatchUpSubscriber` (push-based, live tail after replay).** Pairs with `stack.Materialize` for ordered, durable projections. See §2.3 "Canonical projection pattern" below and advanced.md §6.9 for the full `projectionhost` lifecycle.

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

#### Choosing a projection tier: KV vs Relational vs Graph

There are **three projection tiers** — pick by read-access pattern:

| Tier            | Module                               | Writes ONE event to…       | Best for                                          |
| --------------- | ------------------------------------ | -------------------------- | ------------------------------------------------- |
| **Document/KV** | `stack.Materialize` + `kv.ViewStore` | one record in one table    | single-entity lookups, CRUD-style reads           |
| **Relational**  | `storage.RelationalProjection`       | several related SQL tables | multi-table joins, WHERE/ORDER BY, set predicates |
| **Graph**       | `graph.GraphProjection`              | nodes + edges              | variable-depth traversal, path-finding, adjacency |

`SQLViewStore` (above) is the document tier with queryable columns — still one
record per event. When a single event must update several related tables
atomically (a message + its attachments[] + a member_roles junction), use
`RelationalProjection` instead. For N-hop queries, use `GraphProjection` (see advanced.md §6.13).

```go
import "github.com/larsartmann/go-cqrs-lite/storage/v3"

// Relational: one event → many tables, atomic, dialect-agnostic.
schema := storage.RelationalSchema{Tables: []storage.RelationalTable{
    {Name: "messages", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
        {Name: "id", Type: "TEXT"}, {Name: "channel_id", Type: "TEXT"}, {Name: "content", Type: "TEXT"},
    }},
    {Name: "attachments", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
        {Name: "id", Type: "TEXT"}, {Name: "message_id", Type: "TEXT"}, {Name: "filename", Type: "TEXT"},
    }},
}}
proj, _ := storage.NewRelationalProjection("messages", schema, db, sqlpkg.SQLiteDialect{},
    func(ctx context.Context, evt event.Event, sink storage.ProjectionSink) error {
        var p MessageCreated
        _ = json.Unmarshal(evt.Payload(), &p)
        sink.Upsert(ctx, "messages", storage.Row{"id": p.ID, "channel_id": p.ChannelID, "content": p.Content})
        for _, a := range p.Attachments {
            sink.Ensure(ctx, "attachments", storage.Row{"id": a.ID, "message_id": p.ID, "filename": a.Name})
        }
        return nil // all writes commit atomically; error → full rollback
    }, []event.Type{"MESSAGE_CREATED"})
// proj implements projection.Projection → register with projectionhost or CatchUpSubscriber.
// SQL-ONLY (SQLite/Postgres). For KV backends use stack.Materialize; for graph see advanced.md §6.13.
```

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
