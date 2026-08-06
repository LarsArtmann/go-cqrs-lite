# ADR-0115: Move SQLite Engine from Metaengine Core to Separate Module

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0062 addendum (dep boundary removed), ADR-0061 (original SQLite engine)

## Context

The SQLite engine (`sqliteEngine`) currently lives in the metaengine core module
(`metaengine/v4`). This means the core module's `go.mod` requires `database/sql`
(a stdlib dependency, but conceptually an engine implementation detail).

Every other engine is in a separate module:
- `metaengine/pebbleengine/` (cockroachdb/pebble)
- `metaengine/duckdbengine/` (duckdb-go, CGo)
- `metaengine/pgengine/` (pgx)

The SQLite engine is the exception. Originally justified because `database/sql`
is stdlib, but the principle is inconsistent: if Pebble gets its own module for
deployment isolation, so should SQLite.

With the zero-dependency boundary now removed (ADR-0062 addendum), the
metaengine core's job is **planning**, not data access. Engine implementations
belong in separate modules.

## Decision

**Move the SQLite engine from `metaengine/v4` to `metaengine/sqliteengine/v4`.**

### What Moves

- `sqlite_engine.go`, `sqlite_backends.go` → `metaengine/sqliteengine/`
- SQLite-specific tests → `metaengine/sqliteengine/`
- The `modernc.org/sqlite` dependency moves out of metaengine core `go.mod`

### What Stays in Core

- `memory_engine.go` / `memory_backends.go` — the in-memory engine is the
  zero-dependency default that ships with the planner. It needs no external deps.
- All interfaces (Engine, MapBackend, GraphBackend replacement, etc.)
- The planner, cost model, ADT classification
- `adttest/` (already separate)

### Benefits

1. **Consistent module pattern.** Every engine implementation is a separate
   module. The core defines interfaces; modules provide implementations.
2. **Consumer choice.** Consumers who want only Memory + Dgraph don't pull SQLite
   deps. Currently every metaengine consumer gets `database/sql` + `modernc.org/sqlite`.
3. **Cleaner core go.mod.** The core module depends only on the Record type
   (ADR-0111) and stdlib.

### Migration

1. Create `metaengine/sqliteengine/` with its own `go.mod`
2. Move SQLite engine files
3. Update `go.work` to include the new module
4. Update all imports from `metaengine` to `metaengine/sqliteengine`
5. Update `adttest` matrix tests to import from the new location

## Consequences

- **Positive:** Consistent engine isolation pattern.
- **Positive:** Lighter core module for consumers who don't need SQLite.
- **Negative:** Breaking change for any consumer importing SQLite engine types
  from `metaengine` directly.
- **Neutral:** `database/sql` is stdlib, so this is about module organization, not
  dependency weight.
