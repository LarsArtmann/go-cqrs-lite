# Time Travel in Event Sourced Systems

> Comprehensive research on all known approaches to temporal queries in event-sourced systems, mapped to go-cqrs-lite's architecture.

**Date:** 2026-05-01

---

## Table of Contents

1. [What Is Time Travel?](#1-what-is-time-travel)
2. [The Two Time Dimensions](#2-the-two-time-dimensions)
3. [Time Travel Mechanisms (All Known Approaches)](#3-time-travel-mechanisms)
4. [How Other Systems Do It](#4-how-other-systems-do-it)
5. [Current State of go-cqrs-lite](#5-current-state-of-go-cqrs-lite)
6. [Options for go-cqrs-lite](#6-options-for-go-cqrs-lite)
7. [Decision Matrix](#7-decision-matrix)
8. [Recommended Roadmap](#8-recommended-roadmap)
9. [Appendix: Sources](#appendix-sources)

---

## 1. What Is Time Travel?

Time travel is the ability to **observe or reconstruct the state of a system at any point in its history** — not just the current state. In event-sourced systems, this is theoretically trivial (replay events), but practically complex (performance, correctness, API design, storage cost).

Greg Young's fundamental insight: _"Your bank balance isn't a column in a table; it's the sum of all transactions."_ Current state is always a derivative of immutable facts. If you preserve the facts, you can always reconstruct any past state.

Martin Fowler: _"We can determine the application state at any point in time. Notionally we do this by starting with a blank state and rerunning the events up to a particular time or event. We can take this further by considering multiple time-lines (analogous to branching in a version control system)."_

### Time Travel Use Cases

| Category                     | Example                                                                        | Value                                        |
| ---------------------------- | ------------------------------------------------------------------------------ | -------------------------------------------- |
| **Debugging**                | "Why was this order rejected?"                                                 | Replay to exact state at time of decision    |
| **Audit & Compliance**       | "What was user X's permission level on March 15?"                              | Legal requirement in finance, healthcare     |
| **Temporal Analytics**       | "How many accounts were active vs. churned each month?"                        | Business intelligence from history           |
| **Retroactive Corrections**  | "Salary change should have been effective Jan 1, not Feb 1"                    | Correct reality without destroying history   |
| **What-If / Simulation**     | "What would the balance be if we applied this fee?"                            | Speculative transactions without persistence |
| **Race Condition Diagnosis** | "What was the aggregate state when this concurrent write arrived?"             | Production debugging                         |
| **Regulatory Replay**        | "Show me the system state at the time of this trade"                           | MiFID II, SOX compliance                     |
| **Migration Testing**        | "Does the new projection logic produce the same state from historical events?" | Zero-downtime deployments                    |

---

## 2. The Two Time Dimensions

All time travel in event sourcing operates on one or both of two temporal axes:

### Transaction Time (System Time / "As-At")

**When the event was recorded in the system.** Immutable. Automatically assigned. This is the `OccurredAt()` on our events.

```
Event written to store at 2026-05-01T14:30:00Z
→ Transaction time = 2026-05-01T14:30:00Z (cannot change)
```

### Valid Time (Business Time / "As-Of")

**When the event was true in the real world.** Can differ from transaction time. Must be provided by the domain. This is _not_ currently on our events.

```
Salary change recorded on 2026-02-01 but effective 2026-01-01
→ Transaction time = 2026-02-01
→ Valid time = 2026-01-01
```

### The Four Query Modes

| Query Mode      | Time Axis        | Question                                                               | Example                                           |
| --------------- | ---------------- | ---------------------------------------------------------------------- | ------------------------------------------------- |
| **Current**     | Neither          | "What is the state now?"                                               | Default aggregate load                            |
| **As-At**       | Transaction time | "What did the system know at time T?"                                  | "What was the balance as recorded on March 15?"   |
| **As-Of**       | Valid time       | "What was true in reality at time T?"                                  | "What salary was effective on Jan 1?"             |
| **Bi-Temporal** | Both             | "What did we believe was true at recording-time R about valid-time V?" | "What did we record on Feb 1 as effective Jan 1?" |

Most systems only need **As-At**. Financial, HR, and healthcare domains often need **As-Of** or **Bi-Temporal**.

---

## 3. Time Travel Mechanisms

### Mechanism 1: Full Event Replay

**The fundamental mechanism.** Load all events for an aggregate, replay them in order, stop at the desired point.

```
events := store.Load(ctx, aggType, aggID)
state := zeroState
for _, evt := range events {
    if evt.Version() > targetVersion { break }
    state = state.Apply(evt)
}
```

**Pros:** Simple, always correct, no extra storage, works with any event store
**Cons:** O(n) where n = number of events, gets slow for large streams (1000+ events)
**Used by:** Every event sourcing system (the baseline)

### Mechanism 2: Snapshot-Optimized Replay

**Load the nearest snapshot before the target point, then replay only events after the snapshot.**

```
snapshot := snapshotStore.LoadAtVersion(ctx, aggType, aggID, targetVersion)
state := snapshot.State
events := store.LoadFromVersion(ctx, aggType, aggID, snapshot.Version)
for _, evt := range events {
    if evt.Version() > targetVersion { break }
    state = state.Apply(evt)
}
```

**Pros:** O(k) where k = events after snapshot, practical for large streams
**Cons:** Requires snapshot storage, snapshot must be at or before target version, more complex
**Used by:** Axon Framework, go-cqrs-lite (partially — `LoadAtVersion` exists but no version-targeted load)

### Mechanism 3: Version-Based Stream Read

**Read events up to a specific version number.** The store itself filters, not the application.

```
// Hypothetical API
events := store.LoadToVersion(ctx, aggType, aggID, targetVersion)
state := fold(events)
```

**Pros:** Clean API, store can optimize (e.g., SQL `WHERE version <= N`), no over-reading
**Cons:** Only works for per-aggregate version targeting, not for timestamp-based queries
**Used by:** EventStoreDB (stream revision reads), Eventuous (`ReadEvents(ctx, stream, startVersion, count)`), Axon (`readEvents(aggregateId, firstSequenceNumber)`)

### Mechanism 4: Timestamp-Based Stream Read

**Read events up to a specific wall-clock time.** Uses `OccurredAt()` for filtering.

```
// Hypothetical API
events := store.LoadToTimestamp(ctx, aggType, aggID, targetTime)
```

**Pros:** Natural for "as-at" queries, aligns with business language
**Cons:** Timestamps are not monotonic (clock skew, corrections), ambiguous for concurrent events at same timestamp, requires store-level timestamp indexing
**Used by:** Marten (`AggregateStreamAsync<T>(id, timestamp: pointOfTime)`), EventSourcingDB (`upperBound` parameter)

### Mechanism 5: Global Position / Transaction Log Read

**Read from the global event log at a specific position.** Enables cross-aggregate time travel.

```
// Hypothetical API
allEvents := store.ReadAllFromPosition(ctx, globalPosition)
```

**Pros:** Cross-aggregate temporal consistency, enables "$all stream" patterns, supports subscription/checkpoint replay
**Cons:** Requires a global monotonic position counter, more storage overhead, read performance depends on global log size
**Used by:** EventStoreDB (`$all`stream with commit/prepare position), Kafka (offset-based reads), Datomic (monotonic`t` value)

### Mechanism 6: Database-as-Value (Datomic Model)

**The entire database is an immutable value.** You hold a reference to the database at a point in time and query it like any other value.

```clojure
;; Datomic
(def db-at-jan (d/as-of db #inst "2026-01-01"))
(d/q '[:find ?e ?v :where [?e :user/email ?v]] db-at-jan)
```

**Pros:** Pure functional, no coordination needed, composable, cacheable, works across all entities
**Cons:** Requires structural changes to the data model (datom-based), fundamentally different from stream-per-aggregate model, complex to implement in Go
**Used by:** Datomic exclusively (the defining feature)

### Mechanism 7: Speculative Application (Dry-Run)

**Apply events to a database value without persisting.** Enables "what-if" queries.

```clojure
;; Datomic
(def speculative-db (d/with db [[:db/add 42 :user/email "test@example.com"]]))
(d/q '[:find ?email :where [?e :user/email ?email]] speculative-db)
```

**Pros:** Zero risk, enables business rule validation before commit, supports preview features
**Cons:** Requires a "database value" abstraction, not just a write-then-read store
**Used by:** Datomic (`d/with`), can be approximated with in-memory stores

### Mechanism 8: History Query (All Versions)

**Query all versions of an entity across time.** Returns both assertions and retractions.

```clojure
;; Datomic
(d/q '[:find ?status ?inst ?op
       :in $ ?e
       :where [?e :book/status ?status ?tx ?op]
              [?tx :db/txInstant ?inst]]
     (d/history db) entity-id)
;; Returns: [["available" #inst "2026-01" true]
;;           ["borrowed"    #inst "2026-02" true]
;;           ["available"   #inst "2026-03" true]]
```

**Pros:** Complete audit trail, sees both additions and retractions, essential for compliance
**Cons:** Requires tracking operation type (assert/retract), large result sets for long-lived entities
**Used by:** Datomic (`d/history`)

### Mechanism 9: Temporal Projection (Versioned Read Models)

**Maintain read models that are themselves versioned over time.** The projection stores state snapshots keyed by time.

```
type TemporalProjection struct {
    Name     string
    States   map[time.Time]State  // keyed by transaction time
}
```

**Pros:** O(1) lookup for pre-computed time points, supports analytics queries
**Cons:** Storage multiplication (one state per time point), update cost on every event, only practical for coarse time granularity (daily/hourly snapshots)
**Used by:** EventSourcingDB (incremental read models), custom implementations

### Mechanism 10: Bi-Temporal Event Streams

**Events carry both transaction time and valid time.** Projections can be built along either axis.

```
type BiTemporalEvent struct {
    Event          Event
    TransactionTime time.Time  // when recorded
    ValidTime      time.Time  // when true in reality
}
```

**As-At query:** Replay events ordered by `TransactionTime`
**As-Of query:** Replay events ordered by `ValidTime`

**Pros:** Full bi-temporal capability, supports retroactive corrections
**Cons:** Complex projection logic, event ordering ambiguity (same valid time, different transaction times), requires careful conflict resolution
**Used by:** Rails EventStore (`as_at` / `as_of` scopes), EventSourcingDB, planetgeek.ch patterns

### Mechanism 11: Reverse Reads

**Read events backwards from the end of a stream.** Efficient for "what was the last X?" queries.

```
events := store.ReadBackwards(ctx, aggType, aggID, maxCount)
```

**Pros:** O(k) for last-k queries without scanning entire stream
**Cons:** Only useful for "last N" queries, not arbitrary time travel
**Used by:** EventStoreDB (BACKWARDS direction), Eventuous (`ReadEventsBackwards`), KurrentDB

### Mechanism 12: Event Compaction / Truncation (Rewriting History)

**Delete or compact old events.** The opposite of time travel — destroys the ability to time travel.

```
store.TruncateBefore(ctx, aggType, aggID, minVersion)
```

**Pros:** Reduces storage, improves replay performance
**Cons:** Destroys history, violates immutability principle, makes some time travel impossible
**Used by:** EventStoreDB (stream truncation via `$maxCount` metadata), Kafka (log compaction — but incompatible with ES, use `delete` retention instead)

> **Important:** Log compaction (Kafka-style "keep last value per key") is **incompatible** with event sourcing. Event sourcing requires the _full history_, not just the latest state per key.

### Mechanism 13: Forking / Branching Event Streams

**Create alternative timelines from a point in history.** Like git branches for events.

```
// Fork at version 5
forkedStreamID := store.Fork(ctx, aggType, aggID, atVersion: 5)
```

**Pros:** Enables what-if analysis, A/B testing of business logic, parallel simulations
**Cons:** Complex stream management, merge semantics unclear, no standard pattern
**Used by:** Martin Fowler (theoretical mention), no major framework implements this natively

---

## 4. How Other Systems Do It

### EventStoreDB / KurrentDB

| Capability                    | Implementation                                       |
| ----------------------------- | ---------------------------------------------------- |
| Stream revision read          | `ReadStream(fromRevision: N)`                        |
| $all stream (global position) | `ReadAll(fromPosition: {commit, prepare})`           |
| Backward reads                | `direction: BACKWARDS`                               |
| Projections                   | JavaScript-based continuous queries                  |
| Filtering                     | By event type, stream, metadata                      |
| Bi-temporal                   | Via `valid_at` metadata + `as_at`/`as_of` read modes |
| Global ordering               | Commit/prepare position in transaction log           |
| Optimistic concurrency        | Expected stream revision                             |

**Key insight:** Two positioning systems — stream revision (per-aggregate) and global position (cross-aggregate). The `$all` stream is the global event log, enabling cross-stream temporal queries.

### Marten (.NET / PostgreSQL)

| Capability           | Implementation                                                  |
| -------------------- | --------------------------------------------------------------- |
| Version-based load   | `AggregateStreamAsync<T>(id, version: N)`                       |
| Timestamp-based load | `AggregateStreamAsync<T>(id, timestamp: T)`                     |
| Event storage        | `mt_events` table with `seq_id`, `version`, `timestamp` columns |
| Inline projections   | Transactional (within same DB tx)                               |
| Async projections    | Background daemon with checkpoint tracking                      |
| Live projections     | On-demand computation from raw events                           |
| Bi-temporal          | System time (db timestamp) + business time (in event data)      |

**Key insight:** PostgreSQL's JSONB + timestamp indexing makes temporal queries efficient at the store level. `AggregateStreamAsync<T>` with `version:` or `timestamp:` parameters is the gold standard for aggregate-level time travel API.

### Axon Framework (Java)

| Capability          | Implementation                                            |
| ------------------- | --------------------------------------------------------- |
| Sequence-based read | `eventStore.readEvents(aggregateId, firstSequenceNumber)` |
| Domain event stream | `DomainEventStream` with `hasNext()`, `next()`, `peek()`  |
| Event replay        | `TrackingEventProcessor.resetTokens()`                    |
| Snapshots           | `Snapshotter` with configurable policies                  |
| Position tracking   | `TrackingToken` persisted in `TokenStore`                 |
| Speculative loads   | Manual: read events from sequence N, apply manually       |

**Key insight:** The `DomainEventStream` is a lazy iterator — events are loaded on demand, enabling efficient temporal reads without loading entire streams. The `readEvents(id, firstSequenceNumber)` API is the simplest version-based time travel.

### Datomic

| Capability      | Implementation                                              |
| --------------- | ----------------------------------------------------------- |
| As-of           | `d/as-of db t-or-instant` → filtered database value         |
| Since           | `d/since db t-or-instant` → database value since point      |
| History         | `d/history db` → all assertions + retractions               |
| Speculative     | `d/with db tx-data` → new database value without persisting |
| Global ordering | Monotonic `t` value per transaction                         |
| Indexes         | EAVT, AVET, AEVT, VAET — all include time dimension         |
| Retractions     | `:db/retract` stored as new datoms with `added=false`       |

**Key insight:** Database-as-value is the most powerful time travel model. Every query operates on an immutable snapshot of the entire database at a point in time. No special "time travel API" — time travel is just querying a different value.

### Eventuous (Go)

| Capability             | Implementation                                      |
| ---------------------- | --------------------------------------------------- |
| Version-based read     | `ReadEvents(ctx, stream, startVersion, count)`      |
| Backward read          | `ReadEventsBackwards(ctx, stream, start, count)`    |
| Global position        | `GlobalPosition uint64` in `StreamEvent`            |
| Checkpoint tracking    | `CheckpointCommitter` for subscription resume       |
| Optimistic concurrency | `ExpectedVersion` (NoStream, Any, specific version) |
| State reconstruction   | `LoadState` with `fold` function                    |
| As-of queries          | Manual: read events + fold up to target             |

**Key insight:** Go-idiomatic, no built-in "as-of" API. Time travel is achieved by reading events at specific positions and folding manually. The `GlobalPosition` enables cross-stream temporal coordination.

### Rails EventStore (Ruby)

| Capability     | Implementation                                           |
| -------------- | -------------------------------------------------------- |
| Bi-temporal    | `valid_at` metadata on events                            |
| As-at reads    | `event_store.read.stream("x").as_at.to_a` (by timestamp) |
| As-of reads    | `event_store.read.stream("x").as_of.to_a` (by valid_at)  |
| Position reads | Stream position-based reads                              |

**Key insight:** First-class bi-temporal support with explicit `as_at`/`as_of` read modes. The simplest API for the common "as-of" query pattern.

### Kafka (Infrastructure-Level)

| Capability            | Implementation                                          |
| --------------------- | ------------------------------------------------------- |
| Offset-based reads    | `consumer.seek(partition, offset)`                      |
| Timestamp-based reads | `offsetsForTimes()` to find offset by timestamp         |
| Log retention         | Configurable (time-based or size-based)                 |
| Log compaction        | ⚠️ **INCOMPATIBLE with ES** — keeps only latest per key |
| Global ordering       | Per-partition only, not cross-partition                 |

**Key insight:** Kafka provides offset-based and timestamp-based time travel at the infrastructure level, but log compaction destroys event sourcing history. For ES, use `retention.ms=-1` with `cleanup.policy=delete`.

---

## 5. Current State of go-cqrs-lite

### What We Have

| Feature                  | Location                                                    | Status     |
| ------------------------ | ----------------------------------------------------------- | ---------- |
| Per-aggregate versioning | `event.Version` on every event                              | ✅ Working |
| Load all events          | `Store.Load(ctx, aggType, aggID)`                           | ✅ Working |
| Load from version        | `Store.LoadFromVersion(ctx, aggType, aggID, version)`       | ✅ Working |
| Snapshot at version      | `SnapshotStore.LoadAtVersion(ctx, aggType, aggID, version)` | ✅ Working |
| Event timestamps         | `Event.OccurredAt()` (transaction time)                     | ✅ Working |
| Optimistic concurrency   | `Store.Save(ctx, ..., expectedVersion)`                     | ✅ Working |
| Event upcasting          | `Upcaster` interface for schema evolution                   | ✅ Working |
| Projection checkpoint    | `CheckpointStore` in `InMemoryRunner`                       | ✅ Working |

### What We Lack

| Feature                            | Impact                                                    | Priority |
| ---------------------------------- | --------------------------------------------------------- | -------- |
| **No LoadToVersion**               | Can't read events up to a specific version at store level | HIGH     |
| **No timestamp-based queries**     | Can't query "state at time T"                             | HIGH     |
| **No global transaction position** | No cross-aggregate temporal consistency                   | MEDIUM   |
| **No valid-time (as-of)**          | No bi-temporal support                                    | MEDIUM   |
| **No backward reads**              | Can't efficiently query "last N events"                   | LOW      |
| **No speculative application**     | Can't dry-run events                                      | LOW      |
| **No retraction model**            | No semantic "no longer true"                              | LOW      |
| **No history query**               | Can't see all versions of a value over time               | LOW      |
| **No StoreView**                   | Store returns slices, not an immutable value              | LOW      |
| **No global $all stream**          | No way to read all events across aggregates               | MEDIUM   |

### Current Time Travel Workaround

To reconstruct an aggregate at version 5 today:

```go
// The only way: load ALL events, filter manually
events, _ := store.Load(ctx, aggType, aggID)
var target []event.Event
for _, e := range events {
    if e.Version() <= 5 {
        target = append(target, e)
    }
}
// Then fold target into aggregate state
```

This is **O(n)** for every time-travel query, even if you only need the first 5 events of a 10,000-event stream.

---

## 6. Options for go-cqrs-lite

### Option A: LoadToVersion (Version-Based Time Travel)

**Add `Store.LoadToVersion(ctx, aggType, aggID, maxVersion)` that returns events up to and including `maxVersion`.**

```go
// New method on Store interface
type Store interface {
    // ... existing methods ...

    // LoadToVersion retrieves events up to and including maxVersion.
    LoadToVersion(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxVersion Version,
    ) ([]Event, error)
}
```

**Implementation in MemoryStore:**

```go
func (s *MemoryStore) LoadToVersion(
    _ context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxVersion Version,
) ([]Event, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    key := streamKey(aggregateType, aggregateID)
    events, exists := s.events[key]
    if !exists {
        return nil, ErrAggregateNotFound
    }

    // Events are 1-indexed, so version N is at index N-1
    end := min(maxVersion.Int(), len(events))
    result := make([]Event, end)
    copy(result, events[:end])
    return result, nil
}
```

**Implementation in SQL Store:**

```sql
SELECT * FROM events
WHERE aggregate_type = $1 AND aggregate_id = $2 AND version <= $3
ORDER BY version ASC
```

**Impact:**

- Interface change (new method on `Store`)
- MemoryStore: trivial
- SQL Store: index on `(aggregate_type, aggregate_id, version)` already exists
- EventSourcedRepository: add `LoadAtVersion(ctx, root, version)` method

**Verdict:** ✅ **Minimal effort, high value. This should be the first addition.**

---

### Option B: LoadToTimestamp (Timestamp-Based Time Travel)

**Add `Store.LoadToTimestamp(ctx, aggType, aggID, maxTime)` that returns events where `OccurredAt() <= maxTime`.**

```go
// New method on Store interface
type Store interface {
    // ... existing methods ...

    // LoadToTimestamp retrieves events up to and including maxTime.
    LoadToTimestamp(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxTime time.Time,
    ) ([]Event, error)
}
```

**Implementation in SQL Store:**

```sql
SELECT * FROM events
WHERE aggregate_type = $1 AND aggregate_id = $2 AND occurred_at <= $3
ORDER BY version ASC
```

**Impact:**

- Interface change (new method on `Store`)
- Requires timestamp index in SQL store
- MemoryStore: linear scan or secondary index by time
- Timestamps are non-monotonic — document that concurrent events at the same timestamp may produce non-deterministic ordering

**Verdict:** ✅ **High value, moderate effort. Second priority after LoadToVersion.**

---

### Option C: Repository-Level Time Travel

**Add time-travel methods on `EventSourcedRepository` instead of (or in addition to) the Store.**

```go
// New methods on Repository
type Repository interface {
    // ... existing methods ...

    // LoadAtVersion loads the aggregate at a specific version.
    LoadAtVersion(ctx context.Context, root Root, version Version) error

    // LoadAtTime loads the aggregate at a specific point in time.
    LoadAtTime(ctx context.Context, root Root, t time.Time) error
}
```

**Implementation:**

```go
func (r *EventSourcedRepository) LoadAtVersion(
    ctx context.Context,
    root Root,
    version event.Version,
) error {
    events, err := r.store.LoadToVersion(ctx, root.Type(), root.ID(), version)
    if err != nil {
        return opError("load at version", root.Type(), root.ID(), err)
    }
    return root.LoadEvents(events)
}

func (r *EventSourcedRepository) LoadAtTime(
    ctx context.Context,
    root Root,
    t time.Time,
) error {
    events, err := r.store.LoadToTimestamp(ctx, root.Type(), root.ID(), t)
    if err != nil {
        return opError("load at time", root.Type(), root.ID(), err)
    }
    return root.LoadEvents(events)
}
```

**Impact:**

- No interface change to Store if Store already has `LoadToVersion`/`LoadToTimestamp`
- Convenience methods on Repository
- Mark loaded aggregate as read-only (temporal aggregates should not be saved)

**Verdict:** ✅ **Natural consumer of Options A+B. Implement alongside or after.**

---

### Option D: Global Transaction Position

**Add a monotonic global transaction ID to every event.** Enables cross-aggregate temporal queries.

```go
// Add to Event interface
type Event interface {
    // ... existing methods ...
    TransactionID() id.TransactionID  // NEW: global monotonic position
}

// Add to Store interface
type Store interface {
    // ... existing methods ...

    // LoadAllFromPosition retrieves events from the global log starting at a position.
    LoadAllFromPosition(
        ctx context.Context,
        position id.TransactionID,
        maxCount int,
    ) ([]Event, error)
}
```

**Implementation approaches:**

1. **Database sequence** (PostgreSQL `SEQUENCE`, auto-increment) — simplest, single-writer assumption
2. **Hybrid logical clock (HLC)** — distributed, combines physical clock + logical counter
3. **ULID-based ordering** — already using `oklog/ulid`, ULIDs are time-sortable

**Impact:**

- Breaking change to `Event` interface (new method)
- All `NewEvent` calls need a `TransactionID` assigned
- MemoryStore: atomic counter
- SQL Store: `BIGSERIAL` or `SEQUENCE` + index
- Enables: `$all` stream, cross-aggregate replay, subscription checkpointing at global position

**Verdict:** ⚠️ **High value but breaking change. Plan for next major version.**

---

### Option E: Valid-Time (Bi-Temporal Support)

**Add `ValidAt` to event metadata, enabling "as-of" queries alongside "as-at" queries.**

```go
// New metadata field
type Metadata struct {
    // ... existing fields ...
    ValidAt time.Time `json:"validAt,omitempty"` // NEW: when event was true in reality
}

// New option
func WithValidAt(t time.Time) Option {
    return apply(func(m *Metadata, t time.Time) { m.ValidAt = t }, t)
}

// New store method for as-of queries
type Store interface {
    // ... existing methods ...

    // LoadToValidTime retrieves events where ValidAt <= maxTime.
    LoadToValidTime(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxTime time.Time,
    ) ([]Event, error)
}
```

**Impact:**

- Non-breaking (additive change to Metadata, optional field)
- Requires valid-time index in SQL store
- Projection logic must decide: fold by transaction time or valid time?
- Most domains don't need this — document as opt-in feature

**Verdict:** ⚠️ **Niche but important for financial/HR domains. Optional, non-breaking.**

---

### Option F: Backward Reads

**Add `Store.ReadBackwards(ctx, aggType, aggID, maxCount)` to read from end of stream.**

```go
type Store interface {
    // ... existing methods ...

    // ReadBackwards retrieves the last maxCount events from a stream.
    ReadBackwards(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxCount int,
    ) ([]Event, error)
}
```

**Implementation in SQL Store:**

```sql
SELECT * FROM (
    SELECT * FROM events
    WHERE aggregate_type = $1 AND aggregate_id = $2
    ORDER BY version DESC
    LIMIT $3
) sub
ORDER BY version ASC
```

**Impact:**

- New interface method
- MemoryStore: trivial (slice reverse)
- SQL Store: requires DESC index
- Useful for "last event", "last N events" patterns

**Verdict:** ✅ **Low effort, moderate value. Third priority.**

---

### Option G: Speculative Application (Dry-Run)

**Add a way to apply events to an aggregate without persisting.**

```go
// On Repository
func (r *EventSourcedRepository) ApplySpeculative(
    ctx context.Context,
    root Root,
    events []event.Event,
) error {
    // Apply events to root in memory
    for _, evt := range events {
        if err := root.ApplyEvent(evt); err != nil {
            return err
        }
    }
    // DO NOT call store.Save or bus.Publish
    // DO NOT call root.MarkChangesAsCommitted
    return nil
}
```

**Alternative: Functional approach with StoreView**

```go
type StoreView struct {
    events []event.Event
}

// Create from existing store
view, _ := store.LoadToVersion(ctx, aggType, aggID, version)

// Apply speculative events
speculative := view.Apply(speculativeEvents)

// Query speculative state
state := fold(speculative.Events())
```

**Impact:**

- No interface changes needed for the simple approach
- StoreView would be a new abstraction (larger effort)
- Useful for preview/validation features

**Verdict:** ⚠️ **Low priority. Simple version can be done at application level.**

---

### Option H: Retraction Model

**Add a first-class `Retract` operation that semantically marks a fact as "no longer true" without destroying history.**

```go
// New event option — marks this event as a retraction
func WithRetracts(eventID id.EventID) Option { /* ... */ }

// New store method to query history including retractions
type Store interface {
    // LoadHistory returns all events including retractions.
    LoadHistory(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
    ) ([]Event, error)
}
```

**Impact:**

- Changes how projections interpret events (must check retraction metadata)
- Complex: what does "retract" mean for different event types?
- Could be expressed as a domain event pattern instead (e.g., `UserEmailRetracted`)
- Datomic's approach works because of the EAV model — retracting a specific attribute is clear. In our model, events are coarse-grained (a payload), not fine-grained (a datom).

**Verdict:** ❌ **Not recommended for go-cqrs-lite.** Our event model is too coarse-grained for Datomic-style retractions. Domain-specific events handle this better.

---

### Option I: History Query (All Versions Over Time)

**Add ability to query all versions of an aggregate's attribute over time.**

This is a Datomic-style feature that requires a fundamentally different data model (datoms vs. events). Not practical for go-cqrs-lite without a complete rewrite.

**Verdict:** ❌ **Not applicable.** Requires datom-based storage, not event-stream-based storage.

---

### Option J: Temporal Projections (Versioned Read Models)

**Enhance projections to produce versioned/time-stamped read model snapshots.**

```go
type TemporalProjection struct {
    Name       string
    intervals  map[time.Time][]byte  // state at each time boundary
    resolution time.Duration         // e.g., hourly, daily
}

func (p *TemporalProjection) Handle(ctx context.Context, evt Event) error {
    // Apply event to current state
    // At each time boundary, snapshot the current state
    // Enables "what was the read model state at time T?" queries
}
```

**Impact:**

- New projection type
- Storage cost proportional to (number of time snapshots × state size)
- Only practical for coarse granularity (daily, hourly)
- Useful for reporting/analytics

**Verdict:** ⚠️ **Medium effort, niche value. Consider for analytics module.**

---

## 7. Decision Matrix

| Option                          | Value      | Effort             | Breaking?                 | Priority            |
| ------------------------------- | ---------- | ------------------ | ------------------------- | ------------------- |
| **A: LoadToVersion**            | ⭐⭐⭐⭐⭐ | 🔨 Low             | No (additive)             | **P0**              |
| **B: LoadToTimestamp**          | ⭐⭐⭐⭐   | 🔨 Low             | No (additive)             | **P1**              |
| **C: Repository time travel**   | ⭐⭐⭐⭐   | 🔨 Low             | No (additive)             | **P1**              |
| **F: Backward reads**           | ⭐⭐⭐     | 🔨 Low             | No (additive)             | **P2**              |
| **D: Global transaction ID**    | ⭐⭐⭐⭐⭐ | 🔨🔨🔨 High        | **Yes** (Event interface) | **P3** (next major) |
| **E: Valid-time (bi-temporal)** | ⭐⭐⭐     | 🔨🔨 Medium        | No (additive)             | **P3** (opt-in)     |
| **G: Speculative application**  | ⭐⭐⭐     | 🔨 Low             | No                        | **P4**              |
| **J: Temporal projections**     | ⭐⭐       | 🔨🔨 Medium        | No                        | **P4**              |
| **H: Retraction model**         | ⭐⭐       | 🔨🔨🔨 High        | Maybe                     | **Skip**            |
| **I: History query**            | ⭐         | 🔨🔨🔨🔨 Very High | Yes                       | **Skip**            |

---

## 8. Recommended Roadmap

### Phase 1: Foundation (Non-Breaking)

Add version-based and timestamp-based time travel to `Store` and `Repository`.

**New Store methods:**

```
LoadToVersion(ctx, aggType, aggID, maxVersion) → []Event
LoadToTimestamp(ctx, aggType, aggID, maxTime) → []Event
ReadBackwards(ctx, aggType, aggID, maxCount) → []Event
```

**New Repository methods:**

```
LoadAtVersion(ctx, root, version) → error
LoadAtTime(ctx, root, time) → error
```

**Changes:**

- `event.Store` interface: 3 new methods
- `memory.MemoryStore`: trivial implementations
- `EventSourcedRepository`: 2 new methods
- `storage` module: SQL implementations
- Tests for all new methods
- Mark temporal aggregates as read-only (prevent accidental Save after time-travel load)

### Phase 2: Global Position (Breaking — Next Major Version)

Add a monotonic global transaction position to enable cross-aggregate time travel.

**Event interface change:**

```
TransactionID() id.TransactionID
```

**New Store methods:**

```
LoadAllFromPosition(ctx, position, maxCount) → []Event
```

**Changes:**

- `Event` interface: new method (breaking)
- `event.Core`: new field
- `NewEvent`: assign TransactionID from store
- `MemoryStore`: atomic counter
- SQL Store: `BIGSERIAL` or sequence
- Index on `transaction_id` for efficient global reads
- Update all implementations (memory, storage, testhelpers)

### Phase 3: Bi-Temporal (Opt-In)

Add valid-time support for domains that need it.

**New Metadata field:**

```
ValidAt time.Time `json:"validAt,omitempty"`
```

**New option:**

```
event.WithValidAt(t)
```

**New Store method:**

```
LoadToValidTime(ctx, aggType, aggID, maxTime) → []Event
```

**New Repository method:**

```
LoadAsOf(ctx, root, validTime) → error
```

### Phase 4: Advanced (As Needed)

- Temporal projections (versioned read models)
- Speculative application (dry-run transactions)
- Backward reads (if not done in Phase 1)

---

## Appendix: Sources

### Systems Researched

| System                   | Language   | Time Travel Features                                                                                                  |
| ------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------------- |
| EventStoreDB / KurrentDB | Any (gRPC) | Stream revision, $all stream position, backward reads, projections, bi-temporal via metadata                          |
| Marten                   | .NET       | `AggregateStreamAsync<T>(id, version: N)`, `AggregateStreamAsync<T>(id, timestamp: T)`, live/inline/async projections |
| Axon Framework           | Java       | `readEvents(id, firstSeqNum)`, `DomainEventStream`, `TrackingEventProcessor` replay, snapshots                        |
| Datomic                  | Clojure    | `d/as-of`, `d/since`, `d/history`, `d/with` (speculative), database-as-value, EAVT/AVET/AEVT/VAET indexes             |
| Eventuous                | Go         | `ReadEvents(ctx, stream, startVersion, count)`, `ReadEventsBackwards`, `GlobalPosition`, `LoadState` with fold        |
| Rails EventStore         | Ruby       | `as_at`/`as_of` read scopes, `valid_at` metadata, bi-temporal                                                         |
| EventSourcingDB          | Any (HTTP) | `upperBound` parameter, subject-based scoping, time-bucketed aggregations                                             |
| Kafka                    | Any        | Offset-based reads, timestamp-based offset lookup, ⚠️ log compaction incompatible with ES                             |

### Key References

- Martin Fowler, "Event Sourcing" — https://martinfowler.com/eaaDev/EventSourcing.html
- Greg Young, "Event Store as a Time Machine" — various talks and writings
- Datomic documentation — https://docs.datomic.com/
- EventStoreDB / KurrentDB documentation — https://docs.kurrent.io/
- Marten documentation — https://martendb.io/
- Eventuous documentation — https://eventuous.dev/
- Rails EventStore bi-temporal docs — https://railseventstore.org/docs/advanced-topics/bi-temporal/
- EventSourcingDB temporal query patterns — https://docs.eventsourcingdb.io/best-practices/patterns-for-temporal-queries/
- planetgeek.ch, "Our Experience with Bi-Temporal Event Sourcing" — https://www.planetgeek.ch/2023/12/04/our-experience-with-bi-temporal-event-sourcing/

---

_Research synthesized from official documentation, source code, blog posts, and community resources across 8 event sourcing systems and the broader CQRS/ES community._
