# Skill: go-cqrs-lite — AI Consumer Guide

> **Activate when** a project imports any `github.com/larsartmann/go-cqrs-lite/*/v2` module, or when the user asks how to build CQRS / Event Sourcing systems in Go with this library.
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

| Axis              | Question                                    | Modules                                                            |
| ----------------- | ------------------------------------------- | ------------------------------------------------------------------ |
| **Write model**   | How do I decide + persist changes?          | `event`, `command`, `decider`, `id`                                |
| **Read model**    | How do I build queryable state from events? | `projection`, `listing`, `query`                                   |
| **Storage**       | Where do events/snapshots/checkpoints live? | `memory`, `storage`, `pebble`, `turso`, `kv`, `stack`              |
| **Read models**   | How do I store/query typed projections?     | `readmodel`, `readmodel/cache`                                     |
| **Cross-cutting** | Security, evolution, observability, docs    | `signing`, `encryption`, `schema`, `middleware`, `otel`, `catalog` |

You do NOT need all of them. Start with the minimal recipe (§2), then bolt on capabilities.

---

## 1. Module Decision Matrix — "I want to…"

| If you want to…                                     | Use                                                              | See recipe |
| --------------------------------------------------- | ---------------------------------------------------------------- | ---------- |
| Create/store/load events                            | `event`                                                          | §2.1       |
| Dispatch type-safe commands                         | `command`                                                        | §2.1       |
| Run an event-sourced aggregate                      | `decider`                                                        | §2.1       |
| Generate unique, type-safe IDs                      | `id`                                                             | §2.1       |
| Encode payloads as JSON/CBOR                        | `codec`                                                          | §2.1       |
| Build a read model from events                      | `projection`                                                     | §2.3       |
| Dispatch type-safe queries                          | `query`                                                          | §2.3       |
| List all aggregates + their status                  | `listing`                                                        | §6.4       |
| Persist to PostgreSQL / SQLite                      | `storage`                                                        | §2.2       |
| Persist to embedded PebbleDB                        | `pebble`                                                         | §2.2       |
| Offline-first sync via LibSQL                       | `turso`                                                          | §6.6       |
| Generic key-value abstraction                       | `kv`                                                             | §6.7       |
| Snapshot aggregates for speed                       | `snapshot`                                                       | §2.4       |
| Evolve event schemas over time                      | `schema`                                                         | §2.5       |
| Make event streams tamper-proof                     | `signing`                                                        | §2.6       |
| Encrypt confidential payloads                       | `encryption`                                                     | §2.7       |
| Add logging/retry/recovery/circuit-breaker          | `middleware`                                                     | §2.8       |
| Add OpenTelemetry tracing/metrics                   | `otel` + `middleware`                                            | §2.8       |
| Auto-generate AsyncAPI/OpenAPI/EventCatalog/D2 docs | `catalog`                                                        | §2.9       |
| Soft-delete aggregates without data loss            | `event` (tombstone metadata)                                     | §6.1       |
| Generate typed handler boilerplate                  | `cmd/cqrs-gen`                                                   | §6.8       |
| Publish events to Watermill router                  | `watermill`                                                      | §6.5       |
| In-memory implementations for tests/dev             | `memory`                                                         | §2.1       |
| One-call infrastructure wiring (Bundle presets)     | `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres` | §2.0       |
| Typed read-model store over KV backend              | `readmodel`                                                      | §2.0       |
| Cache decorator for read models                     | `readmodel/cache`                                                | §2.0       |

---

## 2. Composition Recipes (copy-paste, verified APIs)

### 2.0 Bundle Presets — one-call infrastructure wiring

> **New in v2.7.** Consumers should NOT decide on infrastructure manually.
> The deployer picks a preset; the app developer never imports a backend.

```go
import cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"

// One call wires: event store + bus, command store, query store,
// snapshot store, checkpoint store, read-model backend.
b, err := cqrspebble.New("/var/lib/myapp/pebble")
defer b.Close()

// Typed read model over the Bundle's shared KV backend
store, _ := stack.ReadModel[TodoView, TodoID](b, codec.JSONCodec{},
    readmodel.WithKeyPrefix[TodoView, TodoID]("todos:"))

// Command handlers use b.EventSink (asserts to event.Store)
// Queries use the read model store
// Projections use b.Journal + b.Subscriber + b.CheckpointStore
```

