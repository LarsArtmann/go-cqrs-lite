# ADR-0092: DuckDB Columnar-Native Storage via LayoutPlanApplier

Date: 2026-08-02

## Status

Accepted

## Context

The metaengine's DuckDB engine stores values as JSON blobs in a `meta_map`
table. Even with the existing `LayoutPlanner` (filter/sort field extraction
into a planned table), the extracted columns used name-heuristic type
inference (`inferColumnType`), which guessed INTEGER for field names
containing "price", "count", etc., and TEXT for everything else.

This caused two problems:

1. **No native columnar scans.** DuckDB's vectorized execution engine could
   not leverage columnar scans for aggregations (GROUP BY, SUM, AVG) because
   non-extracted fields were only available via JSON decode.

2. **Incorrect type inference.** A `float64` field named "price" was
   classified as INTEGER by the name heuristic, silently truncating values.
   Fields with non-obvious names fell back to TEXT, preventing numeric
   pushdown.

## Decision

Introduce a **columnar-native layout** via two new abstractions:

### 1. `LayoutPlanApplier` interface

```go
type LayoutPlanApplier interface {
    LayoutPlanner
    ApplyLayoutPlan(plan LayoutPlan) error
}
```

Engines that implement this receive the fully-built `LayoutPlan` with
reflection-derived column types, instead of rebuilding from field names.
This is a backward-compatible extension: `LayoutPlanApplier` embeds
`LayoutPlanner`, so the dispatch in `applyLayoutPlan` prefers the new
interface and falls back to the old one.

Currently implemented by DuckDB only. SQLite and Postgres fall back to
`LayoutPlanner.ApplyLayout` with name heuristics.

### 2. `WithColumnarLayout()` query option

```go
metaengine.Query[Input, ProductView]("products",
    metaengine.On(ProductCreated{}, ...),
    metaengine.WithColumnarLayout(),
)
```

When set, the planner calls `BuildColumnarLayoutPlan` which reflects on the
result type `R` and extracts **every exported field** into a native SQL
column with an accurate type:

| Go type          | SQL type |
| ---------------- | -------- |
| int, int8..int64 | INTEGER  |
| uint, uint8..uint64 | INTEGER |
| float32          | REAL     |
| float64          | DOUBLE   |
| bool             | INTEGER  |
| string           | TEXT     |
| everything else  | TEXT     |

Fields named "key" or "value" (case-insensitive) are skipped to avoid
collisions with the base table's primary key and JSON blob columns.

### Type coercion

JSON decoding turns all numbers into `float64`. The DuckDB engine's
`coerceForColumn` maps these back to the declared SQL type:
INTEGER columns get `int64` values, DOUBLE/REAL columns get `float64`.

## Consequences

### Positive

- DuckDB runs vectorized GROUP BY, SUM, AVG directly on native columns
  without JSON decode overhead.
- Accurate float64 precision via DOUBLE (FLOAT8), not REAL (FLOAT4).
- Reflection-based type mapping replaces fragile name heuristics for
  columnar queries.
- Backward compatible: existing `LayoutPlanner` implementations are
  unaffected.

### Negative

- Only DuckDB benefits from accurate types. SQLite/Postgres still use name
  heuristics via the `LayoutPlanner` fallback path.
- No schema evolution: if the result type adds a new field, the existing
  table does not get `ALTER TABLE ADD COLUMN`. The new field is silently
  dropped from the planned columns. A future enhancement could add
  `ALTER TABLE` support.
- The `LayoutPlanApplier` interface adds a second dispatch path, increasing
  the cognitive load of the layout system. The tradeoff is backward
  compatibility vs. a single unified interface.

## Alternatives Considered

### Change `LayoutPlanner.ApplyLayout` signature

Rejected: would break all existing `LayoutPlanner` implementations (SQLite,
Postgres, Pebble). The `LayoutPlanApplier` extension preserves backward
compatibility.

### Native DuckDB columnar storage format

Rejected: would break the generic `Store` abstraction that all engines
share. The planned-table approach maintains cross-engine parity.

### Auto-enable columnar for Counter/Aggregate read patterns

Considered for the future but rejected as the default for now: implicit
behavior changes are dangerous in a library. `WithColumnarLayout()` is
explicit opt-in.
