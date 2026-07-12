# Foundation Plan: Sink/Source + Journal + Stream Read Model

**Date:** 2026-05-28
**Status:** Implementation plan
**Prerequisite reading:**

- `2026-05-28_SINK_SOURCE_SPLIT_AND_GENERIC_BOUNDARIES.md`
- `2026-05-28_JOURNAL_NAMING_PROPOSAL.md`
- `2026-05-28_STREAM_API_V4_PROPOSAL.md`
- `2026-05-28_STREAM_API_V4_SELF_CRITIQUE.md`

---

## Executive Summary

The foundation needs three structural changes:

1. **Decompose `Store` into `Sink` + `Source`** — capability separation, enables read replica routing
2. **Rename `GlobalLoader` → `Journal`** — standard ES terminology, clearer intent
3. **Add `stream/` read model module** — CQRS-compliant aggregate listing with tombstones

These are sequential but mostly additive. Each step improves the foundation without breaking the previous.

---

## Guiding Principles

| #   | Principle                                                                                    | Source      |
| --- | -------------------------------------------------------------------------------------------- | ----------- |
| 1   | **History is immutable** — No `Delete`. Append-only.                                         | v4          |
| 2   | **Sink is write, Source is read** — Separate concerns, separate deployment.                  | Zig + v4    |
| 3   | **Transport stays untyped** — Generics at boundaries, not in persistence.                    | Watermill   |
| 4   | **Read models are projections** — They subscribe to the bus, not query the store.            | v3/v4       |
| 5   | **Composable, not forced** — Small interfaces, type assertions, no wrappers that lose types. | v2 critique |
| 6   | **`uint` for counts** — Negative values are impossible states.                               | v3          |
| 7   | **Cursor pagination only** — `Page[T]` with `HasMore`, no `TotalCount`.                      | v4          |
| 8   | **Additive first, breaking last** — Implement new interfaces before removing old ones.       | v4 critique |

---

## Phase 1: Sink + Source Decomposition

### 1.1 Add `Sink` and `Source` interfaces

**File:** `core/event/store.go`

```go
// Sink is the write side of event persistence.
// Appends events, never reads.
type Sink interface {
    io.Closer

    Save(
        ctx context.Context,
        aggregateType AggregateType,
        aggregateID id.AggregateID,
        events []Event,
        expectedVersion Version,
    ) error

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

    Load(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) ([]Event, error)
    LoadFromVersion(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID, version Version) ([]Event, error)
    LoadToVersion(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID, maxVersion Version) ([]Event, error)
    LoadToTimestamp(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID, maxTime time.Time) ([]Event, error)
}

// Store composes Sink + Source.
// All existing implementations continue to satisfy Store automatically.
type Store interface {
    Sink
    Source
}
```

### 1.2 Rename read extensions to Source extensions

| Before               | After                                                                             |
| -------------------- | --------------------------------------------------------------------------------- |
| `GlobalLoader`       | `GlobalSource`                                                                    |
| `PositionalLoader`   | `PositionalSource`                                                                |
| `BackwardsLoader`    | `BackwardsSource`                                                                 |
| `StreamLoader`       | `StreamSource`                                                                    |
| `TransactionalStore` | `TransactionalSink` (extends Sink) + keep `TransactionalStore` as composite alias |

```go
// GlobalSource loads all events across aggregates.
type GlobalSource interface {
    Source
    ReadAll(ctx context.Context) ([]Event, error)  // was LoadAll
}

// PositionalSource extends GlobalSource with cursor-based loading.
type PositionalSource interface {
    GlobalSource
    ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)  // was LoadAllFromPosition
}
```

### 1.3 Update wrapper types to depend on minimal interfaces

| Wrapper              | Current accepts | Should accept                              |
| -------------------- | --------------- | ------------------------------------------ |
| `VersionedStore`     | `Store`         | `Source` → rename to `VersionedSource`     |
| `StoreStreamAdapter` | `Store`         | `Source` → rename to `SourceStreamAdapter` |
| `projection.Runner`  | `Store`         | `Source` + `PositionalSource`              |

### 1.4 Update `decider.Repository`

