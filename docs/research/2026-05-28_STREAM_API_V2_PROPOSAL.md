# Stream API v2 Proposal: Composable Store Capabilities

> **Date:** 2026-05-28 | **Status:** Proposal v2 | **Supersedes:** `2026-05-28_STREAM_API_PROPOSAL.md`

## What was wrong with v1

The v1 proposal had five structural problems:

1. **Fat option structs** — `StreamOptions` and `EventOptions` are "God parameters" with 8–10 fields. They're not composable: you can't say "active users' created events" without understanding all the fields and their interactions.

2. **Tombstone detection is expensive** — `TombstoneDetector func(aggType, events) bool` loads the _entire event stream_ for every aggregate to determine if it's deleted. At 10K aggregates with 100 events each, listing page 1 loads 1M events just for tombstone checks.

3. **Store decorator loses type assertions** — `TombstoneStore` wraps `event.Store`. If the inner store implemented `TransactionalStore`, `BackwardsLoader`, or `PositionalLoader`, the wrapper silently drops them. This breaks `decider.Repository` which type-asserts to `TransactionalStore`.

4. **Reader is a new abstraction that duplicates Store** — A `Reader` that wraps a `Store` means two ways to read events. Consumers must learn which one to use when. The existing Go-idiomatic pattern in this codebase is small interfaces composed via type assertion (`BackwardsLoader`, `PositionalLoader`, `TransactionalStore`).

5. **Mixed pagination models** — `StreamOptions` uses offset pagination (Page/PageSize), `EventOptions` uses cursor pagination (AfterEventID). Two mental models for the same feature.

## v2 Design Principles

| #   | Principle                                    | How                                                           |
| --- | -------------------------------------------- | ------------------------------------------------------------- |
| 1   | **Extend existing Store via type assertion** | New small interfaces in `core/event/`, like `BackwardsLoader` |
| 2   | **Tombstone is metadata, not callbacks**     | Metadata key like signing uses, O(1) detection                |
| 3   | **Builder pattern for queries**              | Fluent, composable, impossible to construct invalid queries   |
| 4   | **One pagination model**                     | Cursor-based everywhere (ULIDs are natural cursors)           |
| 5   | **SQL stores implement natively**            | Efficient queries, not fallback in-memory filtering           |
| 6   | **No wrapper that loses types**              | Tombstone as store-level capability, not decorator            |

---

## Architecture

### Three layers, three homes

```
core/event/     ← Interfaces + types (Store extensions)
stream/         ← Builder + in-memory fallback adapter
storage/        ← SQL-native implementations
memory/         ← In-memory implementations
```

No single new module owns everything. Each layer lives where it belongs.

---

## Layer 1: `core/event/` — Store extensions

New interfaces alongside `BackwardsLoader`, `PositionalLoader`, `TransactionalStore`:

```go
// core/event/store.go — append to existing file

// AggregateRef is a lightweight reference to an aggregate stream.
// It carries enough information for listings without loading any events.
type AggregateRef struct {
    ID         id.AggregateID
    Type       AggregateType
    Version    Version
    EventCount int
    LastEvent  Event // nil if stream is empty (shouldn't happen in practice)
}

// IsTombstoned reports whether this aggregate has been soft-deleted.
// Checks the last event's metadata for the tombstone key.
func (r AggregateRef) IsTombstoned() bool {
    if r.LastEvent == nil {
        return false
    }
    return HasTombstone(r.LastEvent)
}

// --- Interfaces ---

// StreamLister enumerates aggregate streams.
// Stores implement this to support listing without loading full event streams.
// MemoryStore iterates its internal map. SQLEventStore queries an aggregates table.
type StreamLister interface {
    // ListStreams returns a page of aggregate references.
    // Pass a zero-value cursor for the first page.
    ListStreams(ctx context.Context, filter StreamFilter) (StreamPage, error)
}

// StreamFilter controls stream listing. Built via StreamQuery (see stream/ module).
type StreamFilter struct {
    AggregateType AggregateType
    AfterRef      AggregateRef // cursor: return streams after this one
    Limit         int
    Tombstone     TombstonePolicy
}

// StreamPage is a cursor-based page of aggregate references.
type StreamPage struct {
    Streams    []AggregateRef
    TotalCount int
    Next       *AggregateRef // nil if no more pages
}

// TombstonePolicy controls how soft-deleted aggregates appear.
type TombstonePolicy int

const (
    TombstoneExclude TombstonePolicy = iota // hide deleted (default)
    TombstoneInclude                        // show all, check IsTombstoned()
    TombstoneOnly                           // show only deleted
)
```