Available presets:

| Preset   | Module           | Backend          | Read Models         |
| -------- | ---------------- | ---------------- | ------------------- |
| Memory   | `stack/memory`   | In-memory        | Memory KV           |
| SQLite   | `stack/sqlite`   | SQLite (modernc) | SQL KV (persistent) |
| Pebble   | `stack/pebble`   | PebbleDB (LSM)   | Pebble KV           |
| Postgres | `stack/postgres` | PostgreSQL (pgx) | SQL KV (persistent) |

Read-model cache decorator:

```go
cached, _ := cache.New(store,
    cache.WithCapacity[TodoView, TodoID](10_000),
    cache.WithTTL[TodoView, TodoID](5*time.Minute))
```

See [`docs/PRESETS.md`](docs/PRESETS.md) for full documentation.

### 2.1 Minimal Event Sourcing (event + command + decider + id + memory)

The foundation. Every app starts here.

```go
package main

import (
    "context"
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/codec/v2"
    "github.com/larsartmann/go-cqrs-lite/command/v2"
    "github.com/larsartmann/go-cqrs-lite/decider/v2"
    "github.com/larsartmann/go-cqrs-lite/event/v2"
    "github.com/larsartmann/go-cqrs-lite/id/v2"
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
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
import "github.com/larsartmann/go-cqrs-lite/storage/v2"

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
import "github.com/larsartmann/go-cqrs-lite/pebble/v2"

backend, _ := pebble.Open(dir, &pebble.Options{}, logger)
defer backend.Close() // closes DB AND all stores

eventStore  := backend.EventStore()
snapStore   := backend.SnapshotStore()
cpStore     := backend.CheckpointStore()
```

> **Rule:** `backend.Close()` closes the stores it owns, NOT an externally-passed `*sql.DB`. For Pebble `Open()`, it closes the DB too.

### 2.3 Read Models (projection + query)

Projections rebuild queryable state from the event stream. The runner does **replay-first, then live-tail** with checkpoint management.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/projection/v2"
    "github.com/larsartmann/go-cqrs-lite/query/v2"
)

// Define a projection implementing event.Projection
type TodoProjection struct{ store ReadModel }
func (p *TodoProjection) Name() string { return "todo-read-model" }
func (p *TodoProjection) EventTypes() []event.Type {
    return []event.Type{"todo.created", "todo.updated"} // filter the stream
}
func (p *TodoProjection) Handle(ctx context.Context, evt event.Event) error {
    // mutate your read model
    return nil
}

// Wire the runner: eventStore + eventBus + checkpointStore
runner, _ := projection.NewRunner(eventStore, eventBus, checkpointStore)
_ = runner.Register(&TodoProjection{store: readModel})

// Read-your-writes: replay synchronously, then tail live in the background.
if err := runner.RunReplay(ctx); err != nil { /* handle */ }  // blocks until caught up
go func() { _ = runner.RunLive(ctx) }()                      // background tail
// ← read model is guaranteed caught up here, no sleep needed
```

Query the read model with type-safe dispatch:

```go
qDisp := query.NewDispatcher()
query.RegisterTyped(qDisp, "todo.get",
    func(ctx context.Context, q *GetTodoQuery) (*GetTodoResult, error) {
        return readModel.Get(q.ID)
    })

result, err := query.DispatchTyped[*GetTodoResult](ctx, qDisp, &GetTodoQuery{ID: id})
```

### 2.4 Snapshots for Performance (snapshot)

Avoid replaying long event streams. Snapshots cache aggregate state at a version.

```go
import "github.com/larsartmann/go-cqrs-lite/snapshot/v2"

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
import "github.com/larsartmann/go-cqrs-lite/schema/v2"

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
import "github.com/larsartmann/go-cqrs-lite/signing/v2"

signer, _ := signing.NewHMAC(secret)
bus.UsePublish(signing.SignMiddleware(signer))   // sign on publish
bus.Use(signing.VerifyMiddleware(signer))        // verify on receive
// Ed25519: signing.NewEd25519(privateKey, publicKey)
// Multisig: signing/v2/multisig
```

### 2.7 Encrypted Payloads (encryption)

Confidential event payloads encrypted at rest.

```go
import "github.com/larsartmann/go-cqrs-lite/encryption/v2"

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
    "github.com/larsartmann/go-cqrs-lite/middleware/v2"
    "github.com/larsartmann/go-cqrs-lite/otel/v2"
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

