# Stream API v3 Proposal: CQRS Read Model

> **Date:** 2026-05-28 | **Status:** Proposal v3 | **Supersedes:** `2026-05-28_STREAM_API_V2_PROPOSAL.md`

## What was wrong with v2

Five structural problems, all rooted in the same confusion: **we put read-model concerns into the write model.**

### 1. `int` instead of `uint`

`EventCount int`, `Limit int`, `TotalCount int`. The codebase already uses `uint` for counts in `query.Pagination`. Using `int` is a type safety regression — negative counts and limits are unrepresentable states that `uint` makes impossible.

### 2. `Q()` is a bad API

```go
Q(store) // What does Q mean? Query? Queue? Quokka?
    .Type("User")
    .PageSize(20)
    .Streams(ctx)
```

`Q` is not a Go idiom. And `Limit: 20` is hardcoded inside the function with no way to discover it without reading source. The consumer gets magic defaults they didn't ask for.

### 3. `StreamLister` on `Store` violates CQRS

`Store` is the **write model**. It persists events, handles concurrency, manages streams. Listing aggregates is a **read-model** concern — it queries state, it doesn't mutate it. Putting `ListStreams` on `Store` conflates write and read responsibilities.

The right separation:

| Concern | Write Model | Read Model |
|---|---|---|
| Append events | `Store.Save()` | — |
| Load events | `Store.Load()` | — |
| List aggregates | — | `stream.AggregateReader` |
| Tombstone detection | — | `stream.AggregateReader` (metadata) |
| Paginated queries | — | `stream.ListBuilder` |

### 4. `TombstoneStore` decorator breaks type assertions

Wrapping `event.Store` with a tombstone decorator means `TransactionalStore`, `PositionalLoader`, `BackwardsLoader` are silently lost. The `decider.Repository` type-asserts to `TransactionalStore` — it would fail. This is a footgun.

### 5. `SQLEventStore` bloat

Adding an `aggregates` table and `ListStreams` to `SQLEventStore` makes it responsible for:
- Event persistence (write)
- Optimistic concurrency (write)
- Aggregate enumeration (read)
- Tombstone tracking (read)

That's four responsibilities for one struct. The `aggregates` table is a **read-model projection** — it should be maintained separately, not baked into the event store.

---

## v3 Design Principles

| # | Principle | Rationale |
|---|---|---|
| 1 | **Store is write model only** | No read-model methods on `Store` |
| 2 | **Read model is separate** | `stream/` provides readers, not store extensions |
| 3 | **`uint` for all counts/limits** | Negative values are impossible states |
| 4 | **Descriptive constructors, no magic** | `NewInMemoryReader(loader)`, `NewSQLReader(db)` — no `Q()` |
| 5 | **Tombstone is metadata, not wrapper** | Same pattern as `signing` — metadata key on events |
| 6 | **SQL read model is a projection** | Separate table, maintained by event subscription |
| 7 | **Cursor pagination only** | `Page[T]` has `HasMore`, no `TotalCount` (expensive for append-only logs) |

---

## Architecture

```
core/event/         ← Metadata key for tombstones (write model concern)
stream/             ← Read model: readers, builder, tombstone middleware
memory/             ← Write model: Store (unchanged)
storage/            ← Write model: SQLEventStore (unchanged)
projection/         ← Already exists; stream projection registers here
```

The `stream/` module is **pure read model**. It never writes events. It queries.

---

## Layer 1: `core/event/` — Tombstone metadata (only)

No new interfaces on `Store`. Just the metadata key and helpers:

```go
// core/event/tombstone.go — new file

package event

import "fmt"

// MetadataKeyTombstone marks an aggregate as soft-deleted.
// When present with value "true" on an event, that event's aggregate
// is considered tombstoned. The tombstone status is determined by the
// LAST event in the stream.
const MetadataKeyTombstone MetadataKey = "tombstone"

// MarkTombstone copies an event and sets the tombstone metadata key.
// Returns a new event; the original is unmodified.
func MarkTombstone(evt Event) (*ImmutableEvent, error) {
    if evt == nil {
        return nil, fmt.Errorf("mark tombstone: %w", ErrNilEvent)
    }
    return NewEvent(
        evt.Type(),
        evt.AggregateID(),
        evt.AggregateType(),
        evt.Version(),
        evt.Payload(),
        WithEventID(evt.ID()),
        WithOccurredAt(evt.OccurredAt()),
        WithSchemaVersion(evt.SchemaVersion()),
        WithMetadata(evt.Metadata()),
        WithCustom(MetadataKeyTombstone, "true"),
    )
}

// HasTombstone reports whether the event carries a tombstone marker.
func HasTombstone(evt Event) bool {
    if evt == nil {
        return false
    }
    md := evt.Metadata()
    if md == nil || md.Custom == nil {
        return false
    }
    return md.Custom[MetadataKeyTombstone] == "true"
}
```

