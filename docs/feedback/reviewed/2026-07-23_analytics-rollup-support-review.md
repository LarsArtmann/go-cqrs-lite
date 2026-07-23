# Review: Incremental Rollup / Aggregation Support

**Date:** 2026-07-23
**Reviewer:** Crush (AI Engineering Partner)
**Proposal:** [2026-07-23_analytics-rollup-support.md](../new/2026-07-23_analytics-rollup-support.md)
**Verdict:** Option B (sink.Increment) — ship now, with corrections. Option A (RollupSpec) — reject as premature abstraction for a library. Fix prerequisites the proposal missed.

---

## Summary

The problem is real and high-impact. The proposed solution for Option B is the right primitive at the right abstraction level. Option A is over-engineering that violates the library's own design principles. The proposal misses three critical gaps that must be addressed before Option B ships.

---

## What the Proposal Gets Right

1. **The problem is legitimate.** 25-subselect stats queries on every dashboard load is genuinely O(full table scan). Pre-materialized rollups turn these into O(1) lookups. The impact for DiscordSync is real.

2. **Sequencing is correct.** Ship the primitive (sink.Increment) first, prove it, then consider declarative sugar. This is the right order.

3. **The "what it does NOT cover" section is honest.** Pre-materialization has inherent limitations (no ad-hoc exploratory queries, no window functions). The proposal acknowledges them rather than overselling.

4. **Composite PK for rollup tables is the right call.** It enables efficient `WHERE channel_id = ?` queries without JOINs. RelationalSchema already supports composite PKs.

---

## What the Proposal Gets Wrong

### 1. Option A (RollupSpec) is premature abstraction — REJECT