```go
// core/event/tombstone.go — new file

package event

// MetadataKeyTombstone is the metadata key for tombstone markers.
// When present with value "true", the aggregate is considered soft-deleted.
const MetadataKeyTombstone MetadataKey = "tombstone"

// Tombstone marks an aggregate as soft-deleted by setting metadata on its last event.
// Returns a new event with the tombstone flag attached.
func Tombstone(evt Event) (*ImmutableEvent, error) {
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
    md := evt.Metadata()
    if md == nil || md.Custom == nil {
        return false
    }
    return md.Custom[MetadataKeyTombstone] == "true"
}
```

**Why this is better than v1:**

- `AggregateRef.IsTombstoned()` is O(1) — checks one metadata field on the last event, not a callback over all events
- `TombstonePolicy` lives in `core/event/` because it's a store-level concept, not a separate module concern
- `StreamLister` follows the exact same pattern as `BackwardsLoader` — small interface, stores opt in via type assertion
- No wrapper: stores that implement `StreamLister` keep all their other interface implementations

---

## Layer 2: `stream/` — Builder + adapter

The builder provides a fluent query API that compiles to `StreamFilter`:

```go
// stream/query.go

// Query builds a StreamFilter for listing aggregate streams.
// Start with stream.Q(store), chain filters, call Streams() or Events().
type Query struct {
    store  event.Store
    filter event.StreamFilter
}

// Q creates a stream query builder. Pass any event.Store.
func Q(store event.Store) *Query {
    return &Query{
        store: store,
        filter: event.StreamFilter{
            Limit:     20,
            Tombstone: event.TombstoneExclude,
        },
    }
}

// Type filters to a specific aggregate type.
func (q *Query) Type(aggType event.AggregateType) *Query {
    q.filter.AggregateType = aggType
    return q
}

// PageSize sets the page size.
func (q *Query) PageSize(n int) *Query {
    q.filter.Limit = n
    return q
}

// After sets the cursor for the next page. Pass StreamPage.Next.
func (q *Query) After(ref *event.AggregateRef) *Query {
    q.filter.AfterRef = *ref
    return q
}

// IncludeDeleted shows all aggregates, including tombstoned ones.
func (q *Query) IncludeDeleted() *Query {
    q.filter.Tombstone = event.TombstoneInclude
    return q
}

// OnlyDeleted shows only tombstoned aggregates.
func (q *Query) OnlyDeleted() *Query {
    q.filter.Tombstone = event.TombstoneOnly
    return q
}

// Streams executes the query and returns a page of aggregate references.
// Falls back to in-memory enumeration if the store doesn't implement StreamLister.
func (q *Query) Streams(ctx context.Context) (*event.StreamPage, error) {
    if lister, ok := q.store.(event.StreamLister); ok {
        page, err := lister.ListStreams(ctx, q.filter)
        if err != nil {
            return nil, fmt.Errorf("stream listing: %w", err)
        }
        return &page, nil
    }

    return q.fallbackStreams(ctx)
}

// fallbackStreams uses GlobalLoader/LoadAll to enumerate in-memory.
// For stores that don't implement StreamLister.
func (q *Query) fallbackStreams(ctx context.Context) (*event.StreamPage, error) {
    // ... enumerate via GlobalLoader, apply filters in-memory
}
```

**Why this is better than v1:**

- **Impossible to construct invalid queries** — the builder only exposes valid operations
- **Discoverable** — IDE autocomplete shows `.Type()`, `.PageSize()`, `.IncludeDeleted()`, `.Streams()`
- **Composable** — chain any combination of filters
- **Graceful degradation** — if store doesn't implement `StreamLister`, falls back to in-memory enumeration via `GlobalLoader`

---

## Layer 3: `memory/` + `storage/` — Native implementations

### MemoryStore

```go
// memory/stream.go

// ListStreams enumerates the internal events map.
// Implements event.StreamLister.
func (s *MemoryStore) ListStreams(ctx context.Context, f event.StreamFilter) (event.StreamPage, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var refs []event.AggregateRef

    for key, events := range s.events {
        aggType, aggID := event.ParseStreamKey(key)
        // ... apply filters, build AggregateRef from last event
        ref := event.AggregateRef{
            ID: aggID, Type: aggType,
            Version: event.Version(len(events)),
            LastEvent: events[len(events)-1],
        }
        refs = append(refs, ref)
    }

    // Sort, paginate, return StreamPage
}
```

### SQLEventStore

The SQL implementation is where this design really shines. Instead of `SELECT DISTINCT` on the events table (expensive at scale), add a lightweight `aggregates` tracking table:

