# ADR-0087: Postgres Metaengine Engine

**Date:** 2026-08-01
**Status:** Accepted
**Depends on:** ADR-0061 (metaengine-sqlite-engine), ADR-0084 (cost matrix)

## Context

The metaengine planner picks the best engine per query based on cost profiles. With only Memory and SQLite engines, the planner cannot route queries to Postgres for production workloads that need ACID durability, concurrent readers, and JSONB-specific optimizations.

Postgres is the most common production database for Go services. Adding a Postgres metaengine engine enables consumers to use the planner with their existing Postgres infrastructure — no additional database required.

## Decision

Create `metaengine/pgengine/` as a separate Go module (like `pebbleengine/` and `duckdbengine/`).

### Architecture

- **Driver:** `pgx/v5/stdlib` (database/sql-compatible, pure Go, no CGo)
- **ADTs implemented:** MapBackend + CounterBackend
- **Storage:** JSONB columns for map values (efficient JSON storage + queryable), BIGINT for counters
- **Upsert pattern:** `INSERT ... ON CONFLICT (collection, key) DO UPDATE` (Postgres-native)
- **Layout:** `LayoutRow` (B-tree indexed rows, matching Postgres's storage model)

### Cost Model

| Metric | Value | Rationale |
|--------|-------|-----------|
| `NsPerOp` | 12,000 | INSERT with JSONB encode + WAL fsync + network round-trip |
| `NsPerRead` | 5,000 | Indexed SELECT + JSONB decode + buffer cache hit |

Postgres is slower than Memory (500 ns/op) and SQLite (in-process) due to network round-trips and WAL fsync, but offers concurrent readers, ACID durability, and crash recovery.

### Module Isolation

Separate Go module with `replace github.com/larsartmann/go-cqrs-lite/metaengine/v4 => ../`. Consumers who don't import `pgengine` never pull in `pgx/v5` as a dependency.

### Testing

Tests skip gracefully when Postgres is unavailable (`POSTGRES_TEST_DSN` or `DATABASE_URL` env var). This mirrors the `stack/postgres` testcontainer pattern. With Docker, testcontainers-go provisions a fresh Postgres 16 container per test suite.

## Consequences

### Positive

- Production-ready engine for consumers with existing Postgres infrastructure
- JSONB enables future filtered scan pushdown (`WHERE value->>'field' = ?`)
- No CGo required (pure Go `pgx` driver)
- ACID durability for event-sourced projections

### Negative

- Network round-trip latency (~1ms per op vs Memory's ~500ns)
- Separate module adds versioning complexity
- JSONB decode overhead on every read (not zero-copy)
- No native vector/search/spatial backends (PostGIS/tsvector are future work)

## Future Enhancements

- **ScanBackend:** Filtered scan with `json_extract_path_text` pushdown
- **SearchBackend:** Using `tsvector` + `tsquery` (Postgres native FTS)
- **SpatialBackend:** Using PostGIS or `earthdistance` extension
- **GIN index generation:** Automatic GIN index on JSONB columns for query patterns