### 2.9 Auto-Documentation (catalog)

Generate AsyncAPI 3.0, EventCatalog, OpenAPI, and D2 diagrams from your Go types.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/catalog/v2"
    "github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi"
    "github.com/larsartmann/go-cqrs-lite/catalog/v2/d2"
    "github.com/larsartmann/go-cqrs-lite/catalog/v2/eventcatalog"
    "github.com/larsartmann/go-cqrs-lite/catalog/v2/openapi"
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

Modules must import `github.com/larsartmann/go-cqrs-lite/otel/v2`, not `go.opentelemetry.io/otel`. The otel module re-exports the needed types and keeps the SDK indirect in go.mod.

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
| `id`         | `id/v2`         | Branded IDs: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. All 8 markers exported (`AggregateMarker`, `EventMarker`, `CommandMarker`, …) for `BrandNamer` integration. Custom via `id.Of[struct{}]`.                                     |
| `dispatcher` | `dispatcher/v2` | Generic `Dispatcher[H, M]` with `LifecycleMixin`. Base for command/query dispatchers.                                                                                                                                            |
| `codec`      | `codec/v2`      | Payload encoding: `JSONCodec{}`, `CBORCodec{}` (deterministic), `RawCodec{}`.                                                                                                                                                    |
| `event`      | `event/v2`      | `Event`, `Store` (=`EventSink`+`EventSource`), `Bus`, `Journal`, `SeekableJournal`, `NewEvent`, `NewEvents`, `DecodePayload[T]`, 5-family errors, tombstone (`TombstoneMark`), causality (`Causation`), `Tracing`, `Checkpoint`. |
| `command`    | `command/v2`    | `Dispatcher`, `Handler`, `RegisterTyped`, `BasicCommand`, `PersistedCommand`, `CommandSink`/`Source`, `CommandBus` (pub/sub).                                                                                                    |
| `query`      | `query/v2`      | `Dispatcher`, `TypedHandler[Q,R]`, `RegisterTyped`, `PaginatedResult[T]`, `PersistedQuery`, `QuerySink`/`Source`.                                                                                                                |
| `decider`    | `decider/v2`    | `Decider[State]{Initial, Fold}`, `Repository[State]` (`Execute`, `Load`, `LoadAtVersion`), snapshot integration.                                                                                                                 |

### Read models (Layer 4–5)

| Module       | Import          | One-liner                                                                                                              |
| ------------ | --------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `projection` | `projection/v2` | `Runner` (replay + live tail), `Builder` with `On[T]()`, `HandlerRegistry`, checkpoint-managed.                        |
| `listing`    | `listing/v2`    | `AggregateListing`, `AggregateStatus` (Active/Tombstoned/Undetermined), `StatusMiddleware`, `InMemoryAggregateReader`. |
| `query`      | `query/v2`      | (see Core) — query the read model.                                                                                     |

### Storage (Layer 5)

| Module     | Import        | One-liner                                                                                                                                                                                                               |
| ---------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `memory`   | `memory/v2`   | `MemoryStore`, `MemorySnapshotStore`, `MemoryCheckpointStore`, `MemoryCommandStore`, `MemoryQueryStore`. Tests & dev. (`MemoryBus`/`MemoryCommandBus` deprecated — use `watermill.EventBus`)                            |
| `storage`  | `storage/v2`  | `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`, `SQLCommandStore`, `SQLQueryStore`. PG/SQLite. `NewSQLiteBackend`/`NewSQLBackend` facade. `sql/` sub-package: `RunInTx`, `IsDuplicateKeyError`, `ScanSlice`. |
| `pebble`   | `pebble/v2`   | `EventStore`, `SnapshotStore`, `CheckpointStore`, `NewKVStore`. CBOR envelope. Shared DB via disjoint key prefixes. `Open()` facade.                                                                                    |
| `turso`    | `turso/v2`    | Turso/LibSQL connector, embedded sync, `indexing/` sub-package for index management. Delegates to `storage`.                                                                                                            |
| `kv`       | `kv/v2`       | `Store` (Reader+Writer+Closer), `MemStore`, `Iterator`, `Batch`, `TypedStore[T,K]`, `Cache[T,K]` (Otter LRU).                                                                                                           |
| `snapshot` | `snapshot/v2` | `Snapshot`, `SnapshotSink`/`Source`/`Store`, `SnapshotStrategy`, `EveryNEvents(n)`.                                                                                                                                     |
| `schema`   | `schema/v2`   | `Upcaster`, `VersionedStore`, `upcasterRegistry`. Schema evolution on read.                                                                                                                                             |

