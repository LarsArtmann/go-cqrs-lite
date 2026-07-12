# Hybrid Architecture: Best of Both Worlds

\n> **Status:** SUPERSEDED — informed actual implementation decisions

**Bridging Traditional Aggregate-Based ES and Aggregateless ES in go-cqrs-lite**

**Date**: May 2026
**Status**: Architectural Proposal
**Prerequisite Reading**:

- `docs/research/2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md`
- `docs/research/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The Core Insight](#2-the-core-insight)
3. [Current Architecture](#3-current-architecture)
4. [What Aggregateless Gets Right](#4-what-aggregateless-gets-right)
5. [What Aggregateless Gets Wrong](#5-what-aggregateless-gets-wrong)
6. [The Hybrid Design](#6-the-hybrid-design)
7. [New Abstractions](#7-new-abstractions)
8. [SQL Schema](#8-sql-schema)
9. [Implementation: event.Filter](#9-implementation-eventfilter)
10. [Implementation: ContextQuerier](#10-implementation-contextquerier)
11. [Implementation: ContextAppender (CTE)](#11-implementation-contextappender-cte)
12. [Implementation: ContextRepository](#12-implementation-contextrepository)
13. [Usage Patterns](#13-usage-patterns)
14. [Complete Example: Single-Entity (Traditional)](#14-complete-example-single-entity-traditional)
15. [Complete Example: Cross-Entity (Context Mode)](#15-complete-example-cross-entity-context-mode)
16. [Complete Example: The Hybrid Service](#16-complete-example-the-hybrid-service)
17. [Decision Flowchart](#17-decision-flowchart)
18. [Comparison Matrix](#18-comparison-matrix)
19. [What We Keep From Each World](#19-what-we-keep-from-each-world)
20. [Migration Strategy](#20-migration-strategy)
21. [Risks & Mitigations](#21-risks--mitigations)
22. [Open Questions](#22-open-questions)

---

## 1. Executive Summary

**One event store. Two access patterns. Zero breaking changes.**

Traditional aggregate-based event sourcing excels at single-entity operations. Aggregateless event sourcing excels at cross-entity operations. Rather than choosing one, we add aggregateless-inspired context queries as a **second access pattern** on the same event store.

The result: aggregates remain the default for 95% of commands. A new `Filter` + `Query` + `AppendContext` mechanism provides an escape hatch for the 5% of operations that span multiple entities — without sagas, process managers, or dual-writes.

This is a purely **additive** proposal. No existing interfaces change. No existing code breaks. The new capabilities live in separate interfaces that `SQLEventStore` optionally implements.

---

## 2. The Core Insight

Aggregates and context queries are not competing architectures. They are **two ways to define a consistency boundary** over the same facts:

| Boundary Type | Defined By                                       | Scope                   | Best For                           |
| ------------- | ------------------------------------------------ | ----------------------- | ---------------------------------- |
| **Stream**    | `(aggregate_type, aggregate_id)`                 | Single entity           | CRUD-like operations on one entity |
| **Context**   | `EventFilter` (event types + payload predicates) | Arbitrary set of events | Cross-entity decisions             |

Both boundaries use the same events table. Both use optimistic concurrency (stream via version, context via max sequence). Both produce the same events. They differ only in **how they scope the consistency check**.

```
Stream Boundary:  WHERE aggregate_type = $1 AND aggregate_id = $2 AND version = $3
Context Boundary: WHERE event_type = ANY($1) AND payload @> $2

                    ┌─────────────────────────────────────────┐
                    │              events table                │
                    │                                          │
   Stream reads ──► │  indexed by (type, id, version)         │
                    │                                          │
   Context reads ─►│  indexed by (event_type) + GIN(payload)  │
                    │                                          │
   Both append ──► │  INSERT INTO events (...)                │
                    │                                          │
                    └─────────────────────────────────────────┘
```

The aggregateless approach has no exclusive claim to context queries. They are simply a useful access pattern that traditional ES frameworks never exposed.

---

## 3. Current Architecture

### Interfaces

```
event.Store          Save(stream, version) / Load(stream)     ← stream boundary
event.Bus            Publish / Subscribe / SubscribeAll       ← push-based projections
event.Projection     Handle(event) / EventTypes()             ← read model building
event.SnapshotStore  Save / Load / LoadAtVersion              ← aggregate optimization
event.Outbox         Append / PollPending / Ack               ← reliable publishing
event.Codec          Encode / Decode                          ← serialization

aggregate.Root       ID / Type / Version / Apply / RecordEvent   ← aggregate state
aggregate.Repository Save(root) / Load(root) / Delete(root)     ← repository pattern
```

### Event Flow (Write Side)

```
Command Handler
    │
    ▼
aggregate.Root.RecordEvent(ctx, event)
    │
    ▼
EventSourcedRepository.Save(ctx, root)
    │
    ├──► store.Save(ctx, aggType, aggID, events, expectedVersion)
    │        │
    │        ▼
    │    BEGIN;
    │    SELECT MAX(version) WHERE (aggType, aggID)     ← consistency check
    │    INSERT INTO events (...)                        ← versioned per stream
    │    COMMIT;
    │
    ├──► bus.Publish(ctx, events...)                     ← push to subscribers
    │        or
    │    outbox.Append(ctx, events)                      ← reliable eventual publish
    │
    └──► snapshotStore.Save(ctx, snapshot)               ← optional optimization
```

### Event Flow (Read Side)

```
event.Bus.Subscribe / SubscribeAll
    │
    ▼
InMemoryRunner.Handle(ctx, event)
    │
    ├──► Projection A (eventTypes: ["OrderPlaced"])
    ├──► Projection B (eventTypes: ["OrderPlaced", "OrderShipped"])
    └──► Projection C (eventTypes: nil  ← subscribes to all)
            │
            ▼
        checkpoint.Save(ctx, projName, eventID)
```

### What's Missing

The current architecture handles single-entity operations well. What it **cannot** do:

1. **Query across aggregate streams** — `event.Store` only loads by `(aggregate_type, aggregate_id)`.
2. **Atomically write to multiple streams** — Each `Save` is scoped to one aggregate.
3. **Define consistency boundaries by event content** — Boundaries are always by aggregate identity.
4. **Make cross-entity decisions without sagas** — Requires loading multiple aggregates separately.

---

## 4. What Aggregateless Gets Right

### 4.1 Cross-Entity Consistency Without Sagas

**The problem**: An operation like "bind device to asset" touches two aggregates. Traditional approaches:

| Approach                    | Complexity             | Risk                                         |
| --------------------------- | ---------------------- | -------------------------------------------- |
| Domain service + dual-write | Low code, high risk    | Inconsistency if second write fails          |
| Saga orchestrator           | High code, medium risk | Complex state machine, compensation logic    |
| Process manager             | High code, medium risk | Additional event handlers, ordering concerns |

**The aggregateless solution**: One query that spans both entities, one CTE that checks both. No saga, no dual-write, no process manager.

### 4.2 Feature-Specific State Reconstruction

**The problem**: An aggregate loads ALL its events and reconstructs ALL its state, even when a command only needs a subset.

Consider a `BankAccount` aggregate with 50 event types and 30 fields. The `DepositMoney` command only needs to know: does the account exist? It doesn't need balance, currency, overdraft settings, frozen status, or any of the other 27 fields.

**The aggregateless solution**: Each feature slice folds only the events and state it needs:

```
DepositMoney:  fold [AccountOpened]           → {exists: bool}
WithdrawMoney: fold [Opened, Deposited, ...]  → {exists: bool, balance: int}
TransferMoney: fold [all account events]      → {exists: bool, balance: int, frozen: bool}
```

### 4.3 Transparent Consistency Boundaries

**The problem**: Aggregate boundaries are defined implicitly — you must read the code to understand which events belong to which aggregate and what invariants are enforced.

**The aggregateless solution**: The `EventFilter` is the consistency boundary. It's explicit, readable, and defined at the call site:

```go
filter := event.NewFilter(
    event.Type("DeviceRegistered"),
    event.Type("AssetRegistered"),
    event.Type("DeviceBoundToAsset"),
).Where("deviceId", deviceID).
  OrWhere("assetId", assetID)
