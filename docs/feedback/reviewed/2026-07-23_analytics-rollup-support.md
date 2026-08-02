# Proposal: Incremental Rollup / Aggregation Support for go-cqrs-lite

**Date:** 2026-07-23
**Source:** DiscordSync consumer feedback (ADR-031 architectural shift toward zero-manual-SQL materialized views)
**Priority:** ~~P1 — blocks analytics materialization in DiscordSync~~ **RESOLVED** — Option B (`sink.Increment`) implemented in `storage/relational/sink.go` + `RelationalProjection.Reset`. 11 tests passing. See [implementation review](../../reviews/2026-07-23_adr-review.md) and [architectural correction](../../status/2026-07-23_17-56_rollup-increment-architectural-correction.md). Option A (`RollupSpec`) rejected as premature abstraction.

---

## Problem

go-cqrs-lite has excellent primitives for entity-level materialized views: `ViewStore[V,K]` (Get/Set/Delete/Query/Count with keyset pagination) and `RelationalProjection` (multi-table atomic writes via ProjectionSink). But it has **no first-class support for incremental rollups** — pre-computed aggregations maintained by projections as events flow.

Consumers like DiscordSync currently hand-write SQL like:

```sql
-- 25-subselect single-roundtrip stats query (internal/db/stats.go)
SELECT
  (SELECT COUNT(*) FROM messages) AS total_messages,
  (SELECT COUNT(*) FROM messages WHERE deleted_at IS NOT NULL) AS deleted_messages,
  (SELECT COUNT(*) FROM messages WHERE edited_at IS NOT NULL) AS edited_messages,
  (SELECT COUNT(*) FROM attachments) AS total_attachments,
  (SELECT COUNT(*) FROM attachments WHERE download_status = 'downloaded') AS downloaded,
  ... -- 20 more subselects
```

and:

```sql
-- Activity by channel per day (internal/db/activity.go)
SELECT c.name, SUBSTR(m.created_at,1,10) AS day, COUNT(*) AS count
FROM messages m JOIN channels c ON m.channel_id = c.id
WHERE {guild_condition} AND {attachment_condition}
GROUP BY c.id, day ORDER BY count DESC LIMIT 20
```

These are O(full_table_scan) queries run on every dashboard load. They should be O(1) lookups against pre-materialized rollup tables.

---

## Vision

A projection listens to domain events and **incrementally maintains rollup rows** so reads are O(1):

```
MessageCreated event
  → RollupProjection.Handle()
    → sink.Upsert("message_stats", Row{"key": "total", "count": <incremented>})
    → sink.Upsert("channel_activity_by_day", Row{
        "channel_id": p.ChannelID, "day": p.Date, "count": <incremented>})
```

Dashboard reads become:

```go
stats := statsStore.Get(ctx, "total")             // O(1) — one row
activity := activityStore.Query(ctx, ViewQuery{   // O(window_size) — few rows
    Conditions: []Condition{{Column: "day", Op: OpGte, Value: startDate}},
    Order: []OrderClause{{Column: "day", Desc: true}},
    Limit: 30,
})
```

No GROUP BY, no JOINs, no subqueries at read time.

---

## Proposed API

### Option A: RollupProjection (declarative)

A new high-level projection type in `stack/v4` or a new `rollup/v4` module:

```go
// RollupSpec declares one rollup table maintained incrementally.
type RollupSpec struct {
    Name       string                   // projection name
    Table      string                   // SQL table name
    KeyColumns []string                 // group-by dimensions (e.g. ["channel_id", "day"])
    CounterColumn string                // column to increment/decrement (e.g. "count")
    Events     []RollupEventMapping     // which events trigger increments
}

// RollupEventMapping maps an event type to an increment/decrement action.
type RollupEventMapping struct {
    EventType   event.Type
    Delta       int64                   // +1 for create, -1 for delete
    KeyFromEvent func(evt event.Event) (map[string]any, error)  // extract group-by values
    Condition   func(evt event.Event) (bool, error)             // optional filter (e.g. only non-bot)
}

// NewRollupProjection builds a projection from one or more RollupSpecs.
func NewRollupProjection(
    specs []RollupSpec,
    db *sql.DB,
    dialect sqlpkg.Dialect,
    opts ...RollupOption,
) (projection.Projection, error)
```

**Usage:**

