# Comprehensive Improvement Plan

**Created:** 2026-05-01 · **Session 25**
**Sorted by:** Impact × Ease (highest first)

## Issues Found

### BUG: SQLSnapshotStore double-marshals State
- `Save()` calls `json.Marshal(snap.State)` but `snap.State` is already `[]byte`
- This wraps raw bytes in JSON quotes: `"c3RhdGU="` instead of just storing the raw bytes
- `Load()` returns the JSONB bytes directly, but they're now double-encoded
- **Fix:** Store `snap.State` directly (it's already `[]byte`), no marshaling needed

### BUG: MemoryCheckpointStore doesn't implement io.Closer
- `event.CheckpointStore` doesn't embed `io.Closer` (unlike Store, Bus, SnapshotStore, Outbox)
- `MemoryCheckpointStore` has no `Close()` method
- Inconsistent with all other store interfaces
- **Fix:** Add `io.Closer` to CheckpointStore interface, add Close to MemoryCheckpointStore

### DESIGN: Storage Close() closes caller-owned *sql.DB
- SQLEventStore, SQLSnapshotStore, SQLCheckpointStore all close the shared *sql.DB
- If someone passes the same `*sql.DB` to all three, the first Close() breaks the others
- **Fix:** Don't close `*sql.DB` — it's borrowed, not owned. Document this clearly.

### DESIGN: Event.Version() returns `int`, not `event.Version`
- Interface says `Version() int` but internally uses `event.Version`
- Inconsistent: `AggregateID()` returns branded `id.AggregateID`, but `Version()` returns raw `int`
- `Root.Version()` also returns `int`
- **Fix:** Change `Version()` to return `event.Version` everywhere (breaking change, but correct)

### IMPROVEMENT: Add tests for SQLSnapshotStore and SQLCheckpointStore
- Both new files have zero tests
- **Fix:** Add comprehensive go-sqlmock tests

## Execution Plan

| # | Task                                              | Impact | Effort | Priority |
|---|---------------------------------------------------|--------|--------|----------|
| 1 | Fix SQLSnapshotStore double-marshal bug           | HIGH   | LOW    | P0       |
| 2 | Fix storage Close() ownership (don't close DB)    | HIGH   | LOW    | P0       |
| 3 | Add io.Closer to CheckpointStore + impls          | MED    | LOW    | P1       |
| 4 | Add SQLSnapshotStore tests (go-sqlmock)           | MED    | MED    | P1       |
| 5 | Add SQLCheckpointStore tests (go-sqlmock)         | MED    | MED    | P1       |
| 6 | Change Version() to return event.Version          | HIGH   | HIGH   | P2       |