```go
type Repository[State any] struct {
    sink          event.Sink
    source        event.Source
    publisher     event.Publisher
    outbox        event.Outbox
    snapshotStore event.SnapshotStore
    codec         event.Codec
    snapshotStrategy event.SnapshotStrategy
    enricher      event.ContextEnricher
    decider       Decider[State]
}

// NewRepository takes Sink + Source separately.
func NewRepository[State any](
    sink event.Sink,
    source event.Source,
    publisher event.Publisher,
    decider Decider[State],
    opts ...RepositoryOption[State],
) (*Repository[State], error)

// NewRepositoryWithStore convenience constructor.
func NewRepositoryWithStore[State any](
    store event.Store,
    publisher event.Publisher,
    decider Decider[State],
    opts ...RepositoryOption[State],
) (*Repository[State], error) {
    return NewRepository(store, store, publisher, decider, opts...)
}
```

### 1.5 Add `FakeSink` + `FakeSource` to testhelpers

Extract from `FakeStore` so tests can depend on minimal interfaces.

### 1.6 Verify no implementation changes needed

`MemoryStore`, `SQLEventStore`, `PebbleEventStore`, `FakeStore` already implement all methods. Just add compile-time assertions:

```go
var _ event.Sink = (*memory.MemoryStore)(nil)
var _ event.Source = (*memory.MemoryStore)(nil)
```

---

## Phase 2: Journal Naming

### 2.1 Rename `GlobalLoader` → `Journal`

```go
// Journal is the complete, ordered, append-only log of all domain events.
// Standard event sourcing term for cross-aggregate, time-ordered replay.
type Journal interface {
    ReadAll(ctx context.Context) ([]Event, error)
}

// SeekableJournal extends Journal with position-based loading.
// Enables efficient projection catch-up without loading all events.
type SeekableJournal interface {
    Journal
    ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}
```

### 2.2 Update all references

- `core/event/store.go` — interface definitions
- `memory/store_load.go` — method names (`LoadAll` → `ReadAll`, `LoadAllFromPosition` → `ReadFrom`)
- `storage/event_store.go` — method names
- `projection/runner.go` — field type and constructor
- All tests

### 2.3 Keep `GlobalLoader` as deprecated alias (one release)

```go
// Deprecated: use Journal instead.
type GlobalLoader = Journal

// Deprecated: use SeekableJournal instead.
type PositionalLoader = SeekableJournal
```

---

## Phase 3: Tombstone Foundation

### 3.1 Add tombstone metadata to `core/event/`

**File:** `core/event/tombstone.go`

```go
package event

// TombstoneStatus represents the soft-delete state of an aggregate.
type TombstoneStatus int

const (
    TombstoneActive       TombstoneStatus = iota // aggregate is live
    TombstoneTombstoned                            // aggregate is soft-deleted
    TombstoneUndetermined                          // cannot determine (no detector, no metadata)
)

func (s TombstoneStatus) String() string { ... }
func (s TombstoneStatus) IsActive() bool { return s == TombstoneActive }
func (s TombstoneStatus) IsTombstoned() bool { return s == TombstoneTombstoned }
func (s TombstoneStatus) IsKnown() bool { return s != TombstoneUndetermined }

// Metadata keys.
const MetadataKeyTombstone MetadataKey = "tombstone"
const MetadataKeyRebirth   MetadataKey = "rebirth"

// MarkTombstone copies an event and sets the tombstone metadata key.
func MarkTombstone(evt Event) (*ImmutableEvent, error)

// MarkRebirth copies an event and sets the rebirth metadata key.
func MarkRebirth(evt Event) (*ImmutableEvent, error)

// DetectTombstone inspects an event stream and returns the tombstone status.
// Returns Undetermined if the stream is empty or no tombstone/rebirth metadata is found.
func DetectTombstone(events []Event) TombstoneStatus

// HasTombstone reports whether the event carries a tombstone marker.
func HasTombstone(evt Event) bool
```

### 3.2 StatusMiddleware for the bus

**File:** `stream/middleware.go` (new module)

