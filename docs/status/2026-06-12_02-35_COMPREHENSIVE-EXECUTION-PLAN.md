# Comprehensive Execution Plan: Turso Indexing Hardening Sprint

**Date:** 2026-06-12 02:35 UTC
**Branch:** master (6 commits ahead of origin)
**Goal:** Transform turso/indexing from "good" to "production-grade"

## Scoring Legend

| Score | Meaning |
|-------|---------|
| Impact | 1=cosmetic → 10=critical (foot-gun, data loss, blocker) |
| Effort | minutes (assumes focused work, no context switching) |
| Value | Composite score = Impact / Effort × 10 (rounded) |

## Task Inventory (sorted by Value desc, then by Impact)

| # | ID | Task | Impact | Effort | Value | Category |
|---|----|------|--------|--------|-------|----------|
| 1 | T-001 | Rename `Recommendation.Reason` → `Explanation` in struct | 6 | 8 | 75 | Type clarity |
| 2 | T-002 | Update all `Recommendation{Reason: x}` literal sites | 6 | 6 | 100 | Type clarity |
| 3 | T-003 | Update doc.go example using `r.Reason` | 6 | 3 | 200 | Type clarity |
| 4 | T-004 | Remove dead `Recommendation.EstimatedCost` field | 4 | 3 | 133 | Dead code |
| 5 | T-005 | Add `TestAdvisor_WithExcludedTables` test | 7 | 8 | 88 | Coverage |
| 6 | T-006 | Add `TestAutoIndexer_WithAutoAnalyze` test | 7 | 8 | 88 | Coverage |
| 7 | T-007 | Add `TestAutoIndexer_maybeAnalyze` direct test | 5 | 5 | 100 | Coverage |
| 8 | T-008 | Add benchmark `BenchmarkReadFrom_WithIndexes` | 8 | 10 | 80 | Proof of value |
| 9 | T-009 | Add benchmark `BenchmarkReadFrom_WithoutIndexes` | 8 | 10 | 80 | Proof of value |
| 10 | T-010 | Add `telemetry.go` with OTel Tracer setup | 6 | 10 | 60 | Observability |
| 11 | T-011 | Add OTel span to `Advisor.AnalyzeQuery` | 7 | 8 | 88 | Observability |
| 12 | T-012 | Add OTel span to `AutoIndexer.Apply` | 7 | 8 | 88 | Observability |
| 13 | T-013 | Add OTel span to `AutoIndexer.ApplyCQRSIndexes` | 7 | 5 | 140 | Observability |
| 14 | T-014 | Add OTel span to `ApplyOptimizations` | 6 | 5 | 120 | Observability |
| 15 | T-015 | Add `init.go` with `Tracer()` function | 5 | 8 | 63 | Foundation |
| 16 | T-016 | Add `InitSchemaWithIndexesAndOptimizations` convenience | 7 | 5 | 140 | Ergonomics |
| 17 | T-017 | Update root `turso/indexing.go` with new convenience | 6 | 3 | 200 | Ergonomics |
| 18 | T-018 | Add test for `InitSchemaWithIndexesAndOptimizations` | 6 | 5 | 120 | Coverage |
| 19 | T-019 | Add `IndexUsageStats` helper using `PRAGMA index_info` | 7 | 10 | 70 | Observability |
| 20 | T-020 | Add test for `IndexUsageStats` | 6 | 8 | 75 | Coverage |
| 21 | T-021 | Add `AutoIndexer.Close()` method (lifecycle) | 4 | 3 | 133 | Lifecycle |
| 22 | T-022 | Add `Recommendation.Priority` enum (Critical/Recommended/Optional) | 6 | 8 | 75 | Type model |
| 23 | T-023 | Wire Priority into `inferIndex` and `recommendationFromDetail` | 7 | 10 | 70 | Type model |
| 24 | T-024 | Add benchmark comparison report generator | 6 | 10 | 60 | DX |
| 25 | T-025 | Add `indexing.DryRun` mode (print DDL, don't execute) | 5 | 8 | 63 | Safety |
| 26 | T-026 | Add test for `DryRun` mode | 5 | 5 | 100 | Coverage |
| 27 | T-027 | Add `turso/indexing/README.md` | 5 | 10 | 50 | Discoverability |
| 28 | T-028 | Fix `advisor.go` `fmt.Sprintf` in `analyze_table` (perfsprint) | 3 | 3 | 100 | Lint |
| 29 | T-029 | Rename `db` → `database` in 6 test files (varnamelen) | 2 | 10 | 20 | Lint |
| 30 | T-030 | Remove redundant `//nolint` comments (nolintlint) | 1 | 3 | 33 | Lint |
| 31 | T-031 | Add test for `isUnsupportedPragma` path | 5 | 8 | 63 | Coverage |
| 32 | T-032 | Add `AutoIndexer.Drop(ctx, indexes)` method | 5 | 8 | 63 | Lifecycle |
| 33 | T-033 | Add test for `AutoIndexer.Drop` | 5 | 5 | 100 | Coverage |
| 34 | T-034 | Add `indexing/CHANGELOG.md` entry | 3 | 5 | 60 | Docs |
| 35 | T-035 | Add `turso/indexing/example_test.go` for `InitSchemaWithIndexesAndOptimizations` | 4 | 5 | 80 | DX |
| 36 | T-036 | Add `AutoIndexer.RecommendAndApply` convenience | 6 | 8 | 75 | Ergonomics |
| 37 | T-037 | Add benchmark `BenchmarkAdvisor_MissingIndexes` | 5 | 8 | 63 | Performance |
| 38 | T-038 | Add `turso.WithIndexingHooks` option pattern | 4 | 10 | 40 | Extensibility |
| 39 | T-039 | Add test for `turso.WithIndexingHooks` | 4 | 5 | 80 | Coverage |
| 40 | T-040 | Add `turso/indexing/migration.go` for schema evolution | 8 | 12 | 67 | Schema evolution |
| 41 | T-041 | Add `turso/indexing/health.go` for index health checks | 7 | 10 | 70 | Health |
| 42 | T-042 | Add test for `health.go` | 6 | 5 | 120 | Coverage |
| 43 | T-043 | Add `indexing.Stats` to track advisor effectiveness | 6 | 8 | 75 | Metrics |
| 44 | T-044 | Update `docs/adr/` with CQRS indexing decision | 6 | 10 | 60 | Architecture |
| 45 | T-045 | Add `indexing/policy.go` with `Policy` type for table-specific rules | 5 | 10 | 50 | Extensibility |
| 46 | T-046 | Add `indexing.Version` field to track schema version | 5 | 8 | 63 | Type model |
| 47 | T-047 | Add `indexing.Compact` for dead tuple cleanup guidance | 4 | 8 | 50 | Operations |
| 48 | T-048 | Add `turso.ScheduleCheckpoint` background worker | 7 | 10 | 70 | Sync operations |

## Execution Order (by value, grouped by category)

### Group 1: Type Model Cleanup (15 min, T-001 → T-004)
- T-001: Rename `Recommendation.Reason` → `Explanation`
- T-002: Update struct literal sites
- T-003: Update doc.go example
- T-004: Remove `EstimatedCost`

### Group 2: OTel Foundation (40 min, T-010 → T-015)
- T-015: Add `init.go` with `Tracer()`
- T-010: Add `telemetry.go` with tracer setup
- T-011: OTel span for `AnalyzeQuery`
- T-012: OTel span for `Apply`
- T-013: OTel span for `ApplyCQRSIndexes`
- T-014: OTel span for `ApplyOptimizations`

### Group 3: Coverage Gaps (35 min, T-005 → T-007, T-018, T-020, T-026, T-031, T-033, T-042)
- T-005: Test `WithExcludedTables`
- T-006: Test `WithAutoAnalyze`
- T-007: Test `maybeAnalyze`
- T-018: Test `InitSchemaWithIndexesAndOptimizations`
- T-020: Test `IndexUsageStats`
- T-026: Test `DryRun`
- T-031: Test `isUnsupportedPragma`
- T-033: Test `AutoIndexer.Drop`
- T-042: Test `health.go`

### Group 4: Convenience Exports (13 min, T-016 → T-018, T-035)
- T-016: Add `InitSchemaWithIndexesAndOptimizations`
- T-017: Update root convenience
- T-018: Test
- T-035: Example test

### Group 5: Performance & Proof (38 min, T-008, T-009, T-024, T-037)
- T-008: Benchmark `ReadFrom_WithIndexes`
- T-009: Benchmark `ReadFrom_WithoutIndexes`
- T-024: Comparison report generator
- T-037: Benchmark `MissingIndexes`

### Group 6: Lifecycle & Safety (24 min, T-021, T-025, T-032, T-036)
- T-021: `AutoIndexer.Close()`
- T-025: `DryRun` mode
- T-032: `AutoIndexer.Drop()`
- T-036: `RecommendAndApply` convenience

### Group 7: Type Model Enrichment (18 min, T-022, T-023, T-046)
- T-022: `Recommendation.Priority` enum
- T-023: Wire Priority into inferIndex
- T-046: Add `Version` field

### Group 8: Observability (40 min, T-019, T-041, T-043, T-048)
- T-019: `IndexUsageStats` helper
- T-041: `health.go`
- T-043: `Stats` tracking
- T-048: `ScheduleCheckpoint`

### Group 9: Lint Cleanup (16 min, T-028, T-029, T-030)
- T-028: Fix `fmt.Sprintf` perfsprint
- T-029: Rename `db` → `database`
- T-030: Remove redundant nolint

### Group 10: Documentation (20 min, T-027, T-034, T-044)
- T-027: `turso/indexing/README.md`
- T-034: `CHANGELOG.md`
- T-044: ADR

### Group 11: Extensibility (20 min, T-038, T-039, T-045, T-047)
- T-038: `WithIndexingHooks` option
- T-039: Test
- T-045: `Policy` type
- T-047: `Compact` guidance

### Group 12: Schema Evolution (12 min, T-040)
- T-040: `migration.go`

**Total estimated effort:** ~4.5 hours of focused work
