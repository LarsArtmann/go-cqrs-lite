# Time Travel Capabilities: Current State & Ideal Design

> Comprehensive analysis of temporal query capabilities in go-cqrs-lite — what exists, what's missing, and what the library should provide to be world-class.

**Date:** 2026-05-20

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What Is Time Travel in Event Sourcing?](#2-what-is-time-travel-in-event-sourcing)
3. [Current Capabilities (What We Provide Today)](#3-current-capabilities)
4. [Capability Gap Analysis](#4-capability-gap-analysis)
5. [What We SHOULD Provide (Ideal Design)](#5-what-we-should-provide)
6. [Recommended Roadmap](#6-recommended-roadmap)
7. [What We Should NOT Provide (And Why)](#7-what-we-should-not-provide)
8. [Industry Comparison](#8-industry-comparison)
9. [Appendix: Mechanism Inventory](#9-appendix-mechanism-inventory)

---

## 1. Executive Summary

go-cqrs-lite has **solid foundational building blocks** for time travel — events are immutable, versioned, and replayable — but **lacks the top-level API surface** that makes time travel easy for consumers. Today, reconstructing state at a past point requires manual event filtering and folding. The library provides the raw ingredients but not the finished dish.

**Key finding:** We have 6 out of ~13 known time-travel mechanisms partially implemented. The 3 highest-impact additions — `LoadToVersion`, `LoadToTimestamp`, and position-based `GlobalLoader` — are all non-breaking, low-effort, and solve the most common time-travel use cases.

---

## 2. What Is Time Travel in Event Sourcing?

Time travel is the ability to **observe or reconstruct the state of a system at any point in its history** — not just the current state. In event-sourced systems, this is theoretically trivial (replay events up to a point), but practically complex (performance, correctness, API design).

### Two Time Dimensions

| Dimension            | Name                    | Source                             | Our Current Representation |
| -------------------- | ----------------------- | ---------------------------------- | -------------------------- |
| **Transaction Time** | "As-At" / System Time   | When the system recorded the event | `Event.OccurredAt()`       |
| **Valid Time**       | "As-Of" / Business Time | When the event was true in reality | **Not represented**        |

### The Four Query Modes

| Mode            | Time Axis        | Question                                                      | Example                            |
| --------------- | ---------------- | ------------------------------------------------------------- | ---------------------------------- |
| **Current**     | Neither          | "What is the state now?"                                      | Default `Repository.Load()`        |
| **As-At**       | Transaction time | "What did the system know at time T?"                         | "Balance as recorded on March 15?" |
| **As-Of**       | Valid time       | "What was true in reality at time T?"                         | "Salary effective on Jan 1?"       |
| **Bi-Temporal** | Both             | "What did we believe at recording-time R about valid-time V?" | Financial audit trails             |

**Most domains only need As-At.** Financial, HR, and healthcare domains need As-Of or Bi-Temporal.

---

## 3. Current Capabilities

### 3.1 Aggregate-Level Event Loading

| Method                                                | Interface     | What It Does                           | Time-Travel Role                              |
| ----------------------------------------------------- | ------------- | -------------------------------------- | --------------------------------------------- |
| `Store.Load(ctx, aggType, aggID)`                     | `event.Store` | Load ALL events for an aggregate       | Full history access (the baseline)            |
| `Store.LoadFromVersion(ctx, aggType, aggID, version)` | `event.Store` | Load events AFTER a given version      | Partial stream reconstruction (from snapshot) |
| `Store.AppendBatch(ctx, aggType, aggID, events)`      | `event.Store` | Bulk import without concurrency checks | Event replay / migration                      |

**Implementations:** MemoryStore, SQLEventStore, Pebble CQRSAdapter

### 3.2 Snapshot-Based Point-in-Time

| Method                                                      | Interface                | What It Does                         | Time-Travel Role                         |
| ----------------------------------------------------------- | ------------------------ | ------------------------------------ | ---------------------------------------- |
| `SnapshotStore.Load(ctx, aggType, aggID)`                   | `event.SnapshotStore`    | Load latest snapshot                 | Fast-forward to most recent known state  |
| `SnapshotStore.LoadAtVersion(ctx, aggType, aggID, version)` | `event.SnapshotStore`    | Load snapshot at or before a version | **Closest thing to time travel we have** |
| `SnapshotStore.Save(ctx, snapshot)`                         | `event.SnapshotStore`    | Persist a snapshot                   | Create future time-travel waypoints      |
| `SnapshotStrategy.ShouldSnapshot(aggType, version)`         | `event.SnapshotStrategy` | Decide when to snapshot              | Automatic version-based snapshotting     |
| `EveryNEvents(n)`                                           | `event.SnapshotStrategy` | Snapshot every N events              | The only built-in strategy               |

**Implementations:** MemorySnapshotStore, SQLSnapshotStore

### 3.3 State Reconstruction (Fold/Apply)

| Method                                         | Interface              | What It Does                                | Time-Travel Role                  |
| ---------------------------------------------- | ---------------------- | ------------------------------------------- | --------------------------------- |
| `aggregate.Core.LoadFromHistory(root, events)` | `aggregate.Core`       | Replay events onto aggregate root           | OO-style state reconstruction     |
| `aggregate.Repository.Load(ctx, root)`         | `aggregate.Repository` | Full load pipeline (snapshot + replay)      | Current-state reconstruction only |
| `decider.Repository.Load(ctx, aggID, aggType)` | `decider.Repository`   | Fold-based state reconstruction (read-only) | Current-state reconstruction only |
| `decider.Repository.foldEvents(state, events)` | `decider.Repository`   | Pure function fold loop                     | The fundamental replay primitive  |

### 3.4 Projection Replay

| Method                                         | Interface               | What It Does                               | Time-Travel Role                  |
| ---------------------------------------------- | ----------------------- | ------------------------------------------ | --------------------------------- |
| `GlobalLoader.LoadAll(ctx)`                    | `event.GlobalLoader`    | Load ALL events across all aggregates      | Cross-aggregate projection replay |
| `CheckpointStore.Load(ctx, projName)`          | `event.CheckpointStore` | Get last processed event ID per projection | Resume-from-checkpoint            |
| `CheckpointStore.Save(ctx, projName, eventID)` | `event.CheckpointStore` | Update checkpoint after processing         | Replay progress tracking          |
| `projection.Runner.replay(ctx)`                | `projection.Runner`     | LoadAll → filter by checkpoint → process   | Projection state rebuild          |
| `InMemoryRunner.Handle(ctx, evt)`              | `event.InMemoryRunner`  | Dispatch + checkpoint per event            | Single-process projection         |

**Known limitation:** `LoadAll` loads ALL events into memory, then `filterEvents` linearly scans for checkpoint position. O(n) memory and O(n) time for every replay, even if only a few events need processing.

### 3.5 Schema Version Evolution (Lazy Time Travel)

| Method                                      | Interface                | What It Does                                | Time-Travel Role                                    |
| ------------------------------------------- | ------------------------ | ------------------------------------------- | --------------------------------------------------- |
| `Upcaster.SourceType()` / `SourceVersion()` | `event.Upcaster`         | Identify which events to transform          | Lazy migration: old schema → new schema             |
| `Upcaster.Upcast(evt)`                      | `event.Upcaster`         | Transform old event to new schema           | Events "travel forward in time" schema-wise         |
| `UpcasterRegistry.Upcast(evt)`              | `event.UpcasterRegistry` | Apply all applicable upcasters in order     | Automatic chain migration                           |
| `event.SchemaVersion`                       | `event/types.go`         | Branded type distinct from stream `Version` | Prevents mixing schema version with stream position |

### 3.6 Outbox Replay (Last-Mile Reliability)

| Method                            | Interface               | What It Does                          | Time-Travel Role                                    |
| --------------------------------- | ----------------------- | ------------------------------------- | --------------------------------------------------- |
| `Outbox.PollPending(ctx, limit)`  | `event.Outbox`          | Get unacknowledged events             | Replay events that were persisted but not published |
| `OutboxPublisher.PublishNow(ctx)` | `event.OutboxPublisher` | Synchronous one-shot poll-publish-ack | Manual replay trigger                               |

---

## 4. Capability Gap Analysis

### 4.1 What's Missing — Ranked by Impact

| #   | Gap                                              | Use Case                                       | Current Workaround                                             | Severity                  |
| --- | ------------------------------------------------ | ---------------------------------------------- | -------------------------------------------------------------- | ------------------------- |
| 1   | **No `Store.LoadToVersion`**                     | "Reconstruct state at version N"               | Load ALL events → filter manually → O(n) even for small slices | **CRITICAL**              |
| 2   | **No `Store.LoadToTimestamp`**                   | "What was the state at time T?"                | Same manual filter approach                                    | **HIGH**                  |
| 3   | **No position-based `GlobalLoader`**             | Production-scale projection replay             | `LoadAll()` loads everything into memory → O(n) memory         | **HIGH**                  |
| 4   | **No `Repository.LoadAtVersion` / `LoadAtTime`** | "Give me the aggregate at this point in time"  | Must use Store directly + manually fold                        | **HIGH**                  |
| 5   | **No backward reads**                            | "Last N events", "most recent event of type X" | Load all → take last N manually                                | **MEDIUM**                |
| 6   | **No global transaction position**               | Cross-aggregate temporal consistency           | No equivalent — per-aggregate versioning only                  | **MEDIUM**                |
| 7   | **No `ValidAt` / bi-temporal**                   | "Salary effective on Jan 1"                    | No representation of valid-time                                | **LOW** (domain-specific) |
| 8   | **No speculative application**                   | "What-if" dry-run without persisting           | Clone aggregate in memory manually                             | **LOW**                   |
| 9   | **No temporal projections**                      | Versioned read-model snapshots over time       | No equivalent                                                  | **LOW**                   |

### 4.2 The Time Travel Workflow Today (Painful)

To answer "What was the aggregate state at version 5?" a consumer must write:

```go
// STEP 1: Load ALL events (even if the stream has 10,000 events)
events, err := store.Load(ctx, aggType, aggID)
if err != nil { return err }

// STEP 2: Manually filter to version <= 5
var target []event.Event
for _, e := range events {
    if e.Version() <= event.Version(5) {
        target = append(target, e)
    }
}

// STEP 3: Manually fold into state (decider) or apply (aggregate)
state := decider.Initial
for _, e := range target {
    state, err = decider.Fold(state, e)
    if err != nil { return err }
}
```

**This is O(n) for every time-travel query, even if you only need the first 5 events of a 10,000-event stream.**

### 4.3 The Ideal Workflow (What It Should Be)

```go
// LoadToVersion does the filtering at the store level (SQL: WHERE version <= $3)
state, version, err := deciderRepo.LoadAtVersion(ctx, aggID, aggType, event.Version(5))
```

One call. Store-level optimization. No over-reading.

---

## 5. What We SHOULD Provide

### 5.1 P0: `Store.LoadToVersion` — Version-Based Time Travel

**The single highest-impact addition.** Enables reconstructing aggregate state at any past version with store-level filtering instead of application-level filtering.

```go
// New method on event.Store
LoadToVersion(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxVersion Version,
) ([]Event, error)
```

**SQL implementation:** `WHERE version <= $3 ORDER BY version ASC` — index already exists.

**Memory implementation:** `events[:min(maxVersion.Int(), len(events))]` — O(1) slice.

**Why P0:**

- Non-breaking (additive interface change)
- Minimal implementation effort (all 3 stores are trivial)
- Solves the most common time-travel question: "What was the state at version N?"
- Foundation for Repository-level time-travel methods
- Required by audit, compliance, debugging, and migration testing use cases

### 5.2 P1: `Store.LoadToTimestamp` — Timestamp-Based Time Travel

**Enables "as-at" queries:** "What did the system know at time T?"

```go
// New method on event.Store
LoadToTimestamp(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxTime time.Time,
) ([]Event, error)
```

**SQL implementation:** `WHERE occurred_at <= $3 ORDER BY version ASC` — needs timestamp index.

**Memory implementation:** Linear scan by `OccurredAt()`, or binary search if sorted.

**Caveat to document:** Timestamps are not monotonic (clock skew, corrections). Concurrent events at the same timestamp may produce non-deterministic ordering. Version-based queries are always deterministic; timestamp queries are best-effort.

**Why P1:** Natural for the "as-at" question. Required for regulatory replay ("show me the system state at the time of this trade"). Moderate effort — needs timestamp index in SQL store.

### 5.3 P1: Repository-Level Time Travel

**Convenience methods that compose Store-level primitives with fold/apply.**

```go
// On aggregate.Repository
LoadAtVersion(ctx context.Context, root Root, version event.Version) error
LoadAtTime(ctx context.Context, root Root, t time.Time) error

// On decider.Repository
LoadAtVersion(ctx context.Context, aggID id.AggregateID, aggType event.AggregateType, version event.Version) (State, event.Version, error)
LoadAtTime(ctx context.Context, aggID id.AggregateID, aggType event.AggregateType, t time.Time) (State, event.Version, error)
```

**Implementation:** Load snapshot at/before target → `LoadToVersion` or `LoadToTimestamp` for remaining events → fold/apply → return state.

**Safety:** Document that temporal aggregates should not be saved (read-only). Consider adding `root.SetReadOnly()` or returning a distinct type that doesn't satisfy the `Root` interface.

**Why P1:** Natural consumer of P0/P1 Store methods. Gives consumers a single-call API instead of requiring manual composition.

### 5.4 P1: Position-Based `GlobalLoader` — Production-Scale Projection Replay

**Replace O(n) `LoadAll` + `filterEvents` with position-aware loading.**

```go
// Extend GlobalLoader or create new interface
type PositionalLoader interface {
    LoadAllFromPosition(
        ctx context.Context,
        afterEventID id.EventID,
        limit int,
    ) ([]Event, error)
}
```

**SQL implementation:** `WHERE id > $1 ORDER BY occurred_at ASC LIMIT $2` — uses primary key.

**Memory implementation:** Binary search by event ID, then slice.

**Why P1:** This is already documented as a known issue (TODO_LIST.md, architecture reviews). `LoadAll` + `filterEvents` is the #1 performance bottleneck for production-scale projection replay. At 1M events, replay takes seconds; with position-based loading, it takes milliseconds.

### 5.5 P2: `Store.ReadBackwards` — Reverse Stream Reads

**Efficiently query the last N events from a stream.**

```go
// New method on event.Store
ReadBackwards(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxCount int,
) ([]Event, error)
```

**SQL implementation:** `ORDER BY version DESC LIMIT $3` then re-sort ASC.

**Use cases:** "What was the last event?", "Last N events before this point", "Find the most recent snapshot trigger event".

**Why P2:** Low effort, moderate value. EventStoreDB, Eventuous, and KurrentDB all provide this. Useful for optimization patterns but not critical for initial time-travel support.

### 5.6 P3: Global Transaction Position (Breaking — Next Major Version)

**Add a monotonic global transaction ID to every event.** Enables cross-aggregate temporal queries.

```go
// Add to Event interface (BREAKING)
TransactionID() id.TransactionID

// New GlobalLoader method
LoadAllFromPosition(ctx context.Context, position id.TransactionID, maxCount int) ([]Event, error)
```

**Implementation approaches:**

1. **Database sequence** (PostgreSQL `SEQUENCE`) — simplest, single-writer assumption
2. **Hybrid logical clock (HLC)** — distributed, combines physical clock + logical counter
3. **ULID-based ordering** — already using `oklog/ulid`, ULIDs are time-sortable

**Why P3 (not P0):** Breaking change to the `Event` interface. All `NewEvent` calls need a `TransactionID`. All implementations must be updated. High value (unlocks `$all` stream, cross-aggregate replay, efficient projection checkpointing), but requires a major version bump.

**Alternative (non-breaking):** Make `TransactionID()` optional via interface assertion:

```go
type TransactionalEvent interface {
    Event
    TransactionID() id.TransactionID
}
```

SQL store can set it on Save; MemoryStore can use an atomic counter. Consumers type-assert when needed.

### 5.7 P3: Bi-Temporal Support (Opt-In)

**Add `ValidAt` to event metadata for domains that need "as-of" queries.**

```go
// New metadata field (non-breaking, optional)
type Metadata struct {
    // ... existing fields ...
    ValidAt time.Time `json:"validAt,omitempty"`
}

// New option
event.WithValidAt(t time.Time) Option

// New store method
LoadToValidTime(ctx, aggType, aggID, maxTime) ([]Event, error)
```

**Why P3:** Most domains don't need this. Financial, HR, and healthcare domains do. Non-breaking (additive). Requires a `valid_at` column and index in SQL store. Projection logic must decide whether to fold by transaction time or valid time — document clearly.

### 5.8 P4: Speculative Application

**Apply events to an aggregate without persisting.** Enables "what-if" queries.

```go
// On decider.Repository
func (r *Repository[State]) ApplySpeculative(
    ctx context.Context,
    aggID id.AggregateID,
    aggType event.AggregateType,
    decide DecideFunc[State],
) (State, event.Version, error) {
    // Execute the full decide→fold pipeline but DO NOT save or publish
}
```

**Why P4:** Can be approximated at the application level today (load state, fold speculative events in memory). A library-level API would be convenient but not essential. Low priority.

### 5.9 P4: Temporal Projections

**Versioned read-model snapshots over time.**

```go
type TemporalProjection struct {
    Name       string
    resolution time.Duration  // e.g., daily, hourly
    // stores state at each time boundary
}
```

**Why P4:** Niche. Storage cost proportional to (number of time snapshots × state size). Only practical for coarse granularity. Could be a separate module (`projection/temporal/`).

---

## 6. Recommended Roadmap

### Phase 1: Foundation (Non-Breaking, ~2-3 days)

| Task                                                                               | Effort | Impact     |
| ---------------------------------------------------------------------------------- | ------ | ---------- |
| Add `Store.LoadToVersion` to interface + MemoryStore + SQLEventStore + PebbleStore | 2h     | ⭐⭐⭐⭐⭐ |
| Add `Store.LoadToTimestamp` to interface + MemoryStore + SQLEventStore             | 2h     | ⭐⭐⭐⭐   |
| Add `aggregate.Repository.LoadAtVersion` / `LoadAtTime`                            | 1h     | ⭐⭐⭐⭐   |
| Add `decider.Repository.LoadAtVersion` / `LoadAtTime`                              | 1h     | ⭐⭐⭐⭐   |
| Add `GlobalLoader.LoadAllFromPosition` or new `PositionalLoader` interface         | 2h     | ⭐⭐⭐⭐⭐ |
| Update `projection.Runner` to use position-based loading                           | 2h     | ⭐⭐⭐⭐⭐ |
| Tests for all new methods                                                          | 3h     | —          |
| Documentation                                                                      | 1h     | —          |

**Total: ~14h**

### Phase 2: Enrichment (Non-Breaking, ~1-2 days)

| Task                                                                       | Effort | Impact   |
| -------------------------------------------------------------------------- | ------ | -------- |
| Add `Store.ReadBackwards` to interface + MemoryStore + SQLEventStore       | 2h     | ⭐⭐⭐   |
| Add temporal aggregate read-only safety (prevent Save after LoadAtVersion) | 2h     | ⭐⭐⭐   |
| Add timestamp index to SQL DDL                                             | 1h     | ⭐⭐⭐⭐ |
| Documentation + examples                                                   | 2h     | —        |

**Total: ~7h**

### Phase 3: Global Position (Breaking — v2 Planning, ~5-7 days)

| Task                                                 | Effort | Impact     |
| ---------------------------------------------------- | ------ | ---------- |
| Add `TransactionID` branded type                     | 1h     | ⭐⭐⭐⭐⭐ |
| Add `Event.TransactionID()` to interface (BREAKING)  | 2h     | ⭐⭐⭐⭐⭐ |
| Update `NewEvent` to assign TransactionID from store | 2h     | —          |
| Update MemoryStore (atomic counter)                  | 2h     | —          |
| Update SQLEventStore (BIGSERIAL / SEQUENCE)          | 3h     | —          |
| Update PebbleStore                                   | 2h     | —          |
| Add `LoadAllFromPosition` to GlobalLoader            | 2h     | —          |
| Update all test helpers, fakes, examples             | 4h     | —          |
| Tests                                                | 4h     | —          |

**Total: ~22h**

### Phase 4: Bi-Temporal (Opt-In, ~3-4 days)

| Task                                             | Effort | Impact |
| ------------------------------------------------ | ------ | ------ |
| Add `ValidAt` to Metadata + `WithValidAt` option | 1h     | ⭐⭐⭐ |
| Add `valid_at` column to SQL DDL + index         | 1h     | —      |
| Add `Store.LoadToValidTime`                      | 2h     | ⭐⭐⭐ |
| Add `Repository.LoadAsOf`                        | 1h     | ⭐⭐⭐ |
| Document bi-temporal projection semantics        | 2h     | —      |
| Tests                                            | 3h     | —      |

**Total: ~10h**

---

## 7. What We Should NOT Provide (And Why)

| Mechanism                                                  | Verdict               | Rationale                                                                                                                                                                                                                                                                                                                                       |
| ---------------------------------------------------------- | --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Retraction model** (`:db/retract`-style)                 | ❌ **Skip**           | Our event model is coarse-grained (payload blobs, not datoms). Datomic's retraction works because it operates on `[E A V]` triples — retracting a specific attribute is semantically clear. With our model, "retract event X" is ambiguous (what part of the payload?). Domain-specific events (e.g., `UserEmailRetracted`) handle this better. |
| **History query** (all versions of an attribute over time) | ❌ **Skip**           | Requires datom-based storage (Datomic's EAV model), not event-stream storage. Fundamentally incompatible with our architecture.                                                                                                                                                                                                                 |
| **Event compaction / truncation**                          | ❌ **Skip** (for now) | Destroys time-travel ability. Useful for storage optimization but should only be considered after time-travel APIs are complete, and only with clear documentation of what's lost. Can be revisited as an opt-in destructive operation.                                                                                                         |
| **Forking / branching event streams**                      | ❌ **Skip**           | No major framework implements this natively. Complex stream management with unclear merge semantics. What-if analysis is better served by speculative application (P4).                                                                                                                                                                         |
| **Database-as-Value** (Datomic `d/as-of` model)            | ❌ **Skip**           | Would require fundamental architecture change (datom-based storage, immutable store views). Our stream-per-aggregate model is the right one for Go. The time-travel API we're adding provides the same capability with a different abstraction.                                                                                                 |

---

## 8. Industry Comparison

How go-cqrs-lite compares to other event sourcing systems for time-travel capabilities:

| Capability            | go-cqrs-lite (today)  | go-cqrs-lite (ideal) | EventStoreDB     | Marten                | Axon                     | Eventuous (Go)                             | Datomic                  |
| --------------------- | --------------------- | -------------------- | ---------------- | --------------------- | ------------------------ | ------------------------------------------ | ------------------------ |
| Load all events       | ✅ `Store.Load`       | ✅                   | ✅               | ✅                    | ✅                       | ✅                                         | ✅                       |
| Load from version     | ✅ `LoadFromVersion`  | ✅                   | ✅               | ✅                    | ✅                       | ✅                                         | ✅                       |
| **Load TO version**   | ❌                    | ✅ `LoadToVersion`   | ✅ revision read | ✅ `version:` param   | ✅ `readEvents(id, seq)` | ✅ `ReadEvents(ctx, stream, start, count)` | ✅ `d/as-of`             |
| **Load to timestamp** | ❌                    | ✅ `LoadToTimestamp` | ❌               | ✅ `timestamp:` param | ❌                       | ❌                                         | ✅ `d/as-of`             |
| Snapshot at version   | ✅ `LoadAtVersion`    | ✅                   | ✅               | ✅                    | ✅                       | ❌                                         | N/A (all-points)         |
| Backward reads        | ❌                    | ✅ `ReadBackwards`   | ✅ `BACKWARDS`   | ❌                    | ❌                       | ✅ `ReadEventsBackwards`                   | ✅                       |
| Global position       | ❌                    | ✅ (P3)              | ✅ `$all` stream | ✅ `seq_id`           | ✅ `TrackingToken`       | ✅ `GlobalPosition`                        | ✅ monotonic `t`         |
| Bi-temporal           | ❌                    | ✅ (P3)              | ✅ via metadata  | ✅ system+business    | ❌                       | ❌                                         | ✅ `d/as-of` + `d/since` |
| Speculative apply     | ❌                    | ✅ (P4)              | ❌               | ❌                    | ❌                       | ❌                                         | ✅ `d/with`              |
| Upcasting             | ✅ `UpcasterRegistry` | ✅                   | ❌               | ❌                    | ✅                       | ❌                                         | N/A                      |
| Projection checkpoint | ✅ `CheckpointStore`  | ✅                   | ✅               | ✅                    | ✅                       | ✅                                         | N/A                      |

**Verdict:** With Phase 1 + Phase 2 implemented, go-cqrs-lite would match or exceed the time-travel capabilities of EventStoreDB, Eventuous, and Axon. With Phase 3, it would approach Marten-level capabilities. Bi-temporal (Phase 4) puts it in Datomic/Rails EventStore territory for that specific dimension.

---

## 9. Appendix: Mechanism Inventory

All 13 known time-travel mechanisms, mapped to go-cqrs-lite status:

| #   | Mechanism                   | Status         | Implementation                                                               |
| --- | --------------------------- | -------------- | ---------------------------------------------------------------------------- |
| 1   | Full Event Replay           | ✅ **Working** | `Store.Load` → manual fold                                                   |
| 2   | Snapshot-Optimized Replay   | ✅ **Working** | `SnapshotStore.LoadAtVersion` + `LoadFromVersion`                            |
| 3   | Version-Based Stream Read   | ❌ **Missing** | → `Store.LoadToVersion` (P0)                                                 |
| 4   | Timestamp-Based Stream Read | ❌ **Missing** | → `Store.LoadToTimestamp` (P1)                                               |
| 5   | Global Position Read        | ❌ **Missing** | → `PositionalLoader.LoadAllFromPosition` (P1) / `Event.TransactionID()` (P3) |
| 6   | Database-as-Value           | ❌ **Skip**    | Incompatible with our architecture                                           |
| 7   | Speculative Application     | ❌ **Missing** | → `Repository.ApplySpeculative` (P4)                                         |
| 8   | History Query               | ❌ **Skip**    | Requires datom-based storage                                                 |
| 9   | Temporal Projections        | ❌ **Missing** | → `TemporalProjection` type (P4)                                             |
| 10  | Bi-Temporal Events          | ❌ **Missing** | → `WithValidAt` + `LoadToValidTime` (P3)                                     |
| 11  | Reverse Reads               | ❌ **Missing** | → `Store.ReadBackwards` (P2)                                                 |
| 12  | Event Compaction            | ❌ **Skip**    | Destroys time-travel ability                                                 |
| 13  | Forking/Branching           | ❌ **Skip**    | No standard pattern, unclear semantics                                       |

---

_This analysis synthesized from: codebase audit of all 12 modules, `docs/research/time-travel-options.md`, `docs/research/datomic-lessons.md`, `docs/research/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md`, `docs/research/2026-05-01_INNOVATIVE_CQRS_EVENT_SOURCING_PROJECTS.md`, `docs/research/2026-05-01_LIVESTORE_DEEP_DIVE.md`, EventStoreDB/KurrentDB documentation, Marten documentation, Axon Framework documentation, Eventuous documentation, Datomic documentation, and Rails EventStore bi-temporal documentation._