```

You can see exactly which events are relevant. No implicit aggregate invariants.

---

## 5. What Aggregateless Gets Wrong

### 5.1 No Push-Based Projections

Aggregateless ES is pull-only. Real-time projections require additional infrastructure (polling, LISTEN/NOTIFY, CDC). go-cqrs-lite already has `event.Bus` + `event.Projection` + `InMemoryRunner` — a mature push-based projection system.

### 5.2 No Strong Typing on Identity

Aggregateless uses raw JSONB fields (`payload->>'deviceId'`). go-cqrs-lite uses branded IDs (`id.AggregateID`, `id.EventID`) with compile-time type safety. Dropping branded IDs would be a significant regression.

### 5.3 Single Table Without Compaction

All events in one table with no archival or compaction strategy. The stream-based layout in go-cqrs-lite naturally supports per-stream operations (load, delete, snapshot) and can integrate with KurrentDB-style archiving.

### 5.4 No Schema Enforcement

JSONB is schemaless. Events can contain any fields, and typos silently create phantom data. go-cqrs-lite's `Codec` interface + Go's type system enforce event schemas at compile time.

### 5.5 No Snapshotting

Each feature must implement its own snapshot strategy. The aggregateless approach has no concept of snapshots because it has no concept of aggregates. go-cqrs-lite's `SnapshotStore` + `SnapshotStrategy` provide mature snapshotting.

### 5.6 Immature Tooling

No dedicated database, admin UI, event browser, or community. go-cqrs-lite can leverage existing PostgreSQL tooling and the broader ES ecosystem.

---

## 6. The Hybrid Design

### Principle: Additive, Not Replacing

The hybrid design adds **one new capability** to the existing architecture: context-based consistency boundaries. Everything else remains unchanged.

```
┌──────────────────────────────────────────────────────────────────────┐
│                        EVENT STORE (PostgreSQL)                       │
│                                                                       │
│   ┌─────────────────────────────┐  ┌────────────────────────────┐   │
│   │      STREAM ACCESS           │  │     CONTEXT ACCESS          │   │
│   │      (existing)              │  │     (new)                   │   │
│   │                              │  │                             │   │
│   │  WHERE agg_type = $1         │  │  WHERE type = ANY($1)       │   │
│   │  AND   agg_id   = $2         │  │  AND   payload @> $2        │   │
│   │  AND   version  = $3         │  │                             │   │
│   │                              │  │  CTE check + INSERT         │   │
│   │  UNIQUE(agg_type, agg_id,    │  │                             │   │
│   │         version)             │  │  GIN(payload) index         │   │
│   │                              │  │                             │   │
│   │  Used by:                    │  │  Used by:                   │   │
│   │  • aggregate.Repository      │  │  • ContextQuerier           │   │
│   │  • event.Store.Save/Load     │  │  • ContextAppender          │   │
│   │  • SnapshotStore             │  │  • ContextRepository        │   │
│   └─────────────────────────────┘  └────────────────────────────┘   │
│                                                                       │
│   ┌─────────────────────────────────────────────────────────────┐    │
│   │                     SHARED INFRASTRUCTURE                     │    │
│   │                                                               │    │
│   │  • event.Bus (Publish/Subscribe)                             │    │
│   │  • event.Projection + InMemoryRunner                         │    │
│   │  • event.Outbox (reliable publishing)                        │    │
│   │  • event.Codec (typed serialization)                         │    │
│   │  • catalog (AsyncAPI + EventCatalog documentation)           │    │
│   │  • middleware (logging, retry, recovery, validation)         │    │
│   └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
```

### The Three New Abstractions

| Abstraction         | Interface  | Purpose                                                              |
| ------------------- | ---------- | -------------------------------------------------------------------- |
| **Filter**          | Value type | Defines which events are relevant (event types + payload predicates) |
| **ContextQuerier**  | Interface  | Queries events across streams using a Filter                         |
| **ContextAppender** | Interface  | Atomically appends events with context-based optimistic concurrency  |

These are **separate interfaces** in the `event` package. `SQLEventStore` implements both `event.Store` (existing) and the new interfaces (additive).

### What Does NOT Change

| Component              | Change Required                                 |
| ---------------------- | ----------------------------------------------- |
| `event.Store`          | **None** — stream-based Save/Load untouched     |
| `event.Bus`            | **None** — push-based subscriptions untouched   |
| `event.Projection`     | **None** — read model building untouched        |
| `event.SnapshotStore`  | **None** — snapshotting untouched               |
| `event.Outbox`         | **None** — reliable publishing untouched        |
| `event.Codec`          | **None** — serialization untouched              |
| `aggregate.Root`       | **None** — aggregate state management untouched |
| `aggregate.Repository` | **None** — repository pattern untouched         |
| `command.Dispatcher`   | **None** — command dispatch untouched           |
| `query.Dispatcher`     | **None** — query dispatch untouched             |
| `catalog`              | **None** — documentation generation untouched   |
| `middleware`           | **None** — cross-cutting concerns untouched     |
| `memory`               | **None** — test implementations untouched       |

---

## 7. New Abstractions

### 7.1 event.Filter — Typed Context Query Definition

```go
// event/filter.go

package event

// Filter defines which events are relevant for a cross-entity decision.
// It specifies event types and payload predicates that form the
// consistency boundary for context-based operations.
type Filter struct {
    eventTypes []Type
    predicates []Predicate
}

// Predicate matches events where the payload contains a specific field-value pair.
// Translated to PostgreSQL's JSONB containment: payload @> '{"field": "value"}'
type Predicate struct {
    Field string
    Value any
}

// NewFilter creates a Filter that matches events of the given types.
func NewFilter(eventTypes ...Type) *Filter {
    return &Filter{
        eventTypes: eventTypes,
    }
}

// Where adds a payload predicate. Events must contain the given field-value pair.
// Multiple Where calls are ANDed within the same group.
func (f *Filter) Where(field string, value any) *Filter {
    f.predicates = append(f.predicates, Predicate{Field: field, Value: value})
    return f
}

// EventTypes returns the event types this filter matches.
func (f *Filter) EventTypes() []Type {
    return f.eventTypes
}

// Predicates returns the payload predicates.
func (f *Filter) Predicates() []Predicate {
    return f.predicates
}
```

**Design decisions**:

- Builder pattern (`NewFilter().Where().Where()`) for readability
- `Predicate.Value` is `any` — will be marshaled to JSON for `@>` operator
- Immutable after construction (no mutation methods)
- Event types use existing `event.Type` — no new type needed

### 7.2 ContextQuerier — Cross-Stream Event Loading

```go
// event/context.go

package event

import "context"

// QueryResult contains events loaded via a context query, along with
// the maximum sequence number for optimistic concurrency checks.
type QueryResult struct {
    Events []Event
    MaxSeq int
}

