# Stream API Proposal: Unified Listing & Tombstones

> **Date:** 2026-05-28 | **Status:** Proposal | **Modules affected:** new `stream/`

## Problem

`event.Store` is write-heavy (`Save`/`AppendBatch`/`Delete`) with only per-aggregate `Load*` reads. Two read-oriented capabilities are missing:

1. **Listing** — enumerating which aggregate streams exist (paginated, filtered)
2. **Tombstones** — soft-delete semantics so deleted aggregates can be listed/hidden on demand

These aren't separate concerns — tombstone detection *needs* stream enumeration, and listing *needs* tombstone awareness to show/hide deleted aggregates. A unified API avoids circular dependencies and duplicated types.

## Proposal: `stream/` module

New standalone module, depends on `core/event` only. Mirrors the `signing/` pattern (cross-cutting store concern → standalone module).

```
stream/
├── go.mod
├── doc.go
├── options.go       # StreamOptions, EventOptions, FilterFunc
├── cursor.go        # AggregateInfo, AggregatePage, EventPage
├── reader.go        # Reader interface (core abstraction)
├── store_reader.go  # StoreReader adapts any event.Store → Reader
├── tombstone.go     # TombstonePolicy, TombstoneDetector, IsTombstoned
├── middleware.go     # TombstoneStore decorator (soft-delete on Delete)
├── reader_test.go
├── tombstone_test.go
└── middleware_test.go
```

## Core interface

```go
// Reader provides read-only, paginated, filtered access to event streams.
// Compose it from any event.Store via NewStoreReader(store).
type Reader interface {
    // Streams lists aggregate streams matching the filter.
    Streams(ctx context.Context, opts StreamOptions) (*AggregatePage, error)

    // Events returns filtered, paginated events across matching streams.
    Events(ctx context.Context, opts EventOptions) (*EventPage, error)
}
```

## Options

```go
type StreamOptions struct {
    AggregateType event.AggregateType // filter by type, empty = all
    Page          uint
    PageSize      uint
    Tombstone     TombstonePolicy
}

type EventOptions struct {
    // Scope — at least one recommended
    AggregateType event.AggregateType
    AggregateID   id.AggregateID
    AfterEventID  id.EventID

    // Filters
    EventTypes  []event.Type
    FromTime    time.Time
    ToTime      time.Time
    FromVersion event.Version
    ToVersion   event.Version
    Metadata    map[event.MetadataKey]string

    // Tombstone
    Tombstone TombstonePolicy

    // Pagination
    Limit int // max events, 0 = default
}
```

## Tombstone design

```go
// TombstonePolicy controls how soft-deleted aggregates appear in listings.
type TombstonePolicy int

const (
    TombstoneExclude TombstonePolicy = iota // hide deleted (default)
    TombstoneInclude                        // show all, with Deleted flag
    TombstoneOnly                           // show only deleted
)

// TombstoneDetector determines tombstone status from an aggregate's event stream.
// Consumers provide their own logic (e.g., last event type is "user.deleted").
type TombstoneDetector func(aggType event.AggregateType, events []event.Event) bool

// IsTombstoned reports whether an aggregate has been soft-deleted.
func IsTombstoned(events []event.Event, deleteTypes map[event.Type]bool) bool
```

Key decision: **tombstone is a filter strategy, not a store wrapper**. The `TombstonePolicy` in options lets consumers query *any* store by providing a `TombstoneDetector`. The `TombstoneStore` decorator below is optional convenience.

## Cursor types

```go
type AggregateInfo struct {
    ID          id.AggregateID
    Type        event.AggregateType
    Version     event.Version
    EventCount  int
    LastEventAt time.Time
    Tombstoned  bool
}

type AggregatePage struct {
    Aggregates []AggregateInfo
    TotalCount uint
    Page       uint
    PageSize   uint
    TotalPages uint
}

type EventPage struct {
    Events     []event.Event
    TotalCount uint
    NextCursor id.EventID // pass as AfterEventID for next page
    HasMore    bool
}
```

Reuses the `query.Pagination` pattern (defaults, max page size, `HasNext`/`HasPrev`).

## Store adapter

