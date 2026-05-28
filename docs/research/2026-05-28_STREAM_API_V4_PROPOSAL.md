# Stream API v4 Proposal: Sink/Source Decomposition + Read Model

> **Date:** 2026-05-28 | **Status:** Proposal v4 | **Supersedes:** `2026-05-28_STREAM_API_V3_PROPOSAL.md`

## Principles (non-negotiable)

1. **History is immutable** — No `Delete`. No hard deletes. No exceptions.
2. **Sink is write, Source is read** — `Store = Sink + Source`. Separate concerns, separate modules.
3. **Read models are projections** — They subscribe to the bus, not query the store.
4. **Tombstones are domain events** — `AutoTombstone` middleware detects domain event types. Consumers list their delete/rebirth event types.
5. **Tri-state tombstone** — `Active`, `Tombstoned`, `Undetermined`. No `bool`.
6. **`uint` for all counts** — Negative values are impossible states.
7. **Cursor pagination only** — `Page[T]` with `HasMore`, no `TotalCount`.
8. **Composable, not forced** — Consumers import what they need. No all-in-one monster.

---

## Phase 1: Decompose `Store` into `Sink` + `Source`

### New interfaces in `core/event/store.go`

```go
// Sink is the write side of event persistence.
// Appends events, never reads, never deletes.
type Sink interface {
    io.Closer

    // Save appends events with optimistic concurrency check.
    Save(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        events []Event,
        expectedVersion Version,
    ) error

    // AppendBatch appends without concurrency checks.
    // For bulk imports, replays, migrations.
    AppendBatch(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        events []Event,
    ) error
}

// Source is the read side of event persistence.
// Loads events, never writes.
type Source interface {
    io.Closer

    // Load all events for an aggregate.
    Load(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
    ) ([]Event, error)

    // LoadFromVersion returns events starting after version (exclusive).
    LoadFromVersion(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        version Version,
    ) ([]Event, error)

    // LoadToVersion returns events up to and including maxVersion.
    LoadToVersion(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxVersion Version,
    ) ([]Event, error)

    // LoadToTimestamp returns events where OccurredAt <= maxTime.
    LoadToTimestamp(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        maxTime time.Time,
    ) ([]Event, error)
}
```

### `Store` becomes composite

```go
// Store is the composite of Sink + Source.
// All existing implementations continue to satisfy Store.
type Store interface {
    Sink
    Source
}
```

### Remove `Delete`

`Store.Delete` is removed entirely. History is append-only. For GDPR compliance, consumers use crypto-shredding or a separate `redaction` module (out of scope for this proposal).

### Existing read extensions become Source extensions

```go
// GlobalSource loads all events across aggregates.
type GlobalSource interface {
    Source
    LoadAll(ctx context.Context) ([]Event, error)
}

// PositionalSource extends GlobalSource with cursor-based loading.
type PositionalSource interface {
    GlobalSource
    LoadAllFromPosition(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// BackwardsSource loads events in reverse order.
type BackwardsSource interface {
    Source
    LoadBackwards(ctx context.Context, aggType AggregateType, aggID id.AggregateID) ([]Event, error)
}

// TransactionalSink extends Sink with atomic save+outbox.
type TransactionalSink interface {
    Sink
    SaveWithOutbox(...)
}

// StreamSource streams events without loading all into memory.
type StreamSource interface {
    Source
    LoadStream(ctx context.Context, aggType AggregateType, aggID id.AggregateID) (EventStream, error)
}
```

### Migration path

| Before (v3) | After (v4) |
|---|---|
| `event.Store` | `event.Store` (still exists, now `Sink + Source`) |
| `event.GlobalLoader` | `event.GlobalSource` (extends Source) |
| `event.PositionalLoader` | `event.PositionalSource` (extends GlobalSource) |
| `event.BackwardsLoader` | `event.BackwardsSource` (extends Source) |
| `event.TransactionalStore` | `event.TransactionalSink` (extends Sink) |
| `Store.Delete()` | **REMOVED** |
| `MemoryStore` implements `Store` | Still implements `Store` (no consumer changes) |
| `SQLEventStore` implements `Store` | Still implements `Store` (no consumer changes) |

**Backward compatibility:** `Store` is still `Store`. The renamed interfaces (`GlobalLoader` → `GlobalSource`) are breaking for consumers who type-assert. Since backward compatibility isn't a priority, we rename cleanly.