// ContextQuerier loads events across aggregate streams using a Filter.
// This is the aggregateless-inspired access pattern: instead of loading
// by aggregate identity, you load by event type and payload content.
//
// Implementations should use PostgreSQL's GIN index on payload for
// efficient JSONB containment queries.
type ContextQuerier interface {
    // Query loads events matching the filter, ordered by sequence number.
    // Returns the events and the maximum sequence number for the context.
    // The MaxSeq value is used with ContextAppender for optimistic concurrency.
    Query(ctx context.Context, filter Filter) (QueryResult, error)
}
```

**Design decisions**:

- Separate interface from `Store` — existing implementations don't need to change
- Returns `QueryResult` with `MaxSeq` — needed for subsequent `AppendContext` call
- Uses existing `event.Event` type — no new event representation
- `MaxSeq` of 0 means "no matching events exist" (empty context)

### 7.3 ContextAppender — Atomic Cross-Stream Writes

```go
// event/context.go (continued)

// ContextAppender atomically appends events with context-based optimistic
// concurrency. The implementation uses a PostgreSQL CTE that checks the
// maximum sequence number for the filter context before inserting.
//
// This enables cross-entity operations without sagas or process managers:
// the CTE ensures atomicity across what would be multiple aggregate streams.
type ContextAppender interface {
    // AppendContext atomically inserts events if the context hasn't changed.
    //
    // The filter must be the same filter used in the preceding Query call.
    // expectedMaxSeq must be the MaxSeq from that Query result.
    //
    // Returns ErrContextChanged if another writer modified the context
    // between the Query and this AppendContext call. The caller should
    // retry: re-query, re-decide, re-append.
    AppendContext(
        ctx context.Context,
        filter Filter,
        events []Event,
        expectedMaxSeq int,
    ) error
}

// ErrContextChanged indicates the context was modified between query and append.
// The caller should retry the operation.
var ErrContextChanged = errors.New("context changed between query and append")
```

**Design decisions**:

- `ErrContextChanged` is a sentinel error — works with `errors.Is()`, consistent with `event.ErrVersionConflict`
- Same `event.Event` type — no special "context events"
- Events appended this way still have `aggregate_type`, `aggregate_id`, `version` fields (they live in streams too)
- The CTE check is over the _filter scope_, not individual streams

### 7.4 ContextQuerierAppender — Combined Interface

```go
// ContextQuerierAppender combines querying and appending for context-based operations.
// Convenience type for dependency injection.
type ContextQuerierAppender interface {
    ContextQuerier
    ContextAppender
}
```

### 7.5 Composite — SQLEventStore Implements Everything

```go
// SQLEventStore already implements event.Store.
// After this proposal, it also implements:

var _ event.Store = (*SQLEventStore)(nil)               // existing
var _ event.ContextQuerier = (*SQLEventStore)(nil)       // new
var _ event.ContextAppender = (*SQLEventStore)(nil)      // new
var _ event.ContextQuerierAppender = (*SQLEventStore)(nil) // new
```

The `SQLEventStore` gains two new methods (`Query`, `AppendContext`) but its existing methods are untouched.

---

## 8. SQL Schema

### Current Schema (Stream Mode)

```sql
CREATE TABLE events (
    sequence_number  BIGSERIAL PRIMARY KEY,
    id               UUID NOT NULL,
    event_type       TEXT NOT NULL,
    aggregate_type   TEXT NOT NULL,
    aggregate_id     UUID NOT NULL,
    version          BIGINT NOT NULL,
    payload          JSONB NOT NULL,
    metadata         JSONB NOT NULL DEFAULT '{}',
    occurred_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT unique_stream_version UNIQUE (aggregate_type, aggregate_id, version)
);

-- Stream mode index (already exists)
CREATE INDEX ix_events_stream ON events(aggregate_type, aggregate_id, version);
```

### Additional Indexes (Context Mode)

```sql
-- Context mode: fast filtering by event type
CREATE INDEX ix_events_event_type ON events(event_type);

-- Context mode: fast JSONB containment queries (payload @> '{"field": "value"}')
CREATE INDEX ix_events_payload_gin ON events USING GIN (payload);
```

**Two new indexes. Zero schema changes to the table itself.** The existing `sequence_number` column provides the global ordering needed for context-based optimistic concurrency.

### Why This Works

```
Stream query (existing):
  SELECT * FROM events
  WHERE aggregate_type = $1 AND aggregate_id = $2
  ORDER BY version ASC
  → Uses ix_events_stream → Fast point lookup

Context query (new):
  SELECT * FROM events
  WHERE event_type = ANY($1)
    AND payload @> ANY($2::jsonb[])
  ORDER BY sequence_number ASC
  → Uses ix_events_event_type + ix_events_payload_gin → Fast filtered scan
```

---

## 9. Implementation: event.Filter

### Full Implementation

```go
// event/filter.go
package event

// Filter defines which events are relevant for a context-based operation.
// It specifies a set of event types and payload predicates that together
// form the consistency boundary.
//
// Events match the filter if:
//   - Their type is in the event types list (OR), AND
//   - Their payload matches at least one predicate (OR between predicates)
//
// Example:
//
//	filter := event.NewFilter(
//	    event.Type("DeviceRegistered"),
//	    event.Type("AssetRegistered"),
//	    event.Type("DeviceBoundToAsset"),
//	).Where("deviceId", deviceID).
//	  OrWhere("assetId", assetID)
type Filter struct {
    eventTypes []Type
    predicates []Predicate
}

// Predicate matches events whose payload contains a specific field-value pair.
// Translated to PostgreSQL's JSONB containment: payload @> '{"field": value}'
type Predicate struct {
    Field string
    Value any
}

// NewFilter creates a Filter matching events of the specified types.
func NewFilter(eventTypes ...Type) *Filter {
    return &Filter{
        eventTypes: eventTypes,
    }
}

// Where adds a payload predicate. Events matching this filter must contain
// the given field-value pair in their payload. Multiple predicates are
// combined with OR logic (match any predicate).
func (f *Filter) Where(field string, value any) *Filter {
    f.predicates = append(f.predicates, Predicate{Field: field, Value: value})
    return f
}

// EventTypes returns the event types this filter matches.
func (f *Filter) EventTypes() []Type {
    return f.eventTypes
}