```go
activityRollup := rollup.NewRollupProjection([]rollup.RollupSpec{
    {
        Name: "channel-activity-by-day",
        Table: "channel_activity_by_day",
        KeyColumns: []string{"guild_id", "channel_id", "day"},
        CounterColumn: "message_count",
        Events: []rollup.RollupEventMapping{
            {
                EventType: events.MessageCreated,
                Delta: +1,
                KeyFromEvent: func(evt event.Event) (map[string]any, error) {
                    p, err := event.DecodePayloadAuto[events.MessageCreatedPayload](evt)
                    if err != nil { return nil, err }
                    return map[string]any{
                        "guild_id": p.GuildID,
                        "channel_id": p.ChannelID,
                        "day": p.CreatedAt.Format("2006-01-02"),
                    }, nil
                },
            },
            {
                EventType: events.MessageDeleted,
                Delta: -1,
                KeyFromEvent: func(evt event.Event) (map[string]any, error) {
                    p, err := event.DecodePayloadAuto[events.MessageDeletedPayload](evt)
                    if err != nil { return nil, err }
                    return map[string]any{
                        "guild_id": p.GuildID,
                        "channel_id": p.ChannelID,
                        "day": p.DeletedAt.Format("2006-01-02"),
                    }, nil
                },
            },
        },
    },
}, db, sqlpkg.SQLiteDialect{})
```

The projection generates:

- `CREATE TABLE IF NOT EXISTS channel_activity_by_day (guild_id TEXT, channel_id TEXT, day TEXT, message_count INTEGER DEFAULT 0, PRIMARY KEY (guild_id, channel_id, day))`
- On MessageCreated: `INSERT INTO ... ON CONFLICT(guild_id, channel_id, day) DO UPDATE SET message_count = message_count + 1`
- On MessageDeleted: `... DO UPDATE SET message_count = message_count - 1`

### Option B: Counter helpers on ProjectionSink (composable)

Instead of a standalone RollupProjection, add counter primitives to the existing `ProjectionSink`:

```go
type ProjectionSink interface {
    // ...existing methods...

    // Increment atomically increments (or decrements with negative delta)
    // a counter column on the row matching key. If the row doesn't exist,
    // it's inserted with the delta as initial value.
    Increment(ctx context.Context, table string,
        key Row, counterCol string, delta int64) error

    // IncrementWhere is like Increment but matches by WHERE instead of PK,
    // for cases where the rollup key isn't the table's primary key.
    IncrementWhere(ctx context.Context, table string,
        match Row, counterCol string, delta int64) error
}
```

This lets rollups live inside existing `RelationalProjection` handlers:

```go
func handleActivityRollup(ctx context.Context, evt event.Event, sink relational.ProjectionSink) error {
    p, _ := event.DecodePayloadAuto[events.MessageCreatedPayload](evt)
    return sink.Increment(ctx, "channel_activity_by_day",
        relational.Row{
            "guild_id": p.GuildID,
            "channel_id": p.ChannelID,
            "day": p.CreatedAt.Format("2006-01-02"),
        },
        "message_count", +1)
}
```

### Recommendation: Both

- **Option A (RollupProjection)** for the common case: "count events grouped by N dimensions." Covers DiscordSync's activity, stats, and analytics rollups with zero handler code.
- **Option B (sink.Increment)** for the flexible case: rollup logic that's intertwined with other projection writes (e.g. "increment channel activity AND upsert the message in the same transaction").

Both share the same SQL generation (`INSERT ... ON CONFLICT DO UPDATE SET col = col + ?`), just at different abstraction levels.

---

## What Rollup Spec Needs to Cover

### Multi-measure rollups

A single rollup table may track multiple counters:

```go
RollupSpec{
    Table: "attachment_stats",
    KeyColumns: []string{"guild_id", "day"},
    Counters: map[string]string{  // column → semantic
        "total": "total attachments",
        "downloaded": "successfully downloaded",
        "failed": "permanently failed",
    },
    Events: []RollupEventMapping{
        {EventType: events.AttachmentInserted, CounterColumn: "total", Delta: +1, ...},
        {EventType: events.AttachmentDownloaded, CounterColumn: "downloaded", Delta: +1, ...},
        {EventType: events.AttachmentFailed, CounterColumn: "failed", Delta: +1, ...},
    },
}
```

### Conditional increments (filters)

Some events should only increment if a condition holds (e.g. "don't count bot messages"):

```go
RollupEventMapping{
    EventType: events.MessageCreated,
    Delta: +1,
    Condition: func(evt event.Event) (bool, error) {
        p, _ := event.DecodePayloadAuto[events.MessageCreatedPayload](evt)
        return p.Author.Kind != domain.UserKindBot, nil
    },
    KeyFromEvent: ...,
}
```

### Time-bucketing

The most common rollup dimension is time (day, hour, week). A helper for this:

```go
// TimeBucket extracts a time-bucketed key component from an event field.
type TimeBucket struct {
    Field    string  // payload field name (e.g. "CreatedAt")
    Granularity string // "day", "hour", "week", "month"
}

func (tb TimeBucket) FromEvent(evt event.Event) (string, error)
```

### Projection rebuild / reset

Rollup projections must support `projectionhost.Host.Reset()` — dropping the table and replaying from zero. The `Resettable` interface should `DELETE FROM <table>` on reset.

---

## SQL Generation

All rollup SQL is `INSERT ... ON CONFLICT DO UPDATE SET col = col + ?`:

```sql
-- Increment:
INSERT INTO channel_activity_by_day (guild_id, channel_id, day, message_count)
VALUES (?, ?, ?, ?)
ON CONFLICT(guild_id, channel_id, day) DO UPDATE SET message_count = message_count + excluded.message_count

-- Decrement (delta = -1):
-- Same statement with excluded.message_count = -1
-- Guards against going below 0: MAX(0, message_count + excluded.message_count)
```

PostgreSQL portability: the same `ON CONFLICT` syntax works with the `sqlpkg.Dialect` abstraction.

---

## What This Does NOT Cover (and Shouldn't)

1. **Ad-hoc exploratory queries** — if a user needs "show me messages per hour for channel X between 2pm and 4pm last Tuesday," and the rollup is per-day, the query can't be answered from the rollup. This is an inherent limitation of pre-materialization. Two escape hatches:
   - Provide a raw SQL query interface on `RelationalStore` for these cases.
   - Or accept that the dashboard must pre-declare its time granularities.

2. **Window functions** — "top 5 channels by message count in the last 7 days" requires ordering by a computed sum. This can be done via `ViewStore.Query` with ORDER BY + LIMIT on a rollup table where each row is `(channel_id, day, count)`. The sum-over-window is replaced by summing rollup rows at read time, which is O(7) per channel (7 days).

3. **JOIN-based rollups** — rollup tables are denormalized by design. The guild_id is stored directly in the rollup row; there's no need to JOIN to get it.

---

## DiscordSync Impact

DiscordSync's analytics surface (~80+ SQL queries) would be replaced by:

| Current query                    | Rollup replacement                                                              | Read cost          |
| -------------------------------- | ------------------------------------------------------------------------------- | ------------------ |
| 25-subselect stats               | `StatsRollup` projection maintaining ~25 counter rows                           | O(1) per counter   |
| Activity by channel              | `ChannelActivityByDayRollup`                                                    | O(topN)            |
| Activity by author               | `AuthorActivityByDayRollup`                                                     | O(topN)            |
| Activity matrix                  | `ChannelAuthorActivityByDayRollup`                                              | O(channels × days) |
| Activity over time               | `DailyMessageCountRollup`                                                       | O(days_in_window)  |
| Attachment analytics (8 queries) | `AttachmentAnalyticsRollup` (content-type buckets, size buckets, status counts) | O(buckets)         |

Total: ~5-7 rollup projections replacing ~80+ hand-written SQL queries. Every dashboard read becomes O(1) or O(small_constant).

---

## Priority and Sequencing

1. **sink.Increment (Option B)** — smallest change, immediately useful, unblocks DiscordSync's simplest rollups. Ship first.
2. **RollupSpec + RollupProjection (Option A)** — declarative sugar on top of Option B. Ship once the Increment primitive is proven.
3. **TimeBucket helper** — convenience for the most common dimension. Ship with Option A.
4. **Multi-measure support** — needed for stats rollups. Ship as extension to Option A.

---

## Open Questions

1. **Should rollup tables use composite PRIMARY KEY (group-by columns) or a single `key TEXT` column?** Composite PK is more natural for rollups and enables efficient `WHERE channel_id = ?` queries. The `RelationalSchema` enrichment (P0 from the gap analysis) must support composite PKs — it already does.

2. **Should rollup projections be a separate tier (T3) module or live in `stack/v4`?** Given they build on `relational/` (T4) and `projection/` (T2), they'd be T3. A dedicated `rollup/v4` module is cleanest.

3. **How to handle rollup drift after manual DB edits?** If someone manually edits a counter row, the rollup is wrong until reset. The `projectionhost.Host.Reset()` + replay path is the recovery mechanism. Should we add a checksum/verification?

4. **Should the library support "derived rollups"** — rollups computed from other rollups rather than from raw events? E.g. a weekly rollup derived from daily rollups. This avoids re-processing the entire event history for coarser granularities. Probably future work.