```go
// StoreReader adapts any event.Store (+ optional GlobalLoader) into a Reader.
func NewStoreReader(store event.Store, opts ...ReaderOption) *StoreReader

type ReaderOption func(*StoreReader)

func WithTombstoneDetector(fn TombstoneDetector) ReaderOption

// WithDeleteTypes is shorthand for a type-set-based detector.
func WithDeleteTypes(types ...event.Type) ReaderOption
```

Implementation strategy by store type:

| Store | Streams() strategy | Events() strategy |
|---|---|---|
| `memory.MemoryStore` | Iterate `events` map keys, apply filters in-memory | `collectAllSorted()` + filter |
| `storage.SQLEventStore` | `SELECT DISTINCT aggregate_type, aggregate_id ... GROUP BY` | `SELECT * FROM events WHERE ... ORDER BY occurred_at` |
| Any `event.Store` | Fallback: load all via `GlobalLoader`, enumerate in-memory | Fallback: `GlobalLoader.LoadAll()` + filter |

## Store decorator (optional)

```go
// TombstoneStore wraps any event.Store so Delete() appends a tombstone
// event instead of removing events.
type TombstoneStore struct {
    event.Store
    detector TombstoneDetector
}

func Wrap(store event.Store, detector TombstoneDetector) *TombstoneStore

// Delete appends a tombstone marker event instead of deleting events.
func (t *TombstoneStore) Delete(ctx context.Context, aggType event.AggregateType, aggID id.AggregateID) error

// HardDelete performs the actual deletion. Intentionally named to require explicit opt-in.
func (t *TombstoneStore) HardDelete(ctx context.Context, aggType event.AggregateType, aggID id.AggregateID) error
```

## Usage examples

```go
// --- Setup ---
store := memory.NewMemoryStore()

reader := stream.NewStoreReader(store,
    stream.WithDeleteTypes("user.deleted", "order.cancelled"),
)

// --- List active user aggregates, page 2 ---
page, err := reader.Streams(ctx, stream.StreamOptions{
    AggregateType: event.MustParseAggregateType("User"),
    Page:          2,
    PageSize:      20,
    Tombstone:     stream.TombstoneExclude, // default
})

// --- List deleted orders for admin purge ---
deleted, err := reader.Streams(ctx, stream.StreamOptions{
    AggregateType: event.MustParseAggregateType("Order"),
    Tombstone:     stream.TombstoneOnly,
})

// --- All "user.created" events, cursor-paginated ---
page, err := reader.Events(ctx, stream.EventOptions{
    EventTypes:    []event.Type{"user.created"},
    AfterEventID:  lastSeenID,
    Limit:         100,
})

// --- Wrap store so Delete becomes soft-delete ---
ts := stream.Wrap(store, myDetector)
ts.Delete(ctx, "User", userID)       // appends tombstone marker
ts.HardDelete(ctx, "User", userID)   // actual deletion, explicit opt-in
```

## Why one module, not two

| Aspect | Combined `stream/` | Separate `listing/` + `tombstone/` |
|---|---|---|
| Dependency | Tombstone needs stream enumeration. Listing needs tombstone awareness. Unified avoids circular dep. | Circular dependency or shared `core/` types for both to depend on. |
| Consumer ergonomics | One import, one `Reader`, compose options. | Two imports, two objects, consumer wires them together. |
| Size | ~8 files, ~500 lines. Not oversized. | Two ~250-line modules with duplicated cursor types. |
| Precedent | `projection/` bundles replay + checkpoint + builder. `signing/` bundles sign + verify + middleware. | No existing pattern of two modules that depend on each other. |

## Implementation order

1. `options.go`, `cursor.go`, `tombstone.go` — pure types, no dependencies beyond `core/event`
2. `reader.go` — interface definition
3. `store_reader.go` — `memory/` adapter first (fastest to test), `storage/` adapter second
4. `middleware.go` — `TombstoneStore` decorator
5. Tests for each layer

## Open questions

- **Tombstone marker format**: Should `TombstoneStore.Delete` append a synthetic event, or set a metadata flag on the last event? Synthetic event is cleaner (audit trail) but changes the stream version.
- **SQL schema**: `SQLEventStore` doesn't currently have an aggregate-level metadata table. Listing requires either a `SELECT DISTINCT` on the events table, or a new `aggregates` tracking table. The latter is more efficient but adds schema complexity.
- **Event counts in `AggregateInfo`**: For `memory/` this is cheap. For `storage/`, `COUNT(*)` per aggregate may be expensive. Consider making it optional or lazy.