// Predicates returns the payload predicates.
// Returns nil if no predicates are set (matches all events of the given types).
func (f *Filter) Predicates() []Predicate {
    return f.predicates
}
```

### Filter → SQL Translation (in SQLEventStore)

```go
func (s *SQLEventStore) buildContextQuery(filter event.Filter) (string, []any) {
    args := []any{filter.EventTypes()}
    argIdx := 2 // $1 is event types

    whereClause := "event_type = ANY($1)"

    if len(filter.Predicates()) > 0 {
        predicates := make([]string, len(filter.Predicates()))
        jsonbValues := make([]string, len(filter.Predicates()))

        for i, p := range filter.Predicates() {
            predicates[i] = fmt.Sprintf("payload @> $%d", argIdx)
            // Build JSONB: {"field": "value"}
            jsonObj, _ := json.Marshal(map[string]any{p.Field: p.Value})
            jsonbValues[i] = string(jsonObj)
            argIdx++
        }

        // Cast to jsonb array for ANY()
        args = append(args, jsonbValues)
        whereClause += " AND payload @> ANY($" + fmt.Sprintf("%d", argIdx-1) + "::jsonb[])"
    }

    query := fmt.Sprintf(
        "SELECT %s FROM events WHERE %s ORDER BY sequence_number ASC",
        eventColumns,
        whereClause,
    )

    return query, args
}
```

---

## 10. Implementation: ContextQuerier

### Query Method on SQLEventStore

```go
// Query loads events matching the filter across aggregate streams.
// Returns events ordered by sequence number and the max sequence number
// for optimistic concurrency checking with AppendContext.
func (s *SQLEventStore) Query(
    ctx context.Context,
    filter event.Filter,
) (event.QueryResult, error) {
    query := `SELECT id, event_type, aggregate_type, aggregate_id, version,
                     payload, metadata, occurred_at
              FROM events
              WHERE event_type = ANY($1)`

    args := []any{filter.EventTypes()}
    argIdx := 2

    if len(filter.Predicates()) > 0 {
        jsonbValues := make([]string, len(filter.Predicates()))
        for i, p := range filter.Predicates() {
            jsonObj, _ := json.Marshal(map[string]any{p.Field: p.Value})
            jsonbValues[i] = string(jsonObj)
        }
        query += fmt.Sprintf(" AND payload @> ANY($%d::jsonb[])", argIdx)
        args = append(args, jsonbValues)
        argIdx++
    }

    query += " ORDER BY sequence_number ASC"

    rows, err := s.db.QueryContext(ctx, query, args...)
    if err != nil {
        return event.QueryResult{}, fmt.Errorf("context query: %w", err)
    }
    defer rows.Close()

    events, err := scanEvents(rows)
    if err != nil {
        return event.QueryResult{}, fmt.Errorf("scan context events: %w", err)
    }

    var maxSeq int
    if len(events) > 0 {
        // The last event in sequence order has the highest sequence number
        // We need a separate query for max_seq to handle the CTE correctly
        maxQuery := `SELECT COALESCE(MAX(sequence_number), 0) FROM events WHERE event_type = ANY($1)`
        maxArgs := []any{filter.EventTypes()}

        if len(filter.Predicates()) > 0 {
            // Reuse same predicate logic
            jsonbValues := make([]string, len(filter.Predicates()))
            for i, p := range filter.Predicates() {
                jsonObj, _ := json.Marshal(map[string]any{p.Field: p.Value})
                jsonbValues[i] = string(jsonObj)
            }
            maxQuery += fmt.Sprintf(" AND payload @> ANY($2::jsonb[])")
            maxArgs = append(maxArgs, jsonbValues)
        }

        err = s.db.QueryRowContext(ctx, maxQuery, maxArgs...).Scan(&maxSeq)
        if err != nil {
            return event.QueryResult{}, fmt.Errorf("get max sequence: %w", err)
        }
    }

    return event.QueryResult{
        Events: events,
        MaxSeq: maxSeq,
    }, nil
}
```

**Note**: The `MaxSeq` query uses the same filter as the event query. This ensures the sequence number reflects the exact context boundary that `AppendContext` will check.

---

## 11. Implementation: ContextAppender (CTE)

### The CTE-Based Append

This is the heart of the aggregateless consistency mechanism, adapted to work with go-cqrs-lite's stream-based event structure.

```go
// AppendContext atomically inserts events if the filter context hasn't changed.
// Uses a PostgreSQL CTE to check max(sequence_number) matches expectedMaxSeq
// before inserting — the same pattern as aggregateless event sourcing.
func (s *SQLEventStore) AppendContext(
    ctx context.Context,
    filter event.Filter,
    events []Event,
    expectedMaxSeq int,
) error {
    if len(events) == 0 {
        return nil
    }

    // Build the context condition for the CTE
    contextCondition := "event_type = ANY($1)"
    args := []any{filter.EventTypes()}
    argIdx := 2

    if len(filter.Predicates()) > 0 {
        jsonbValues := make([]string, len(filter.Predicates()))
        for i, p := range filter.Predicates() {
            jsonObj, _ := json.Marshal(map[string]any{p.Field: p.Value})
            jsonbValues[i] = string(jsonObj)
        }
        contextCondition += fmt.Sprintf(" AND payload @> ANY($%d::jsonb[])", argIdx)
        args = append(args, jsonbValues)
        argIdx++
    }

    // Build the CTE query
    // The CTE checks max(sequence_number) for the context, then
    // inserts only if it matches expectedMaxSeq.
    // If another writer modified the context, INSERT produces 0 rows.
    cte := fmt.Sprintf(`
        WITH context AS (
            SELECT MAX(sequence_number) AS max_seq
            FROM events
            WHERE %s
        )
        INSERT INTO events (id, event_type, aggregate_type, aggregate_id,
                           version, payload, metadata, occurred_at)
        SELECT
            unnest($%d::uuid[]),
            unnest($%d::text[]),
            unnest($%d::text[]),
            unnest($%d::uuid[]),
            unnest($%d::bigint[]),
            unnest($%d::jsonb[]),
            unnest($%d::jsonb[]),
            unnest($%d::timestamptz[])
        FROM context
        WHERE COALESCE(max_seq, 0) = $%d`,
        contextCondition,
        argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7,
        argIdx+8,
    )

    // Prepare batch arrays
    ids := make([]string, len(events))
    types := make([]string, len(events))
    aggTypes := make([]string, len(events))
    aggIDs := make([]string, len(events))
    versions := make([]int64, len(events))
    payloads := make([][]byte, len(events))
    metadatas := make([][]byte, len(events))
    timestamps := make([]time.Time, len(events))

    for i, evt := range events {
        ids[i] = evt.ID().String()
        types[i] = string(evt.Type())
        aggTypes[i] = string(evt.AggregateType())
        aggIDs[i] = evt.AggregateID().String()
        versions[i] = int64(evt.Version())
        payloads[i] = evt.Payload()
        metaBytes, _ := marshalMetadata(evt.Metadata())
        metadatas[i] = metaBytes
        timestamps[i] = evt.OccurredAt()
    }

    args = append(args,
        ids,           // $argIdx
        types,         // $argIdx+1
        aggTypes,      // $argIdx+2
        aggIDs,        // $argIdx+3
        versions,      // $argIdx+4
        payloads,      // $argIdx+5
        metadatas,     // $argIdx+6
        timestamps,    // $argIdx+7
        expectedMaxSeq, // $argIdx+8
    )

    result, err := s.db.ExecContext(ctx, cte, args...)
    if err != nil {
        return fmt.Errorf("context append: %w", err)
    }

    rowCount, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("check rows affected: %w", err)
    }

    if rowCount == 0 {
        return fmt.Errorf(
            "%w: expected max_seq %d for context with %d event types",
            ErrContextChanged,
            expectedMaxSeq,
            len(filter.EventTypes()),
        )
    }

    return nil
}
```

### How the CTE Works (Step by Step)

```
1. CTE "context" computes:  MAX(sequence_number) WHERE [filter conditions]
   → Returns a single row with max_seq

2. INSERT ... SELECT ... FROM context WHERE COALESCE(max_seq, 0) = $expected

3. If max_seq == expected:
   → SELECT returns 1 row → INSERT fires → events written
   → rowCount > 0 → success

4. If max_seq != expected (another writer changed the context):
   → SELECT returns 0 rows → INSERT doesn't fire
   → rowCount == 0 → ErrContextChanged → caller retries
```

### Important: Events Still Belong to Streams

Events appended via `AppendContext` still have `aggregate_type`, `aggregate_id`, and `version` fields. They can be loaded by both:

- **Stream mode**: `store.Load(ctx, aggType, aggID)` — loads events for one aggregate
- **Context mode**: `store.Query(ctx, filter)` — loads events across aggregates

This means projections, snapshots, and all existing infrastructure work unchanged. The context mode is an additional lens, not a replacement.

---

## 12. Implementation: ContextRepository

A high-level convenience that mirrors `aggregate.EventSourcedRepository` for context-based operations:

```go
// event/context_repository.go
package event

import "context"