---

## Phase 2: Tombstone design

### Tri-state enum

```go
// core/event/tombstone.go

package event

// TombstoneStatus represents the soft-delete state of an aggregate.
type TombstoneStatus int

const (
    TombstoneActive TombstoneStatus = iota       // aggregate is live
    TombstoneTombstoned                          // aggregate is soft-deleted
    TombstoneUndetermined                        // cannot determine (no detector, no metadata)
)

func (s TombstoneStatus) String() string {
    switch s {
    case TombstoneActive:
        return "active"
    case TombstoneTombstoned:
        return "tombstoned"
    case TombstoneUndetermined:
        return "undetermined"
    default:
        return fmt.Sprintf("TombstoneStatus(%d)", s)
    }
}

// IsActive reports whether the aggregate is active (not tombstoned).
func (s TombstoneStatus) IsActive() bool { return s == TombstoneActive }

// IsTombstoned reports whether the aggregate is soft-deleted.
func (s TombstoneStatus) IsTombstoned() bool { return s == TombstoneTombstoned }

// IsKnown reports whether the status is determinable (not Undetermined).
func (s TombstoneStatus) IsKnown() bool { return s != TombstoneUndetermined }
```

### Metadata keys

```go
// MetadataKeyTombstone marks an event as a tombstone action.
const MetadataKeyTombstone MetadataKey = "tombstone"

// MetadataKeyRebirth marks an event as undoing a tombstone.
const MetadataKeyRebirth MetadataKey = "rebirth"
```

### Detection logic

```go
// DetectTombstone inspects an event stream and returns the tombstone status.
// Returns Undetermined if the stream is empty or no tombstone/rebirth metadata is found.
func DetectTombstone(events []Event) TombstoneStatus {
    if len(events) == 0 {
        return TombstoneUndetermined
    }

    last := events[len(events)-1]
    md := last.Metadata()
    if md == nil || md.Custom == nil {
        return TombstoneUndetermined
    }

    // Rebirth takes precedence (newest event wins)
    if md.Custom[MetadataKeyRebirth] == "true" {
        return TombstoneActive
    }
    if md.Custom[MetadataKeyTombstone] == "true" {
        return TombstoneTombstoned
    }

    return TombstoneUndetermined
}
```

### Marking helpers

```go
// MarkTombstone copies an event and sets the tombstone metadata key.
func MarkTombstone(evt Event) (*ImmutableEvent, error)

// MarkRebirth copies an event and sets the rebirth metadata key.
func MarkRebirth(evt Event) (*ImmutableEvent, error)
```

---

## Phase 3: `stream/` — Read Model Module

### Types

```go
// stream/types.go

package stream

import (
    "time"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// AggregateRef is a lightweight identity reference to an aggregate stream.
// No derived state. Status is computed separately by the reader.
type AggregateRef struct {
    ID          id.AggregateID
    Type        event.AggregateType
    Version     event.Version
    EventCount  uint
    LastEventAt time.Time
}

// AggregateStatus pairs an aggregate with its computed tombstone state.
type AggregateStatus struct {
    Ref    AggregateRef
    Status event.TombstoneStatus
}

// Page is a cursor-based page of results.
// No TotalCount — append-only logs make counts stale and expensive.
type Page[T any] struct {
    Items   []T
    HasMore bool
}

// TombstonePolicy controls visibility.
type TombstonePolicy int

const (
    TombstoneExclude TombstonePolicy = iota // default: hide tombstoned
    TombstoneInclude                        // show all, with Status
    TombstoneOnly                           // show only tombstoned
)
```

### AggregateReader interface

```go
// stream/aggregate_reader.go

package stream

import "context"

// AggregateReader queries aggregate streams.
type AggregateReader interface {
    // List returns a page of aggregate references.
    // Tombstoned aggregates are excluded by default.
    List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error)

    // ListWithStatus returns aggregates with their computed tombstone status.
    // Use this when you need to know which aggregates are tombstoned.
    ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error)
}

// ListOptions controls listing queries.
type ListOptions struct {
    Type      event.AggregateType
    After     id.AggregateID // cursor
    Limit     uint
    Tombstone TombstonePolicy
}
```

### EventReader interface (comprehensive)

