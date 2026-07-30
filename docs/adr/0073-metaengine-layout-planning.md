# ADR-0073: Metaengine Layout Planning (deployment-time DDL)

|             |                                                                         |
| ----------- | ----------------------------------------------------------------------- |
| **Status**  | Accepted                                                                |
| **Date**    | 2026-07-29                                                              |
| **Context** | json_extract pushdown still scans every row; no secondary index is used |

## Context

ADR-0072 pushes `WHERE json_extract(value,'$.f') = ?` into SQLite. This removes
the Go-side filter/sort, but `json_extract` still evaluates against **every**
row in the collection — there is no index on the extracted field. For selective
filters at scale (e.g. "open" out of 1M tasks) the scan remains O(N).

SQLite cannot index a JSON field inside a generic blob without a generated
column or a dedicated table with real columns.

## Decision

Add **layout planning**: a `LayoutPlan` describes a dedicated table
(`meta_planned_<collection>`) whose declared filter/sort fields become real,
indexed SQL columns instead of JSON-extracted values. `BuildLayoutPlan` (and the
reflection-based `BuildLayoutPlanFromType[R]`) generate the plan; the
`plannedSQLiteEngine` materializes it:

- **Writes** split the JSON value: the full object still goes in a `value TEXT`
  column (so reads stay self-describing), and each planned column gets its own
  extracted column populated on `MapSet`.
- **Pushdown reads** on a planned collection use direct column references
  (`WHERE status = ? ORDER BY priority`) backed by indexes — O(log N) instead of
  O(N). Unplanned collections fall back to the standard `meta_map` +
  `json_extract` path.

The "Don't Be Stupid" rules (one table + N indexes, never index an unfiltered
column, dedup a column that is both filtered and sorted) are encoded in
`BuildLayoutPlan`. Column types are inferred from Go reflection via
`BuildLayoutPlanFromType[R]` (int→INTEGER, float→REAL, bool→INTEGER, else TEXT),
falling back to the name heuristic when the result type is unknown.

A 100K-row stress test (`TestPlannedEngine_Stress100K`) and a fallback test
(`TestPlannedEngine_FallbackToMetaMap`) pin correctness: selective counts,
ordered limited pages, and graceful degradation for unplanned collections.

## Consequences

- Auto-layout is wired into `Plan()`: when the assigned engine implements
  `LayoutPlanner` and the query declares filter/sort fields via
  `FilterOnField`/`SortOnField`, the plan is generated and applied automatically.
  Manual `NewPlannedSQLiteEngine` is still available for explicit control over
  column types (e.g., `BuildLayoutPlanFromType[R]`).
- The `LayoutPlanner` interface is an optional capability — engines that cannot
  create optimized layouts (memory, Pebble today) simply don't implement it,
  and `Plan()` silently skips the auto-layout step.
- Raw value readers (`RawValueReader`, `RawScanReader`) and `TypedReader[V]`
  shipped alongside auto-layout as the Level-2.5 optimization: single-pass JSON
  decode (1 op instead of 3) for both point lookups and filtered scans.
- Planned and unplanned collections coexist on one engine.
- This is the Level-2 optimization: within-engine layout. Level-1 (ADR-0072) was
  within-engine pushdown. A future Level-3 (generated typed read API) would make
  reads fully typed without `ExecuteTyped` or `TypedReader`.

## Alternatives considered

- **Generated columns (`GENERATED ALWAYS AS (json_extract(...))`)**: avoids a
  separate table but ties the schema to JSON field paths and complicates writes.
  Rejected in favor of explicit extracted columns for clarity and portability.
- **One projection per (filter, sort) combo**: explicitly forbidden by the rules
  — it multiplies write amplification.