// ContextRepository provides a Decider-pattern workflow for context-based operations.
// It mirrors the aggregate.Repository pattern but uses Filter-based consistency
// instead of stream-based versioning.
//
// Usage:
//
//	err := contextRepo.Execute(ctx, filter, func(events []Event, maxSeq int) ([]Event, error) {
//	    state := foldMyState(events)
//	    return decideMyOperation(state, command)
//	})
type ContextRepository struct {
    store ContextQuerierAppender
    bus   Bus
}

// NewContextRepository creates a repository for context-based operations.
func NewContextRepository(
    store ContextQuerierAppender,
    bus Bus,
) *ContextRepository {
    return &ContextRepository{
        store: store,
        bus:   bus,
    }
}

// Execute runs a Decider-pattern workflow with context-based consistency.
//
//  1. Queries events matching the filter (gets events + maxSeq)
//  2. Calls decide with the loaded events and maxSeq
//  3. If decide returns events, atomically appends them via CTE
//  4. Publishes the new events to the bus
//
// If the context changed between query and append, returns ErrContextChanged.
// The caller should retry.
func (r *ContextRepository) Execute(
    ctx context.Context,
    filter Filter,
    decide func(events []Event, maxSeq int) ([]Event, error),
) error {
    result, err := r.store.Query(ctx, filter)
    if err != nil {
        return fmt.Errorf("query context: %w", err)
    }

    newEvents, err := decide(result.Events, result.MaxSeq)
    if err != nil {
        return fmt.Errorf("decide: %w", err)
    }

    if len(newEvents) == 0 {
        return nil
    }

    err = r.store.AppendContext(ctx, filter, newEvents, result.MaxSeq)
    if err != nil {
        return fmt.Errorf("append context: %w", err)
    }

    err = r.bus.Publish(ctx, newEvents...)
    if err != nil {
        return fmt.Errorf("publish events: %w", err)
    }

    return nil
}
```

### With Retry Support

```go
// ExecuteWithRetry wraps Execute with automatic retry on ErrContextChanged.
// maxRetries controls how many times to retry before giving up.
func (r *ContextRepository) ExecuteWithRetry(
    ctx context.Context,
    filter Filter,
    decide func(events []Event, maxSeq int) ([]Event, error),
    maxRetries int,
) error {
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := r.Execute(ctx, filter, decide)
        if err == nil {
            return nil
        }

        if !errors.Is(err, ErrContextChanged) {
            return err
        }

        if attempt == maxRetries {
            return fmt.Errorf("context changed after %d retries: %w", maxRetries, err)
        }
    }

    return nil
}
```

---

## 13. Usage Patterns

### Pattern 1: Single-Entity Operation (Traditional — Unchanged)

```go
func (h *WithdrawHandler) Handle(ctx context.Context, cmd *WithdrawMoney) error {
    root := NewAccount(cmd.AccountID)

    if err := h.repo.Load(ctx, root); err != nil {
        return err
    }

    if root.Balance() < cmd.Amount {
        return ErrInsufficientFunds
    }

    root.RecordEvent(ctx, NewMoneyWithdrawn(cmd.AccountID, cmd.Amount))

    return h.repo.Save(ctx, root)
}
```

**When to use**: Always. This is the default. Use it for 95% of commands.

### Pattern 2: Cross-Entity Operation (Context Mode — New)

```go
func (h *BindDeviceToAssetHandler) Handle(ctx context.Context, cmd *BindDeviceToAsset) error {
    filter := event.NewFilter(
        event.Type("DeviceRegistered"),
        event.Type("AssetRegistered"),
        event.Type("DeviceBoundToAsset"),
        event.Type("DeviceUnboundFromAsset"),
    ).Where("deviceId", cmd.DeviceID).
      Where("assetId", cmd.AssetID)

    return h.contextRepo.ExecuteWithRetry(ctx, filter, h.decide, 3)
}

func (h *BindDeviceToAssetHandler) decide(
    events []event.Event,
    maxSeq int,
) ([]event.Event, error) {
    state := foldBindingState(events)

    if !state.deviceExists {
        return nil, ErrDeviceNotFound
    }
    if !state.assetExists {
        return nil, ErrAssetNotFound
    }
    if state.isBound {
        return nil, ErrAlreadyBound
    }

    evt, err := event.NewEvent(
        "DeviceBoundToAsset",
        cmd.DeviceID,       // aggregateID (device owns the event)
        "Device",           // aggregateType
        state.deviceVersion + 1,
        encodePayload(BindingPayload{
            DeviceID: cmd.DeviceID,
            AssetID:  cmd.AssetID,
        }),
    )
    if err != nil {
        return nil, err
    }

    return []event.Event{evt}, nil
}
```

**When to use**: Operations that span multiple aggregate types and need strong consistency.

### Pattern 3: Feature-Specific Read (Context Query Only — No Write)

```go
func (q *BalanceQuery) Handle(ctx context.Context, query *GetBalance) (*BalanceResult, error) {
    filter := event.NewFilter(
        event.Type("AccountOpened"),
        event.Type("MoneyDeposited"),
        event.Type("MoneyWithdrawn"),
    ).Where("accountId", query.AccountID)

    result, err := q.contextQuerier.Query(ctx, filter)
    if err != nil {
        return nil, err
    }

    state := foldBalanceState(result.Events)

    return &BalanceResult{
        AccountID: query.AccountID,
        Balance:   state.balance,
        Exists:    state.exists,
    }, nil
}
```

**When to use**: Building ad-hoc read models or queries that need specific event types without loading the full aggregate.

---

## 14. Complete Example: Single-Entity (Traditional)

The existing pattern — no changes needed.

```go
// --- Aggregate ---

type Account struct {
    aggregate.Core
    customerName string
    balance      int
    frozen       bool
}

func NewAccount(id id.AggregateID) *Account {
    return &Account{
        Core: *aggregate.NewCore(id, "Account"),
    }
}

func (a *Account) Apply(evt event.Event) error {
    switch evt.Type() {
    case "AccountOpened":
        var p AccountOpenedPayload
        _ = event.DecodePayload(evt, JSONCodec{}, &p)
        a.customerName = p.CustomerName
    case "MoneyDeposited":
        var p MoneyDepositedPayload
        _ = event.DecodePayload(evt, JSONCodec{}, &p)
        a.balance += p.Amount
    case "MoneyWithdrawn":
        var p MoneyWithdrawnPayload
        _ = event.DecodePayload(evt, JSONCodec{}, &p)
        a.balance -= p.Amount
    case "AccountFrozen":
        a.frozen = true
    }
    return nil
}

// --- Command Handler ---

type WithdrawHandler struct {
    repo *aggregate.EventSourcedRepository
}

func (h *WithdrawHandler) Handle(ctx context.Context, cmd *WithdrawMoney) error {
    root := NewAccount(cmd.AccountID)

    if err := h.repo.Load(ctx, root); err != nil {
        return fmt.Errorf("load account: %w", err)
    }

    if root.frozen {
        return ErrAccountFrozen
    }
    if root.balance < cmd.Amount {
        return ErrInsufficientFunds
    }

    payload, _ := json.Marshal(MoneyWithdrawnPayload{Amount: cmd.Amount})
    evt, err := event.NewEvent(
        "MoneyWithdrawn",
        root.ID(),
        root.Type(),
        root.Version()+1,
        payload,
    )
    if err != nil {
        return err
    }

    root.RecordEvent(ctx, evt)

    return h.repo.Save(ctx, root)
}
```

---

## 15. Complete Example: Cross-Entity (Context Mode)

```go
// --- State (feature-specific) ---