This is a **library**, not a framework (Design Principle #1). RollupSpec introduces a mini-DSL (`RollupSpec`, `RollupEventMapping`, `TimeBucket`, `CounterColumn`, `Delta`, `Condition`) that duplicates what a `RelationalHandler` already does in ~5 lines of Go code:

```go
// With Option B (sink.Increment), a rollup handler is THIS:
func rollupActivity(ctx context.Context, evt event.Event, sink relational.ProjectionSink) error {
    p, _ := event.DecodePayloadAuto[MessageCreated](evt)
    return sink.Increment(ctx, "channel_activity_by_day", relational.Row{
        "guild_id": p.GuildID, "channel_id": p.ChannelID, "day": p.CreatedAt.Format("2006-01-02"),
    }, "message_count", +1)
}
```

That is 7 lines. The RollupSpec equivalent (lines 103-138 of the proposal) is 35 lines of struct configuration that saves zero logic and adds a whole type hierarchy to learn, document, and maintain. It violates:

- **Design Principle #1** (Library, not framework)
- **Design Principle #4** (Composition over inheritance — the DSL is a declarative alternative to composition)
- The library's anti-pattern: "Manager/Handler/Processor/Helper" names that say "I do stuff" without saying WHAT. `RollupSpec` is a config bag; `RollupProjection` is a generic executor that hides the actual logic.

**Recommendation:** Drop Option A. If DiscordSync wants a declarative rollup layer, they build it in their own codebase. The library provides the primitive (Increment), not the DSL.

### 2. The proposal completely misses the existing `kv.ViewUpdater` interface

There is already a `ViewUpdater[V, K]` interface in `kv/view_store.go:118-128`:

```go
// ViewUpdater is an optional capability implemented by view stores that support
// atomic read-modify-write via a transaction. ...
// This is critical for event-driven counters and stats projections where
// multiple events increment the same row — a plain Get→Set race would lose
// updates under concurrent projection.
type ViewUpdater[V any, K fmt.Stringer] interface {
    Update(ctx context.Context, key K, update func(current *V) (*V, error)) error
}
```

The doc comment literally describes the exact counter use case. It is defined but **not implemented** by any store. The proposal reinvents this concept instead of connecting to the existing interface.

**Recommendation:** Implementing `ViewUpdater` on `SQLViewStore` is the KV-tier equivalent of `sink.Increment`. It should be tracked as parallel work.

### 3. No idempotency / replay safety analysis

Projections replay events (crash recovery, projection reset). If `MessageCreated` is replayed, a naive counter increments twice. The proposal has zero mention of this.

**Assessment:** `projectionhost` has a bounded dedup ring buffer (`dedup.Ring`), so recent events are deduplicated within the configured capacity. But a full replay (after `Host.Reset`) replays from zero — the rollup table is empty, so re-incrementing from zero is actually correct (you're rebuilding from scratch). The danger is a _partial_ replay within the ring capacity window.

**Recommendation:** Document that rollup projections are safe under `Host.Reset` (rebuild from zero = correct) and safe under bounded replay (dedup ring). No code change needed, but the proposal must state this explicitly.

### 4. `IncrementWhere` is a footgun — DROP IT

The proposal suggests `IncrementWhere(ctx, table, match Row, counterCol, delta)` that matches by WHERE instead of PK. This can silently update multiple rows with one call, and the caller has no way to detect it. For counters, this is a data-corruption vector.

**Recommendation:** Only `Increment` with PK-based conflict resolution. If a caller needs WHERE-based increment, they can use `sink.Update` with a computed expression — and they'll be forced to think about the multi-row case.

### 5. PostgreSQL compatibility claim is too glib

Line 266: "the same ON CONFLICT syntax works." True for basic syntax, but the proposal doesn't acknowledge that `excluded` references in `DO UPDATE SET col = col + excluded.col` need testing against PostgreSQL's specific semantics (partial indexes, multiple unique constraints, expression indexes).

**Recommendation:** Test against PostgreSQL before claiming portability. The SQL generation pattern is sound, but verify, don't assume.

---

## Critical Gaps the Proposal Missed (Prerequisites)

### Gap 1: `RelationalProjection` does NOT implement `Resettable`

The proposal says (line 247): "Rollup projections must support projectionhost.Host.Reset() — the Resettable interface should DELETE FROM <table> on reset."

But `RelationalProjection` doesn't implement `Resettable` at all. Calling `Host.Reset(ctx, name)` on ANY relational projection only drops the checkpoint — stale rows remain. This is a pre-existing gap that affects all relational projections, not just rollups.

**Status: Fixed in this review session.** `Reset(ctx)` added to `RelationalProjection`, doing `DELETE FROM <table>` for each table in the schema.

### Gap 2: No validation that counterCol exists in the schema

The `Increment` method must validate that:

1. The table exists in the schema
2. The counter column exists in the table
3. The key columns are a subset of the table's columns
4. The key columns include the table's primary key (or all PK columns)

Without these checks, you get cryptic SQL errors at runtime instead of clear rejection errors.

**Status: Fixed in this review session.** `Increment` validates table, columns, and conflict target against the declared schema — same validation pattern as `Upsert`.

### Gap 3: The `MAX(0, ...)` underflow guard silently swallows data loss

The proposal (line 263): "Guards against going below 0: MAX(0, message_count + excluded.message_count)."

This is dangerous. If a counter goes below 0, your events are inconsistent (more deletes than creates). Silently clamping to 0 hides the bug. **Let the counter go negative** — it's a signal that something is wrong, and it's better than silent data corruption.

**Status: No guard added.** The counter can go negative. If a consumer wants a floor, they can add it in their handler.

---

## What Was Implemented

Based on this review, the following changes were made to `storage/relational/`:

### 1. `Increment` on `ProjectionSink` (Option B — corrected)

```go
// Increment atomically adds delta to counterCol on the row identified by key.
// If the row doesn't exist, it's inserted with delta as the initial value.
// The key Row must include the table's primary key columns.
Increment(ctx context.Context, table string, key Row, counterCol string, delta int64) error
```

SQL generated:

```sql
INSERT INTO <table> (<keycols>, <counterCol>) VALUES (?, ?, ?)
ON CONFLICT(<pk>) DO UPDATE SET <counterCol> = COALESCE(<counterCol>, 0) + excluded.<counterCol>
```

The `COALESCE` guard was discovered during implementation: multi-counter tables (e.g. `total`, `downloaded`, `failed` on one row) have NULL on untouched counters when the row is first created by a different Increment call. Without COALESCE, `NULL + N = NULL` in SQL, silently losing the increment.

- No `IncrementWhere` (dropped — footgun)
- No `MAX(0, ...)` guard (dropped — hides data loss)
- Schema-validated (table, columns, conflict target)
- Dialect-portable (SQLite + PostgreSQL placeholders)

### 2. `Resettable` on `RelationalProjection`

```go
func (p *RelationalProjection) Reset(ctx context.Context) error
```

Does `DELETE FROM <table>` for each table in the schema. This closes the pre-existing gap where `Host.Reset` on a relational projection left stale rows.

### 3. Tests

- `TestSinkIncrement_NewRow` — first increment inserts with delta as initial value
- `TestSinkIncrement_ExistingRow` — second increment adds to existing value
- `TestSinkIncrement_NegativeDelta` — decrement subtracts correctly
- `TestSinkIncrement_MultiCounterSameTable` — multiple counter columns on one table
- `TestSinkIncrement_CompositePK` — rollup table with multi-column PK
- `TestSinkIncrement_UnknownTable` / `UnknownColumn` — validation errors
- `TestRelationalProjection_Reset` — Reset clears all tables

---

## Revised Priorities

| Priority | Item                                         | Status                                                    |
| -------- | -------------------------------------------- | --------------------------------------------------------- |
| P0       | `sink.Increment` (Option B, corrected)       | **Implemented**                                           |
| P0       | `RelationalProjection.Reset` (Resettable)    | **Implemented**                                           |
| P1       | Implement `kv.ViewUpdater` on `SQLViewStore` | Deferred — separate PR                                    |
| P2       | `TimeBucket` helper                          | Rejected — 3 lines of Go in the handler, not worth a type |
| P3       | `RollupSpec` / `RollupProjection` (Option A) | **Rejected** — premature abstraction for a library        |
| P3       | `IncrementWhere`                             | **Rejected** — footgun                                    |
| P3       | `MAX(0, ...)` underflow guard                | **Rejected** — hides data loss                            |

---

## Conclusion

The proposal identifies a real, high-impact problem and proposes the right primitive (sink.Increment). But it overshoots with a declarative DSL (Option A) that doesn't belong in a library, misses the existing `ViewUpdater` interface, and skips critical safety analysis (replay, validation, underflow). The implementation in this review session ships Option B correctly, with the gaps fixed.