```sql
CREATE TABLE IF NOT EXISTS aggregates (
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    version        INT  NOT NULL DEFAULT 0,
    tombstoned     BOOL NOT NULL DEFAULT FALSE,
    last_event_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (aggregate_type, aggregate_id)
);
```

Updated `Save` maintains this table in the same transaction:

```go
// storage/event_store.go — updated Save

func (s *SQLEventStore) Save(ctx context.Context, ...) error {
    tx, _ := s.db.BeginTx(ctx, nil)
    // ... existing checkVersion + insertEvents ...

    // Update aggregates tracking table
    last := events[len(events)-1]
    _, err = tx.ExecContext(ctx, upsertAggregateQuery,
        aggregateType, aggregateID,
        expectedVersion.Int() + len(events),
        event.HasTombstone(last), // check tombstone metadata
        last.OccurredAt(),
    )

    return commitTx(tx)
}
```

`ListStreams` queries the aggregates table — O(limit) not O(total):

```go
// storage/stream.go

func (s *SQLEventStore) ListStreams(ctx context.Context, f event.StreamFilter) (event.StreamPage, error) {
    query := "SELECT aggregate_type, aggregate_id, version, tombstoned, last_event_at FROM aggregates WHERE 1=1"
    args := []any{}

    if f.AggregateType != "" {
        query += " AND aggregate_type = ?"
        args = append(args, f.AggregateType)
    }

    switch f.Tombstone {
    case event.TombstoneExclude:
        query += " AND tombstoned = FALSE"
    case event.TombstoneOnly:
        query += " AND tombstoned = TRUE"
    }

    if f.AfterRef.ID != (id.AggregateID{}) {
        query += " AND (aggregate_type, aggregate_id) > (?, ?)"
        args = append(args, f.AfterRef.Type, f.AfterRef.ID)
    }

    query += " ORDER BY aggregate_type, aggregate_id LIMIT ?"
    args = append(args, f.Limit + 1) // +1 to detect HasMore

    // ... execute, build StreamPage
}
```

**Why this is better than v1:**

- **No N+1 problem** — tombstone status is a column, not a callback per aggregate
- **No full scan** — aggregates table is indexed, paginated queries are O(limit)
- **Transactional consistency** — aggregates table updated in same tx as events
- **Migration path** — `ALTER TABLE` for existing deployments, `CREATE TABLE` for new ones

---

## Tombstone lifecycle

The full lifecycle uses metadata consistently, like signing:

```go
// 1. Consumer's domain code appends a "deleted" event normally
store.Save(ctx, "User", userID, []event.Event{
    userDeletedEvent, // normal domain event, nothing special
}, expectedVersion)

// 2. Bus middleware auto-marks tombstone on the last event
bus.UsePublish(tombstone.AutoMark("user.deleted", "order.cancelled"))

// 3. Or: store-level option marks tombstone on Delete calls
store := storage.NewSQLEventStore(db, storage.WithTombstoneOnDelete())

// 4. Listing respects tombstone automatically
page, _ := stream.Q(store).Type("User").Streams(ctx)            // excludes deleted
page, _ := stream.Q(store).Type("User").OnlyDeleted().Streams(ctx) // only deleted
page, _ := stream.Q(store).IncludeDeleted().Streams(ctx)          // everything
```

The `AutoMark` bus middleware — lives in `stream/`:

```go
// stream/middleware.go

// AutoMark returns PublishMiddleware that sets the tombstone metadata key
// on events whose type is in deleteTypes. This is the recommended way to
// mark tombstones — it's automatic, consistent, and happens before storage.
func AutoMark(deleteTypes ...event.Type) event.PublishMiddleware {
    typeSet := make(map[event.Type]bool, len(deleteTypes))
    for _, t := range deleteTypes {
        typeSet[t] = true
    }

    return func(next event.Publisher) event.Publisher {
        return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
            marked := make([]event.Event, 0, len(events))
            for _, evt := range events {
                if typeSet[evt.Type()] {
                    m, err := event.Tombstone(evt)
                    if err != nil {
                        return fmt.Errorf("tombstone event %s: %w", evt.Type(), err)
                    }
                    marked = append(marked, m)
                    continue
                }
                marked = append(marked, evt)
            }
            return next.Publish(ctx, marked...)
        })
    }
}
```

---

## Full usage examples

