# ADR-0007: Pebble Module Scope — Event Store Only

**Status:** Accepted
**Date:** 2026-05-29

## Context

The `pebble/` module implements `event.Store` (Save, Load, LoadFromVersion, LoadToTimestamp, AppendBatch) using CockroachDB's Pebble embedded KV store. The `storage/` module provides `SQLEventStore` which additionally implements `event.Journal`, `event.SeekableJournal`, and `event.BackwardsSource`.

Should `pebble/` be expanded to cover the full `storage/` surface (Journal, SeekableJournal, CheckpointStore, SnapshotStore)?

## Decision

**Pebble implements `event.Store` only. No Journal, no SeekableJournal, no secondary index.**

> **Updated (2026-05-29):** The outbox pattern was removed from go-cqrs-lite. `CheckpointStore` and `SnapshotStore` remain available. Crash recovery uses `SeekableJournal` + `CheckpointStore` (see `docs/planning/REMOVE-OUTBOX-PLAN.md`).

## Rationale

### Why Pebble can't efficiently provide Journal

`event.Journal.ReadAll()` returns all events across all aggregates ordered by time — the foundation of projection replay.

SQL does this trivially:

```sql
SELECT * FROM events ORDER BY occurred_at ASC
```

Pebble's key scheme stores events per-aggregate by version:

```
cqrs_event:User:abc:0000000001  →  {UserCreated...}
cqrs_event:User:abc:0000000002  →  {UserUpdated...}
cqrs_event:Order:xyz:0000000001 →  {OrderPlaced...}
```

A prefix scan gives you one aggregate's events in version order — not all events in time order. Global time ordering doesn't exist in this key space.

### The secondary index approach (rejected)

Maintain a second set of keys alongside the primary:

```
cqrs_journal:01JX5K2ABC...  →  cqrs_event:User:abc:0000000001
cqrs_journal:01JX5K3DEF...  →  cqrs_event:Order:xyz:0000000001
cqrs_journal:01JX5K4GHI...  →  cqrs_event:User:abc:0000000002
```

Since event IDs are ULIDs (time-sortable), scanning `cqrs_journal:` gives global time order.

Costs: double writes on every Save, extra storage, more compaction overhead.

### Why it's not worth it

Pebble is for embedded/single-process use cases. Consumers who need global Journal for projection replay should use `storage/` (SQL) — that's exactly what it's for. The Pebble module should stay focused on what embedded KV does well: per-aggregate event persistence with optimistic concurrency.

## Consequences

- `pebble/` will never implement `event.Journal` or `event.SeekableJournal`
- `pebble/` may still add `CheckpointStore` and `SnapshotStore` (per-key operations, no global ordering needed)
- Consumers needing full projection replay should use `storage/` with PostgreSQL/SQLite
- The Pebble example (`example/todo`) uses `memory.MemoryBus` for live projections instead of Journal replay