```go
// stream/event_reader.go

package stream

import "context"

// EventReader queries events across streams.
type EventReader interface {
    // Read returns events matching the filter, cursor-paginated.
    Read(ctx context.Context, opts ReadOptions) (*Page[event.Event], error)
}

// ReadOptions controls event queries.
type ReadOptions struct {
    // Scope — narrow to specific aggregates or types
    AggregateType event.AggregateType // optional
    AggregateID   id.AggregateID      // optional
    EventTypes    []event.Type        // optional

    // Time range
    FromTime time.Time // inclusive
    ToTime   time.Time // inclusive

    // Version range (per-aggregate)
    FromVersion event.Version
    ToVersion   event.Version

    // Cursor
    AfterEventID id.EventID

    // Pagination
    Limit uint
}
```

### Builder

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

// AggregateQueryBuilder provides a fluent API for aggregate listings.
type AggregateQueryBuilder struct {
    reader AggregateReader
    opts   ListOptions
}

// NewAggregateQuery creates a builder for aggregate listings.
func NewAggregateQuery(reader AggregateReader) *AggregateQueryBuilder {
    return &AggregateQueryBuilder{
        reader: reader,
        opts: ListOptions{
            Limit:     defaultPageSize,
            Tombstone: TombstoneExclude,
        },
    }
}

func (b *AggregateQueryBuilder) OfType(t event.AggregateType) *AggregateQueryBuilder {
    b.opts.Type = t
    return b
}

func (b *AggregateQueryBuilder) After(id id.AggregateID) *AggregateQueryBuilder {
    b.opts.After = id
    return b
}

func (b *AggregateQueryBuilder) PageSize(n uint) *AggregateQueryBuilder {
    switch {
    case n == 0:
        b.opts.Limit = defaultPageSize
    case n > maxPageSize:
        b.opts.Limit = maxPageSize
    default:
        b.opts.Limit = n
    }
    return b
}

func (b *AggregateQueryBuilder) IncludeDeleted() *AggregateQueryBuilder {
    b.opts.Tombstone = TombstoneInclude
    return b
}

func (b *AggregateQueryBuilder) OnlyDeleted() *AggregateQueryBuilder {
    b.opts.Tombstone = TombstoneOnly
    return b
}

func (b *AggregateQueryBuilder) List(ctx context.Context) (*Page[AggregateRef], error) {
    return b.reader.List(ctx, b.opts)
}

func (b *AggregateQueryBuilder) ListWithStatus(ctx context.Context) (*Page[AggregateStatus], error) {
    return b.reader.ListWithStatus(ctx, b.opts)
}
```

### In-memory readers (generic fallback)

```go
// stream/in_memory.go

package stream

import "context"

// InMemoryAggregateReader implements AggregateReader using a Source.
// Enumerates all streams, applies filters in-memory.
type InMemoryAggregateReader struct {
    source event.Source
}

func NewInMemoryAggregateReader(source event.Source) *InMemoryAggregateReader {
    return &InMemoryAggregateReader{source: source}
}

func (r *InMemoryAggregateReader) List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error) {
    // Load all streams, build refs, filter, paginate
}

func (r *InMemoryAggregateReader) ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error) {
    // Same as List, but also calls event.DetectTombstone on each stream
}
```

### SQL readers (projection-backed)

```go
// stream/sql_reader.go

package stream

import "context"
import "database/sql"

// SQLAggregateReader queries a projection table maintained by AggregateProjection.
type SQLAggregateReader struct {
    db        *sql.DB
    tableName string
}

func NewSQLAggregateReader(db *sql.DB, tablePrefix string) *SQLAggregateReader {
    return &SQLAggregateReader{
        db:        db,
        tableName: tablePrefix + "stream_aggregates",
    }
}

func (r *SQLAggregateReader) List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error) {
    // Query aggregates projection table, apply tombstone filter, paginate
}

func (r *SQLAggregateReader) ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error) {
    // Same, but joins with tombstone status column
}
```

### AggregateProjection (maintains read model table)

```go
// stream/projection.go

package stream

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/larsartmann/go-cqrs-lite/core/event"
)

// AggregateProjection maintains the stream_aggregates read model table.
// Register with projection.Runner to keep it in sync.
type AggregateProjection struct {
    db        *sql.DB
    tableName string
}

