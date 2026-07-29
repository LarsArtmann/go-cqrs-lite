## Core Guide — mental model, module selection, conventions, cheat sheet

> Read this first. It holds the decision-making material you need on every task: the mental model, the module decision matrix, the non-negotiable conventions, the anti-patterns, and the API cheat sheet. For long copy-paste recipes, jump to [`recipes.md`](recipes.md); for read-model patterns see [`readmodels.md`](readmodels.md); for pitfalls see [`faq.md`](faq.md).

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
| **Read model**    | How do I build queryable state from events? | `stack.Materialize`, `kv` (`TypedStore`, `Cache`), `listing`, `query`                |
| **Storage**       | Where do events/snapshots/checkpoints live? | `storage/memory`, `storage`, `storage/pebble`, `storage/turso`, `kv`, `stack`        |
| **Cross-cutting** | Security, evolution, observability, docs    | `signing`, `encryption`, `schema`, `middleware`, `otel`, `catalog`, `transport/http` |

You do NOT need all of them. Start with the 30-second quickstart below, then use §1 to pick modules.

### 30-second quickstart — "Hello CQRS"

The minimal loop: define state + events → wire infrastructure → execute → query. This is the
complete example from `example/getting-started/main.go` distilled to its essence:

```go
// 1. Define your domain (pure functions, no framework coupling)
type CounterState struct{ Value int }
type Incremented struct{ Amount int }

func apply(s CounterState, evt event.Event) (CounterState, error) {
    p, _ := event.DecodePayloadAuto[Incremented](evt)
    s.Value += p.Amount
    return s, nil
}

// 2. Wire infrastructure (one call — swap memory→sqlite→pebble by changing ONE line)
bundle, _ := stack.New(
    stack.WithEventStore(memory.NewMemoryStore()),
    stack.WithBus(cqrswatermill.NewEventBus()),
    stack.WithReadModels(kv.NewMemStore()),
    stack.WithCheckpointStore(memory.NewMemoryCheckpointStore()),
)
defer bundle.Close()

// 3. Create repository (load → fold → decide → save → publish in one call)
repo, _ := stack.Repository(bundle, decider.Decider[CounterState]{
    Initial: CounterState{}, Apply: apply,
})

// 4. Execute commands — events are sourced and published
ctx := context.Background()
aggID := id.NewStreamID()
_ = repo.Execute(ctx, aggID, "Counter", func(_ CounterState, v event.Version) ([]event.Event, error) {
    evt, _ := event.New("counter.incremented", aggID, "Counter", v.Increment(), Incremented{Amount: 5})
    return []event.Event{evt}, nil
})

// 5. Query the materialized view (projection not shown — see example/getting-started)
//    view, _ := mat.View(ctx, aggID)
//    fmt.Println(view.Value) // 5
```

Swap `memory.NewMemoryStore()` → `sqlite.New("app.db")` → `pebble.New("./data")` for persistence.
**The domain code doesn't change.** See `example/getting-started/` for the full runnable version
with projection + read model.

---

## 1. Module Decision Matrix — "I want to…"