```go
// StatusMiddleware returns PublishMiddleware that auto-marks tombstone/rebirth metadata
// on events whose type is in the provided sets.
//
// Usage:
//   bus.UsePublish(stream.StatusMiddleware(
//       []event.Type{"user.deleted", "order.cancelled"},
//       []event.Type{"user.restored"},
//   ))
func StatusMiddleware(deleteTypes, rebirthTypes []event.Type) event.PublishMiddleware
```

---

## Phase 4: `stream/` Read Model Module

### 4.1 Module structure

```
stream/
├── go.mod              # depends on core/ only
├── doc.go
├── types.go            # AggregateRef, AggregateStatus, Page[T], TombstonePolicy, ListOptions
├── reader.go           # AggregateReader, AggregateLister interfaces
├── in_memory.go        # InMemoryAggregateReader (GlobalSource-based)
├── sql_reader.go       # SQLAggregateReader (projection table)
├── builder.go          # ListBuilder fluent API
├── middleware.go       # StatusMiddleware
├── projection.go       # AggregateProjection (maintains aggregates table)
└── *_test.go
```

### 4.2 Types

```go
// AggregateRef is identity — no derived state.
type AggregateRef struct {
    ID          id.AggregateID
    Type        event.AggregateType
    Version     event.Version
    LastEventAt time.Time
}

// AggregateStatus pairs identity with computed state.
type AggregateStatus struct {
    Ref    AggregateRef
    Status event.TombstoneStatus
}

// Page is cursor-based. No TotalCount.
type Page[T any] struct {
    Items   []T
    HasMore bool
}

// TombstonePolicy controls visibility.
type TombstonePolicy int

const (
    TombstoneExclude TombstonePolicy = iota
    TombstoneInclude
    TombstoneOnly
)

// ListOptions controls aggregate listing.
// Type is REQUIRED for cursor pagination.
type ListOptions struct {
    Type      event.AggregateType // required
    After     id.AggregateID      // cursor
    Limit     uint
    Tombstone TombstonePolicy
}
```

### 4.3 Reader interface

```go
// AggregateReader queries aggregate streams.
type AggregateReader interface {
    List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error)
}

// AggregateLister extends AggregateReader with status.
type AggregateLister interface {
    AggregateReader
    ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error)
}
```

### 4.4 In-memory reader (fallback)

```go
// InMemoryAggregateReader implements AggregateReader using GlobalSource.
// Loads ALL events and filters in-memory. Suitable for testing and small datasets.
type InMemoryAggregateReader struct {
    source event.GlobalSource
}

func NewInMemoryAggregateReader(source event.GlobalSource) *InMemoryAggregateReader
```

### 4.5 SQL reader (projection-backed)

```go
// SQLAggregateReader queries a dedicated aggregates projection table.
type SQLAggregateReader struct {
    db     *sql.DB
    prefix string // table prefix
}

func NewSQLAggregateReader(db *sql.DB, tablePrefix string) *SQLAggregateReader
```

### 4.6 Builder (consumer-facing API)

```go
// ListBuilder provides a fluent API for aggregate listings.
type ListBuilder struct {
    reader AggregateReader
    opts   ListOptions
}

func NewList(reader AggregateReader) *ListBuilder
func NewInMemoryList(source event.GlobalSource) *ListBuilder  // convenience
func NewSQLList(db *sql.DB, prefix string) *ListBuilder       // convenience

func (b *ListBuilder) OfType(t event.AggregateType) *ListBuilder
func (b *ListBuilder) After(id id.AggregateID) *ListBuilder
func (b *ListBuilder) PageSize(n uint) *ListBuilder  // clamped to [1, maxPageSize]
func (b *ListBuilder) IncludeDeleted() *ListBuilder
func (b *ListBuilder) OnlyDeleted() *ListBuilder
func (b *ListBuilder) List(ctx context.Context) (*Page[AggregateRef], error)
```

### 4.7 Projection (maintains aggregates table)