### Cross-cutting (Layer 4–5)

| Module           | Import              | One-liner                                                                                                                                      |
| ---------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `signing`        | `signing/v2`        | `NewHMAC`, `NewEd25519`, `multisig`, `SignMiddleware`/`VerifyMiddleware`. Tamper-proof streams.                                                |
| `encryption`     | `encryption/v2`     | `NewXChaCha20Poly1305`, `NewAES256GCM`, `Codec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`, `StaticKeyResolver`.                         |
| `middleware`     | `middleware/v2`     | `Logging`, `Retry`, `Recovery`, `Validation`, `Metrics`, `CircuitBreaker`, `EventTracing`, `CommandMetrics`, etc. For command + event + query. |
| `transport/http` | `transport/http/v2` | `NewSSEBroker`, `SSEHandler`. Bridges `event.Bus` to Server-Sent Events HTTP clients.                                                          |
| `otel`           | `otel/v2`           | `Tracer`, `Meter`, `Spans`, `Attributes`. Re-exports — import this, not go.opentelemetry.io.                                                   |
| `catalog`        | `catalog/v2`        | `Registry`, `SchemaFromType[T]()`, exporters: `asyncapi`, `d2`, `eventcatalog`, `openapi`.                                                     |
| `watermill`      | `watermill/v2`      | `EventBus` (GoChannel-backed, replaces `memory.MemoryBus`), `CatchUpSubscriber`, `EventPublisher`, `MessageToEvent`. ADR-0028.                 |

### Tooling (Layer 6)

| Module              | Import        | One-liner                                                                                  |
| ------------------- | ------------- | ------------------------------------------------------------------------------------------ |
| `testutil`          | `testutil/v2` | `NewCmd(tb, ...)`, `NoopCommandHandler`. Shared test helpers (zero panics).                |
| `cmd/cqrs-gen`      | (go install)  | Code generator: typed handler registration from `//cqrs:command` / `//cqrs:query` markers. |
| `cmd/api-stability` | (go install)  | API surface checker: compares exports against `docs/api_surface.txt` golden file.          |

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
    "github.com/larsartmann/go-cqrs-lite/listing/v2"
    "github.com/larsartmann/go-cqrs-lite/storage/v2"
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
import "github.com/larsartmann/go-cqrs-lite/turso/v2"

// Offline-first: local embedded LibSQL with background sync to Turso cloud
db, _ := turso.OpenSync(ctx, "file:local.db", "libsql://my-db.turso.io", authToken)
backend, _ := turso.NewBackend(db)
// Or without sync: db, _ := turso.Open("file:local.db")
```

### 6.6 Pebble as KV Store

```go
import (
    "github.com/cockroachdb/pebble"
    cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v2"
)

db, _ := pebble.Open(dir, &pebble.Options{})               // raw cockroachdb/pebble
kvStore := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
defer kvStore.Close()
kvStore.Set([]byte("k"), []byte("v"))
val, _ := kvStore.Get([]byte("k"))
```

### 6.7 Code Generation (cqrs-gen)

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen/v2@latest
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

---

## 7. Testing Patterns

```go
// In-memory test implementations
store := memory.NewMemoryStore()
bus := watermill.NewEventBus()
snapStore := memory.NewMemorySnapshotStore()
cpStore := memory.NewMemoryCheckpointStore()

// Event test helpers (for golden tests, assertions)
import "github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"

eventtest.AssertGolden(t, "testdata/event.golden", got, update)

// Shared test utilities
import "github.com/larsartmann/go-cqrs-lite/testutil/v2"

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
kvStore := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
```

### "My SQL backend needs a dialect parameter"

**Cause:** `NewSQLBackend` infers the dialect from the `*sql.DB` driver name — no explicit dialect needed:

```go
// Wrong
backend, _ := storage.NewSQLBackend(db, storage.DialectPostgres)

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