type BindingState struct {
    deviceExists bool
    assetExists  bool
    currentBinding *BindingInfo
    deviceVersion  int
    assetVersion   int
}

type BindingInfo struct {
    DeviceID id.AggregateID
    AssetID  id.AggregateID
}

func foldBindingState(events []event.Event) BindingState {
    state := BindingState{}

    for _, evt := range events {
        switch evt.Type() {
        case "DeviceRegistered":
            state.deviceExists = true
            state.deviceVersion = evt.Version()
        case "AssetRegistered":
            state.assetExists = true
            state.assetVersion = evt.Version()
        case "DeviceBoundToAsset":
            var p DeviceBoundToAssetPayload
            _ = event.DecodePayload(evt, event.JSONCodec{}, &p)
            state.currentBinding = &BindingInfo{
                DeviceID: p.DeviceID,
                AssetID:  p.AssetID,
            }
        case "DeviceUnboundFromAsset":
            state.currentBinding = nil
        }
    }

    return state
}

// --- Command ---

type BindDeviceToAsset struct {
    DeviceID id.AggregateID
    AssetID  id.AggregateID
}

// --- Handler ---

type BindDeviceToAssetHandler struct {
    contextRepo *event.ContextRepository
}

func (h *BindDeviceToAssetHandler) Handle(ctx context.Context, cmd *BindDeviceToAsset) error {
    filter := event.NewFilter(
        event.Type("DeviceRegistered"),
        event.Type("AssetRegistered"),
        event.Type("DeviceBoundToAsset"),
        event.Type("DeviceUnboundFromAsset"),
    ).Where("deviceId", cmd.DeviceID).
      Where("assetId", cmd.AssetID)

    return h.contextRepo.ExecuteWithRetry(ctx, filter, func(
        events []event.Event,
        maxSeq int,
    ) ([]event.Event, error) {
        state := foldBindingState(events)

        if !state.deviceExists {
            return nil, fmt.Errorf("device %s not found", cmd.DeviceID)
        }
        if !state.assetExists {
            return nil, fmt.Errorf("asset %s not found", cmd.AssetID)
        }
        if state.currentBinding != nil {
            return nil, fmt.Errorf("device already bound to asset %s", state.currentBinding.AssetID)
        }

        payload, _ := json.Marshal(DeviceBoundToAssetPayload{
            DeviceID: cmd.DeviceID,
            AssetID:  cmd.AssetID,
        })

        evt, err := event.NewEvent(
            "DeviceBoundToAsset",
            cmd.DeviceID,
            "Device",
            state.deviceVersion+1,
            payload,
        )
        if err != nil {
            return nil, err
        }

        return []event.Event{evt}, nil
    }, 3)
}
```

### Why This Is Better Than a Saga

| Aspect             | Saga Approach                                  | Context Approach                       |
| ------------------ | ---------------------------------------------- | -------------------------------------- |
| **Classes needed** | 3-5 (orchestrator, steps, compensation)        | 1 (handler with decide function)       |
| **Consistency**    | Eventual (each step is a separate transaction) | Strong (single CTE atomic operation)   |
| **Compensation**   | Must write explicit undo logic                 | Not needed — atomic success or failure |
| **Testing**        | Integration test with multiple services        | Unit test of pure decide function      |
| **Debugging**      | Distributed trace across services              | Single CTE, single error point         |

---

## 16. Complete Example: The Hybrid Service

A real service that uses both patterns:

```go
type Service struct {
    // Traditional aggregate access (95% of operations)
    accountRepo *aggregate.EventSourcedRepository

    // Context-based access (5% of operations — cross-entity)
    contextRepo *event.ContextRepository

    // Shared infrastructure
    bus     event.Bus
    store   *storage.SQLEventStore
    codec   event.Codec
}

func NewService(db *sql.DB, bus event.Bus) *Service {
    store := storage.NewSQLEventStore(db)
    codec := event.JSONCodec{}

    return &Service{
        accountRepo: aggregate.NewRepository(store, bus),
        contextRepo: event.NewContextRepository(store, bus),
        store:       store,
        bus:         bus,
        codec:       codec,
    }
}

// OpenAccount — single entity, traditional aggregate
func (s *Service) OpenAccount(ctx context.Context, cmd *OpenAccount) error {
    root := NewAccount(cmd.AccountID)

    if err := s.accountRepo.Load(ctx, root); err != nil {
        return err
    }

    if root.Version() > 0 {
        return ErrAccountAlreadyExists
    }

    payload, _ := json.Marshal(AccountOpenedPayload{
        CustomerName: cmd.CustomerName,
        Currency:     cmd.Currency,
    })

    evt, err := event.NewEvent("AccountOpened", root.ID(), root.Type(), 1, payload)
    if err != nil {
        return err
    }

    root.RecordEvent(ctx, evt)

    return s.accountRepo.Save(ctx, root)
}

// Deposit — single entity, traditional aggregate
func (s *Service) Deposit(ctx context.Context, cmd *DepositMoney) error {
    root := NewAccount(cmd.AccountID)

    if err := s.accountRepo.Load(ctx, root); err != nil {
        return err
    }

    if root.Version() == 0 {
        return ErrAccountNotFound
    }

    payload, _ := json.Marshal(MoneyDepositedPayload{Amount: cmd.Amount})

    evt, err := event.NewEvent(
        "MoneyDeposited", root.ID(), root.Type(),
        root.Version()+1, payload,
    )
    if err != nil {
        return err
    }

    root.RecordEvent(ctx, evt)

    return s.accountRepo.Save(ctx, root)
}

// Transfer — cross-entity, context mode
// Transfers money between two accounts atomically.
// No saga needed — the CTE ensures atomicity across both accounts.
func (s *Service) Transfer(ctx context.Context, cmd *TransferMoney) error {
    filter := event.NewFilter(
        event.Type("AccountOpened"),
        event.Type("MoneyDeposited"),
        event.Type("MoneyWithdrawn"),
        event.Type("MoneyTransferred"),
    ).Where("accountId", cmd.FromAccount).
      Where("accountId", cmd.ToAccount)

    return s.contextRepo.ExecuteWithRetry(ctx, filter, func(
        events []event.Event,
        maxSeq int,
    ) ([]event.Event, error) {
        fromState := foldAccountState(events, cmd.FromAccount)
        toState := foldAccountState(events, cmd.ToAccount)

        if !fromState.exists {
            return nil, fmt.Errorf("source account %s not found", cmd.FromAccount)
        }
        if !toState.exists {
            return nil, fmt.Errorf("destination account %s not found", cmd.ToAccount)
        }
        if fromState.balance < cmd.Amount {
            return nil, ErrInsufficientFunds
        }

        withdrawPayload, _ := json.Marshal(MoneyTransferredPayload{
            FromAccount: cmd.FromAccount,
            ToAccount:   cmd.ToAccount,
            Amount:      cmd.Amount,
        })

        // Event on source account
        withdrawEvt, err := event.NewEvent(
            "MoneyTransferred",
            cmd.FromAccount,
            "Account",
            fromState.version+1,
            withdrawPayload,
        )
        if err != nil {
            return nil, err
        }

        // Event on destination account
        depositEvt, err := event.NewEvent(
            "MoneyTransferred",
            cmd.ToAccount,
            "Account",
            toState.version+1,
            withdrawPayload,
        )
        if err != nil {
            return nil, err
        }

        return []event.Event{withdrawEvt, depositEvt}, nil
    }, 3)
}

