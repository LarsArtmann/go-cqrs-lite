# ADR-0121: ApplyLayoutPlan Post-Construction Registration

## Status

Accepted

## Context

The metaengine's `LayoutPlanApplier` interface (implemented by DuckDB and
SQLite engines) generates DDL from declared query patterns. However, when a
consumer constructs an engine and later registers additional queries (e.g.,
via `projectionadapter` after the engine is already created), the layout plan
must be applied to the already-running engine.

The current API requires the layout plan to be passed at construction time
(`NewPlannedSQLiteEngine(db, plans)`). This forces consumers to know all
queries upfront, which is impractical for:

- Plugins that register projections after engine creation
- Dynamic query registration (runtime-discovered schemas)
- Test setups that build engines incrementally

## Decision

Add an `ApplyLayoutPlan(plans []LayoutPlan) error` method to engines that
implement `LayoutPlanApplier`. This allows post-construction DDL execution
for dynamically registered queries.

```go
eng, _ := sqliteengine.NewSQLiteEngine(db)
// ... later, after discovering query patterns ...
eng.(metaengine.LayoutPlanApplier).ApplyLayoutPlan(plans)
```

The method is idempotent: applying the same plan twice is a no-op (CREATE
TABLE IF NOT EXISTS semantics). Conflicting plans (different columns for the
same table) return `ErrLayoutConflict`.

## Consequences

- Consumers can register queries incrementally
- The engine must handle DDL execution on a live database (SQLite supports
  this natively; DuckDB requires careful transaction handling)
- The layout plan is NOT enforced at query time — consumers who skip
  ApplyLayoutPlan get unordered table scans (correct, just slower)
