# ADR-0120: Aggregate Pushdown Architecture

## Status

Accepted

## Context

The metaengine planner assigns queries to engines based on ADT classification
and cost estimation. For Map ADT queries, the default read path loads all rows
into Go memory and computes aggregates (COUNT, SUM, MIN, MAX, AVG) in-process.
For collections with thousands of rows, this is wasteful: SQL engines like
DuckDB, SQLite, and Postgres can compute these aggregates natively with
vectorized execution, avoiding the deserialization tax entirely.

Prior to this ADR, the metaengine had no mechanism for engines to declare
aggregate pushdown capabilities. Every aggregate computation was Go-side,
regardless of the underlying engine's SQL power.

## Decision

Introduce **five optional aggregate-pushdown interfaces** in
`metaengine/aggregations.go`. Engines implement whichever subset they support;
the planner and consumer API (TypedReader) type-assert at runtime.

### The Five Interfaces

| Interface | SQL Pattern | Use Case |
|---|---|---|
| `AggregateReader` | `SELECT COUNT/SUM/MIN/MAX/AVG(...)` | Scalar aggregates |
| `GroupedAggregateReader` | `SELECT group, AGG(...) ... GROUP BY group` | One aggregate per group |
| `MultiAggregateReader` | `SELECT AGG1(...), AGG2(...), ...` | Multiple scalars in one pass |
| `MultiGroupedAggregateReader` | `SELECT group, AGG1(...), AGG2(...) ... GROUP BY group` | Multiple aggregates per group |
| `DistinctReader` | `SELECT DISTINCT column` | Unique value enumeration |
| `ExplainableAggregate` | Returns SQL without executing | Debugging and plan inspection |

All six interfaces are **optional** — the Memory engine implements none of them
(Go-side accumulation is its only option). DuckDB implements all five readers
plus ExplainableAggregate. SQLite implements all five. Postgres implements all
five.

### Design Principles

1. **Aggregation goes to the engine, not to Go.** The interfaces push the SQL
   computation into the engine's native query processor. DuckDB's vectorized
   columnar execution makes GROUP BY 4.4x faster and MultiAggregate 2.1x faster
   at 100K rows compared to Go-side accumulation.

2. **Two execution paths per engine.** Each engine supports both a standard
   path (`json_extract(value, '$.field')` on the `meta_map` table) and a planned
   path (native SQL columns on a planned table). The planned path is faster
   because it avoids JSON parsing entirely. The engine checks its internal
   layout plan map to select the path.

3. **Cross-engine parity via shared test harness.** The `bench/aggregate_parity`
   test suite verifies DuckDB and SQLite produce identical results. The
   `pgengine/aggregations_test.go` suite provides equivalent coverage for
   Postgres. This ensures all three engines agree on edge cases (empty
   collections, negative values, NULL handling).

4. **Shared scan-value normalization.** `metaengine.DecodeFloat` normalizes
   driver-specific scan values (`float64`, `int64`, `*big.Int`, `[]byte`) to
   `float64`. Extracted from three duplicated copies (DuckDB, SQLite, PG) into
   the metaengine core as a single shared function.

5. **Explain without executing.** `ExplainableAggregate.ExplainAggregateQuery`
   returns the SQL string the engine WOULD run, without executing it. This
   lets consumers debug pushdown, verify index usage, and inspect query plans.

### Consumer API

`TypedReader[V]` exposes typed convenience methods that prefer pushdown:

```go
count, _ := reader.Count(ctx)
sum, _   := reader.Sum(ctx, "price")
groups, _ := reader.GroupedCount(ctx, "status")
```

These methods type-assert to the aggregate interfaces at runtime. If the engine
implements the interface, the computation is pushed to SQL. If not, TypedReader
falls back to loading all rows and computing in Go.

## Consequences

### Positive

- **Performance**: GROUP BY pushdown is 4.4x faster, MultiAggregate 2.1x faster
  at 100K rows on DuckDB. Scalar aggregates avoid loading any rows.
- **Cross-engine consistency**: The parity test suite guarantees all three SQL
  engines agree on results.
- **Progressive enhancement**: Engines opt-in by implementing interfaces. The
  Memory engine and graph adapter are unaffected.
- **Observability**: `Doctor()` shows pushdown capabilities per collection;
  `ExplainAggregateQuery` shows the generated SQL.

### Negative

- **Five new interfaces**: Increases the interface surface area. However, all
  are optional and engine-specific — consumer code only sees TypedReader.
- **Two code paths per aggregate operation**: Each engine must implement both
  standard (json_extract) and planned (native column) paths. This is necessary
  because layout plans are optional, but it doubles the test surface.
- **Driver-specific scan values**: Each database driver returns different Go
  types for aggregate results. The shared `DecodeFloat` normalizes this, but
  adding a new engine requires verifying its driver's scan types.

### Neutral

- **Plan serialization**: `SerializableQuery` now includes `ReadPattern`, which
  lets `PlanDiff` detect when a query's read pattern changes (e.g., from
  point_lookup to aggregate). This is additive to the JSON schema.