// foldAccountState folds events for a specific account from a mixed result set
func foldAccountState(events []event.Event, accountID id.AggregateID) AccountState {
    state := AccountState{}

    for _, evt := range events {
        if evt.AggregateID() != accountID {
            continue
        }

        switch evt.Type() {
        case "AccountOpened":
            state.exists = true
        case "MoneyDeposited":
            var p MoneyDepositedPayload
            _ = event.DecodePayload(evt, event.JSONCodec{}, &p)
            state.balance += p.Amount
        case "MoneyWithdrawn":
            var p MoneyWithdrawnPayload
            _ = event.DecodePayload(evt, event.JSONCodec{}, &p)
            state.balance -= p.Amount
        case "MoneyTransferred":
            var p MoneyTransferredPayload
            _ = event.DecodePayload(evt, event.JSONCodec{}, &p)
            if p.FromAccount == accountID {
                state.balance -= p.Amount
            } else {
                state.balance += p.Amount
            }
        }
        state.version = evt.Version()
    }

    return state
}
```

---

## 17. Decision Flowchart

When writing a new command handler, use this decision tree:

```
                        ┌─────────────────────────────┐
                        │  New Command Handler          │
                        └──────────────┬──────────────┘
                                       │
                        ┌──────────────▼──────────────┐
                        │  Does it touch ONE entity?   │
                        └──────────────┬──────────────┘
                                       │
                          ┌────────────┴────────────┐
                          │ YES                     │ NO
                          ▼                         ▼
               ┌─────────────────────┐  ┌─────────────────────────┐
               │  Use Aggregate      │  │  Does it need strong    │
               │  Repository         │  │  consistency?           │
               │  (traditional)      │  │                         │
               │                     │  │  i.e., must all writes  │
               │  • Load aggregate   │  │  succeed or fail as     │
               │  • Mutate state     │  │  one unit?              │
               │  • RecordEvent      │  └──────────┬──────────────┘
               │  • Save aggregate   │             │
               └─────────────────────┘  ┌──────────┴──────────┐
                                        │ YES                 │ NO
                                        ▼                     ▼
                              ┌───────────────────┐  ┌───────────────────┐
                              │  Use Context       │  │  Use eventual     │
                              │  Repository        │  │  consistency      │
                              │  (aggregateless)   │  │                   │
                              │                    │  │  • Publish event   │
                              │  • Define filter   │  │  • Bus.Subscribe  │
                              │  • Query context   │  │  • React async    │
                              │  • Decide (pure)   │  │  • Idempotent     │
                              │  • CTE append      │  │    handlers       │
                              └───────────────────┘  └───────────────────┘