**Why this belongs in `core/event/`:** Tombstone metadata is a **write-model convention**. Events carry it. The write model produces it. The read model reads it. This is exactly the same pattern as `signing.MetadataKey`.

---

## Layer 2: `stream/` — Read model module

### Types

```go
// stream/types.go

package stream

import (
    "time"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// AggregateRef is a lightweight reference to an aggregate stream.
// Used for listings without loading full event streams.
type AggregateRef struct {
    ID          id.AggregateID
    Type        event.AggregateType
    Version     event.Version
    EventCount  uint        // uint: cannot be negative
    LastEventAt time.Time
    Tombstoned  bool        // derived from last event's metadata
}

// Page is a cursor-based page of results.
// No TotalCount — append-only logs make counts stale and expensive.
type Page[T any] struct {
    Items   []T
    HasMore bool // true if there are more pages
}

// TombstonePolicy controls visibility of soft-deleted aggregates.
type TombstonePolicy int

const (
    TombstoneExclude TombstonePolicy = iota // default: hide deleted
    TombstoneInclude                        // show all
    TombstoneOnly                           // show only deleted
)

// AggregateFilter controls listing queries.
// Consumers construct this via ListBuilder, not directly.
type AggregateFilter struct {
    Type      event.AggregateType
    After     id.AggregateID  // cursor: return aggregates after this ID
    Limit     uint
    Tombstone TombstonePolicy
}
```

### Reader interface

```go
// stream/reader.go

package stream

import "context"

// AggregateReader is a CQRS read model for aggregate listings.
// Implementations may query projected tables, the events table,
// or enumerate via GlobalLoader.
type AggregateReader interface {
    List(ctx context.Context, f AggregateFilter) (*Page[AggregateRef], error)
}
```

### In-memory reader (generic fallback)

```go
// stream/in_memory.go

package stream

import (
    "context"
    "slices"
    "github.com/larsartmann/go-cqrs-lite/core/event"
)

// InMemoryReader implements AggregateReader using GlobalLoader.
// Loads ALL events and filters in-memory. Suitable for testing
// and small datasets. NOT recommended for production SQL stores.
type InMemoryReader struct {
    loader event.GlobalLoader
}

// NewInMemoryReader creates a reader that enumerates via GlobalLoader.
func NewInMemoryReader(loader event.GlobalLoader) *InMemoryReader {
    return &InMemoryReader{loader: loader}
}

func (r *InMemoryReader) List(ctx context.Context, f AggregateFilter) (*Page[AggregateRef], error) {
    all, err := r.loader.LoadAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("stream in-memory list: %w", err)
    }

    // Group by aggregate, build refs, apply filters, paginate
    refs := buildRefs(all)
    refs = filterAndSort(refs, f)

    var limit uint = f.Limit
    if limit == 0 {
        limit = defaultPageSize // 20, configurable
    }

    if uint(len(refs)) > limit {
        return &Page[AggregateRef]{
            Items:   refs[:limit],
            HasMore: true,
        }, nil
    }

    return &Page[AggregateRef]{Items: refs, HasMore: false}, nil
}
```

### SQL reader (projection-backed)

```go
// stream/sql_reader.go

package stream

import (
    "context"
    "database/sql"
    "fmt"
)

// SQLReader queries a dedicated aggregates table.
// This table must be maintained separately (e.g., via AggregateProjection).
// The table is a CQRS read model — the event store knows nothing about it.
type SQLReader struct {
    db          *sql.DB
    tablePrefix string // e.g., "cqrs_" → table is "cqrs_stream_aggregates"
}

// NewSQLReader creates a reader that queries the aggregates projection table.
func NewSQLReader(db *sql.DB, tablePrefix string) *SQLReader {
    return &SQLReader{db: db, tablePrefix: tablePrefix}
}

func (r *SQLReader) tableName() string {
    return r.tablePrefix + "stream_aggregates"
}

func (r *SQLReader) List(ctx context.Context, f AggregateFilter) (*Page[AggregateRef], error) {
    query, args := r.buildQuery(f)
    // Execute query, build refs, return page
}
```