```go
// AggregateProjection maintains the stream_aggregates table.
// Register with projection.Runner to keep the read model in sync.
type AggregateProjection struct {
    db        *sql.DB
    tableName string
}

func NewAggregateProjection(db *sql.DB, tablePrefix string) (*AggregateProjection, error)
```

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS cqrs_stream_aggregates (
    aggregate_type   TEXT NOT NULL,
    aggregate_id     TEXT NOT NULL,
    version          INT  NOT NULL,
    event_count      INT  NOT NULL DEFAULT 0,
    last_event_at    TIMESTAMPTZ NOT NULL,
    tombstone_status INT  NOT NULL DEFAULT 0,  -- 0=active, 1=tombstoned, 2=undetermined
    PRIMARY KEY (aggregate_type, aggregate_id)
);
```

---

## Phase 5: Remove `Delete`

### 5.1 Remove from `Store` interface

```go
// Store no longer includes Delete.
// History is append-only. For GDPR, use crypto-shredding or a separate redaction module.
type Store interface {
    Sink
    Source
}
```

### 5.2 Remove from all implementations

- `memory/store.go`
- `storage/event_store.go`
- `testhelpers/fake_store.go`
- `storage/pebble_helpers.go`
- All tests that use `Delete`

### 5.3 Test cleanup pattern

```go
// Before: store.Delete(ctx, aggType, aggID)
// After: create a fresh store per test
store := memory.NewMemoryStore()
// ... test ...
// store is discarded when test ends
```

---

## Phase 6: Generic Boundaries (Future Work)

After the foundation is solid, add type safety at application boundaries:

| Feature                 | File                            | Description                                                      |
| ----------------------- | ------------------------------- | ---------------------------------------------------------------- |
| `AggregateStore[State]` | `core/event/aggregate_store.go` | Binds `aggType` at construction, eliminates it from every call   |
| `TypedSource[State]`    | `core/event/typed_source.go`    | Auto-folds events into state, removing `load → fold` boilerplate |
| `SubscribeTyped[T]`     | `core/event/bus_typed.go`       | Type-safe bus handlers with automatic codec integration          |

These are **additive** — they don't change existing interfaces.

---

## Implementation Order

| Step | Phase   | Description                                                               | Breaking?        |
| ---- | ------- | ------------------------------------------------------------------------- | ---------------- |
| 1    | 1.1     | Add `Sink` + `Source` interfaces                                          | No               |
| 2    | 1.1     | Verify all stores satisfy `Sink` + `Source`                               | No               |
| 3    | 1.2     | Rename `GlobalLoader` → `GlobalSource`, etc.                              | Yes (mechanical) |
| 4    | 1.3     | Update wrappers (`VersionedSource`, `SourceStreamAdapter`)                | Yes              |
| 5    | 1.4     | Update `decider.Repository` to take `Sink` + `Source`                     | Yes              |
| 6    | 1.5     | Extract `FakeSink` + `FakeSource`                                         | No               |
| 7    | 2.1     | Rename `GlobalSource` → `Journal`, `PositionalSource` → `SeekableJournal` | Yes              |
| 8    | 2.2     | Update all method names (`ReadAll`, `ReadFrom`)                           | Yes              |
| 9    | 3.1     | Add tombstone types to `core/event/`                                      | No               |
| 10   | 3.2     | Add `StatusMiddleware`                                                    | No               |
| 11   | 4.1–4.7 | Build `stream/` module                                                    | No (new module)  |
| 12   | 5.1     | Remove `Delete` from `Store`                                              | Yes              |
| 13   | 5.2     | Remove `Delete` from all implementations                                  | Yes              |
| 14   | 5.3     | Update tests to use fresh stores                                          | No               |

---

## Files to Change

### `core/event/`

| File                 | Change                                                                                                                                                                  |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `store.go`           | Add `Sink`, `Source`, rename `GlobalLoader` → `Journal`, `PositionalLoader` → `SeekableJournal`, `BackwardsLoader` → `BackwardsSource`, `StreamLoader` → `StreamSource` |
| `tombstone.go`       | New: `TombstoneStatus`, `MarkTombstone`, `MarkRebirth`, `DetectTombstone`, `HasTombstone`                                                                               |
| `versioned_store.go` | Rename to `versioned_source.go`, wrap `Source` not `Store`                                                                                                              |
| `stream.go`          | Rename `StoreStreamAdapter` → `SourceStreamAdapter`, `StreamLoader` → `StreamSource`                                                                                    |

### `core/decider/`

| File         | Change                                                                                                   |
| ------------ | -------------------------------------------------------------------------------------------------------- |
| `decider.go` | `Repository` fields: `sink`, `source`; constructor takes `Sink` + `Source`; add `NewRepositoryWithStore` |
| `options.go` | No changes needed                                                                                        |

### `projection/`

| File             | Change                                            |
| ---------------- | ------------------------------------------------- |
| `runner.go`      | Accept `Source` + `SeekableJournal` (was `Store`) |
| `runner_test.go` | Update test helpers                               |

### `memory/`

| File            | Change                                                            |
| --------------- | ----------------------------------------------------------------- |
| `store.go`      | Remove `Delete`; add compile-time assertions for `Sink`, `Source` |
| `store_load.go` | Rename `LoadAll` → `ReadAll`, `LoadAllFromPosition` → `ReadFrom`  |

### `storage/`

| File                     | Change                                                                                                            |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| `event_store.go`         | Remove `Delete`; rename `LoadAll` → `ReadAll`, `LoadAllFromPosition` → `ReadFrom`; add `Sink`/`Source` assertions |
| `pebble_event_store.go`  | Remove `Delete`                                                                                                   |
| `pebble_helpers.go`      | Remove `Delete`                                                                                                   |
| `transactional_store.go` | Rename `TransactionalStore` → `TransactionalSink` (keep alias); `SaveWithOutbox` extends `Sink`                   |

### `testhelpers/`

| File            | Change                                             |
| --------------- | -------------------------------------------------- |
| `fake_store.go` | Extract `FakeSink` + `FakeSource`; remove `Delete` |

### New: `stream/`

| File            | Purpose                                                     |
| --------------- | ----------------------------------------------------------- |
| `go.mod`        | Module definition                                           |
| `types.go`      | `AggregateRef`, `AggregateStatus`, `Page[T]`, `ListOptions` |
| `reader.go`     | `AggregateReader`, `AggregateLister` interfaces             |
| `in_memory.go`  | `InMemoryAggregateReader`                                   |
| `sql_reader.go` | `SQLAggregateReader`                                        |
| `builder.go`    | `ListBuilder` fluent API                                    |
| `middleware.go` | `StatusMiddleware`                                          |
| `projection.go` | `AggregateProjection`                                       |
| `*_test.go`     | Tests                                                       |

---

## Open Questions

1. **Should `Journal` include `io.Closer`?**
   - Current `GlobalLoader` has no `Closer`. `Store` does.
   - If Journal is backed by the same resource as Store, lifecycle is shared.
   - If separate (e.g., read replica), it needs its own.
   - **Recommendation:** Keep `Journal` minimal (no `Closer`). Let implementations optionally satisfy `io.Closer` if they have their own resources.

2. **What about `EventReader` (cross-aggregate event queries)?**
   - v4 proposed it but it's underspecified.
   - SQL implementation either queries the write model (violates CQRS) or needs a separate events projection table.
   - **Recommendation:** Defer to v4.1. Phase 4 focuses on aggregate listing only.

3. **Should `TransactionalStore` be kept as an alias?**
   - `decider.Repository` type-asserts to it.
   - **Recommendation:** Keep `TransactionalStore` as a deprecated alias for `TransactionalSink` during transition, then remove.

4. **How does the saga store fit?**
   - `saga/store.go` has its own `Store` interface (`Save`, `Load`).
   - **Recommendation:** Leave unchanged for now. Saga store is domain-specific, not part of the event store hierarchy.

---

## Success Criteria

- [ ] All existing tests pass after each phase
- [ ] `nix run .#build` succeeds
- [ ] `nix run .#lint` passes
- [ ] `nix run .#test` passes
- [ ] No `any` types introduced in new interfaces
- [ ] Max 250 lines/file, max 30 lines/function
- [ ] Every exported type has Go doc comment
- [ ] New module (`stream/`) has its own `go.mod`
