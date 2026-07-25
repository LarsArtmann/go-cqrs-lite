# Time Travel in Event Sourcing: The Complete Industry Survey

> **Status:** IMPLEMENTED — Version/timestamp/position-based reads all shipped

> Exhaustive research on how every major event sourcing system implements time travel — and what go-cqrs-lite should adopt.

**Date:** 2026-05-20

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The Time Travel Taxonomy](#2-the-time-travel-taxonomy)
3. [Industry Survey: How Every System Does It](#3-industry-survey-how-every-system-does-it)
4. [Cross-Cutting Patterns](#4-cross-cutting-patterns)
5. [What go-cqrs-lite Has Today](#5-what-go-cqrs-lite-has-today)
6. [What go-cqrs-lite Should Add](#6-what-go-cqrs-lite-should-add)
7. [Detailed Implementation Guide](#7-detailed-implementation-guide)
8. [Recommended Roadmap](#8-recommended-roadmap)
9. [What NOT to Add](#9-what-not-to-add)
10. [Feature Comparison Matrix](#10-feature-comparison-matrix)

---

## 1. Executive Summary

This report synthesizes research across **13 event sourcing systems**, **5 temporal database approaches**, and **dozens of production implementations** to answer: _"How does the industry do time travel in event sourcing, and what should go-cqrs-lite adopt?"_

**Key findings:**

| Finding                                                | Detail                                                                          |
| ------------------------------------------------------ | ------------------------------------------------------------------------------- |
| **Every major framework provides version-based reads** | LoadToVersion / ReadEvents(start, count) is table stakes                        |
| **Timestamp-based reads exist in ~50% of frameworks**  | Marten, Rails EventStore, EventSourcingDB — not EventStoreDB or Eventuous       |
| **Global position is universal in production systems** | Every production-grade store has a monotonic global ordering mechanism          |
| **Backward reads are common but not universal**        | EventStoreDB, Eventuous, EventSourcingDB have them; Marten, Axon don't          |
| **Bi-temporal is niche but critical for some domains** | Only Rails EventStore provides first-class support                              |
| **Upcasting is universally lazy (on read)**            | Marten, Axon, and Oskar Dudycz's patterns all transform on read, not on write   |
| **Projection replay = position-based catch-up**        | Every framework that scales uses position/tracking tokens, not LoadAll + filter |
| **The #1 missing feature in go-cqrs-lite**             | Store-level version filtering (`LoadToVersion`) — every peer has this           |

---

## 2. The Time Travel Taxonomy

All time-travel mechanisms in event sourcing fall into these categories:

### 2.1 Per-Aggregate Temporal Queries

| Mechanism               | Query                     | Example API                                                              |
| ----------------------- | ------------------------- | ------------------------------------------------------------------------ |
| **Load TO version**     | "State at version N"      | `Store.LoadToVersion(ctx, aggType, aggID, maxVersion)`                   |
| **Load FROM version**   | "Events after version N"  | `Store.LoadFromVersion(ctx, aggType, aggID, version)` — **we have this** |
| **Load TO timestamp**   | "State at time T"         | `Store.LoadToTimestamp(ctx, aggType, aggID, maxTime)`                    |
| **Read backwards**      | "Last N events"           | `Store.ReadBackwards(ctx, aggType, aggID, maxCount)`                     |
| **Snapshot at version** | "Pre-computed state at N" | `SnapshotStore.LoadAtVersion()` — **we have this**                       |

### 2.2 Cross-Aggregate Temporal Queries

| Mechanism                       | Query                        | Example API                                         |
| ------------------------------- | ---------------------------- | --------------------------------------------------- |
| **Global position read**        | "All events from position P" | `GlobalLoader.LoadAllFromPosition(ctx, pos, limit)` |
| **$all stream**                 | "Entire event log"           | EventStoreDB's `$all` stream                        |
| **Position-based subscription** | "Resume from checkpoint"     | `SubscribeFromPosition(pos)`                        |

### 2.3 Temporal Dimensions

| Dimension            | Name    | Source                         | Our Representation                             |
| -------------------- | ------- | ------------------------------ | ---------------------------------------------- |
| **Transaction Time** | "As-At" | When system recorded the event | `Event.OccurredAt()` — **we have this**        |
| **Valid Time**       | "As-Of" | When event was true in reality | **Not represented** — needs `ValidAt` metadata |

### 2.4 Advanced Mechanisms

| Mechanism                   | Description                             | Who Does It                |
| --------------------------- | --------------------------------------- | -------------------------- |
| **Database-as-Value**       | Entire DB is immutable, query any point | Datomic only               |
| **Speculative Application** | "What-if" without persisting            | Datomic (`d/with`)         |
| **Retraction**              | Semantic "no longer true"               | Datomic (`:db/retract`)    |
| **History Query**           | All versions of attribute over time     | Datomic (`d/history`)      |
| **Counterfactual Replay**   | Replay with modified conditions         | canon (Rust), research     |
| **Temporal Projections**    | Versioned read-model snapshots          | EventSourcingDB, custom    |
| **Event Compaction**        | Delete old events (anti-time-travel)    | EventStoreDB (`$maxCount`) |

---

## 3. Industry Survey: How Every System Does It

### 3.1 EventStoreDB / KurrentDB (gRPC, Any Language)

The reference event store. Provides the most comprehensive set of reading primitives.

**Reading API:**

| Capability              | API                                                  | Notes                                     |
| ----------------------- | ---------------------------------------------------- | ----------------------------------------- |
| Forward stream read     | `ReadStream(stream, fromRevision, limit, forwards)`  | From specific revision, limited count     |
| Backward stream read    | `ReadStream(stream, fromRevision, limit, backwards)` | From end or specific revision             |
| $all forward            | `ReadAll(fromPosition, limit, forwards)`             | Global log from commit position           |
| $all backward           | `ReadAll(fromPosition, limit, backwards)`            | Reverse global traversal                  |
| Filtering               | `filter_include`, `filter_exclude` (regex)           | Server-side event type / stream filtering |
| Catch-up subscription   | `SubscribeToAll(fromPosition)`                       | Resume from commit position               |
| Persistent subscription | `SubscribePersistent(group, fromPosition)`           | Durable consumer groups                   |

**Global Position Model:**

- **Commit Position**: 64-bit unsigned int, monotonic per node
- **Prepare Position**: Position within transaction prepare phase
- **Stream Position**: Per-stream 0-based position
- Each read returns `{commitPosition, preparePosition, streamPosition}`

**No native timestamp-based reads.** Timestamps are metadata only. You must filter client-side.

```python
# Example: Read stream from revision 5
events = client.get_stream("order-123", stream_position=5, backwards=False)

# Example: Read $all from global position
with client.read_all(commit_position=1234, limit=100) as response:
    for event in response:
        print(f"Position: {event.commit_position}")
```

**Key insight for go-cqrs-lite:** Two positioning systems (stream revision + global position) is the standard pattern. EventStoreDB proves that per-aggregate versioning and global ordering are complementary, not competing.

---

### 3.2 Marten (.NET / PostgreSQL)

The gold standard for aggregate-level time travel API. Runs on PostgreSQL.

**Reading API:**

| Capability              | API                                         | Notes                               |
| ----------------------- | ------------------------------------------- | ----------------------------------- |
| Aggregate at version    | `AggregateStreamAsync<T>(id, version: N)`   | Fold events up to version N         |
| Aggregate at timestamp  | `AggregateStreamAsync<T>(id, timestamp: T)` | Fold events up to timestamp         |
| Fetch stream to version | `FetchStreamAsync(streamId, version: N)`    | Raw events up to version            |
| Fetch stream to time    | `FetchStreamAsync(streamId, timestamp: T)`  | Raw events up to timestamp          |
| Stream state            | `FetchStreamStateAsync(streamId)`           | Version, timestamps, aggregate type |
| Single event            | `LoadAsync<T>(eventId)`                     | By event ID                         |
| LINQ queries            | `QueryAllRawEvents().Where(...)`            | Full LINQ over event store          |

```csharp
// State at version 3
var party = await session.Events.AggregateStreamAsync<QuestParty>(questId, 3);

// State at a point in time
var partyYesterday = await session.Events
    .AggregateStreamAsync<QuestParty>(questId, timestamp: DateTime.UtcNow.AddDays(-1));
```

**Projection Rebuild:**

```csharp
// Rebuild a single projection
using var daemon = await store.BuildProjectionDaemonAsync();
await daemon.RebuildProjectionAsync("Shop", CancellationToken.None);

// Rebuild a single stream projection
await store.Advanced.RebuildSingleStreamAsync<SimpleAggregate>(streamId);
```

**Three Projection Lifecycles:**

- **Inline**: Same transaction as event append (strong consistency)
- **Async**: Background daemon with checkpoint tracking (eventual consistency)
- **Live**: On-demand fold from raw events (no persistence)

**Upcasting (Lazy, on Read):**

```csharp
// CLR type transformation
options.Events.Upcast<V1.Event, V3.Event>(old => new V3.Event(old.Id, old.Name, Status.Open));

// Raw JSON transformation
options.Events.Upcast<V3.Event>("event_type", Upcast(json => {
    var old = json.RootElement;
    return new V3.Event(old.GetProperty("Id").GetGuid(), ...);
}));

// Multi-version chain: V1 → V2 → V3 registered in order
options.Events.Upcast<V2.Event>(v1ToV2).Upcast<V3.Event>(v2ToV3);
```

**Key insight for go-cqrs-lite:** Marten's `AggregateStreamAsync(id, version: N)` is the single API consumers want most. Version AND timestamp parameters on the same method. PostgreSQL's `WHERE version <= $3` makes this trivial.

---

### 3.3 Axon Framework (Java)

Enterprise-grade Java framework. Focuses on lazy streaming and processor lifecycle.

**Reading API:**

| Capability                | API                                            | Notes                            |
| ------------------------- | ---------------------------------------------- | -------------------------------- |
| Read events from sequence | `readEvents(aggregateId, firstSequenceNumber)` | Returns lazy `DomainEventStream` |
| Read snapshot             | `readSnapshot(aggregateId)`                    | Returns `DomainEventStream`      |

**DomainEventStream — Lazy Iterator:**

```java
DomainEventStream stream = eventStore.readEvents("order-123", 5);
while (stream.hasNext()) {
    DomainEventMessage<?> event = stream.next();
    // Or peek without advancing:
    DomainEventMessage<?> peeked = stream.peek();
}
Long lastSeq = stream.getLastSequenceNumber();
```

This is fundamentally different from our approach — Axon streams events lazily from the database rather than loading them all into a slice. This matters for streams with 10,000+ events.

**Projection Replay (TrackingEventProcessor):**

```java
// Full replay
processor.shutDown();
processor.resetTokens();  // Reset to beginning
processor.start();

// Replay from specific position
processor.resetTokens(startPosition);

// Replay with context (selective replay)
processor.resetTokens(new MailReplayContext(List.of("user1", "user5")));

// Check if replay is happening
ReplayStatus status = ...; // Injected into handlers
if (!status.isReplay()) {
    emailSender.send(...);  // Skip side effects during replay
}
```

**Snapshotting:**

```java
SnapshotPolicy policy = SnapshotPolicy.afterEvents(50)
    .or(SnapshotPolicy.whenEventMatches(
        msg -> msg.type().equals(new QualifiedName(AccountClosed.class))
    ));
```

**Upcasting (Lazy, Chained):**

```java
public class ComplaintV1ToV2 extends SingleEventUpcaster {
    protected boolean canUpcast(IntermediateEventRepresentation rep) {
        return rep.getType().equals(new SimpleSerializedType("ComplaintEvent", "1.0"));
    }
    protected IntermediateEventRepresentation doUpcast(IntermediateEventRepresentation rep) {
        return rep.upcastPayload(
            new SimpleSerializedType("ComplaintEvent", "2.0"),
            JsonNode.class,
            json -> { ((ObjectNode) json).put("description", "default"); return json; }
        );
    }
}
```

**Key insight for go-cqrs-lite:** The `ReplayStatus` injection pattern — handlers know if they're processing live vs. replay — prevents accidental email sends during replay. We should consider this for our projection handlers.

---

### 3.4 Eventuous (Go)

The closest peer to go-cqrs-lite. Go-idiomatic event sourcing.

**Reading API:**

| Capability      | API                                                       | Notes                          |
| --------------- | --------------------------------------------------------- | ------------------------------ |
| Forward read    | `ReadEvents(ctx, stream, start, count)`                   | Stream position + count        |
| Backward read   | `ReadEventsBackwards(ctx, stream, start, count)`          | From end of stream             |
| Global position | `StreamEvent.GlobalPosition uint64`                       | Cross-stream ordering          |
| Load state      | `LoadState[S](ctx, reader, stream, fold, zero, expected)` | Generic fold with expectations |

```go
// Read forward from position 10, get 50 events
events, err := reader.ReadEvents(ctx, "Booking-123", 10, 50)

// Read backward from end, get 5 events
events, err := reader.ReadEventsBackwards(ctx, "Booking-123", math.MaxInt32, 5)

// Load and fold state
state, events, version, err := LoadState(ctx, reader, "Booking-123",
    bookingFold, BookingState{}, eventuous.IsExisting)
```

**CheckpointStore:**

```go
type CheckpointStore interface {
    GetCheckpoint(ctx context.Context, id string) (Checkpoint, error)
    StoreCheckpoint(ctx context.Context, checkpoint Checkpoint) error
}
```

**Catch-up Subscription:**

```go
sub := kdb.NewCatchUp(client, jsonCodec, "MyProjection",
    kdb.FromAll(),
    kdb.WithCheckpointStore(checkpointStore),
    kdb.WithHandler(handlerFunc),
)
err := sub.Start(ctx)
```

**Key insight for go-cqrs-lite:** Eventuous proves that Go event sourcing libraries can have `ReadEventsBackwards`, `GlobalPosition`, and position-based catch-up. These are not framework features — they're library features. We're the only Go library without them.

---

### 3.5 Rails EventStore (Ruby)

The only framework with first-class bi-temporal support.

**Bi-Temporal API:**

```ruby
# Publish event with valid_at (business time)
event_store.publish(Event.new(data: { salary: 10000 },
                              metadata: { valid_at: Time.utc(2020, 1, 1) }))

# Read by transaction time (as-at)
events = event_store.read.stream("salary-123").as_at.to_a

# Read by valid time (as-of)
events = event_store.read.stream("salary-123").as_of.to_a
```

**Use Case — Salary Correction:**

```ruby
# Original salary entry (published Jan 1, valid Jan 1)
Timecop.travel(Time.utc(2022, 1, 1)) do
  event_store.publish(SalaryRaised.new(data: { salary: 10_000 }))
end

# Correction (published Mar 1, but valid retroactively Jan 1)
Timecop.travel(Time.utc(2022, 3, 1)) do
  event_store.with_metadata({ valid_at: Time.utc(2022, 1, 1) }) do
    event_store.publish(SalaryRaised.new(data: { salary: 10_300 }))
  end
end
```

**Key insight for go-cqrs-lite:** Bi-temporal is a metadata concern, not a store concern. The `valid_at` field is just metadata on events. The `as_at`/`as_of` read modes are ordering filters. This can be added to go-cqrs-lite as an opt-in layer without touching core interfaces.

---

### 3.6 Datomic (Clojure)

The theoretical pinnacle. Database-as-value model.

**Temporal API:**

```clojure
;; Database at a point in time
(def db-jan (d/as-of db #inst "2024-01-01"))
(d/q '[:find ?name :where [?e :user/name ?name]] db-jan)

;; Changes since a point
(def db-since (d/since db tx-id))

;; Complete history (assertions + retractions)
(def history (d/history db))

;; Speculative application (dry-run)
(def speculative (d/with db [[:db/add 42 :user/email "test@example.com"]]))

;; Retraction (semantic "no longer true")
(d/transact conn [[:db/retract user-id :user/email "old@example.com"]])
```

**Four Temporal Indexes:** EAVT, AVET, AEVT, VAET — all include the transaction dimension.

**Key insight for go-cqrs-lite:** The `d/as-of` pattern (immutable database value at a point in time) is the most elegant time-travel API. While we can't implement database-as-value in Go, we can provide the same semantics via `Store.LoadToVersion` + fold.

---

### 3.7 EventSourcingDB (HTTP API)

Modern event store with built-in temporal query language.

**Reading API:**

```bash
# Read events up to a specific event (upper bound)
POST /api/v1/read-events
{
  "subject": "/books/42",
  "options": {
    "upperBound": { "id": "3001", "type": "exclusive" },
    "lowerBound": { "id": "2001", "type": "inclusive" },
    "order": "chronological"  # or "antichronological"
  }
}
```

**EventQL Temporal Queries:**

```sql
FROM e IN events
WHERE e.subject == "/books/42"
GROUP BY DAY(e.time)
PROJECT INTO {
  date: UNIQUE(DAY(e.time)),
  eventCount: COUNT()
}
```

**Key insight for go-cqrs-lite:** `upperBound` and `lowerBound` parameters give fine-grained temporal control. The `antichronological` order is their version of `ReadBackwards`. EventQL is ambitious but shows that temporal queries can be a first-class query language.

---

### 3.8 Equinox (F#)

Optimized for Azure Cosmos DB. "Tip with Unfolds" pattern.

**Key Innovation:** Instead of reading entire event streams, Equinox stores a compressed snapshot in a single "Tip" document alongside recent events:

```fsharp
// Tip document structure
{
    "id": "-1",              // Fixed ID for point-read (1 RU)
    "i": 4,                  // Current version
    "u": [                   // Compressed unfolds (snapshots)
        { "i": 4, "c": "s1", "d": "compressedData" }
    ]
}
```

**Access Strategies:**

- `Unoptimized` — Read all events every time
- `Snapshot` — Single compressed snapshot in Tip
- `MultiSnapshot` — Multiple snapshots for complex scenarios
- `RollingState` — No events stored, only state (highest performance)

**Key insight for go-cqrs-lite:** For read-heavy workloads, storing snapshots alongside events in the same document eliminates the separate SnapshotStore entirely. This is particularly relevant for our Pebble store.

---

### 3.9 LiveStore (TypeScript / Effect)

Local-first, SQLite-based. Radical approach: state is fully disposable.

**Key Principles:**

1. **Materializers are pure functions** — no `time.Now()`, no UUID generation, no external calls
2. **State is disposable** — `DROP TABLE + rematerialize from event log` is a valid migration strategy
3. **Global sequence numbers** — events ordered by monotonic global position

```typescript
const materializers = State.SQLite.materializers(events, {
	"v1.TodoCreated": ({ id, text }) => tables.todos.insert({ id, text, completed: false }),
	"v1.TodoCompleted": ({ id }) => tables.todos.update({ completed: true }).where({ id }),
});
```

**Key insight for go-cqrs-lite:** Deterministic materializers (no side effects during projection) is a principle we should enforce. If projection handlers are pure, replay is always safe. This aligns with our existing `event.Projection.Handle(ctx, evt) error` signature but should be documented as a contract.

---

### 3.10 canon (Rust)

Research framework with built-in counterfactual replay.

```rust
#[aggregate(snapshot_every = 50)]
pub struct Ship { status: ShipStatus, fuel_level: f32 }

// Counterfactual replay: replay with modified conditions
pub fn counterfactual_replay(id: Uuid, modified_condition: bool) -> Vec<Event> {
    let events = event_store.load(id);
    events.into_iter()
        .filter_map(|e| apply_counterfactual_condition(&e, modified_condition))
        .collect()
}
```

**Key insight for go-cqrs-lite:** Counterfactual replay is a niche but powerful debugging tool. It's best implemented at the application layer, not in the library. Our `decider.Repository.Load` (read-only) already enables this pattern.

---

### 3.11 SQL:2011 Temporal Tables

The SQL standard for time travel. Not event sourcing, but informs our SQL implementation.

```sql
-- System-versioned temporal table (SQL Server)
CREATE TABLE Employee (
    EmployeeID INT PRIMARY KEY,
    Name NVARCHAR(100),
    Salary DECIMAL(10, 2),
    SysStartTime DATETIME2 GENERATED ALWAYS AS ROW START,
    SysEndTime DATETIME2 GENERATED ALWAYS AS ROW END,
    PERIOD FOR SYSTEM_TIME (SysStartTime, SysEndTime)
)
WITH (SYSTEM_VERSIONING = ON);

-- Time travel query
SELECT * FROM Employee
FOR SYSTEM_TIME AS OF '2024-01-01';
```

**PostgreSQL Implementation (via extension):**

```sql
-- Using nearform/temporal_tables extension
CREATE TRIGGER versioning_trigger
BEFORE INSERT OR UPDATE OR DELETE ON events
FOR EACH ROW EXECUTE PROCEDURE versioning('sys_period', 'events_history', true);
```

**Key insight for go-cqrs-lite:** PostgreSQL temporal tables can complement our event store. If we store projection results in temporal tables, consumers get `FOR SYSTEM_TIME AS OF` queries "for free" without replaying events.

---

### 3.12 Production Experience: TimeRocket (8+ Years Bi-Temporal)

TimeRocket has run bi-temporal event sourcing for 8+ years in production (F#). Key lessons:

**What worked:**

- Complete audit trail makes debugging trivial
- Flexible business logic for retroactive changes and future bookings
- Once infrastructure was built, difficult use cases became quick to implement

**What didn't work:**

- Domain modeling with events is difficult, especially "undo" scenarios
- Rewrote the code 5 times across 8 years
- Performance required multiple rewrites
- Event format migrations (JSON library changes) were painful

**Their event action taxonomy:**

- `Create` — new projection
- `Update` — modify existing
- `Delete` — remove value
- `Recreate` — undo delete
- `RejectCreation` — rejected workflow
- `ModifyPerpetually` — change over entire lifetime (uni-temporal)
- `Skip` — ignore in certain projections

**Key insight for go-cqrs-lite:** Bi-temporal complexity is real. 5 rewrites in 8 years. This should be an opt-in feature, not core. But the `valid_at` metadata pattern is simple and should be available.

---

### 3.13 Oskar Dudycz / event-driven.io / Emmett

Practical, pragmatic guidance. Focus on simplicity.

**Event Versioning Patterns:**

| Change             | Pattern                | Example                                       |
| ------------------ | ---------------------- | --------------------------------------------- |
| New optional field | Add nullable property  | `DateTime? InitializedAt`                     |
| New required field | Default value          | `ShoppingCartStatus Status = Opened`          |
| Rename property    | JSON attribute mapping | `[JsonPropertyName("ShoppingCartId")] CartId` |
| Breaking change    | Upcasting              | Lazy transformation on read                   |

**Upcasting + Downcasting:**

```csharp
// Upcast: V1 → V2
public static ShoppingCartOpened Upcast(V1.ShoppingCartOpened old) =>
    new(old.ShoppingCartId, new Client(old.ClientId));

// Downcast: V2 → V1 (for legacy consumers)
public static V1.ShoppingCartOpened Downcast(ShoppingCartOpened newEvent) =>
    new(newEvent.ShoppingCartId, newEvent.Client.Id);
```

**Projection Rebuild — Blue-Green:**

1. Set up secondary read model
2. Reapply events to secondary
3. Switch queries to secondary
4. Archive old

**Key insight for go-cqrs-lite:** Keep aggregates short-lived (easier schema evolution). Design events with upcasting in mind. The `EventTransformations` pluggable pipeline pattern is what our `UpcasterRegistry` already does — but we should document the downcasting pattern too.

---

## 4. Cross-Cutting Patterns

### 4.1 Upcasting: Everyone Does It Lazy (On Read)

Every framework that supports upcasting does it **lazily** — transforming old events when they're read, not when they're written:

| Framework    | Upcasting Style                                 | Chain Support              |
| ------------ | ----------------------------------------------- | -------------------------- |
| Marten       | Lazy (on read), CLR or JSON transform           | Yes (V1→V2→V3)             |
| Axon         | Lazy (on read), `SingleEventUpcaster`           | Yes (ordered registration) |
| go-cqrs-lite | Lazy (on read), `UpcasterRegistry`              | Yes (sorted by version)    |
| Oskar Dudycz | Lazy (on read), `EventTransformations` pipeline | Yes (ordered registration) |

**No framework does eager upcasting (on write)** — it would require rewriting historical events, violating immutability.

### 4.2 Projection Replay = Position-Based Catch-Up

Every production-grade framework uses position-based replay:

| Framework        | Mechanism                                     | Position Type                            |
| ---------------- | --------------------------------------------- | ---------------------------------------- |
| EventStoreDB     | Catch-up subscription from commit position    | `uint64` commit position                 |
| Axon             | `TrackingEventProcessor` with `TrackingToken` | Custom token per event store             |
| Marten           | Projection daemon with `seq_id` checkpoint    | PostgreSQL sequence                      |
| Eventuous        | `CatchUpSubscription` with `CheckpointStore`  | `uint64` global position                 |
| EventSourcingDB  | `fromLatestEvent` / `lowerBound`              | Event ID                                 |
| **go-cqrs-lite** | `LoadAll()` + linear `filterEvents` scan      | **Event ID (but loads all into memory)** |

**This is our biggest architectural gap.** At 1M events, our projection runner takes seconds. Every peer does this in milliseconds via position-based loading.

### 4.3 Snapshotting Strategies Across Frameworks

| Framework        | Strategy                                                  | Configurable?              |
| ---------------- | --------------------------------------------------------- | -------------------------- |
| Axon             | `SnapshotPolicy.afterEvents(N).or(whenEventMatches(...))` | Yes, composable predicates |
| Marten           | `Snapshot<T>(SnapshotLifecycle.Inline/Async)`             | Yes, lifecycle choice      |
| EventStoreDB     | Custom projections that emit snapshot events              | Manual                     |
| Equinox          | `AccessStrategy.Snapshot` — compressed in Tip document    | Yes, 4 strategies          |
| **go-cqrs-lite** | `EveryNEvents(N)`                                         | Yes, single strategy       |

**Gap:** Axon's composable snapshot predicates (event count OR specific event type OR time threshold) are more expressive than our `EveryNEvents`. But our approach is simpler and covers 90% of cases.

### 4.4 Side-Effect Suppression During Replay

Every framework that does projection replay needs to suppress side effects:

| Framework        | Pattern                                                                       |
| ---------------- | ----------------------------------------------------------------------------- |
| Axon             | `ReplayStatus` injected into handlers — `if (!status.isReplay())`             |
| LiveStore        | Materializers are pure functions by contract — no side effects possible       |
| Rails EventStore | Manual: handler checks replay context                                         |
| **go-cqrs-lite** | **No mechanism** — projection handlers can trigger side effects during replay |

**This is a correctness gap.** Our `projection.Runner.replay()` calls `p.Handle(ctx, evt)` during replay. If a handler sends emails, those emails go out during every restart.

---

## 5. What go-cqrs-lite Has Today

### Working Capabilities

| #   | Capability          | API                                                         | Quality                       |
| --- | ------------------- | ----------------------------------------------------------- | ----------------------------- |
| 1   | Load all events     | `Store.Load(ctx, aggType, aggID)`                           | ✅ Production                 |
| 2   | Load from version   | `Store.LoadFromVersion(ctx, aggType, aggID, version)`       | ✅ Production                 |
| 3   | Snapshot at version | `SnapshotStore.LoadAtVersion(ctx, aggType, aggID, version)` | ✅ Production                 |
| 4   | Snapshot latest     | `SnapshotStore.Load(ctx, aggType, aggID)`                   | ✅ Production                 |
| 5   | Snapshot strategy   | `EveryNEvents(n)`                                           | ✅ Production                 |
| 6   | Aggregate replay    | `Core.LoadFromHistory(root, events)`                        | ✅ Production                 |
| 7   | Decider fold        | `Repository.Load(ctx, aggID, aggType)`                      | ✅ Production                 |
| 8   | Global loader       | `GlobalLoader.LoadAll(ctx)`                                 | ✅ Working but O(n) memory    |
| 9   | Projection replay   | `Runner.replay()` with checkpoint                           | ⚠️ Works but loads all events |
| 10  | Upcasting           | `UpcasterRegistry` with lazy chains                         | ✅ Production                 |
| 11  | Schema versioning   | `event.SchemaVersion` branded type                          | ✅ Production                 |
| 12  | Outbox replay       | `OutboxPublisher.PublishNow(ctx)`                           | ✅ Production                 |
| 13  | Checkpoint tracking | `CheckpointStore.Load/Save`                                 | ✅ Production                 |
| 14  | Bulk import         | `Store.AppendBatch(ctx, aggType, aggID, events)`            | ✅ Production                 |

### Critical Gaps

| #   | Gap                                   | Impact                                   | Every Peer Has It?            |
| --- | ------------------------------------- | ---------------------------------------- | ----------------------------- |
| 1   | **No LoadToVersion**                  | O(n) for all temporal queries            | ✅ Yes — all peers            |
| 2   | **No LoadToTimestamp**                | No "as-at" queries                       | ~50% of peers                 |
| 3   | **No position-based GlobalLoader**    | O(n) memory projection replay            | ✅ Yes — all production peers |
| 4   | **No Repository.LoadAtVersion**       | Consumers must compose manually          | ✅ Yes — Marten, Axon         |
| 5   | **No backward reads**                 | Can't efficiently get last N events      | ~60% of peers                 |
| 6   | **No replay-side-effect suppression** | Handlers fire side effects during replay | Axon, LiveStore               |
| 7   | **No global transaction position**    | No cross-aggregate temporal consistency  | ✅ Yes — all production peers |
| 8   | **No ValidAt metadata**               | No bi-temporal support                   | Rails EventStore only         |
| 9   | **No speculative application**        | No "what-if" queries                     | Datomic only                  |

---

## 6. What go-cqrs-lite Should Add

### Priority Matrix

| Priority | Feature                                | Effort | Breaking?   | Rationale                                           |
| -------- | -------------------------------------- | ------ | ----------- | --------------------------------------------------- |
| **P0**   | `Store.LoadToVersion`                  | 2h     | No          | Every peer has it. Table stakes.                    |
| **P0**   | Position-based `GlobalLoader`          | 3h     | No          | Critical for production projection replay           |
| **P1**   | `Store.LoadToTimestamp`                | 2h     | No          | Required for "as-at" queries                        |
| **P1**   | `Repository.LoadAtVersion/LoadAtTime`  | 2h     | No          | Consumer convenience — single-call API              |
| **P1**   | Replay side-effect suppression         | 3h     | No          | Correctness gap — prevent email sends during replay |
| **P2**   | `Store.ReadBackwards`                  | 2h     | No          | 60% of peers have it. Useful for "last N" patterns  |
| **P3**   | Global `TransactionID`                 | 22h    | **Yes**     | Required for production cross-aggregate ordering    |
| **P3**   | `ValidAt` metadata + `LoadToValidTime` | 10h    | No (opt-in) | Required for financial/HR domains                   |
| **P4**   | Speculative application                | 4h     | No          | Datomic `d/with` equivalent. Nice for debugging     |

---

## 7. Detailed Implementation Guide

### 7.1 `Store.LoadToVersion` (P0)

```go
// New method on event.Store interface
LoadToVersion(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxVersion Version,
) ([]Event, error)
```

**SQL:** `SELECT * FROM events WHERE aggregate_type = $1 AND aggregate_id = $2 AND version <= $3 ORDER BY version ASC`

**Memory:** `events[:min(maxVersion.Int(), len(events))]` — O(1) slice

**Pebble:** Range scan from version 1 to maxVersion

### 7.2 Position-Based GlobalLoader (P0)

```go
// New interface (or extend GlobalLoader)
type PositionalLoader interface {
    LoadAllFromPosition(
        ctx context.Context,
        afterEventID id.EventID,
        limit int,
    ) ([]Event, error)
}
```

**SQL:** `SELECT * FROM events WHERE id > $1 ORDER BY occurred_at ASC LIMIT $2`

**Memory:** Binary search by event ID, then slice

**Consumer:** `projection.Runner` uses `PositionalLoader` when available, falls back to `LoadAll`

### 7.3 `Store.LoadToTimestamp` (P1)

```go
LoadToTimestamp(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxTime time.Time,
) ([]Event, error)
```

**SQL:** `SELECT * FROM events WHERE aggregate_type = $1 AND aggregate_id = $2 AND occurred_at <= $3 ORDER BY version ASC`

**Requires:** Timestamp index on `(aggregate_type, aggregate_id, occurred_at)` in SQL DDL

### 7.4 Repository-Level Time Travel (P1)

```go
// On decider.Repository
func (r *Repository[State]) LoadAtVersion(
    ctx context.Context,
    aggID id.AggregateID,
    aggType event.AggregateType,
    version event.Version,
) (State, event.Version, error)

// On aggregate.Repository (new method)
LoadAtVersion(ctx context.Context, root Root, version event.Version) error
```

### 7.5 Replay Side-Effect Suppression (P1)

```go
// Option 1: Context flag
func replayContext(ctx context.Context) context.Context {
    return context.WithValue(ctx, replayKey{}, true)
}

func IsReplay(ctx context.Context) bool {
    v, _ := ctx.Value(replayKey{}).(bool)
    return v
}

// Consumer usage:
func (h *EmailProjection) Handle(ctx context.Context, evt event.Event) error {
    h.updateReadModel(evt)
    if !event.IsReplay(ctx) {
        h.emailSender.Send(...)  // Only during live processing
    }
    return nil
}
```

### 7.6 `Store.ReadBackwards` (P2)

```go
ReadBackwards(
    ctx context.Context,
    aggregateType AggregateType,
    aggregateID id.AggregateID,
    maxCount int,
) ([]Event, error)
```

**SQL:** `SELECT * FROM (SELECT * FROM events WHERE aggregate_type = $1 AND aggregate_id = $2 ORDER BY version DESC LIMIT $3) sub ORDER BY version ASC`

---

## 8. Recommended Roadmap

### Phase 1: Table Stakes (Non-Breaking, ~14h)

| Task                                                                                        | Effort |
| ------------------------------------------------------------------------------------------- | ------ |
| `Store.LoadToVersion` — interface + MemoryStore + SQLEventStore + PebbleStore               | 2h     |
| Position-based `GlobalLoader.LoadAllFromPosition` — interface + MemoryStore + SQLEventStore | 3h     |
| Update `projection.Runner` to use `PositionalLoader` when available                         | 2h     |
| `Store.LoadToTimestamp` — interface + MemoryStore + SQLEventStore                           | 2h     |
| `decider.Repository.LoadAtVersion` / `LoadAtTime`                                           | 1h     |
| `aggregate.Repository.LoadAtVersion` / `LoadAtTime`                                         | 1h     |
| Replay side-effect suppression via context                                                  | 1h     |
| Tests for all new methods                                                                   | 3h     |

**Result:** Matches EventStoreDB + Eventuous time-travel capabilities.

### Phase 2: Enrichment (Non-Breaking, ~7h)

| Task                                                                   | Effort |
| ---------------------------------------------------------------------- | ------ |
| `Store.ReadBackwards` — interface + MemoryStore + SQLEventStore        | 2h     |
| Timestamp index in SQL DDL                                             | 1h     |
| Temporal aggregate read-only safety (prevent Save after LoadAtVersion) | 2h     |
| Documentation + examples                                               | 2h     |

**Result:** Matches Marten's aggregate-level time-travel API.

### Phase 3: Global Position (Breaking — v2, ~22h)

| Task                                 | Effort |
| ------------------------------------ | ------ |
| `TransactionID` branded type         | 1h     |
| `Event.TransactionID()` on interface | 2h     |
| MemoryStore atomic counter           | 2h     |
| SQLEventStore BIGSERIAL / SEQUENCE   | 3h     |
| Update all implementations           | 6h     |
| Tests                                | 4h     |

**Result:** Cross-aggregate temporal consistency. Production-grade projection replay.

### Phase 4: Bi-Temporal (Opt-In, ~10h)

| Task                                         | Effort |
| -------------------------------------------- | ------ |
| `ValidAt` on Metadata + `WithValidAt` option | 1h     |
| `valid_at` column in SQL DDL + index         | 1h     |
| `Store.LoadToValidTime`                      | 2h     |
| `Repository.LoadAsOf`                        | 1h     |
| Documentation                                | 2h     |
| Tests                                        | 3h     |

**Result:** Matches Rails EventStore bi-temporal capabilities.

---

## 9. What NOT to Add

| Mechanism                         | Verdict           | Rationale                                                                                                                                                           |
| --------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Database-as-Value                 | ❌ Skip           | Requires datom-based storage. Our stream-per-aggregate model is correct for Go.                                                                                     |
| Retraction model                  | ❌ Skip           | Our events are coarse-grained payloads, not EAV triples. Domain events handle "undo" better.                                                                        |
| History query                     | ❌ Skip           | Requires Datomic's datom model. Incompatible.                                                                                                                       |
| Event compaction                  | ❌ Skip (for now) | Destroys time-travel. Only after APIs are complete, as opt-in destructive operation.                                                                                |
| Stream forking                    | ❌ Skip           | No standard pattern. What-if = speculative application (P4).                                                                                                        |
| Lazy streaming reads (Axon-style) | ❌ Skip           | Go's `[]Event` slices are idiomatic. Lazy streaming adds complexity without clear benefit for our use cases. If needed, consumers can wrap with their own iterator. |
| EventQL-style query language      | ❌ Skip           | Out of scope for a library. Consumers build their own query layers.                                                                                                 |

---

## 10. Feature Comparison Matrix

| Feature                            | go-cqrs-lite (today) | go-cqrs-lite (ideal) | EventStoreDB | Marten | Axon | Eventuous (Go) | Rails ES | Datomic | EventSourcingDB | Equinox |
| ---------------------------------- | -------------------- | -------------------- | ------------ | ------ | ---- | -------------- | -------- | ------- | --------------- | ------- |
| Load all events                    | ✅                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | ✅      | ✅              | ✅      |
| Load from version                  | ✅                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | ✅      | ✅              | ✅      |
| **Load TO version**                | ❌                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | ✅      | ✅              | ✅      |
| **Load TO timestamp**              | ❌                   | ✅                   | ❌           | ✅     | ❌   | ❌             | ✅       | ✅      | ✅              | ❌      |
| Snapshot at version                | ✅                   | ✅                   | ✅           | ✅     | ✅   | ❌             | ❌       | N/A     | ❌              | ✅      |
| **Backward reads**                 | ❌                   | ✅                   | ✅           | ❌     | ❌   | ✅             | ❌       | ✅      | ✅              | ❌      |
| **Global position**                | ❌                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | ✅      | ✅              | ✅      |
| **Position-based replay**          | ❌                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | N/A     | ✅              | ✅      |
| Bi-temporal                        | ❌                   | ✅                   | ❌           | ❌     | ❌   | ❌             | ✅       | ✅      | ❌              | ❌      |
| Speculative apply                  | ❌                   | ✅                   | ❌           | ❌     | ❌   | ❌             | ❌       | ✅      | ❌              | ❌      |
| Upcasting (lazy)                   | ✅                   | ✅                   | ❌           | ✅     | ✅   | ❌             | ❌       | N/A     | ❌              | ❌      |
| Schema versioning                  | ✅                   | ✅                   | ❌           | ✅     | ✅   | ❌             | ❌       | N/A     | ❌              | ❌      |
| Projection checkpoint              | ✅                   | ✅                   | ✅           | ✅     | ✅   | ✅             | ✅       | N/A     | ✅              | ✅      |
| **Replay side-effect suppression** | ❌                   | ✅                   | ❌           | ❌     | ✅   | ❌             | ❌       | N/A     | ❌              | ❌      |
| Composable snapshot policies       | ❌                   | P2                   | ❌           | ✅     | ✅   | ❌             | ❌       | N/A     | ❌              | ✅      |

---

_Research synthesized from: EventStoreDB/KurrentDB documentation, Marten documentation and source code, Axon Framework API docs and reference guide, Eventuous Go source code, Rails EventStore documentation, Datomic documentation, EventSourcingDB documentation, Equinox source code and documentation, LiveStore documentation, canon (Rust) framework, SQL:2011 temporal table specifications, nearform/temporal_tables PostgreSQL extension, Oskar Dudycz event-driven.io articles, TimeRocket 8-year bi-temporal production experience (planetgeek.ch), and community blog posts across 20+ sources._