### Builder (the consumer-facing API)

```go
// stream/builder.go

package stream

import (
    "context"
    "database/sql"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

const (
    defaultPageSize = 20
    maxPageSize     = 100
)

// ListBuilder provides a fluent, discoverable API for aggregate listing.
// Created via NewList() for in-memory readers,
// or NewSQLList() for SQL-backed readers.
type ListBuilder struct {
    reader AggregateReader
    filter AggregateFilter
}

// NewList creates a builder using an InMemoryReader.
// Suitable for MemoryStore and any store implementing GlobalLoader.
// NOT recommended for production SQL stores — use NewSQLList instead.
func NewList(loader event.GlobalLoader) *ListBuilder {
    return &ListBuilder{
        reader: NewInMemoryReader(loader),
        filter: AggregateFilter{
            Limit:     defaultPageSize,
            Tombstone: TombstoneExclude,
        },
    }
}

// NewSQLList creates a builder using a SQLReader.
// Queries the aggregates projection table directly.
func NewSQLList(db *sql.DB, tablePrefix string) *ListBuilder {
    return &ListBuilder{
        reader: NewSQLReader(db, tablePrefix),
        filter: AggregateFilter{
            Limit:     defaultPageSize,
            Tombstone: TombstoneExclude,
        },
    }
}

// OfType filters to a specific aggregate type.
func (b *ListBuilder) OfType(t event.AggregateType) *ListBuilder {
    b.filter.Type = t
    return b
}

// After sets the cursor for the next page.
// Pass the last AggregateRef.ID from the previous Page.
func (b *ListBuilder) After(id id.AggregateID) *ListBuilder {
    b.filter.After = id
    return b
}

// PageSize sets the page size. Clamped to [1, maxPageSize].
func (b *ListBuilder) PageSize(n uint) *ListBuilder {
    switch {
    case n == 0:
        b.filter.Limit = defaultPageSize
    case n > maxPageSize:
        b.filter.Limit = maxPageSize
    default:
        b.filter.Limit = n
    }
    return b
}

// IncludeDeleted shows all aggregates, including tombstoned ones.
func (b *ListBuilder) IncludeDeleted() *ListBuilder {
    b.filter.Tombstone = TombstoneInclude
    return b
}

// OnlyDeleted shows only tombstoned aggregates.
func (b *ListBuilder) OnlyDeleted() *ListBuilder {
    b.filter.Tombstone = TombstoneOnly
    return b
}

// List executes the query.
func (b *ListBuilder) List(ctx context.Context) (*Page[AggregateRef], error) {
    return b.reader.List(ctx, b.filter)
}
```

**Why this is better than v2:**
- No `Q()` — `NewList()` and `NewSQLList()` are descriptive and typed
- No hardcoded defaults — `PageSize()` overrides, with clamping
- No `StreamLister` on Store — read model is entirely separate
- Backend-agnostic builder — same API for memory and SQL
- `uint` throughout — type-safe counts and limits

---

## Layer 3: Projection (for SQL read model)

The aggregates table is maintained by a projection that subscribes to the event bus. This is the CQRS pattern: event store writes, projection maintains read model.

```go
// stream/projection.go

package stream

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/larsartmann/go-cqrs-lite/core/event"
)

// AggregateProjection maintains the stream_aggregates table.
// Register it with projection.Runner to keep the read model in sync.
type AggregateProjection struct {
    db        *sql.DB
    tableName string
}

// NewAggregateProjection creates a projection that maintains the aggregates table.
// The table is created if it doesn't exist.
func NewAggregateProjection(db *sql.DB, tablePrefix string) (*AggregateProjection, error) {
    p := &AggregateProjection{
        db:        db,
        tableName: tablePrefix + "stream_aggregates",
    }
    if err := p.createTable(); err != nil {
        return nil, fmt.Errorf("create aggregates table: %w", err)
    }
    return p, nil
}

func (p *AggregateProjection) Name() string {
    return "stream.aggregate_projection"
}

func (p *AggregateProjection) EventTypes() []event.Type {
    return nil // Subscribe to all events
}

func (p *AggregateProjection) Handle(ctx context.Context, evt event.Event) error {
    // UPSERT: increment event_count, update version/last_event_at/tombstoned
    _, err := p.db.ExecContext(ctx,
        `INSERT INTO `+p.tableName+`
            (aggregate_type, aggregate_id, version, event_count, last_event_at, tombstoned)
         VALUES (?, ?, ?, 1, ?, ?)
         ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
            version = excluded.version,
            event_count = `+p.tableName+`.event_count + 1,
            last_event_at = excluded.last_event_at,
            tombstoned = excluded.tombstoned`,
        evt.AggregateType(), evt.AggregateID(), evt.Version(), evt.OccurredAt(), event.HasTombstone(evt),
    )
    return err
}
```