| If you want to…                                       | Use                                                                             | See recipe      |
| ----------------------------------------------------- | ------------------------------------------------------------------------------- | --------------- |
| Create/store/load events                              | `event`                                                                         | recipes §2.1    |
| Dispatch type-safe commands                           | `command`                                                                       | recipes §2.1    |
| Run an event-sourced stream                           | `decider`                                                                       | recipes §2.1    |
| Generate unique, type-safe IDs                        | `id`                                                                            | recipes §2.1    |
| Typed event metadata (tracing, custom data)           | `metadata`                                                                      | —               |
| Encode payloads as JSON/CBOR                          | `codec`                                                                         | recipes §2.1    |
| Build a read model from events                        | `stack.Materialize` + `kv.ViewStore` (see tier table below)                     | readmodels §2.3 |
| Multi-table projection (composite keys, junctions)    | `storage.RelationalProjection`                                                  | readmodels §2.3 |
| Dispatch type-safe queries                            | `query`                                                                         | readmodels §2.3 |
| List all streams + their status                       | `listing`                                                                       | advanced §6.3   |
| Persist to PostgreSQL / SQLite                        | `storage`                                                                       | recipes §2.2    |
| Persist to embedded PebbleDB                          | `storage/pebble`                                                                | recipes §2.2    |
| Offline-first sync via Turso Database                 | `storage/turso`                                                                 | advanced §6.5   |
| Generic key-value abstraction                         | `kv`                                                                            | advanced §6.6   |
| Snapshot streams for speed                            | `snapshot`                                                                      | recipes §2.4    |
| Evolve event schemas over time                        | `schema`                                                                        | recipes §2.5    |
| Upcast events during projection replay                | `schema` (`VersionedSeekableJournal`)                                           | advanced §6.9   |
| Make event streams tamper-proof                       | `signing`                                                                       | recipes §2.6    |
| Encrypt confidential payloads                         | `encryption`                                                                    | recipes §2.7    |
| Add logging/retry/recovery/circuit-breaker            | `middleware`                                                                    | recipes §2.8    |
| Deduplicate commands on retry (idempotency)           | `idempotency` + `middleware`                                                    | recipes §2.8    |
| Add OpenTelemetry tracing/metrics                     | `otel` + `middleware`                                                           | recipes §2.8    |
| Auto-generate AsyncAPI/OpenAPI/EventCatalog/D2 docs   | `catalog`                                                                       | recipes §2.9    |
| Soft-delete streams without data loss                 | `event` (tombstone metadata)                                                    | advanced §6.1   |
| Generate typed handler boilerplate                    | `cmd/cqrs-gen`                                                                  | advanced §6.7   |
| Publish events to Watermill router                    | `watermill`                                                                     | advanced §6.4   |
| Dispatch commands/queries over gRPC                   | `transport/grpc`                                                                | advanced §6.8   |
| Verify doc code references compile                    | `cmd/doc-check`                                                                 | modules §5      |
| In-memory command bus (typed pub/sub)                 | `command` (`NewMemoryBus`)                                                      | recipes §2.1    |
| In-memory implementations for tests/dev               | `memory`                                                                        | recipes §2.1    |
| One-call infrastructure wiring (Bundle presets)       | `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`, `stack/turso` | recipes §2.0    |
| Typed read-model store over KV backend                | `kv.TypedStore`                                                                 | recipes §2.0    |
| Cache decorator for read models                       | `kv.Cache`                                                                      | recipes §2.0    |
| Run projections with crash-restart + checkpoint + DLQ | `projectionhost`                                                                | advanced §6.9   |
| Test deciders/projections with Given/When/Then        | `scenario`                                                                      | advanced §6.10  |
| Schedule delayed commands / durable deadlines         | `scheduling`                                                                    | advanced §6.11  |
| Dead-letter failed dispatches (retry exhaustion)      | `middleware` (DLQ)                                                              | recipes §2.8    |
| Derive commands reactively from events                | `deriver`                                                                       | advanced §6.12  |
| Build graph/traversal read models (nodes + edges)     | `graph`                                                                         | advanced §6.13  |
| Expose CQRS metrics via Prometheus `/metrics`         | `prometheus`                                                                    | advanced §6.14  |
| Stream events to browsers via SSE                     | `transport/http` (`SSEBroker`)                                                  | advanced §6.15  |
| Replay events to reconnecting clients (catch-up)      | `transport/http` (Last-Event-ID) or `watermill` (`CatchUpSubscriber`)           | advanced §6.15  |
| Pull-based event backfill (REST endpoint)             | `transport/http` (`BackfillHandler`)                                            | advanced §6.15  |

> **§2 (recipes), §5 (module reference), §6 (advanced patterns)** live in the on-demand `references/` files. This is the progressive-disclosure design — this file holds the decision material needed on every trigger; the references hold long copy-paste recipes loaded only when needed.

### Which projection tier?

**Three tiers** — pick by read-access pattern, not by preference:

| Tier            | Module                               | One event writes…       | Use when                                                                            | Do NOT use for                                           |
| --------------- | ------------------------------------ | ----------------------- | ----------------------------------------------------------------------------------- | -------------------------------------------------------- |
| **Document/KV** | `stack.Materialize` + `kv.ViewStore` | one record, one table   | single-entity lookups, CRUD reads, simple WHERE/ORDER BY on columns                 | composite keys, multi-table writes, OR conditions, JOINs |
| **Relational**  | `storage.RelationalProjection`       | several tables (atomic) | composite primary keys, junction tables, multi-table denormalization, complex WHERE | variable-depth traversal                                 |
| **Graph**       | `graph.GraphProjection`              | nodes + edges           | N-hop traversal, adjacency, path-finding, causation DAGs                            | simple CRUD (overkill)                                   |

`SQLViewStore` is the document tier **with queryable SQL columns** — still one record per event, single-column primary key. If you need composite keys or one event writing to multiple tables, that's `RelationalProjection`. Don't try to make ViewStore do relational work — the tiers exist because no single tier serves all read patterns well.

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

Use `listing/` for tombstone-aware stream status read models.

### 3.2 Sink/Source split — use the right interface

`Store` is split for ISP. Don't take `Store` when you only need one side:

```go
var sink event.EventSink = store        // write: Save, AppendBatch
var source event.EventSource = store    // read: Load, LoadFromVersion, LoadToVersion, LoadToTimestamp
var journal event.Journal = store       // cross-stream: ReadAll
var seekable event.SeekableJournal = store // position-based: ReadFrom(afterID, limit)
```