func NewAggregateProjection(db *sql.DB, tablePrefix string) (*AggregateProjection, error) {
    p := &AggregateProjection{db: db, tableName: tablePrefix + "stream_aggregates"}
    if err := p.createTable(); err != nil {
        return nil, fmt.Errorf("create aggregates table: %w", err)
    }
    return p, nil
}

func (p *AggregateProjection) Name() string { return "stream.aggregate_projection" }

func (p *AggregateProjection) EventTypes() []event.Type { return nil } // all events

func (p *AggregateProjection) Handle(ctx context.Context, evt event.Event) error {
    status := event.DetectTombstone([]event.Event{evt})
    _, err := p.db.ExecContext(ctx,
        `INSERT INTO `+p.tableName+`
            (aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
         VALUES (?, ?, ?, 1, ?, ?)
         ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
            version = excluded.version,
            event_count = `+p.tableName+`.event_count + 1,
            last_event_at = excluded.last_event_at,
            tombstone_status = excluded.tombstone_status`,
        evt.AggregateType(), evt.AggregateID(), evt.Version(), evt.OccurredAt(), status.String(),
    )
    return err
}
```

### Bus middleware: AutoTombstone

```go
// stream/middleware.go

package stream

import (
    "context"
    "fmt"
    "github.com/larsartmann/go-cqrs-lite/core/event"
)

// AutoTombstone returns PublishMiddleware that marks tombstone/rebirth metadata
// on events whose type is in the configured sets.
//
// Usage:
//   bus.UsePublish(stream.AutoTombstone(
//       []event.Type{"user.deleted", "order.cancelled"},   // tombstone types
//       []event.Type{"user.reactivated", "order.restored"}, // rebirth types
//   ))
func AutoTombstone(deleteTypes, rebirthTypes []event.Type) event.PublishMiddleware {
    deletes := makeTypeSet(deleteTypes)
    rebirths := makeTypeSet(rebirthTypes)

    return func(next event.Publisher) event.Publisher {
        return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
            marked := make([]event.Event, 0, len(events))
            for _, evt := range events {
                switch {
                case deletes[evt.Type()]:
                    m, err := event.MarkTombstone(evt)
                    if err != nil {
                        return fmt.Errorf("auto-tombstone %s: %w", evt.Type(), err)
                    }
                    marked = append(marked, m)
                case rebirths[evt.Type()]:
                    m, err := event.MarkRebirth(evt)
                    if err != nil {
                        return fmt.Errorf("auto-rebirth %s: %w", evt.Type(), err)
                    }
                    marked = append(marked, m)
                default:
                    marked = append(marked, evt)
                }
            }
            return next.Publish(ctx, marked...)
        })
    }
}
```

---

## Phase 4: Module layout

```
core/event/
├── store.go            # Sink, Source, Store, GlobalSource, PositionalSource, BackwardsSource, TransactionalSink
├── tombstone.go        # TombstoneStatus, DetectTombstone, MarkTombstone, MarkRebirth, MetadataKeyTombstone, MetadataKeyRebirth

stream/
├── go.mod
├── doc.go
├── types.go            # AggregateRef, AggregateStatus, Page[T], TombstonePolicy, ListOptions, ReadOptions
├── aggregate_reader.go # AggregateReader interface
├── event_reader.go     # EventReader interface
├── builder.go          # AggregateQueryBuilder, NewAggregateQuery
├── in_memory.go        # InMemoryAggregateReader, InMemoryEventReader
├── sql_reader.go       # SQLAggregateReader, SQLEventReader
├── projection.go       # AggregateProjection
├── middleware.go        # AutoTombstone
└── *_test.go
```

---

## Usage examples

### Setup (in-memory, testing)

```go
store := memory.NewMemoryStore() // implements Store = Sink + Source
bus := memory.NewMemoryBus()

// Auto-mark tombstones and rebirths on publish
bus.UsePublish(stream.AutoTombstone(
    []event.Type{"user.deleted", "order.cancelled"},
    []event.Type{"user.reactivated", "order.restored"},
))

// Read model
reader := stream.NewInMemoryAggregateReader(store)

// List active users
page, _ := stream.NewAggregateQuery(reader).
    OfType("User").
    PageSize(20).
    List(ctx)

// List with status (includes tombstone state)
statusPage, _ := stream.NewAggregateQuery(reader).
    OfType("User").
    IncludeDeleted().
    ListWithStatus(ctx)