**Schema** (auto-created by projection):

```sql
CREATE TABLE IF NOT EXISTS cqrs_stream_aggregates (
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    version        INT  NOT NULL,
    event_count    INT  NOT NULL DEFAULT 0,
    last_event_at  TIMESTAMPTZ NOT NULL,
    tombstoned     BOOL NOT NULL DEFAULT FALSE,
    PRIMARY KEY (aggregate_type, aggregate_id)
);
```

**Why this is better than v2:**
- `SQLEventStore` is unchanged — no bloat, no new responsibilities
- Aggregates table is a **read model** — maintained by projection, not the store
- Tombstone status is pre-computed in the table — O(1) listing, no N+1
- The projection can be skipped for testing (use `InMemoryReader` instead)

---

## Tombstone lifecycle (bus middleware)

Same pattern as `signing` — bus middleware auto-marks tombstones on publish:

```go
// stream/middleware.go

package stream

import (
    "context"
    "fmt"
    "github.com/larsartmann/go-cqrs-lite/core/event"
)

// AutoTombstone returns PublishMiddleware that sets the tombstone metadata key
// on events whose type is in the provided delete type set.
//
// Usage:
//   bus.UsePublish(stream.AutoTombstone("user.deleted", "order.cancelled"))
func AutoTombstone(deleteTypes ...event.Type) event.PublishMiddleware {
    typeSet := make(map[event.Type]struct{}, len(deleteTypes))
    for _, t := range deleteTypes {
        typeSet[t] = struct{}{}
    }

    return func(next event.Publisher) event.Publisher {
        return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
            marked := make([]event.Event, 0, len(events))
            for _, evt := range events {
                if _, isDelete := typeSet[evt.Type()]; !isDelete {
                    marked = append(marked, evt)
                    continue
                }
                tombstoned, err := event.MarkTombstone(evt)
                if err != nil {
                    return fmt.Errorf("auto-tombstone %s: %w", evt.Type(), err)
                }
                marked = append(marked, tombstoned)
            }
            return next.Publish(ctx, marked...)
        })
    }
}
```

**Domain code is unchanged:**

```go
// Publish a "deleted" event normally — middleware auto-marks tombstone
bus.Publish(ctx, userDeletedEvent)

// Or manually mark tombstone for edge cases
marked, _ := event.MarkTombstone(evt)
```

---

## Usage examples

### Memory store (testing, small deployments)

```go
store := memory.NewMemoryStore()
bus := memory.NewMemoryBus()

// Auto-mark tombstones on publish
bus.UsePublish(stream.AutoTombstone("user.deleted"))

// List active users (uses GlobalLoader — loads all events, filters in-memory)
page, err := stream.NewList(store).
    OfType("User").
    PageSize(20).
    List(ctx)

// Next page
if page.HasMore {
    lastID := page.Items[len(page.Items)-1].ID
    page2, _ := stream.NewList(store).
        OfType("User").
        PageSize(20).
        After(lastID).
        List(ctx)
}

// Admin: list deleted users for GDPR purge
deleted, _ := stream.NewList(store).
    OfType("User").
    OnlyDeleted().
    PageSize(100).
    List(ctx)
```

### SQL store (production)

```go
// Write model — unchanged
store := storage.NewSQLEventStore(db)

// Read model — separate projection table
proj, _ := stream.NewAggregateProjection(db, "cqrs_")
runner, _ := projection.NewRunner(nil, bus, checkpointStore)
runner.Register(proj)

// List users (queries the projection table, O(limit))
page, _ := stream.NewSQLList(db, "cqrs_").
    OfType("User").
    PageSize(20).
    List(ctx)

// Deleted users
page, _ := stream.NewSQLList(db, "cqrs_").
    OfType("User").
    OnlyDeleted().
    List(ctx)
```

### Check if an aggregate is tombstoned

```go
ref := page.Items[0]
if ref.Tombstoned {
    // handle deleted aggregate
}
```

### Hard delete (GDPR — bypasses tombstone)