### 3.3 Decode payloads with a codec — never type-assert

```go
// Recommended — DecodePayloadAuto dispatches based on the event's encoding stamp.
// Works for mixed JSON+CBOR streams (e.g. during JSON→CBOR migration):
payload, err := event.DecodePayloadAuto[UserCreated](evt)

// Explicit codec — use when you know the encoding and want validation:
payload, err := event.DecodePayload[UserCreated](evt, codec.CBORCodec{})

// Wrong — Payload() returns []byte, not your type
payload := evt.Payload().(UserCreated) // DON'T
```

**Default codec is CBOR** (ADR-0051). `event.DefaultCodec = codec.CBORCodec{}`.
Events created via `event.New()` are auto-stamped with their encoding, so mixed
JSON+CBOR streams decode correctly via `DecodePayloadAuto[T]`. To revert to
JSON globally: `event.DefaultCodec = codec.JSONCodec{}`.

For browser-facing SSE endpoints, CBOR payloads must be transformed to JSON
(CBOR is penalized by SSE's text framing — base64 adds 33%):

```go
broker, _ := http.NewSSEBroker(bus, http.WithPayloadTransform(http.CBORToJSONTransform))
```

### 3.4 OTel via otel/ — never go.opentelemetry.io directly

Modules must import `github.com/larsartmann/go-cqrs-lite/otel/v4`, not `go.opentelemetry.io/otel`. The otel module re-exports the needed types and keeps the SDK indirect in go.mod.

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
errorfamily.NewRejection("user.create.empty_email", "...")    // client error, not retryable
errorfamily.NewConflict("user.create.duplicate", "...")        // optimistic concurrency
errorfamily.NewTransient("store.timeout", "...")               // retryable
errorfamily.NewInfrastructure("store.connection", "...")       // system-level
errorfamily.NewCorruption("store.invalid_event", "...")        // data integrity
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

| Anti-pattern                               | Correct approach                                                        |
| ------------------------------------------ | ----------------------------------------------------------------------- |
| Adding a `Delete()` method to Store        | Use tombstone metadata (`event.MarkTombstone`)                          |
| Taking `Store` param when you only read    | Take `EventSource` or `Journal`                                         |
| Type-asserting `evt.Payload()`             | Use `event.DecodePayloadAuto[T](evt)` or `DecodePayload[T](evt, codec)` |
| Importing `go.opentelemetry.io` directly   | Import `otel/v4` re-exports                                             |
| Manually setting event version in `Decide` | Let `event.NewEvents` auto-increment from the passed version            |
| Creating a saga/process-manager module     | Use projection + command dispatch (see `example/taskmanager/`)          |
| Editing dependency go.mod files by hand    | Use `go get` commands                                                   |
| Using `any` types in public APIs           | Use generics / branded types                                            |
| Storing the \*sql.DB lifetime in backend   | `backend.Close()` closes stores, NOT your `*sql.DB`                     |

---

## 7. Testing Patterns

```go
// In-memory test implementations
store := memory.NewMemoryStore()
bus := watermill.NewEventBus()
snapStore := memory.NewMemorySnapshotStore()
cpStore := memory.NewMemoryCheckpointStore()

// Event test helpers (for golden tests, assertions)
import "github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"

eventtest.AssertGolden(t, "testdata/event.golden", got, update)

// Shared test utilities
import "github.com/larsartmann/go-cqrs-lite/testutil/v4"

cmd := testutil.NewCmd(t, "user.create", aggID) // t = *testing.T
```

---

## 8. Dependency Layering (module graph)

```
Layer 0: id/, dispatcher/, codec/, kv/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec), command/ (→id, dispatcher), query/ (→dispatcher), scheduling/ (→id)
Layer 2: schema/ (→event), snapshot/ (→event), graph/ (→event), projection/ (→event), deriver/ (→event, command)
Layer 3: decider/ (→event, snapshot), scenario/ (→event, id, projection), projectionhost/ (→event, id, projection)
Layer 4: storage/memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, listing/, watermill/, transport/http/, transport/grpc/, storage/pebble/, storage/turso/, prometheus/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability, cmd/doc-check
```

**Saga pattern:** No dedicated module. Multi-step orchestration = projection + command dispatch. See `example/taskmanager/`.

---

## 9. Examples in the Repo

| Example             | Path                       | Demonstrates                                                              |
| ------------------- | -------------------------- | ------------------------------------------------------------------------- |
| **taskmanager**     | `example/taskmanager/`     | Flagship: full HTTP service, CQRS/ES, signing, SSE, snapshots, tombstones |
| **getting-started** | `example/getting-started/` | Minimal: 80-line demo of the core loop (bundle → repo → projection)       |

---

## 10. Where to Find More

| Need                    | Source                                                                                                                                                                             |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Per-module API details  | Each module's `README.md` and `doc.go` (renders on pkg.go.dev)                                                                                                                     |
| Architectural decisions | `docs/adr/` (43 ADRs)                                                                                                                                                              |
| Storage deep-dive       | `docs/STORAGE_GUIDE.md`                                                                                                                                                            |
| Error system            | `docs/error-taxonomy.md`                                                                                                                                                           |
| Signing internals       | `docs/signing-architecture.md`                                                                                                                                                     |
| Domain glossary         | `docs/DOMAIN_LANGUAGE.md`                                                                                                                                                          |
| Migration guides        | `docs/MIGRATION.md`, `docs/MIGRATION_v1.md`                                                                                                                                        |
| Feature inventory       | `FEATURES.md`                                                                                                                                                                      |
| Contributor guide       | `AGENTS.md` (in repo)                                                                                                                                                              |
| Consumer feedback       | `docs/feedback/` (7 files, 5 consumers)                                                                                                                                            |
| HTTP/HTMX integration   | [`cqrs-htmx`](https://github.com/LarsArtmann/cqrs-htmx) — wires this library's dispatch into `net/http` with HTMX/SSE/WebSocket. Has its own Crush skill for HTTP-layer questions. |

---

## 11. Quick API Cheat Sheet

```go
// Events
evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1), payload, opts...)
events, _ := event.NewEvents(aggID, "User", baseVersion, []event.Type{...}, []any{...})
p, _ := event.DecodePayloadAuto[T](evt)                 // mixed streams (recommended)
p, _ := event.DecodePayload[T](evt, codec.CBORCodec{})  // explicit codec
ref := id.NewStreamRef("User", aggID)

// Store (Sink/Source split)
store.Save(ctx, ref, events, expectedVersion)    // optimistic concurrency
events, _ := store.Load(ctx, ref)
events, _ := store.LoadFromVersion(ctx, ref, v)
allEvents, _ := journal.ReadAll(ctx)              // cross-stream

// Bus
bus.Publish(ctx, evt1, evt2)
bus.Subscribe("user.created", handler)
bus.Use(middleware...)
bus.UsePublish(middleware...)

// Decider
d := decider.Decider[State]{Initial: initState, Apply: applyFunc}
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
aggID := id.NewStreamID()
eventID := id.NewEventID()
type OrderID = id.Of[struct{}]
orderID := id.New[OrderID]()

// Codec — CBOR is recommended for production (smaller, faster, signing-safe)
data, _ := codec.JSONCodec{}.Encode(payload)
data, _ := codec.CBORCodec{}.Encode(payload)         // 19% smaller, 32% faster encode
payload, _ := codec.CBORCodec{}.Decode(data)
// Stack-level default: stack.WithDefaultCodec(codec.CBORCodec{}) at bundle creation

// Stack Bundle accessors (v3.6+)
store, ok := bundle.EventStore()                    // typed event.Store accessor
status := bundle.DebugStructured()                  // map[string]bool for health checks
lag := projHost.LagDuration()                       // projection lag gauge (time.Duration)

// Query middleware (symmetric with command middleware)
qDisp.Use(middleware.QueryRecovery())
qDisp.Use(middleware.QueryLogging(logger))
qDisp.Use(middleware.QueryTypedMetrics(recorder))

// Scenario testing — GivenState (no unused Cmd type param)
scenario.GivenState[CounterState](t, fold, initial, events...).
    When(nil, decideFunc).Then(expectedTypes...)

// Schema evolution — VersionedSeekableJournal (upcast events during projection replay)
vjournal, _ := schema.NewVersionedSeekableJournal(journal, upcaster1, upcaster2)
host, _ := projectionhost.New(vjournal, cpStore)  // transparent upcasting on ReadFrom

// Prometheus metrics with CQRS histogram views
metricsProvider, _ := cqrsprometheus.Setup(
    cqrsprometheus.WithViews(cqrsotel.NewCQRSViews()...),
)
defer metricsProvider.Shutdown(ctx)

// REST backfill endpoint (pull-based event snapshot for clients)
// Uses broker's journal + payload transform — same config as SSE.
mux.Handle("/events/backfill", http.BackfillHandler(broker))
```

---

## About This Skill

All Go import paths and qualified symbols in these docs are verified by
`cmd/doc-check` — run it after editing any reference file:

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../AGENTS.md ../../references/*.md
```

This skill uses progressive disclosure: `SKILL.md` is a thin index (≤1000 chars),
`core.md` holds the decision material loaded on every trigger, and the other
reference files are loaded on demand.
