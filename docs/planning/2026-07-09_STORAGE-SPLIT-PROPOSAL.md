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

## Proposed Split (3 packages + dispatchlog stays in storage/)

### 1. `storage/sql/` (keep existing)

- SQL dialect, DBHandle, QueryEngine, RunInTx, helpers
- Already exists and is well-scoped

### 2. `storage/eventstore/` (new)

- `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`
- Event store SQL migrations
- Migrations embedded SQL files (postgres.sql, sqlite.sql)

### 3. `storage/readmodel/` (new)

- `SQLKVStore`, `SQLViewStore[V,K]`, view mapper, auto mapper
- `RelationalSchema`, `RelationalProjection`, `RelationalStore`, `ProjectionSink`
- All read-side SQL concerns

### Dispatch log (stays in `storage/` for now)

- `SQLCommandStore`, `SQLQueryStore` — dispatch infrastructure (persisted
  commands and queries for replay/debugging/accountability)
- These are small CRUD wrappers (~1,100 lines combined) that share the backend
  facade. Extracting them into a separate package adds import path churn for
  zero structural benefit. "auditstore" was a lying name — these are dispatch
  logs, not after-the-fact compliance artifacts.
- **Future:** if a `storage/dispatchlog/` package becomes warranted (e.g. the
  stores grow domain logic or gain divergent dependencies), the name is
  reserved. For now, they stay co-located with `SQLBackend`.

### What stays in `storage/` (compatibility facade + dispatch log)

- `SQLBackend` — the composition root that ties stores together
- `NewSQLiteBackend`, `NewSQLBackend` — constructor helpers
- `SQLiteEnableWAL`, `SQLiteEnableForeignKeys` — SQLite helpers
- `SQLCommandStore`, `SQLQueryStore` — dispatch log (see above)
- `PostgresListenNotifyBus` — bus transport that ties into the backend
- `ListingAggregateReader` — reader that ties into the backend
- Re-exports from split packages for backward compat (with `// Deprecated:` comments)

### Sub-modules stay as-is

- `storage/pebble/` — already its own concern (embedded KV)
- `storage/turso/` — already its own concern (Turso connector)
- `storage/memory/` — already its own concern (in-memory test impls)
- `storage/sql/` — already its own concern (SQL helpers)
- `storage/relational/` — already its own concern
- `storage/view/` — already its own concern

## Migration Path

1. Create `storage/eventstore/`, `storage/readmodel/` directories
2. Move files with `git mv`
3. Update import paths in all consumers
4. Keep deprecated re-exports in `storage/` for v3 compat
5. v4 removes the deprecated re-exports

## Risk Assessment

- **Low risk**: The sub-modules (pebble, turso, memory, sql, relational, view) already exist and won't move
- **Medium risk**: The main `storage/` package has many consumers — the deprecated re-exports mitigate this
- **High value**: 109 files → 3 focused packages + facade. Dispatch log stays co-located with the backend facade. Clearer ownership, lower cognitive load.