```go
// Store.Delete is unchanged — it's a hard delete
store.Delete(ctx, "User", userID)
```

---

## Module layout

```
core/event/
├── tombstone.go     # MarkTombstone, HasTombstone, MetadataKeyTombstone

stream/
├── go.mod
├── doc.go
├── types.go         # AggregateRef, Page[T], TombstonePolicy, AggregateFilter
├── reader.go        # AggregateReader interface
├── in_memory.go     # InMemoryReader (GlobalLoader fallback)
├── sql_reader.go    # SQLReader (queries projection table)
├── builder.go       # ListBuilder, NewList, NewSQLList
├── middleware.go     # AutoTombstone bus middleware
├── projection.go    # AggregateProjection (maintains aggregates table)
└── *_test.go
```

---

## v2 → v3 comparison

| Aspect | v2 | v3 |
|---|---|---|
| **Tombstone detection** | `TombstoneDetector` callback (loads all events per aggregate) | Metadata key on last event (O(1)) |
| **Listing location** | `StreamLister` on `Store` | Separate `AggregateReader` in `stream/` |
| **SQL efficiency** | `ListStreams` on `SQLEventStore` with aggregates table | Projection maintains aggregates table; `SQLReader` queries it |
| **Store bloat** | `SQLEventStore` gains aggregates table + `ListStreams` | `SQLEventStore` unchanged |
| **Pagination** | Mixed offset + cursor | Cursor-only (`After` + `HasMore`) |
| **Type safety** | `int` for counts/limits | `uint` for counts/limits |
| **Builder entry** | `Q(store)` with hardcoded defaults | `NewList(loader)`, `NewSQLList(db)` with overridable `PageSize()` |
| **Decorator safety** | `TombstoneStore` wrapper loses type assertions | Bus middleware (`AutoTombstone`) — no wrapping needed |
| **CQRS separation** | Partial (read methods on write model) | Clean (read model is entirely separate) |

---

## Why no Sink+Source split (for now)

The user raised: *"a Store should be composed of Sink and Source."* This is architecturally correct:

```
Sink   = Save + AppendBatch + Delete + Close
Source = Load + LoadFromVersion + LoadToVersion + LoadToTimestamp + LoadAll
Store  = Sink + Source
```

This would let `stream.NewList(source)` accept only the read side. And it would let consumers compose stores from separate sinks and sources (e.g., write to Kafka, read from Postgres).

**Why v3 doesn't require this:**
- Splitting `Store` is a major refactor touching every module, every test, every consumer
- v3 works within the existing `Store` interface — it just doesn't extend it
- The read model (`stream.AggregateReader`) is already separate from `Store`
- A future refactor can split `Store` without changing `stream/` at all

If `Store` is split later, `InMemoryReader` changes from `NewInMemoryReader(loader event.GlobalLoader)` to `NewInMemoryReader(source event.Source)` — a one-line change.

---

## Implementation order

1. `core/event/tombstone.go` — pure types, zero risk, no consumer changes
2. `stream/types.go` — `AggregateRef`, `Page[T]`, `TombstonePolicy`
3. `stream/reader.go` — `AggregateReader` interface
4. `stream/in_memory.go` — `InMemoryReader` (works with any `GlobalLoader` immediately)
5. `stream/builder.go` — `ListBuilder`, `NewList()`, `NewSQLList()`
6. `stream/middleware.go` — `AutoTombstone()` bus middleware
7. `stream/projection.go` — `AggregateProjection` for SQL read model
8. `stream/sql_reader.go` — `SQLReader` (queries aggregates table)
9. Tests for each layer

---

## Open questions

1. **Sorting order for listings:** `InMemoryReader` sorts by `(aggregate_type, aggregate_id)` for cursor stability. Is this acceptable, or should `AggregateFilter` include a `SortBy` field? (Leaning toward keeping it simple — one deterministic sort order.)

2. **`EventCount` in SQL:** The projection maintains `event_count` via `+1` on each event. If events are backfilled via `AppendBatch`, the projection won't see them (it only sees published events). Should `AggregateProjection` support a `Recount()` method that scans the events table and rebuilds the aggregates table?

3. **MemoryStore optimization:** `InMemoryReader` loads ALL events via `GlobalLoader.LoadAll()`. For a `MemoryStore` with 10K aggregates, this loads every event into memory just to build refs. Should `memory/` provide a `NewAggregateReader()` method that iterates the internal map directly? (Module graph: `memory → stream → core`. No cycle.)
