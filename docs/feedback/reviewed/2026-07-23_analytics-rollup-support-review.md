# Review: Incremental Rollup / Aggregation Support for go-cqrs-lite

**Date:** 2026-07-23
**Reviewer:** Crush (AI Engineering Partner)
**Proposal:** [2026-07-23_analytics-rollup-support.md](../new/2026-07-23_analytics-rollup-support.md)
**Verdict:** Option B (`Increment` on `ProjectionSink`) — implemented, correctly layered. Option A (`RollupSpec`) — rejected. Prerequisite gap (`Resettable`) — fixed.

---

## Architectural Analysis: Which Layer Does What

The codebase has three projection tiers, each with its own sink interface for a fundamentally different data model:

| Tier                | Module                                    | Sink Interface                            | Data Model           | Counter Mechanism                                     |
| ------------------- | ----------------------------------------- | ----------------------------------------- | -------------------- | ----------------------------------------------------- |
| KV/Document (T0→T5) | `kv.ViewStore[V,K]` + `stack.Materialize` | `ViewStore` + optional capabilities (ISP) | One document per key | `kv.ViewUpdater[V,K]` — **defined but unimplemented** |
| Relational (T4)     | `storage/relational/`                     | `ProjectionSink` (monolithic)             | Multi-table SQL rows | `sink.Increment` — **implemented in this review**     |
| Graph (T3)          | `graph/`                                  | `GraphSink` (monolithic)                  | Nodes + edges        | Not applicable — counters are node properties         |

### Why `Increment` belongs on `ProjectionSink` (not a separate interface)

The kv/ tier uses ISP (separate `ViewQuerier`, `ViewCounter`, `ViewUpdater`, `ViewResetter`, `ViewBatchSetter`) because backends genuinely differ in capabilities — `kv.TypedStore` (memory/Pebble) can't do `WHERE` clauses, but `SQLViewStore` can.

The relational tier does NOT use ISP. `ProjectionSink` is monolithic (Upsert, Ensure, Update, DeleteWhere, QueryOne) because `sqlSink` is the sole implementation — there is no "relational projection over a non-SQL backend." Adding `Increment` follows the established pattern: it's the same family of SQL operations as `Upsert` (`INSERT...ON CONFLICT DO UPDATE SET col = excluded.col` → `SET col = COALESCE(col, 0) + excluded.col`).

Consumer UX matters too. A handler receives `ProjectionSink` and calls `.Increment()` directly. With a separate `CounterSink` interface, they'd need a type assertion that ALWAYS succeeds (since `sqlSink` is the only implementation) — pure ceremony.

### Why `Reset` belongs on `RelationalProjection`

`projectionhost.Resettable` (Tier 3) defines `Reset(ctx) error`. `RelationalProjection` (Tier 4) implements it by doing `DELETE FROM <table>` for each table in its schema. The projection owns its tables, so it knows how to clear them. This is the same layering as `kv.ViewResetter` → `SQLViewStore.DeleteAll`.

### The complete counter architecture (two tiers, two mechanisms)

Consumers have two counter paths depending on their projection tier:

1. **KV/Document counters** (entity views with a numeric field): `kv.ViewUpdater[V,K]` — atomic read-modify-write callback. Defined at Tier 0, unimplemented on `SQLViewStore`. This is P1 follow-up work.

2. **Relational counters** (multi-dimensional rollup tables): `ProjectionSink.Increment` — atomic SQL-level `INSERT...ON CONFLICT DO UPDATE SET col = COALESCE(col,0) + excluded.col`. Implemented in this review.

Both are correct for their tier. They serve different use cases: `ViewUpdater` for single-document counters, `Increment` for multi-dimensional rollup tables with composite PKs.

---

## What the Proposal Gets Right

1. **The problem is real.** 25-subselect stats queries on every dashboard load is genuinely O(full scan). Pre-materialized rollups turn these into O(1).

2. **Sequencing is correct.** Ship the primitive first, prove it, then consider declarative sugar.

3. **Composite PK for rollup tables** — enables efficient `WHERE channel_id = ?` without JOINs.

4. **The "what it does NOT cover" section is honest** — pre-materialization has inherent limitations.