// Check status
for _, s := range statusPage.Items {
    if s.Status.IsTombstoned() {
        fmt.Printf("User %s is deleted\n", s.Ref.ID)
    }
}
```

### Setup (SQL, production)

```go
// Write model
sink := storage.NewSQLEventSink(db)   // NEW: separate sink
source := storage.NewSQLEventSource(db) // NEW: separate source
store := storage.NewSQLStore(sink, source) // convenience wrapper

// Or just use sink+source directly:
// repo := decider.NewRepository(sink, bus)

// Read model projection
proj, _ := stream.NewAggregateProjection(db, "cqrs_")
runner, _ := projection.NewRunner(nil, bus, checkpoint)
runner.Register(proj)

// Read model queries
reader := stream.NewSQLAggregateReader(db, "cqrs_")

page, _ := stream.NewAggregateQuery(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

### Domain code (unchanged)

```go
// Publish a "deleted" event — middleware auto-marks tombstone metadata
bus.Publish(ctx, userDeletedEvent)

// Publish a "reactivated" event — middleware auto-marks rebirth metadata
bus.Publish(ctx, userReactivatedEvent)
```

---

## v3 → v4 comparison

| Aspect | v3 | v4 |
|---|---|---|
| **Store decomposition** | `Store` is monolithic | `Store = Sink + Source` |
| **Delete** | Still on `Store` | **Removed entirely** |
| **Read interfaces** | `GlobalLoader`, `PositionalLoader` | `GlobalSource`, `PositionalSource` (extend Source) |
| **Tombstone** | `bool` on `AggregateRef` | `TombstoneStatus` enum, separate from `AggregateRef` |
| **Rebirth** | Not addressed | `MarkRebirth`, `MetadataKeyRebirth`, `AutoTombstone` with rebirth types |
| **AggregateRef** | Embedded `Tombstoned bool` | Pure identity, no derived state |
| **AggregateStatus** | Didn't exist | New type: `Ref + Status` |
| **EventReader** | Not in scope | `EventReader` + `ReadOptions` for cross-stream event queries |
| **SQL read model** | Projection concept mentioned | Full `AggregateProjection` + `SQLAggregateReader` implementation |
| **Builder** | `NewList()`, `NewSQLList()` | `NewAggregateQuery(reader)` — accepts any `AggregateReader` |

---

## Implementation order

1. `core/event/tombstone.go` — `TombstoneStatus`, `DetectTombstone`, `MarkTombstone`, `MarkRebirth`
2. `core/event/store.go` — Decompose into `Sink` + `Source`, update `Store`, rename loader interfaces
3. Update all implementations (`memory/`, `storage/`, `testhelpers/`) to satisfy new interfaces
4. Remove `Delete` from all stores, update tests
5. `stream/types.go` — `AggregateRef`, `AggregateStatus`, `Page[T]`, `ListOptions`, `ReadOptions`
6. `stream/aggregate_reader.go` — `AggregateReader` interface
7. `stream/event_reader.go` — `EventReader` interface
8. `stream/in_memory.go` — `InMemoryAggregateReader`, `InMemoryEventReader`
9. `stream/builder.go` — `AggregateQueryBuilder`
10. `stream/middleware.go` — `AutoTombstone`
11. `stream/projection.go` — `AggregateProjection`
12. `stream/sql_reader.go` — `SQLAggregateReader`, `SQLEventReader`
13. Tests for each layer

---

## Open questions (after implementation starts)

1. **Sink/Source constructors** — Should `storage/` provide `NewSQLEventSink` and `NewSQLEventSource` as separate constructors, or a combined `NewSQLStore` that returns both? (Leaning toward: both. Separate for flexibility, combined for convenience.)

2. **EventReader for in-memory** — `InMemoryEventReader` needs to scan all streams. For `MemoryStore` this is efficient (internal map). For `SQLEventSource` it would need `LoadAll` or a query. Should `EventReader` be a separate concern from `AggregateReader`? (Yes — they have different use cases.)

3. **Tombstone column in SQL** — Store as string (`"active"`, `"tombstoned"`, `"undetermined"`) or integer (0, 1, 2)? Integer is more efficient but string is more debuggable. (Leaning toward integer with constants.)

4. **Crypto-shredding for GDPR** — Out of scope for this proposal, but worth noting: if a consumer truly needs data removal, they should encrypt per-user and delete the key. This is a separate `crypto/` or `redaction/` module concern, not a store concern.
