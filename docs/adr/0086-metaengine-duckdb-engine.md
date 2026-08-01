# ADR-0086: DuckDB Metaengine Engine

**Date:** 2026-08-01
**Status:** Accepted
**Supersedes:** N/A
**Related:** [ADR-0085](0085-metaengine-new-adts.md) (new ADTs), [ADR-0062](0062-metaengine-dependency-boundary.md) (dependency boundary), [ADR-0084](0084-metaengine-layered-architecture.md) (layered architecture)

## Context

The metaengine has two existing engine implementations:
1. **MemoryEngine** — in-process, brute-force, O(N) scans. In the core module.
2. **PebbleEngine** — LSM point reads, separate module (cockroachdb/pebble dep).
3. **SQLiteEngine** — in core module, row-oriented, `json_extract` pushdown.

DuckDB is an embedded columnar (OLAP) database that excels at analytical workloads. It uses vectorized execution for GROUP BY aggregations, making Counter reads O(1) effectively. The layered architecture (ADR-0084) declares `LayoutColumnar` as a storage layout, and the cost matrix shows `(Counter, Columnar) → O(1)`. Without a DuckDB engine, this cost matrix entry has no backing implementation.

## Decision

Create `metaengine/duckdbengine/` as a separate Go module (CGo required) implementing `metaengine.Engine` with `MapBackend` and `CounterBackend`.

### Module structure

```
metaengine/duckdbengine/
├── doc.go               # Package documentation
├── drivers.go           # //go:build cgo — blank import for duckdb-go driver
├── engine.go            # duckdbEngine: Engine + MapBackend + CounterBackend
├── go.mod               # Module: duckdb-go + metaengine/v4 deps
└── engine_cgo_test.go   # //go:build cgo — 4 tests (Map, Counter, Profile, Plan integration)
```

### Cost model

```
DuckDBNsPerOp   = 15_000  (INSERT with JSON encode — columnar write amortized)
DuckDBNsPerRead =  3_000  (vectorized GROUP BY on hot columnar cache)
```

The write cost is higher than SQLite (7,000 ns) because DuckDB's columnar flush is more expensive for single-row inserts. The read cost is lower because DuckDB's vectorized execution makes GROUP BY extremely fast.

### Storage layout declarations

The engine declares `LayoutColumnar` for Map, Counter, and SortedMap ADTs in its `EngineProfile.Layouts` map. This is the first engine to use the `LayoutColumnar` layout, validating the cost matrix infrastructure from ADR-0084.

### CGo isolation

Following the pattern from `stack/duckdb`:
- `drivers.go` has `//go:build cgo` — blank imports the DuckDB driver
- `engine.go` has no build tag — uses `database/sql` (pure Go)
- Tests have `//go:build cgo` — skip when CGo is disabled
- The module is isolated so consumers who don't import it never need CGo

## Consequences

### Positive

- **Proves the columnar pushdown pattern** — the first engine to declare `LayoutColumnar`, validating that the cost matrix from ADR-0084 works with real engines
- **Counter reads are O(1)** — DuckDB's vectorized GROUP BY makes aggregate reads extremely fast, validating the cost matrix entry `(Counter, Columnar) → O(1)`
- **CGo is isolated** — the module is separate; non-importers never pay the CGo cost
- **Follows the pebbleengine pattern exactly** — same module structure, same replace directive for the local metaengine dep

### Negative

- **CGo required** — statically links the DuckDB C++ engine (~30-50MB binary). This is the same tradeoff as `stack/duckdb`.
- **Only Map + Counter backends** — Set, Graph, Multimap, Log, Vector, Search, Spatial are not implemented. These require specialized index structures that don't map naturally to columnar SQL.
- **Single-row inserts are expensive** — DuckDB's columnar format amortizes writes across batches. Individual MapSet calls pay the full columnar flush cost. For write-heavy workloads, the Memory engine or Pebble is preferred.

### Neutral

- **Placeholder syntax uses `$1, $2`** (Postgres-compatible) — DuckDB follows Postgres syntax, not SQLite `?` placeholders. This is handled correctly by the engine's SQL strings.
