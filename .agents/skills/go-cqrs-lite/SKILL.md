---
name: go-cqrs-lite
description: Build Go applications with go-cqrs-lite — the composable CQRS + Event Sourcing library (event-sourced aggregates, deciders, projections, read models, SQL/Pebble/Turso storage, snapshots, schema evolution, signing, encryption, scheduling, deriver sagas, graph projections, catalog docs). Use this skill whenever a project imports any github.com/larsartmann/go-cqrs-lite/*/v3 module, OR the user asks how to build CQRS/event-sourcing systems in Go, dispatch commands/queries, build read models or projections, use event stores or buses, or work with any go-cqrs-lite module (event, command, query, decider, id, codec, storage, stack, kv, listing, projection, projectionhost, schema, signing, encryption, middleware, otel, catalog, watermill, scheduling, deriver, graph, prometheus, scenario) — even when the user does NOT name the library explicitly (e.g. "set up event sourcing", "build a read model", "dispatch a command", "snapshot an aggregate", "replay events", "idempotent commands", "soft-delete aggregate", "project events to SQL").
user-invocable: true
---

# go-cqrs-lite — AI Consumer Guide

> This is the **single source of truth for AI consumers**. It replaces the need to discover and read 28 module READMEs. AGENTS.md (in the library repo) is for contributors; this file is for users.

## How to use this skill

This core file holds the mental model, the module decision matrix, the critical conventions, the cheat sheet, and the FAQ. For the long copy-paste material, read the bundled reference files **on demand**:

| You need…                                                                                                                                                             | Read                                                    |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| Copy-paste composition recipes (event-sourcing setup, persistence, read models, snapshots, signing, encryption, observability, auto-docs)                             | [`references/recipes.md`](references/recipes.md) (§2)   |
| The full per-module reference table (imports + one-liners for all 28 modules)                                                                                         | [`references/modules.md`](references/modules.md) (§5)   |
| Advanced patterns (tombstone, persistence/audit, listing, watermill, turso, cqrs-gen, gRPC, projectionhost, scenario-testing, scheduling, deriver, graph, prometheus) | [`references/advanced.md`](references/advanced.md) (§6) |

**Rule of thumb:** start with §1 (decision matrix) below to pick the right modules, then jump to the matching recipe in `references/recipes.md`.

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

You do NOT need all of them. Start with the minimal recipe (§2), then bolt on capabilities.

---

## 1. Module Decision Matrix — "I want to…"

| If you want to…                                       | Use                                                                             | See recipe     |
| ----------------------------------------------------- | ------------------------------------------------------------------------------- | -------------- |
| Create/store/load events                              | `event`                                                                         | recipes §2.1   |
| Dispatch type-safe commands                           | `command`                                                                       | recipes §2.1   |
| Run an event-sourced aggregate                        | `decider`                                                                       | recipes §2.1   |
| Generate unique, type-safe IDs                        | `id`                                                                            | recipes §2.1   |
| Encode payloads as JSON/CBOR                          | `codec`                                                                         | recipes §2.1   |
| Build a read model from events                        | `stack.Materialize` + `kv.TypedStore`                                           | recipes §2.3   |
| Dispatch type-safe queries                            | `query`                                                                         | recipes §2.3   |
| List all aggregates + their status                    | `listing`                                                                       | advanced §6.3  |
| Persist to PostgreSQL / SQLite                        | `storage`                                                                       | recipes §2.2   |
| Persist to embedded PebbleDB                          | `storage/pebble`                                                                | recipes §2.2   |
| Offline-first sync via LibSQL                         | `storage/turso`                                                                 | advanced §6.5  |
| Generic key-value abstraction                         | `kv`                                                                            | advanced §6.6  |
| Snapshot aggregates for speed                         | `snapshot`                                                                      | recipes §2.4   |
| Evolve event schemas over time                        | `schema`                                                                        | recipes §2.5   |
| Make event streams tamper-proof                       | `signing`                                                                       | recipes §2.6   |
| Encrypt confidential payloads                         | `encryption`                                                                    | recipes §2.7   |
| Add logging/retry/recovery/circuit-breaker            | `middleware`                                                                    | recipes §2.8   |
| Deduplicate commands on retry (idempotency)           | `idempotency`                                                                   | recipes §2.8   |
| Add OpenTelemetry tracing/metrics                     | `otel` + `middleware`                                                           | recipes §2.8   |
| Auto-generate AsyncAPI/OpenAPI/EventCatalog/D2 docs   | `catalog`                                                                       | recipes §2.9   |
| Soft-delete aggregates without data loss              | `event` (tombstone metadata)                                                    | advanced §6.1  |
| Generate typed handler boilerplate                    | `cmd/cqrs-gen`                                                                  | advanced §6.7  |
| Publish events to Watermill router                    | `watermill`                                                                     | advanced §6.4  |
| Dispatch commands/queries over gRPC                   | `transport/grpc`                                                                | advanced §6.8  |
| Verify doc code references compile                    | `cmd/doc-check`                                                                 | modules §5     |
| In-memory command bus (typed pub/sub)                 | `command` (`NewMemoryBus`)                                                      | recipes §2.1   |
| In-memory implementations for tests/dev               | `memory`                                                                        | recipes §2.1   |
| One-call infrastructure wiring (Bundle presets)       | `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`, `stack/turso` | recipes §2.0   |
| Typed read-model store over KV backend                | `kv.TypedStore`                                                                 | recipes §2.0   |
| Cache decorator for read models                       | `kv.Cache`                                                                      | recipes §2.0   |
| Run projections with crash-restart + checkpoint + DLQ | `projectionhost`                                                                | advanced §6.9  |
| Test deciders/projections with Given/When/Then        | `scenario`                                                                      | advanced §6.10 |
| Schedule delayed commands / durable deadlines         | `scheduling`                                                                    | advanced §6.11 |
| Dead-letter failed dispatches (retry exhaustion)      | `middleware` (DLQ)                                                              | recipes §2.8   |
| Derive commands reactively from events                | `deriver`                                                                       | advanced §6.12 |
| Build graph/traversal read models (nodes + edges)     | `graph`                                                                         | advanced §6.13 |
| Expose CQRS metrics via Prometheus `/metrics`         | `prometheus`                                                                    | advanced §6.14 |

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
| Importing `go.opentelemetry.io` directly   | Import `otel/v3` re-exports                                  |
| Manually setting event version in `Decide` | Let `event.NewEvents` auto-increment from the passed version |
| Creating a saga/process-manager module     | Use projection + command dispatch (see `example/todo/`)      |
| Editing dependency go.mod files by hand    | Use `go get` commands                                        |
| Using `any` types in public APIs           | Use generics / branded types                                 |
| Storing the \*sql.DB lifetime in backend   | `backend.Close()` closes stores, NOT your `*sql.DB`          |

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
Layer 1: event/ (→id, codec), command/ (→id, dispatcher), query/ (→dispatcher), scheduling/ (→id)
Layer 2: schema/ (→event), snapshot/ (→event), graph/ (→event), projection/ (→event), deriver/ (→event, command)
Layer 3: decider/ (→event, snapshot), scenario/ (→event, id, projection), projectionhost/ (→event, id, projection)
Layer 4: storage/memory/, signing/, encryption/, otel/
Layer 5: middleware/, storage/, listing/, watermill/, transport/http/, transport/grpc/, storage/pebble/, storage/turso/, prometheus/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen, cmd/api-stability, cmd/doc-check
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
