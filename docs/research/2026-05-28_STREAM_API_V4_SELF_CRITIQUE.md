# v4 Proposal Self-Critique

> **Status:** IMPLEMENTED — StatusMiddleware shipped, io.Closer removed (ADR-0010), Sink/Source split (ADR-0006)

> **Date:** 2026-05-28 | **For:** `docs/research/2026-05-28_STREAM_API_V4_PROPOSAL.md`

## Issues to address before implementation

### 1. `Store = Sink + Source` — the `Close()` problem

Both `Sink` and `Source` embed `io.Closer`. When composed into `Store`, Go deduplicates the method. But if a consumer implements `Sink` and `Source` as **separate objects** (e.g., write to Kafka, read from Postgres), each has its own `Close()`. The composite `Store` can only expose one `Close()`.

**Fix:** Document that `Store` is a convenience composite. For separate resources, consumers use `Sink` and `Source` directly. `Store.Close()` is for single-resource implementations (MemoryStore, SQLEventStore). Add a `Closer` interface or `io.Closer` wrapper if needed for multi-resource composition.

**Better fix:** Don't embed `io.Closer` in both. Put it only in `Sink`:

```go
type Sink interface {
    io.Closer
    Save(...)
    AppendBatch(...)
}

type Source interface {
    Load(...)
    LoadFromVersion(...)
    // no Close — lifecycle managed by Sink or caller
}
```

But this breaks the pattern where `Source` can be used independently. Hmm.

**Best fix:** Keep both with `io.Closer`, document that `Store.Close()` should close both resources. For implementations where Sink and Source share a resource, Close is idempotent. For separate resources, a wrapper Store closes both.

### 2. `AutoTombstone` name is misleading

It handles both tombstones AND rebirths. The name says only "tombstone."

**Fix:** Rename to `stream.LifecycleMiddleware` or `stream.StatusMiddleware`. Following the `signing` pattern: `SignMiddleware`, `VerifyMiddleware`. So: `stream.StatusMiddleware(deleteTypes, rebirthTypes)`.

### 3. `InMemoryAggregateReader` needs `GlobalSource`, not `Source`

`Source` only has `Load` (per-aggregate). To enumerate all aggregates, `InMemoryAggregateReader` needs `GlobalSource.LoadAll()` to discover which streams exist.

**Fix:** Update the proposal:

```go
func NewInMemoryAggregateReader(source event.GlobalSource) *InMemoryAggregateReader
```

This means the in-memory reader only works with stores that implement `GlobalSource`. That's fine — `MemoryStore` does, and most SQL stores do too.

### 4. `EventReader` SQL implementation is underspecified

For `EventReader.Read()` with SQL, we'd need to query events across aggregates with filters (event types, time range, etc.). This requires either:

- Querying the write model's `events` table directly (violates CQRS)
- Maintaining a separate `events` projection table (doubles storage)
- Using the existing `events` table but with read-optimized indexes (pragmatic but not pure)

**Fix:** Move `EventReader` to a "Future work" section. v4 scope is aggregate listing + tombstones. Event-level cross-stream querying is v4.1.

### 5. Blast radius of removing `Delete`

A quick grep shows `Delete` is used in:

- `memory/store.go`
- `storage/event_store.go`
- `testhelpers/fake_store.go`
- `storage/pebble_helpers.go` (maybe?)
- Various test files
- Potentially `saga/` store

**Fix:** Add an implementation note: "Remove Delete from Store interface, update all implementations, grep for `.Delete(` across the repo, update or remove tests."

### 6. `Delete` removal — what about test cleanup?

Tests currently use `Delete` to clean up between test runs. Without `Delete`, how do tests reset state?

**Fix:** Tests should create a fresh `MemoryStore` per test. `t.Parallel()` tests already do this. For integration tests with SQL, use `TRUNCATE` or recreate the schema. Document the test pattern change.

### 7. `AggregateProjection` `event_count` caveat

`AppendBatch` bypasses the bus. Events appended directly to the store won't be seen by the projection. `event_count` will be stale.

**Fix:** Document this as a known limitation. Projections are eventually consistent and only see bus-published events. Direct store mutations bypass projections. This is standard CQRS behavior.

### 8. Tombstone status column type

The proposal shows `tombstone_status` as implied string but says "leaning toward integer." Be consistent.

**Fix:** Use `INT` in SQL, map to Go `TombstoneStatus` constants:

```go
const (
    TombstoneActive       TombstoneStatus = 0
    TombstoneTombstoned   TombstoneStatus = 1
    TombstoneUndetermined TombstoneStatus = 2
)
```

SQL schema:

```sql
tombstone_status INT NOT NULL DEFAULT 0
```

### 9. Builder naming

`NewAggregateQuery(reader)` is descriptive but doesn't clearly indicate it's for LISTING. `Query` is generic.

**Fix:** Rename to `NewAggregateList(reader)` or `NewListQuery(reader)`. The existing pattern in the codebase: `projection.NewBuilder`, `query.NewDispatcher`. So `stream.NewListBuilder(reader)` follows convention.

### 10. `AggregateRef` should not have `EventCount` for SQL readers

`EventCount` requires `COUNT(*)` or maintaining a counter. For the projection table, we maintain `event_count`. But for a generic `Source`-based reader, computing `EventCount` requires loading all events.

**Fix:** Make `EventCount` optional. In `AggregateRef`, use a pointer or a separate `AggregateStats` struct. Or just include it in the projection table and omit it from the generic reader.