```

### Rules of Thumb

| Scenario                  | Pattern              | Why                                        |
| ------------------------- | -------------------- | ------------------------------------------ |
| Create order              | Aggregate            | Single entity                              |
| Cancel order              | Aggregate            | Single entity                              |
| Add item to order         | Aggregate            | Single entity                              |
| Transfer between accounts | Context              | Two entities, atomic                       |
| Bind device to asset      | Context              | Two entity types, atomic                   |
| Send welcome email        | Eventual             | Side effect, no consistency needed         |
| Update search index       | Eventual             | Read model, eventual is fine               |
| Process payment           | Context              | Payment + order, atomic                    |
| Freeze account + notify   | Aggregate + Eventual | Freeze is atomic, notification is eventual |

---

## 18. Comparison Matrix

### Full Feature Comparison

| Feature                         | Traditional Only         | Aggregateless Only      | Hybrid (This Proposal)          |
| ------------------------------- | ------------------------ | ----------------------- | ------------------------------- |
| **Single-entity CRUD**          | Excellent                | Works (overkill)        | Excellent (aggregate path)      |
| **Cross-entity atomic ops**     | Poor (needs saga)        | Excellent               | Excellent (context path)        |
| **Push-based projections**      | Excellent                | Missing                 | Excellent (Bus + Projection)    |
| **Snapshotting**                | Excellent                | Missing                 | Excellent (SnapshotStore)       |
| **Strong typing**               | Excellent (Go types)     | Weak (JSONB)            | Excellent (Go types everywhere) |
| **Schema enforcement**          | Compile-time             | None                    | Compile-time                    |
| **Event schema evolution**      | Upcasting                | Weak                    | Upcasting                       |
| **Read model flexibility**      | Projection-based         | Per-feature fold        | Both                            |
| **Consistency model**           | Stream version           | Context CTE             | Both                            |
| **Auto-documentation**          | Catalog (AsyncAPI)       | None                    | Catalog (AsyncAPI)              |
| **Testing**                     | Aggregate unit tests     | Pure function tests     | Both                            |
| **Performance (single entity)** | Fast (stream read)       | Slower (filtered query) | Fast (stream read)              |
| **Performance (cross-entity)**  | Slow (multi-read + saga) | Fast (single query)     | Fast (single query)             |
| **Tooling maturity**            | High                     | Low                     | High (extends existing)         |
| **Migration cost**              | —                        | Complete rewrite        | Additive (new interfaces only)  |
| **Learning curve**              | DDD + ES                 | SQL + FP                | Gradual (start with aggregates) |

### Error Handling Comparison

| Scenario                         | Traditional Error     | Context Error           |
| -------------------------------- | --------------------- | ----------------------- |
| Concurrent write to same entity  | `ErrVersionConflict`  | `ErrContextChanged`     |
| Concurrent write to same context | N/A                   | `ErrContextChanged`     |
| Entity not found                 | `root.Version() == 0` | `state.exists == false` |
| Invalid command                  | Error from `Apply`    | Error from `decide`     |
| Retry strategy                   | Caller retries        | `ExecuteWithRetry`      |

Both `ErrVersionConflict` and `ErrContextChanged` work with `errors.Is()`. Both are optimistic concurrency violations. The retry pattern is the same.

---

## 19. What We Keep From Each World

### From Traditional ES (The Foundation)

| What                                  | Why                                               |
| ------------------------------------- | ------------------------------------------------- |
| `aggregate.Root`                      | Proven pattern for single-entity state management |
| `aggregate.Repository`                | Clean abstraction over load/decide/save cycle     |
| `event.Store` (stream-based)          | Fast point lookups by aggregate identity          |
| `event.Bus` + `event.Projection`      | Push-based real-time projections                  |
| `event.SnapshotStore`                 | Performance optimization for large event streams  |
| `event.Outbox`                        | Reliable eventual publishing                      |
| `event.Codec` + `DecodePayload[T]`    | Type-safe serialization                           |
| Branded IDs (`id.Of[T]`)              | Compile-time identity safety                      |
| Catalog (AsyncAPI, EventCatalog)      | Auto-documentation and AI-ready metadata          |
| Middleware (logging, retry, recovery) | Cross-cutting concerns                            |
| `InMemoryRunner`                      | Test and single-process projection execution      |

### From Aggregateless ES (The Enhancement)

| What                                | Why                                                         |
| ----------------------------------- | ----------------------------------------------------------- |
| `event.Filter`                      | Explicit, readable consistency boundary definition          |
| Context queries (`Query`)           | Cross-entity event loading without multiple aggregate loads |
| CTE atomic append (`AppendContext`) | Cross-entity atomic writes without sagas                    |
| Feature-specific fold functions     | Only reconstruct the state each operation needs             |
| Pure `decide` functions             | Testable business logic without mocks                       |
| Transparent consistency             | Filter is the boundary — visible at the call site           |

### What We Explicitly Reject From Aggregateless

| What                                 | Why                                            |
| ------------------------------------ | ---------------------------------------------- |
| Single events table (no streams)     | Lose aggregate identity, versioning, snapshots |
| No `aggregate_type` / `aggregate_id` | Lose stream-based access pattern               |
| Raw JSONB without typed events       | Lose compile-time safety                       |
| No push-based subscriptions          | Already have Bus + Projection                  |
| No schema enforcement                | Already have Codec + Go types                  |
| No snapshotting                      | Already have SnapshotStore                     |
| No admin tooling                     | Build on PostgreSQL's ecosystem instead        |

---

## 20. Migration Strategy

### Phase 0: Understand (No Code Changes)

- Read this document
- Identify 1-2 cross-entity operations in your domain
- Evaluate whether they currently use sagas, domain services, or dual-writes

### Phase 1: Add Indexes (Infrastructure Only)

```sql
CREATE INDEX ix_events_event_type ON events(event_type);
CREATE INDEX ix_events_payload_gin ON events USING GIN (payload);
```

Zero application changes. Zero risk. Just adds query capability to the existing events table.

### Phase 2: Add Interfaces (Code, No Behavior Change)

Add to `core/event/`:

- `filter.go` — `Filter`, `Predicate` types
- `context.go` — `ContextQuerier`, `ContextAppender`, `ContextRepository`, `QueryResult`

These are pure interface definitions. No existing code changes. No implementations yet.

### Phase 3: Implement on SQLEventStore

Add `Query` and `AppendContext` methods to `storage.SQLEventStore`. These are new methods on an existing struct. Existing methods (`Save`, `Load`, etc.) are untouched.

Add compile-time interface checks:

```go
var _ event.ContextQuerier = (*SQLEventStore)(nil)
var _ event.ContextAppender = (*SQLEventStore)(nil)
```

### Phase 4: Implement ContextRepository

Add `event/context_repository.go` with `ContextRepository.Execute` and `ExecuteWithRetry`.

### Phase 5: Use In One Cross-Entity Handler

Pick the most painful cross-entity operation. Replace its saga/process manager with a context-based handler. Measure:

- Code reduction (expected: 3-5x fewer classes)
- Performance (expected: faster — single CTE vs. multi-step saga)
- Correctness (expected: stronger — atomic vs. eventual)

### Phase 6: Evaluate and Expand

If Phase 5 succeeds, apply the pattern to other cross-entity operations. If it doesn't fit, the traditional path is still there — no loss.

### Migration Order for go-cqrs-lite Itself

| Phase | Module         | Change                                                            |
| ----- | -------------- | ----------------------------------------------------------------- |
| 1     | `storage/`     | Add GIN + event_type indexes                                      |
| 2     | `core/event/`  | Add `Filter`, `Predicate` types                                   |
| 3     | `core/event/`  | Add `ContextQuerier`, `ContextAppender`, `QueryResult` interfaces |
| 4     | `storage/`     | Implement `Query`, `AppendContext` on `SQLEventStore`             |
| 5     | `core/event/`  | Add `ContextRepository`                                           |
| 6     | `memory/`      | Implement `Query` on `MemoryStore` (for testing)                  |
| 7     | `testhelpers/` | Add `NewTestContextRepository` helper                             |
| 8     | `integration/` | Add integration tests for context mode                            |
| 9     | `catalog/`     | Document context-based operations in AsyncAPI                     |
| 10    | `example/`     | Add hybrid service example                                        |

---

## 21. Risks & Mitigations

### Risk: GIN Index Performance on Large Tables

**Impact**: Context queries may be slow on tables with hundreds of millions of events.

**Mitigation**:

- PostgreSQL's GIN index is highly optimized for JSONB containment
- Partitioning by time range (monthly) keeps partitions manageable
- For very large tables, consider materialized context views
- Benchmark before committing to this pattern

### Risk: CTE Serialization on Hot Contexts

**Impact**: If many operations target the same context (e.g., a popular device being bound/unbound repeatedly), the CTE creates a serialization point.

**Mitigation**:

- This is the same problem as aggregate contention — not unique to context mode
- For hot contexts, consider whether the operation truly needs strong consistency
- Eventual consistency via Bus + Projection may be more appropriate for high-contention scenarios
- The decision flowchart (Section 17) guides this choice

### Risk: Filter Correctness

**Impact**: A wrong filter (missing event type, wrong predicate) means incorrect consistency — either too narrow (misses conflicts) or too wide (false conflicts).

**Mitigation**:

- Filters are explicit and reviewable at the call site
- Integration tests should verify the exact events returned for a given filter
- Consider adding filter validation (e.g., "filter must include at least one event type")
- Catalog module can document which event types exist, making it easy to verify completeness

### Risk: Events Without Stream Identity

**Impact**: Events appended via `AppendContext` still need valid `aggregate_type`, `aggregate_id`, and `version`. If these are wrong, stream-based reads break.

**Mitigation**:

- The `decide` function must correctly set these fields
- This is no different from the responsibility of `aggregate.Root.RecordEvent` — the caller must provide correct metadata
- Helper functions can compute the next version from the folded state

### Risk: Dual Consistency Models Confuse Developers

**Impact**: Having both `ErrVersionConflict` (stream) and `ErrContextChanged` (context) might confuse developers about which to use.

**Mitigation**:

- Clear naming and documentation (this document)
- The decision flowchart provides a simple rule: single entity → aggregate, cross-entity → context
- Both errors are optimistic concurrency violations with the same retry strategy
- Code reviews enforce the decision tree

---

## 22. Open Questions

| Question                                                          | Options                                      | Recommendation                                                                                  |
| ----------------------------------------------------------------- | -------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Should `Filter` support OR between predicate groups?              | Simple AND only vs. full boolean algebra     | Start simple. Add complexity if real use cases demand it.                                       |
| Should `Query` support pagination?                                | Load all matching events vs. cursor-based    | Start with load-all. Add pagination if performance requires it.                                 |
| Should `AppendContext` publish to Bus?                            | ContextRepository does it vs. caller does it | ContextRepository handles it — same pattern as aggregate.Repository.                            |
| Should `MemoryStore` implement `ContextQuerier`?                  | Yes (for testing) vs. no (keep simple)       | Yes — test coverage matters. Implement with in-memory filtering.                                |
| Should events appended via context mode participate in snapshots? | Yes vs. no vs. automatic                     | Yes — events have stream identity, so existing snapshot infrastructure works.                   |
| How does the catalog document context-based operations?           | New channel type vs. special annotation      | Use a special annotation in the catalog metadata. The catalog already supports custom metadata. |
| Should context mode support outbox?                               | Yes vs. no                                   | Yes — same reliability guarantees. `ContextRepository` should accept optional `Outbox`.         |

---

## Summary

```
Traditional ES                          Aggregateless ES
┌─────────────────────┐                 ┌─────────────────────┐
│ Aggregates           │                 │ Pure events          │
│ Stream boundaries    │                 │ Context queries      │
│ Strong typing        │                 │ CTE atomicity        │
│ Mature tooling       │                 │ Cross-entity ease    │
│ Snapshots            │                 │ Feature folds        │
│ Push projections     │                 │ Transparent boundary │
└────────┬────────────┘                 └────────┬────────────┘
         │                                       │
         │         ┌──────────────────┐          │
         └────────►│  HYBRID DESIGN   │◄─────────┘
                   │                  │
                   │  Same events     │
                   │  Same table      │
                   │  Two access      │
                   │  patterns        │
                   │                  │
                   │  Aggregates for  │
                   │  single-entity   │
                   │                  │
                   │  Context queries │
                   │  for cross-      │
                   │  entity          │
                   │                  │
                   │  Additive only.  │
                   │  Zero breaking   │
                   │  changes.        │
                   └──────────────────┘
```

**One event store. Two access patterns. The right tool for each job.**
