# ADR-0061: Metaengine SQLite Engine

**Status:** Accepted
**Date:** 2026-07-25

## Context

The `metaengine` module currently has only a `MemoryEngine` — an in-memory
implementation of all 8 ADT backend interfaces. While sufficient for testing
and prototyping, production deployments need a persistent engine that survives
restarts and can handle data volumes exceeding available memory.

The `SQLiteEngineProfile()` function already exists (declared in `engine.go`)
with the expected complexity profile for each ADT on SQLite, but no actual
implementation backs it.

## Decision

Implement `SQLiteEngine` as a direct `*sql.DB` wrapper, NOT as a wrapper
around `storage.SQLViewStore`. Rationale:

1. **SQLViewStore is view-specific** — it maps typed Go structs to SQL columns
   via `ViewMapper`. The metaengine deals with `any`-typed keys and values,
   which don't fit the column-extraction pattern.
2. **Table-per-ADT, not table-per-collection** — each ADT type (map, set,
   counter, etc.) gets its own table. Collections are namespaced within each
   table via a `collection TEXT` column. This keeps the schema simple and
   allows efficient queries per ADT type.
3. **JSON encoding for keys and values** — since the backend interfaces use
   `any` for both keys and values, JSON encoding provides universal
   serialization. This is a correctness-over-performance trade-off: the
   MemoryEngine is O(1) for lookups; SQLiteEngine is O(log N) but persistent.

### Table Schema

```sql
CREATE TABLE IF NOT EXISTS meta_map (
    collection TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    PRIMARY KEY (collection, key)
);

CREATE TABLE IF NOT EXISTS meta_set (
    collection TEXT NOT NULL,
    key        TEXT NOT NULL,
    PRIMARY KEY (collection, key)
);

CREATE TABLE IF NOT EXISTS meta_counter (
    collection TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (collection, key)
);

CREATE TABLE IF NOT EXISTS meta_multimap (
    collection TEXT NOT NULL,
    key        TEXT NOT NULL,
    seq        INTEGER NOT NULL,
    value      TEXT NOT NULL,
    PRIMARY KEY (collection, key, seq)
);

CREATE TABLE IF NOT EXISTS meta_log (
    collection TEXT NOT NULL,
    seq        INTEGER PRIMARY KEY AUTOINCREMENT,
    value      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta_graph_edges (
    collection TEXT NOT NULL,
    from_node  TEXT NOT NULL,
    to_node    TEXT NOT NULL
);
```

### Backend Coverage

| Backend         | Status  | Notes                                           |
| --------------- | ------- | ----------------------------------------------- |
| MapBackend      | Full    | INSERT OR REPLACE, SELECT, DELETE               |
| MapUpdater      | Full    | Transactional read-modify-write                 |
| ScanBackend     | Partial | Loads all rows, filters/sorts in Go             |
| SetBackend      | Full    | INSERT OR IGNORE, SELECT EXISTS                 |
| CounterBackend  | Full    | INSERT ON CONFLICT DO UPDATE (atomic increment) |
| GraphBackend    | Basic   | Adjacency list table, BFS in Go                 |
| MultimapBackend | Full    | seq column for ordered multi-values             |
| LogBackend      | Full    | AUTOINCREMENT seq for append order              |

## Consequences

- Metaengine gains its first persistent backend.
- Consumers can use `NewSQLiteEngine(db)` to get a durable engine.
- The cost model (`nsPerOp`) will need calibration (ADR-0063).
- ScanBackend does NOT push filters to SQL — this is a known limitation
  to be addressed by the FilterOn/SortOn pushdown ADR (ADR-0062).