Actually, simpler: `AggregateRef` always has `EventCount` (uint). For the in-memory reader, it's accurate (count the slice). For the SQL reader, it comes from the projection table. If the projection doesn't track it, it returns 0. The consumer can check `EventCount > 0` to know if stats are available.

Better: Don't include `EventCount` in `AggregateRef` at all. Add it to `AggregateStatus` or a separate struct. Keep `AggregateRef` minimal: identity + version + time.

```go
type AggregateRef struct {
    ID          id.AggregateID
    Type        event.AggregateType
    Version     event.Version
    LastEventAt time.Time
}
```

```go
type AggregateInfo struct {
    Ref         AggregateRef
    EventCount  uint
    Status      event.TombstoneStatus
}
```

This separates identity (Ref) from computed state (Info).

### 11. Cursor pagination for aggregates is tricky

`After id.AggregateID` as cursor assumes aggregate IDs are sortable. `AggregateID` is a `string` (any non-empty string). Lexicographic sort may not match temporal order.

**Fix:** For deterministic cursor pagination, sort by `(last_event_at, aggregate_type, aggregate_id)` and encode the cursor as a composite. For simplicity in v4, require `Type` to be specified in `ListOptions`, then cursor is just `After id.AggregateID` within that type.

```go
type ListOptions struct {
    Type  event.AggregateType // REQUIRED for cursor pagination
    After id.AggregateID      // cursor within the type
    Limit uint
    // ...
}
```

If `Type` is empty, return an error: "ListOptions.Type is required for cursor pagination."

This makes the API simpler and the SQL query simpler.

### 12. The `StreamLoader` / `EventStream` interfaces already exist

In `core/event/stream.go`, `StreamLoader` and `EventStream` are already defined. In v4, `StreamLoader` should become `StreamSource` (extends Source).

**Fix:** Rename `StreamLoader` → `StreamSource`. Keep `EventStream` as-is.

### 13. `TransactionalSink` naming

The user said they like `TransactionalStore` extending `Store`. With the split, it should extend `Sink` (since transactions are about writes). But the name `TransactionalSink` is less intuitive than `TransactionalStore`.

**Fix:** Keep `TransactionalStore` as a composite that extends `Store` (which is `Sink + Source`). But also provide `TransactionalSink` for consumers who only need the write side. This is additive, not breaking.

Actually, simpler: `TransactionalStore` extends `Store` (the composite). `TransactionalSink` is a separate interface that extends `Sink`. Consumers who type-assert to `TransactionalStore` still work. Consumers who only have a `Sink` can type-assert to `TransactionalSink`.

### 14. Implementation order refinement

The current order has step 4 as "Remove Delete from all stores, update tests." This is disruptive and could be done incrementally.

**Better order:**

1. Define `Sink` + `Source` interfaces (additive, no breaking changes)
2. Verify all existing `Store` implementations satisfy `Sink` and `Source`
3. Rename `GlobalLoader` → `GlobalSource`, etc. (breaking but mechanical)
4. Add tombstone types (`TombstoneStatus`, `MarkTombstone`, `MarkRebirth`)
5. Build `stream/` module (new, no breaking changes)
6. Finally: remove `Delete` from `Store` (the most disruptive change)

This way, steps 1-5 can be implemented and tested before the big breaking change.

---

## Summary of recommended changes to v4 proposal

| #   | Change                                                          | Priority |
| --- | --------------------------------------------------------------- | -------- |
| 1   | Rename `AutoTombstone` → `StatusMiddleware`                     | High     |
| 2   | `InMemoryAggregateReader` takes `GlobalSource`, not `Source`    | High     |
| 3   | Move `EventReader` to "Future work"                             | High     |
| 4   | Remove `EventCount` from `AggregateRef`, add `AggregateInfo`    | Medium   |
| 5   | Make `Type` required in `ListOptions`                           | Medium   |
| 6   | Document `Close()` behavior for composite `Store`               | Medium   |
| 7   | Use `INT` for tombstone_status in SQL schema                    | Medium   |
| 8   | Refine implementation order (additive first, Delete last)       | Medium   |
| 9   | Rename builder: `NewAggregateQuery` → `NewListBuilder`          | Low      |
| 10  | Rename `StreamLoader` → `StreamSource`                          | Low      |
| 11  | Keep `TransactionalStore` as composite, add `TransactionalSink` | Low      |

---

## One big question remaining

**Should `stream/` also provide an `AggregateWriter` for creating tombstone events directly?**

The user said: "if we get a Tombstone Command, a Tombstone Event will be triggered."

This implies a domain command → event → middleware marks tombstone metadata. The middleware is the right place because it's cross-cutting and consistent.

But what if a consumer wants to tombstone an aggregate WITHOUT a domain command? E.g., an admin panel "delete user" button. Should `stream/` provide a helper?

```go
stream.Tombstone(ctx, sink, bus, "User", userID, expectedVersion)
```

This loads the aggregate, appends a synthetic `user.tombstoned` event, saves it, publishes it. The middleware then marks the metadata.

Or is this too much? The consumer can just publish their own domain event (`user.deleted`) and the middleware handles the rest.

**My take:** The middleware approach is sufficient. `stream/` doesn't need a writer. The consumer's domain code publishes the event. `stream/` only reads.

But I should mention this in the proposal: "Tombstone events are domain events. The middleware marks metadata. `stream/` does not write events."
