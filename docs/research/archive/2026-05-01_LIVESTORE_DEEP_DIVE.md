# LiveStore Deep Dive — Lessons for go-cqrs-lite

\n> **Status:** RESOLVED — informed the event sourcing design

> **Source:** <https://docs.livestore.dev/llms.txt>
> **Date:** 2026-05-01
> **Context:** LiveStore is a client-centric local-first data layer for high-performance apps based on SQLite and event-sourcing (TypeScript/Effect). go-cqrs-lite is a Go CQRS/Event Sourcing library/SDK.

---

## Table of Contents

- [1. LiveStore Overview](#1-livestore-overview)
- [2. Core Architecture](#2-core-architecture)
- [3. Error Handling](#3-error-handling)
- [4. Dual-SQLite Pattern](#4-dual-sqlite-pattern)
- [5. Materializers (Projections)](#5-materializers-projections)
- [6. Sync Model](#6-sync-model)
- [7. Schema Evolution](#7-schema-evolution)
- [8. Patterns Worth Noting](#8-patterns-worth-noting)
- [9. What to Copy](#9-what-to-copy)
- [10. What to Improve](#10-what-to-improve)
- [11. Concrete Architecture Proposal](#11-concrete-architecture-proposal)
- [12. Actionable Opportunities](#12-actionable-opportunities)

---

## 1. LiveStore Overview

LiveStore is a **client-centric local-first** data layer that uses:

- **Event sourcing** for the write model (ordered log of mutation events)
- **SQLite** for the read model (reactive, in-memory + persisted)
- **Signals-based reactivity** for UI updates (Adapton-inspired)
- **Push/pull sync** (Git-like) for cross-client replication

Key positioning: "Like Redux, but persisted and synced across devices." It's a **framework**, not a library — it has opinions about transport, storage, and reactivity.

go-cqrs-lite is a **library/SDK** — no opinions about transport, storage backend, or reactivity. This distinction matters: LiveStore's tight integration enables features we can't copy directly, but its **patterns and error semantics** are universally applicable.

---

## 2. Core Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CLIENT SESSION                        │
│                                                         │
│  store.commit(event)                                    │
│       │                                                 │
│       ▼                                                 │
│  ┌──────────────┐     ┌──────────────────────────┐      │
│  │  Eventlog     │────▶│  In-Memory SQLite       │      │
│  │  (write model)│     │  (read model / state)    │      │
│  └──────┬───────┘     └──────────┬───────────────┘      │
│         │                        │                      │
│         │  materialize           │  query + subscribe   │
│         │                        │                      │
│         │                        ▼                      │
│         │               ┌──────────────────┐            │
│         │               │  Reactivity     │            │
│         │               │  (Signals)      │            │
│         │               └──────────────────┘            │
│         │                        │                      │
│         │                        ▼                      │
│         │                     UI updates                 │
│         │                                                │
│         ▼ (push/pull sync)                              │
│  ┌──────────────────────────────────┐                   │
│  │  Sync Backend (pluggable)        │                   │
│  │  Cloudflare / ElectricSQL / S2   │                   │
│  └──────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Three Core Concepts

| Concept            | Role                                     | go-cqrs-lite Equivalent                  |
| ------------------ | ---------------------------------------- | ---------------------------------------- |
| **Events**         | Immutable record of things that happened | `Event` interface                        |
| **Materializers**  | Event → state derivation (projection)    | `Projection` interface                   |
| **State (SQLite)** | Read model, fully derived from eventlog  | No direct equivalent (application-level) |

---

## 3. Error Handling

LiveStore has **three distinct error domains**, each with different semantics:

### 3.1 Materializer Errors (Projection Layer)

| Behavior                      | Detail                                                                                                                               |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **Transactional**             | Materializer runs in a DB transaction; failure = **full rollback** of that event's state changes                                     |
| **Event not committed**       | If the error happens on the committing client, the event is **never committed** to the eventlog and never pushed to the sync backend |
| **No partial state**          | It's impossible for a materializer failure to leave the read model in an inconsistent state                                          |
| **Fail-fast by default**      | The system **halts or flags** the event as problematic — no silent skip                                                              |
| **Future: configurable skip** | Plan to allow skipping failed events, but with strong warning about state divergence across clients                                  |

**Key principle:** Materializer errors are treated as **bugs**, not operational events. There is no dead letter queue.

### 3.2 Sync/Rebase Errors

| Scenario                     | Handling                                                                                                       |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------- |
| **Rebase failure**           | SQLite `session` extension enables efficient DB rollback; state rolls back to snapshot + re-applies all events |
| **Client ID collision**      | Detected on first push → client gets new random ID, patches local events, retries                              |
| **Concurrent push conflict** | Pull-before-push model ensures global total order; local events rebased on top of remote before pushing        |
| **Merge conflicts**          | Default: last-write-wins. Custom merge logic supported per application.                                        |

### 3.3 Schema Evolution Errors

| Scenario                                          | Handling                                                                       |
| ------------------------------------------------- | ------------------------------------------------------------------------------ |
| **Unknown events from newer app version**         | Ignore / error / catch-all handler / "update required" read-only screen        |
| **Non-backward-compatible clientDocument change** | Previous events **dropped, state reset** — explicitly unsafe for critical data |
| **Event definition removed**                      | Not allowed — event definitions are permanent                                  |
| **Event field removed**                           | Allowed (forward-compatible)                                                   |
| **Event field added**                             | Allowed if it has a default value or is optional                               |

### 3.4 What LiveStore Gets Wrong (Our Opportunity)

- **No dead letter queue** — failed events halt the system with no recovery path for poison events
- **No retry with backoff** — a transient failure (network blip, temporary constraint violation) causes a hard stop
- **No catch-up after failure** — if a materializer fails and you fix the code, there's no mechanism to replay the failed event

---

## 4. Dual-SQLite Pattern

LiveStore runs **two SQLite databases** per client:

```
┌─────────────────────────────────────────────────────────┐
│                    MAIN THREAD                          │
│                                                         │
│  ┌──────────────────────────────────────┐               │
│  │  In-Memory SQLite (reactive DB)      │               │
│  │  - Synchronous queries (no await)    │               │
│  │  - Sub-millisecond reads            │               │
│  │  - Signals-based reactivity          │               │
│  │  - UI components subscribe here      │               │
│  └──────────────┬───────────────────────┘               │
│                 │                                       │
│                 │ atomically materialize on commit       │
│                 ▼                                       │
│         store.commit(event)                             │
│                 │                                       │
└─────────────────┼───────────────────────────────────────┘
                  │ binary message channel
                  ▼
┌─────────────────────────────────────────────────────────┐
│              LEADER WEB WORKER                          │
│                                                         │
│  ┌──────────────────────────────────────┐               │
│  │  Persisted SQLite (OPFS)             │               │
│  │  - Eventlog DB (immutable history)   │               │
│  │  - State DB (can rebuild from log)   │               │
│  │  - Sync backend connection           │               │
│  │  - Leader election via Web Locks     │               │
│  └──────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────┘
```

### Why Two DBs, Not One

| In-Memory SQLite (Main Thread)             | Persisted SQLite (Worker Thread)    |
| ------------------------------------------ | ----------------------------------- |
| Synchronous reads — no async/await         | Durability across sessions          |
| Sub-millisecond query latency              | OPFS filesystem access              |
| Reactive: UI subscribes to changes         | Sync backend connection             |
| **Disposable** — can rebuild from eventlog | Source of truth for rebase/recovery |
| Higher memory cost per tab                 | Single writer (leader)              |

### Why NOT a Service Worker

| Concern             | Service Worker              | Dedicated Web Worker    |
| ------------------- | --------------------------- | ----------------------- |
| **Lifecycle**       | Shuts down after inactivity | Persistent while needed |
| **OPFS access**     | Not supported               | Supported               |
| **Purpose**         | Network interception        | Computation + storage   |
| **Leader election** | Not designed for this       | Web Locks API           |

### The Critical Insight

**The state DB is fully derived from the eventlog → it is always disposable.** This eliminates schema migration pain for the read model. When the read model schema changes, you simply:

1. Drop the state DB
2. Re-apply all events through materializers
3. State is reconstructed with the new schema

This is LiveStore's **original motivation**: frustration with database schema migrations led to event sourcing as a way to separate read and write models, making the read model disposable.

---

## 5. Materializers (Projections)

### Definition

Materializers are **functions that write to the database in response to events**. They are the equivalent of projections/read-model handlers in go-cqrs-lite.

### Key Properties

| Property                             | Detail                                                                                                 |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| **Executed in eventlog order**       | Guarantees consistency                                                                                 |
| **Transactional**                    | Always runs in a DB transaction; failure = full rollback                                               |
| **Can read current state**           | `ctx.query()` allows reading from the projected state within the transaction                           |
| **Can return multiple writes**       | Single write, array of writes, void, or an Effect                                                      |
| **Side-effect free / deterministic** | Non-deterministic code (e.g. `crypto.randomUUID()`) belongs in the event payload, not the materializer |
| **Auto-migration aware**             | When state schema changes, LiveStore auto-migrates the DB and rematerializes from eventlog             |

### Determinism Rule (Critical)

```typescript
// ❌ DON'T — non-deterministic ID generation inside materializer
const materializers = State.SQLite.materializers(events, {
	"v1.TodoCreated": ({ text }) => tables.todos.insert({ id: crypto.randomUUID(), text }),
	//                          ^^^^^^^^^^^^^^^^^^^^^^^
	//                          Non-deterministic! Different result on replay
});

// ✅ DO — all data from event payload
const events = {
	todoCreated: Events.synced({
		name: "v1.TodoCreated",
		schema: Schema.Struct({ id: Schema.String, text: Schema.String }),
		//                            ^^^^^^^^^^^^^^^^^^
		//                            Include ID in event payload
	}),
};
const materializers = State.SQLite.materializers(events, {
	"v1.TodoCreated": ({ id, text }) => tables.todos.insert({ id, text }),
});
store.commit(events.todoCreated({ id: crypto.randomUUID(), text: "Buy milk" }));
```

**Why this matters:** If a materializer produces different state on replay (because it generated a random ID), the read model diverges from what other clients computed from the same event. In a synced environment, this causes silent state corruption.

### Reading Current State in Materializers

```typescript
const materializers = State.SQLite.materializers(events, {
	todoCreated: ({ id, text, completed }, ctx) => {
		// ctx.query gives a consistent view within the transaction
		const previousIds = ctx.query(todos.select("id"));
		return todos.insert({ id, text, completed: completed ?? false, previousIds });
	},
});
```

### go-cqrs-lite Comparison

| LiveStore Materializer                | go-cqrs-lite Projection               | Gap                                                      |
| ------------------------------------- | ------------------------------------- | -------------------------------------------------------- |
| Event → DB write operations           | `Projection.Handle(ctx, Event) error` | We return `error` only, no structured write operations   |
| `ctx.query()` — read current state    | No query access during projection     | Projections are blind to their own output                |
| Transactional (auto)                  | Manual / application-level            | No transactional guarantee in our `Projection` interface |
| Side-effect free (enforced by design) | Convention only                       | No mechanism to prevent side effects                     |
| Auto-migration + rematerialization    | No rebuild/reset mechanism            | Projections can't recover from schema changes            |

---

## 6. Sync Model

### Push/Pull (Git-like)

LiveStore's sync is fundamentally **pull-before-push**, mirroring Git's rebase workflow:

```
Client A                    Sync Backend                    Client B
   │                              │                              │
   │  1. Pull (get latest events) │                              │
   │─────────────────────────────▶│                              │
   │◀─────────────────────────────│                              │
   │                              │                              │
   │  2. Rebase local events      │                              │
   │     on top of upstream       │                              │
   │                              │                              │
   │  3. Push rebased events      │                              │
   │─────────────────────────────▶│─────────────────────────────▶
   │                              │                              │
   │              4. Backend notifies B of new events            │
   │                              │◀─────────────────────────────│
   │                              │  5. B pulls, materializes    │
```

### Event Structure

| Field          | Purpose                                          |
| -------------- | ------------------------------------------------ |
| `seqNum`       | Monotonically increasing sequence number         |
| `parentSeqNum` | Parent event (like Git parent commit)            |
| `name`         | Event definition name (e.g. `v1.TodoCreated`)    |
| `args`         | Event payload (encoded via schema, usually JSON) |

### Three Heads (like Git branches)

| Head                | Scope                              |
| ------------------- | ---------------------------------- |
| Client session head | Per browser tab/worker             |
| Client leader head  | Per client (elected leader worker) |
| Sync backend head   | Global (source of total order)     |

### Conflict Resolution

- **Default:** Last-write-wins
- **Custom:** Application can implement domain-specific merge logic
- **Rebasing preserves total order:** Local events are always rebased on top of remote events before pushing

### go-cqrs-lite Comparison

| LiveStore Sync                      | go-cqrs-lite              | Gap                                            |
| ----------------------------------- | ------------------------- | ---------------------------------------------- |
| Pull-before-push with rebase        | Outbox push only          | No pull, no rebase mechanism                   |
| Global total order via sync backend | No sync layer             | Future consideration                           |
| Client ID + leader election         | No client identity        | Not applicable (server-side)                   |
| Compaction (snapshots)              | `SnapshotStore` interface | We have the interface; LiveStore implements it |

---

## 7. Schema Evolution

### Write Model (Events)

| Change                  | Allowed? | Constraint                                  |
| ----------------------- | -------- | ------------------------------------------- |
| Add event definition    | ✅       | Always                                      |
| Remove event definition | ❌       | Never — event definitions are permanent     |
| Add struct field        | ✅       | Must have default value or be optional      |
| Remove struct field     | ✅       | Forward-compatible (old data still decodes) |
| Change field type       | ⚠️       | Only if backward-compatible decode works    |

### Read Model (State/SQLite)

| Change             | Allowed? | How                                        |
| ------------------ | -------- | ------------------------------------------ |
| Add table          | ✅       | Add table definition + materializer        |
| Remove table       | ✅       | Just remove — it's derived from events     |
| Add column         | ✅       | Add column definition, update materializer |
| Remove column      | ✅       | Remove column, update materializer         |
| Change column type | ✅       | Change definition, rematerialize           |

**The key insight:** State schema changes are **free** because the state is fully derived. Just update the materializer and rematerialize. This is why LiveStore was built — to eliminate database migration pain.

### Versioned Event Names

```typescript
// Recommended: version prefixes
todoCreated: Events.synced({ name: 'v1.TodoCreated', schema: ... })

// When schema changes:
todoCreatedV2: Events.synced({ name: 'v2.TodoCreated', schema: ... })
```

This makes evolution explicit. Old materializers still handle `v1.*` events; new materializers handle `v2.*` events. No silent breakage.

### go-cqrs-lite: UpcasterRegistry vs. Versioned Names

We already have `UpcasterRegistry` for transforming old event formats to new ones. LiveStore's versioned names are a **complementary** strategy:

| Approach            | Mechanism                                        | Trade-off                                                                  |
| ------------------- | ------------------------------------------------ | -------------------------------------------------------------------------- |
| **Upcasters**       | Transform old payload → new payload at read time | Single event type name; upcaster chain can grow complex                    |
| **Versioned names** | New event type for new schema                    | Multiple event types; simpler per-type handlers; more materializer entries |

Both are valid. For go-cqrs-lite, we should **document both** and let consumers choose.

---

## 8. Patterns Worth Noting

### 8.1 Undo/Redo as Explicit Events

> Undo/redo should be modeled as **compensating events**, not by removing events from history.

Events are immutable append-only. "Undo" is a new event that reverses the effect. This matches our event sourcing model perfectly.

### 8.2 Soft Deletes Over Hard Deletes

> Avoid `DELETE` events — use soft deletes (e.g. `deletedAt` timestamp) to prevent concurrency issues in distributed/replicated scenarios.

A hard delete event can conflict with a concurrent update event. A soft delete (setting `deletedAt`) is commutative.

### 8.3 Side-Effect Scopes (Still TODO in LiveStore)

| Scope              | Description                                           | Implementation               |
| ------------------ | ----------------------------------------------------- | ---------------------------- |
| Per client session | Run on every session (e.g. UI refresh)                | Application-level            |
| Per client         | Run once across sessions (e.g. onboarding)            | Local lock between sessions  |
| Globally once      | Run exactly once across all clients (e.g. send email) | Distributed transaction/lock |

**go-cqrs-lite's `OutboxPublisher`** handles scope 3 (globally-once) via the outbox pattern. Scopes 1 and 2 are application-level.

### 8.4 Synced vs. Client-Only Events

| Type         | Synced?                                                         | Use Case                                          |
| ------------ | --------------------------------------------------------------- | ------------------------------------------------- |
| `synced`     | Yes — replicated across all clients                             | Business events (`TodoCreated`)                   |
| `clientOnly` | No — local only, but synced across sessions (e.g. browser tabs) | UI state (`SelectedTabChanged`, `ScrollPosition`) |

This avoids polluting the global eventlog with transient state. go-cqrs-lite currently treats all events identically.

### 8.5 Client Documents

LiveStore has a special `clientDocument` table type for local UI state:

```typescript
uiState: State.SQLite.clientDocument({
	name: "uiState",
	schema: Schema.Struct({
		newTodoText: Schema.String,
		filter: Schema.Literal("all", "active", "completed"),
	}),
	default: { id: SessionIdSymbol, value: { newTodoText: "", filter: "all" } },
});
```

- Convenience layer (like `React.useState` but persistent)
- Client-only, auto-resets on breaking schema changes
- Don't use for sensitive data

### 8.6 State Machines

> Listen to query results, emit events when results change. State machine side effects commit new mutations to LiveStore.

This is a projection → command feedback loop. Our architecture supports this naturally (projection reads state → publishes command), but we don't document the pattern.

### 8.7 SQLite Session Extension for Rollback

LiveStore uses the SQLite `session` extension for efficient database rollback during rebase:

> When the eventlog is rolled back as part of a rebase, the session extension enables efficient undo of database changes without requiring a full snapshot restore.

Alternative implementation: periodic snapshots + replay from snapshot point. We already have `SnapshotStore` for this.

---

## 9. What to Copy

### 9.1 Transactional Materializers (HIGH)

**LiveStore:** Materializer error = rollback entire event's state changes. No partial projections.

**go-cqrs-lite today:** `InMemoryRunner.Handle()` calls `Projection.Handle()` and skips checkpoint advance on error, but there's no explicit transactional contract.

**Action:** Make the transactional contract explicit in the `Projection` interface. Document that `Handle` must be atomic — if it fails, the read model must be unchanged.

### 9.2 State Is Disposable (HIGH)

**LiveStore:** Read model is fully derived from eventlog → can always `DROP TABLE` and rematerialize. Eliminates schema migration pain.

**go-cqrs-lite today:** `CheckpointStore` tracks position but has no `Delete(name)` method. Projections can't reset and rebuild.

**Action:** Add `CheckpointStore.Delete(name string) error` so projections can reset and replay from the beginning.

### 9.3 Deterministic Materializers (MEDIUM)

**LiveStore:** All data flows through event payload. No `crypto.randomUUID()` or `time.Now()` inside projections. Non-deterministic code produces different state on replay → silent corruption in synced environments.

**go-cqrs-lite today:** Convention only — no enforcement.

**Action:** Document as a rule. Our `Projection.Handle(ctx, Event)` already receives all event data — enforce "no side effects" by convention and linter suggestions.

### 9.4 Schema Auto-Migration via Rematerialization (MEDIUM)

**LiveStore:** When state schema changes, just drop the read model and replay from eventlog.

**go-cqrs-lite today:** `UpcasterRegistry` handles write-model evolution. No read-model counterpart.

**Action:** Add `CheckpointStore.Delete()` + a catch-up runner that can replay from store. The combination enables "reset and rebuild."

### 9.5 Versioned Event Names (LOW)

**LiveStore:** `v1.TodoCreated` → `v2.TodoCreated` — explicit evolution, no silent breakage.

**go-cqrs-lite today:** Unversioned strings like `"user.created"`.

**Action:** Document `v1.EventName` convention in examples and best practices.

### 9.6 Soft Deletes Over Hard Deletes (LOW)

**LiveStore:** Avoid `DELETE` events → use `deletedAt` timestamps to prevent concurrency issues.

**Action:** Document as best practice.

### 9.7 Past-Tense Event Names (LOW)

**LiveStore:** Use `todoCreated` / `createdTodo` instead of `todoCreate` / `createTodo` — indicates something already occurred.

**go-cqrs-lite today:** Mixed conventions in examples.

**Action:** Document and standardize in examples.

---

## 10. What to Improve

LiveStore has significant gaps where go-cqrs-lite can do **better**:

### 10.1 Catch-Up / Replay Mechanism (CRITICAL)

**LiveStore gap:** No "replay from position X" on startup. If a projection was down, there's no way to catch up on missed events. All events must arrive live through the bus.

**Our opportunity:** Build a **catch-up projection runner**:

1. On startup, read `CheckpointStore.Load(name)` to get last-processed position
2. `Store.Load(fromVersion)` or `Store.LoadFromVersion(aggregateID, version)` to get missed events
3. Feed events to `Projection.Handle()`
4. Switch to live mode via `Bus.Subscribe()`

This is the **#1 missing feature** in both libraries. Our `CheckpointStore` + `Store.Load()` already give us the building blocks — we just need the runner.

### 10.2 Global Event Stream Position (HIGH)

**LiveStore gap:** Uses `seqNum` per client, but no global log position for partitioned catch-up across multiple aggregates.

**Our gap:** Checkpoint is `id.EventID` (ULID), not a queryable position. `Store.Load()` loads per-aggregate. There's no `ReadAll(fromGlobalPosition)` method.

**Our opportunity:** Add a **global log position** (monotonic sequence) to the `Store` interface. This enables:

- Efficient catch-up queries across all aggregates
- Competing consumers that partition by position range
- Monitoring (how far behind is each projection?)

### 10.3 Competing Consumers / Distributed Projections (MEDIUM)

**LiveStore gap:** Single-leader model, no distributed projection hosting.

**Our opportunity:** `CheckpointStore` already enables this — add a **lease/lock mechanism** so multiple instances can partition projections. Only one instance runs a given projection at a time.

### 10.4 Dead Letter Queue (MEDIUM)

**LiveStore gap:** Failed events halt the system with no recovery path.

**Our opportunity:** Add a **DLQ** to `InMemoryRunner`:

- Failed events → `DeadLetterStore` instead of stopping the world
- Configurable behavior: `Halt` (LiveStore default) vs. `Quarantine` (operational)
- Admin API to inspect, retry, or discard dead-lettered events

### 10.5 Query Access During Projection (MEDIUM)

**LiveStore:** `ctx.query()` inside materializer gives access to current projected state within the transaction.

**Our gap:** `Projection.Handle(ctx, Event)` doesn't expose the read model. Projections are blind to their own output.

**Our opportunity:** Add an optional `ProjectionWithQuery` interface:

```go
type ProjectionWithQuery interface {
    Projection
    HandleWithQuery(ctx context.Context, event Event, query QueryFunc) error
}

type QueryFunc func(query string, args ...any) (Rows, error)
```

### 10.6 Batch Materialization (LOW)

**LiveStore gap:** Events processed one at a time through materializers.

**Our opportunity:** Add `HandleBatch(ctx context.Context, events []Event) error` to projections for bulk upserts — significant performance win for high-throughput scenarios.

### 10.7 Client-Only Events (LOW)

**LiveStore:** `clientOnly` events skip sync, used for UI state.

**Our opportunity:** Add `event.WithLocalOnly()` option that marks events to skip the outbox and any future sync layer.

### 10.8 Projection Rebuild / Reset API (MEDIUM)

**LiveStore:** Auto-migration + rematerialization when schema changes.

**Our opportunity:** Expose a `Reset(ctx, projectionName)` method that:

1. Deletes the read model data
2. Deletes the checkpoint
3. Replays all events from the store through the projection

---

## 11. Concrete Architecture Proposal

### Dual-Store Pattern for go-cqrs-lite

The dual-SQLite pattern translates to a **dual-store pattern** in Go:

```
┌──────────────────────────────────────────────────┐
│              APPLICATION (HTTP/gRPC)             │
│  Command Handler → Store.Save → Bus.Publish       │
└───────────────┬──────────────────────────────────┘
                │
                ▼
┌──────────────────────────────────────────────────┐
│              EVENT STORE (write model)            │
│  Store.Save(events, expectedVersion)              │
│  Outbox.Append(events)                           │
│  Global log position (monotonic sequence)         │
└───────────────┬──────────────────────────────────┘
                │
        ┌───────┴───────┐
        ▼               ▼
┌──────────────┐  ┌──────────────┐
│  LIVE PATH   │  │  CATCH-UP    │
│  Bus.Publish │  │  PATH        │
│  → handlers  │  │  Store.Load │
│              │  │  → projection│
└──────┬───────┘  └──────┬───────┘
       │                 │
       ▼                 ▼
┌──────────────────────────────────────────────────┐
│           PROJECTIONS (read model)                │
│  Projection.Handle(ctx, event)                    │
│    → Write to read model DB (transactional)       │
│    → CheckpointStore.Save(name, position)          │
│  On failure: rollback tx, no checkpoint advance    │
│  On reset: CheckpointStore.Delete(name) + replay   │
└──────────────────────────────────────────────────┘
```

### Catch-Up Runner Design

```go
// CatchUpRunner replays missed events on startup, then switches to live mode.
type CatchUpRunner struct {
    store       event.Store
    bus         event.Bus
    checkpoint  event.CheckpointStore
    projections []event.Projection
    deadLetter  DeadLetterStore // optional
}

func (r *CatchUpRunner) Start(ctx context.Context) error {
    for _, p := range r.projections {
        // 1. Read checkpoint
        lastPos, err := r.checkpoint.Load(ctx, p.Name())

        // 2. Load missed events from store
        events := r.store.LoadAllFrom(ctx, lastPos)

        // 3. Replay through projection
        for _, evt := range events {
            if err := p.Handle(ctx, evt); err != nil {
                // Transactional: read model unchanged, checkpoint not advanced
                if r.deadLetter != nil {
                    r.deadLetter.Queue(ctx, p.Name(), evt, err)
                }
                continue // or halt, depending on config
            }
            r.checkpoint.Save(ctx, p.Name(), evt.ID())
        }
    }

    // 4. Switch to live mode
    return r.bus.SubscribeAll(r.handleLive)
}
```

### Error Handling Proposal

```go
type ErrorPolicy int

const (
    ErrorPolicyHalt       ErrorPolicy = iota // Stop on first error (LiveStore default)
    ErrorPolicyQuarantine                     // Move to DLQ, continue processing
    ErrorPolicyRetry                          // Retry with backoff, then quarantine
)

type RunnerConfig struct {
    ErrorPolicy    ErrorPolicy
    MaxRetries     int
    RetryBackoff   time.Duration
    DeadLetterStore DeadLetterStore // nil = halt on error
}
```

---

## 12. Actionable Opportunities

### Priority Matrix

| #   | Opportunity                          | Impact                                          | Effort              | Priority |
| --- | ------------------------------------ | ----------------------------------------------- | ------------------- | -------- |
| 1   | Catch-up projection runner           | Critical — projections are useless without it   | Medium              | **P0**   |
| 2   | `CheckpointStore.Delete(name)`       | Enables reset + rebuild                         | Low                 | **P0**   |
| 3   | Transactional projection contract    | Prevents partial state on error                 | Low                 | **P0**   |
| 4   | Global log position in `Store`       | Enables efficient catch-up                      | Medium              | **P1**   |
| 5   | Dead letter queue                    | Operational resilience                          | Medium              | **P1**   |
| 6   | Query access during projection       | Rich projections (like LiveStore `ctx.query()`) | Medium              | **P2**   |
| 7   | Projection rebuild/reset API         | Schema evolution support                        | Low (depends on #2) | **P2**   |
| 8   | Batch `HandleBatch`                  | High-throughput optimization                    | Low                 | **P2**   |
| 9   | Versioned event names convention     | Documentation only                              | Minimal             | **P3**   |
| 10  | Soft delete best practice            | Documentation only                              | Minimal             | **P3**   |
| 11  | Past-tense event name convention     | Documentation only                              | Minimal             | **P3**   |
| 12  | Client-only events (`WithLocalOnly`) | Future sync layer prep                          | Low                 | **P3**   |

### Quick Wins (Can Do Today)

1. **Document "state is disposable"** as the canonical pattern — this is LiveStore's most powerful insight
2. **Document determinism rule** — no `time.Now()`, no `uuid.New()` inside projections
3. **Document versioned event names** — `v1.UserCreated` convention
4. **Document soft deletes** — `deletedAt` over `DELETE` events
5. **Document undo/redo pattern** — compensating events, not event removal
6. **Document side-effect scope taxonomy** — per-session, per-client, globally-once

### Medium-Term (Requires Code Changes)

1. Add `CheckpointStore.Delete(name string) error`
2. Build `CatchUpRunner` with start-from-checkpoint → replay → live-switch
3. Make projection error handling configurable (halt vs. quarantine)
4. Add global log position to `Store` interface
5. Add optional `ProjectionWithQuery` interface

### Long-Term (Architecture)

1. Dead letter store + admin API
2. Competing consumers (lease-based distributed projections)
3. Client-only event marker for future sync layer
4. Batch projection handler for high-throughput

---

## Appendix: LiveStore Feature Checklist

Features LiveStore has that go-cqrs-lite doesn't (by design or gap):

| Feature                            | LiveStore                   | go-cqrs-lite     | Status                                       |
| ---------------------------------- | --------------------------- | ---------------- | -------------------------------------------- |
| Reactive queries (Signals)         | ✅ Built-in                 | ❌ Not our scope | By design — we're a library, not a framework |
| Auto schema migration (read model) | ✅ Drop + rematerialize     | ❌               | Gap — need rebuild mechanism                 |
| Built-in sync (push/pull)          | ✅ Pluggable sync providers | ❌               | Future — outbox is step 1                    |
| Devtools (inspector)               | ✅ Built-in                 | ❌               | Future consideration                         |
| Client documents (UI state)        | ✅ `clientDocument` type    | ❌               | Different scope                              |
| In-memory + persisted dual DB      | ✅ SQLite pair              | N/A              | Architecture pattern to learn from           |
| Offline-first                      | ✅ Out of the box           | Partial (outbox) | Different target (server vs. client)         |

Features go-cqrs-lite has that LiveStore doesn't:

| Feature                        | go-cqrs-lite               | LiveStore                | Our Advantage                      |
| ------------------------------ | -------------------------- | ------------------------ | ---------------------------------- |
| Catch-up / replay from store   | Building blocks exist      | ❌ None                  | We can build it first              |
| Competing consumers            | CheckpointStore enables it | ❌ None                  | Architecture supports distribution |
| Language-agnostic (Go)         | Go (server-side)           | TypeScript (client-side) | Different target market            |
| Modular (import what you need) | 8 independent modules      | Monolithic framework     | Library > framework philosophy     |
| Branded IDs (`id.Of[T]`)       | ✅ Type-safe               | String-based IDs         | Compile-time safety                |
| AsyncAPI catalog generation    | ✅ Built-in                | ❌                       | Documentation from types           |
| Upcaster registry              | ✅ Write-model evolution   | Versioned names only     | Both strategies available          |

---

_Research compiled from <https://docs.livestore.dev/llms.txt> and related documentation pages._