```go
// --- Setup ---
store := memory.NewMemoryStore()
bus := memory.NewMemoryBus()

// Auto-mark tombstones on publish
bus.UsePublish(stream.AutoMark("user.deleted", "order.cancelled"))

// --- Domain code (nothing changes) ---
bus.Publish(ctx, userCreatedEvent)
bus.Publish(ctx, userDeletedEvent) // auto-tombstoned by middleware

// --- Listing ---

// All active users
page, _ := stream.Q(store).
    Type("User").
    PageSize(20).
    Streams(ctx)

// Next page
if page.Next != nil {
    page2, _ := stream.Q(store).
        Type("User").
        PageSize(20).
        After(page.Next).
        Streams(ctx)
}

// Admin: show deleted users for GDPR purge
deleted, _ := stream.Q(store).
    Type("User").
    OnlyDeleted().
    PageSize(100).
    Streams(ctx)

// All aggregates across all types
all, _ := stream.Q(store).
    IncludeDeleted().
    PageSize(50).
    Streams(ctx)

// --- Check if a specific aggregate is tombstoned ---
ref := page.Streams[0]
if ref.IsTombstoned() {
    // handle deleted aggregate
}

// --- Hard delete (GDPR) ---
// Tombstoned aggregates can be hard-deleted via the normal Store interface
store.Delete(ctx, "User", userID)
```

---

## Module layout

```
core/event/
├── store.go         # + StreamLister, StreamFilter, StreamPage, AggregateRef
├── tombstone.go     # MetadataKeyTombstone, Tombstone(), HasTombstone(), TombstonePolicy

stream/
├── go.mod           # depends on core/event only
├── query.go         # Q(), Query builder
├── fallback.go      # in-memory fallback for stores without StreamLister
├── middleware.go     # AutoMark() bus middleware
├── query_test.go
├── middleware_test.go
└── fallback_test.go

memory/
├── stream.go        # ListStreams implementation

storage/
├── stream.go        # ListStreams implementation (aggregates table)
├── migrations.go    # CREATE TABLE aggregates + migration helper
```

---

## v1 → v2 comparison

| Aspect                  | v1                                        | v2                                          |
| ----------------------- | ----------------------------------------- | ------------------------------------------- |
| **Interface style**     | New `Reader` abstraction                  | Extend existing Store via type assertion    |
| **Tombstone detection** | Callback: loads all events                | Metadata key: O(1) check                    |
| **Query construction**  | Fat option struct                         | Builder pattern                             |
| **SQL efficiency**      | `SELECT DISTINCT` on events table         | Indexed `aggregates` table                  |
| **Store wrapping**      | Decorator (loses type assertions)         | No wrapper, stores implement directly       |
| **Pagination**          | Mixed (offset + cursor)                   | Cursor-based everywhere                     |
| **Tombstone marking**   | Manual or store wrapper                   | Bus middleware (auto)                       |
| **Module count**        | 1 new module                              | 1 new module + extensions to 2 existing     |
| **Consumer ergonomics** | `reader.Streams(ctx, StreamOptions{...})` | `stream.Q(store).Type("User").Streams(ctx)` |
| **Discoverability**     | Must know struct fields                   | IDE autocomplete on builder                 |

---

## Implementation order

1. `core/event/tombstone.go` — `Tombstone()`, `HasTombstone()`, `MetadataKeyTombstone` (pure types, zero risk)
2. `core/event/store.go` — `StreamLister`, `StreamFilter`, `StreamPage`, `AggregateRef`, `TombstonePolicy` (interfaces only)
3. `stream/query.go` — `Q()` builder + `fallbackStreams()` (works with any store immediately)
4. `stream/middleware.go` — `AutoMark()` (bus middleware)
5. `memory/stream.go` — `ListStreams` on MemoryStore (fast feedback loop)
6. `storage/stream.go` — aggregates table + `ListStreams` on SQLEventStore
7. Tests for each layer

---

## Open questions

1. **Aggregates table schema** — Should it be auto-created by `NewSQLEventStore` or require an explicit migration? Leaning toward auto-create with an opt-out option.

2. **EventPage alongside StreamPage** — Should we also add a `GlobalEventLister` interface for paginated event listing across streams? The current `GlobalLoader.LoadAll()` returns everything. A `GlobalEventLister` with cursor-based pagination + event type filters would be useful for audit logs and projection rebuilds. This could be a follow-up.

3. **Tombstone on Delete vs tombstone on publish** — `WithTombstoneOnDelete()` option on SQLEventStore would intercept `Store.Delete` calls and convert them to metadata updates. But this changes `Delete` semantics. Safer to use bus middleware and keep `Delete` as hard delete. Agreed?

4. **AggregateRef.LastEvent** — Carrying the last event in the ref is convenient for `IsTombstoned()` but means `ListStreams` returns partial event data. For SQL stores this means a JOIN. Alternative: just carry `Tombstoned bool` and `LastEventAt time.Time` as flat fields, computed at query time. This avoids the JOIN and keeps `AggregateRef` lightweight.