---

## What the Proposal Gets Wrong

### 1. Option A (RollupSpec) is premature abstraction — REJECT

This is a library, not a framework (Design Principle #1). A rollup handler with `sink.Increment` is 7 lines:

```go
func rollupActivity(ctx context.Context, evt event.Event, sink ProjectionSink) error {
    p, _ := event.DecodePayloadAuto[MessageCreated](evt)
    return sink.Increment(ctx, "channel_activity_by_day", Row{
        "guild_id": p.GuildID, "channel_id": p.ChannelID,
        "day": p.CreatedAt.Format("2006-01-02"),
    }, "message_count", +1)
}
```

The RollupSpec equivalent is 35 lines of struct configuration that saves zero logic. It violates Design Principle #1 (library not framework) and #4 (composition over declarative DSL).

### 2. Missed the existing `kv.ViewUpdater` interface

`kv/view_store.go:118-128` already defines `ViewUpdater[V,K]` with a doc comment literally describing the counter use case. The proposal reinvents this concept instead of connecting to the existing abstraction.

### 3. `IncrementWhere` is a footgun — DROPPED

Matching by WHERE instead of PK can silently update multiple rows. For counters, this is a data-corruption vector. Only PK-based `Increment` is implemented.

### 4. `MAX(0, ...)` underflow guard hides data loss — DROPPED

If a counter goes below 0, events are inconsistent (more deletes than creates). Silently clamping hides the bug. Let it go negative — it's a signal.

### 5. No replay safety analysis

Projections replay events. Under `Host.Reset` (full replay from zero), the rollup table starts empty and re-incrementing from zero is correct. Under bounded replay (crash recovery), `projectionhost`'s dedup ring handles recent events. No code change needed, but this must be documented.

---

## What Was Implemented

### 1. `Increment` on `ProjectionSink` (`storage/relational/sink.go`)

```go
Increment(ctx context.Context, table string, key Row, counterCol string, delta int64) error
```

SQL: `INSERT INTO <table> (<keycols>, <counter>) VALUES (?, ?, ?) ON CONFLICT(<pk>) DO UPDATE SET <counter> = COALESCE(<counter>, 0) + excluded.<counter>`

- `COALESCE` guards multi-counter tables where untouched counters are NULL (NULL + N = NULL in SQL)
- Schema-validated: table exists, counter column exists, counter not in key, key includes all PK columns
- `append` aliasing fixed: `allCols` and `allVals` are explicit copies, not `append(keyCols, ...)` which can mutate the backing array

### 2. `Reset` on `RelationalProjection` (`storage/relational/projection.go`)

```go
func (p *RelationalProjection) Reset(ctx context.Context) error
```

Does `DELETE FROM <table>` for each table in the schema. Implements `projectionhost.Resettable` (documented conformance — no new module dependency to avoid pulling OTel SDK into storage for a one-method assertion).

### 3. Error sentinels (`storage/relational/errors.go`)

`errSinkCounterInKey`, `errSinkKeyMissingPK` — both `errorfamily.NewRejection`.

### 4. Tests (`storage/relational/increment_test.go`)

11 tests: new row, existing row, negative delta, multi-counter (COALESCE), composite PK, separate keys, unknown table, unknown counter column, counter in key, missing PK column, atomic rollback, reset + replay.

---

## Revised Priorities

| Priority | Item                                         | Status                                   |
| -------- | -------------------------------------------- | ---------------------------------------- |
| P0       | `sink.Increment` on `ProjectionSink`         | **Implemented**                          |
| P0       | `RelationalProjection.Reset` (Resettable)    | **Implemented**                          |
| P1       | Implement `kv.ViewUpdater` on `SQLViewStore` | Deferred — separate work, different tier |
| P2       | `TimeBucket` helper                          | Rejected — 3 lines of Go in the handler  |
| P3       | `RollupSpec` / `RollupProjection` (Option A) | **Rejected** — premature abstraction     |
| P3       | `IncrementWhere`                             | **Rejected** — footgun                   |
| P3       | `MAX(0, ...)` underflow guard                | **Rejected** — hides data loss           |
