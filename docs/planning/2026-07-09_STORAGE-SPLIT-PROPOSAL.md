# Proposal: Split storage/ into Focused Packages

> Generated as part of the v4 structural improvement plan (ADR-0046).
> Status: **Proposal** — not yet started. Awaits approval.

## Current State

`storage/` is a god module: 109 non-test Go files, 14,330 lines, 14 internal dependencies.
It mixes 8 distinct concerns:

| Concern                    | Files | Lines (approx) |
| -------------------------- | ----- | -------------- |
| Event store (SQL)          | 8     | 1,200          |
| Command store (SQL)        | 4     | 600            |
| Query store (SQL)          | 3     | 500            |
| Snapshot store (SQL)       | 2     | 300            |
| Checkpoint store (SQL)     | 2     | 200            |
| KV store (SQL)             | 1     | 280            |
| View store (SQL)           | 7     | 1,100          |
| Relational projection      | 5     | 800            |
| Timer store                | 1     | 150            |
| SQL helpers/dialect        | 6     | 900            |
| Backend facade             | 2     | 400            |
| Postgres listen/notify bus | 3     | 500            |
| Listing/aggregate reader   | 2     | 300            |
| Pebble sub-module          | 26    | 3,800          |
| Turso sub-module           | 19    | 2,500          |

## Proposed Split (4 packages)

### 1. `storage/sql/` (keep existing)

- SQL dialect, DBHandle, QueryEngine, RunInTx, helpers
- Already exists and is well-scoped

### 2. `storage/eventstore/` (new)

- `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`
- Event store SQL migrations
- Migrations embedded SQL files (postgres.sql, sqlite.sql)

### 3. `storage/auditstore/` (new)

- `SQLCommandStore`, `SQLQueryStore`
- Command and query audit trail stores
- These share the same "persisted request" pattern

### 4. `storage/readmodel/` (new)

- `SQLKVStore`, `SQLViewStore[V,K]`, view mapper, auto mapper
- `RelationalSchema`, `RelationalProjection`, `RelationalStore`, `ProjectionSink`
- All read-side SQL concerns

### What stays in `storage/` (compatibility facade)

- `SQLBackend` — the composition root that ties stores together
- `NewSQLiteBackend`, `NewSQLBackend` — constructor helpers
- `SQLiteEnableWAL`, `SQLiteEnableForeignKeys` — SQLite helpers
- Re-exports from split packages for backward compat (with `// Deprecated:` comments)

### Sub-modules stay as-is

- `storage/pebble/` — already its own concern (embedded KV)
- `storage/turso/` — already its own concern (Turso connector)
- `storage/memory/` — already its own concern (in-memory test impls)
- `storage/sql/` — already its own concern (SQL helpers)
- `storage/relational/` — already its own concern
- `storage/view/` — already its own concern

## Migration Path

1. Create `storage/eventstore/`, `storage/auditstore/`, `storage/readmodel/` directories
2. Move files with `git mv`
3. Update import paths in all consumers
4. Keep deprecated re-exports in `storage/` for v3 compat
5. v4 removes the deprecated re-exports

## Risk Assessment

- **Low risk**: The sub-modules (pebble, turso, memory, sql, relational, view) already exist and won't move
- **Medium risk**: The main `storage/` package has many consumers — the deprecated re-exports mitigate this
- **High value**: 109 files → 4 focused packages + facade. Clearer ownership, lower cognitive load.
