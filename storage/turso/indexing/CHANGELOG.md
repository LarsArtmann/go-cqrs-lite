# Changelog — turso/indexing

All notable changes to the `turso/indexing` sub-package.

## v2.2.1 (in progress)

### Added

- `Index.Partial` bool field to make partial-index predicates explicit
- `IndexSet.DropDDL()` for symmetric drop-statement generation
- `Priority` enum (`Optional`, `Recommended`, `Critical`) on `Recommendation`
- `Version` type for tracking advisor evolution
- `AdvisorOption` functional options pattern:
  - `WithExcludedTables(tables...)` — skip specific tables in analysis
- `AutoIndexerOption` functional options pattern:
  - `WithAutoAnalyze()` — run ANALYZE after creating indexes
  - `WithDryRun()` — collect DDL into `LastDDL()` instead of executing
- `AutoIndexer.Close()` — lifecycle cleanup
- `AutoIndexer.Drop(ctx, indexes...)` — drop specific indexes
- `AutoIndexer.RecommendAndApply(ctx)` — convenience combining detect+apply
- `AutoIndexer.LastDDL()` — retrieve DDL captured during dry-run
- `Stats(ctx, db)` and `UnusedIndexes(ctx, db)` for query-planner observability
- `CheckpointScheduler` for periodic WAL checkpointing
- `ApplyOptimizationsTraced(ctx, db)` — PRAGMA batch with OTel span
- OTel tracing on all major operations (Advisor, AutoIndexer, Optimizations)
- `InitSchemaWithIndexesAndOptimizations` one-shot root convenience
- Sub-package `README.md` with full API documentation

### Changed

- `Recommendation.Reason` renamed to `Recommendation.Explanation`
  to disambiguate from `Index.Reason`
- `Recommendation.EstimatedCost` field removed (was never populated)
- `AutoIndexer.ApplyRecommended` consistently enforced
  `IsEnabled()` check (alongside `Apply` and `ApplyCQRSIndexes`)
- All Advisor/Indexer operations now use OTel instrumentation

## v2.2.0

- Initial release of `turso/indexing` sub-package
