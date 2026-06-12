# Comprehensive Status Report: Turso Auto-Smart Indexing

**Date:** 2026-06-10 21:42 UTC  
**Branch:** master  
**Commit Base:** 6e6e5498 (docs(status): add comprehensive zero-lint completion report)  
**Session Focus:** turso/indexing sub-package — auto-smart index management for CQRS workloads

---

## a) FULLY DONE ✅

### 1. turso/indexing Sub-Package — COMPLETE

A brand-new `turso/indexing` sub-package providing auto-smart index management for Turso/LibSQL databases:

| File               | Lines   | Purpose                                                                  | Tests                     |
| ------------------ | ------- | ------------------------------------------------------------------------ | ------------------------- |
| `doc.go`           | 25      | Package docs with quick-start examples                                   | ✅                        |
| `index.go`         | 108     | `Index`, `IndexSet`, `RecommendedCQRSIndexes()`                          | ✅ 8 tests                |
| `advisor.go`       | 378     | `Advisor` — EXPLAIN QUERY PLAN analyzer, scan detection, index inference | ✅ 7 tests                |
| `auto.go`          | 102     | `AutoIndexer` — optional automatic index creation (disabled by default)  | ✅ 5 tests                |
| `optimizations.go` | 109     | PRAGMA helpers: WAL, cache, mmap, analyze, optimize                      | ✅ 8 tests                |
| `example_test.go`  | 85      | 5 runnable examples for pkg.go.dev                                       | ✅                        |
| **Total**          | **807** | **6 source files**                                                       | **28 tests, all passing** |

### 2. Convenience Re-Exports in turso Root

| Function                                 | Wraps                             | Purpose                        |
| ---------------------------------------- | --------------------------------- | ------------------------------ |
| `turso.NewIndexAdvisor(db)`              | `indexing.NewAdvisor`             | Query plan analysis            |
| `turso.NewAutoIndexer(db)`               | `indexing.NewAutoIndexer`         | Auto-index management          |
| `turso.ApplyTursoOptimizations(ctx, db)` | `indexing.ApplyOptimizations`     | Performance PRAGMAs            |
| `turso.ApplyCQRSIndexes(ctx, db)`        | `auto.ApplyCQRSIndexes`           | Idempotent CQRS index creation |
| `turso.InitSchemaWithIndexes(ctx, db)`   | `InitSchema` + `ApplyCQRSIndexes` | One-shot setup                 |

### 3. turso/README.md Updated

Added comprehensive "Auto-Smart Indexing" section with:

- Quick start (`InitSchemaWithIndexes`)
- Recommended CQRS indexes table
- Index Advisor usage
- Auto-Indexer usage
- Performance optimizations
- Index definition helpers

### 4. turso/indexing_test.go Added

5 integration tests verifying root-level convenience functions work end-to-end:

- `TestNewIndexAdvisor`
- `TestNewAutoIndexer`
- `TestApplyTursoOptimizations`
- `TestApplyCQRSIndexes`
- `TestInitSchemaWithIndexes`

### 5. CQRS-Optimized Indexes Defined

| Index Name               | Table    | Columns                                       | Purpose                                           |
| ------------------------ | -------- | --------------------------------------------- | ------------------------------------------------- |
| `idx_events_cursor`      | events   | `(occurred_at, id)`                           | Cursor pagination for `ReadFrom` / journal replay |
| `idx_events_agg_ver`     | events   | `(aggregate_type, aggregate_id, version)`     | Covering index for version-range loads            |
| `idx_events_type_time`   | events   | `(event_type, occurred_at)`                   | Projection filters by event type                  |
| `idx_commands_agg_time`  | commands | `(aggregate_type, aggregate_id, received_at)` | Command audit trail ordering                      |
| `idx_commands_type_time` | commands | `(command_type, received_at)`                 | Command type analytics                            |

### 6. Cross-Module Test Verification

All 34 test suites pass:

```
✅ event/v2           (89.4% coverage)
✅ event/v2/eventtest (18.4% coverage)
✅ storage/v2         (86.8% coverage)
✅ storage/v2/sql     (34.7% coverage)
✅ turso/v2           (38.1% coverage)
✅ turso/v2/indexing  (72.7% coverage)
✅ All other 28 modules
```

### 7. Build Verification

`nix run .#build` — ✅ PASS

---

## b) PARTIALLY DONE 🟡

### 1. turso/v2 Coverage (38.1%)

The turso root module has lower coverage because the `sync.go` remote sync code requires a real Turso server connection. The in-memory tests cannot exercise `OpenSync`, `Push`, `Pull`, `Checkpoint`, or `Stats`. Coverage is concentrated in `connector.go` and `indexing.go`.

**Mitigation:** The `connector_test.go` already has a test for `OpenSync_MemoryWithRemote` that validates the rejection path.

### 2. Lint — turso/indexing (48 remaining issues)

Breakdown of remaining lint issues in `turso/indexing`:

| Linter      | Count | Severity | Notes                                                         |
| ----------- | ----- | -------- | ------------------------------------------------------------- |
| exhaustruct | 13    | LOW      | Struct literal field initialization — test code only          |
| goconst     | 11    | LOW      | String literals repeated in test SQL                          |
| noinlineerr | 9     | LOW      | `errors.New` without variable — common in test setup          |
| nolintlint  | 1     | LOW      | One `//nolint` comment may be redundant                       |
| perfsprint  | 5     | LOW      | `fmt.Sprintf` in error messages (minor perf)                  |
| unqueryvet  | 3     | LOW      | `SELECT *` in test queries (intentional for EXPLAIN analysis) |
| varnamelen  | 6     | LOW      | `db` variable name in tests (project convention allows this)  |

**None of these are functional issues.** They are style/consistency items. The `varnamelen` and `unqueryvet` issues are in test code where `db` and `SELECT *` are conventional and intentional.

---

## c) NOT STARTED 🔵

### 1. Real Turso Remote Sync Integration Tests

`turso/sync.go` `OpenSync`, `Push`, `Pull`, `Checkpoint`, `Stats` are tested only for the rejection path (in-memory + remote). Real remote sync tests would need:

- A live Turso database URL + auth token
- CI secrets configuration
- Potentially a mock Turso server

### 2. Performance Benchmarks for Indexed vs Unindexed Queries

No before/after benchmark exists showing query plan improvements from the new indexes. Would be valuable to add to `turso/benchmark_test.go`.

### 3. turso/indexing Coverage Target (72.7% → 90%+)

The `advisor.go` `explain()` and `userTables()` methods have paths not fully exercised. The `isUnsupportedPragma` helper is covered indirectly.

### 4. Index Monitoring / Health Check Integration

The `listing` module has `InMemoryAggregateReader` and tombstone detection. No integration exists between `turso/indexing` and the health-check middleware pattern used in other modules.

### 5. Documentation on pkg.go.dev

Examples are present but module needs to be tagged for consumers to see the new `turso/indexing` package on pkg.go.dev.

---

## d) TOTALLY FUCKED UP! 🔴

**Nothing.** All tests pass. Build passes. No panics. No data loss risks. The `isUnsupportedPragma` helper correctly silences errors for LibSQL variants that don't support `mmap_size` or `PRAGMA optimize`, preventing false failures.

---

## e) WHAT WE SHOULD IMPROVE! 🟢

### Immediate (Next Session)

1. **Add benchmark comparing indexed vs unindexed `ReadFrom`** — Quantify the cursor pagination improvement.
2. **Add `turso/indexing` README** — The sub-package deserves its own `indexing/README.md` for consumers who discover it independently.
3. **Cover `isUnsupportedPragma` explicitly** — Add a test that forces the pragma-unsupported path.

### Short-Term (Next 3 Sessions)

4. **Real Turso sync integration test stub** — Add a `sync_integration_test.go` with build tags (`//go:build integration`) that can be run manually with env vars.
5. **Index usage statistics helper** — `indexing.IndexUsageStats(ctx, db)` that queries `sqlite_stat1` or `PRAGMA index_info` to report which indexes are actually being used.
6. **Index dropping / cleanup** — `AutoIndexer.CleanupUnused(ctx)` to remove indexes with zero query hits.
7. **Migrate remaining `fmt.Sprintf` to string concatenation** — Fix 5 remaining `perfsprint` issues.

### Medium-Term (Next 2 Weeks)

8. **Integration with `listing` module** — Add index-aware aggregate listing that validates expected indexes exist before running queries.
9. **Schema evolution index migration** — When `schema` upcasters change event types, auto-indexer could detect and update `idx_events_type_time` coverage.
10. **WAL checkpoint scheduling** — Add `turso.ScheduleCheckpoint(interval)` for long-running sync databases.

---

## f) Top #25 Things We Should Get Done Next!

| #   | Priority    | Task                                                                   | Module         | Effort  |
| --- | ----------- | ---------------------------------------------------------------------- | -------------- | ------- |
| 1   | 🔴 CRITICAL | Benchmark indexed vs unindexed `ReadFrom`                              | turso          | 30 min  |
| 2   | 🔴 CRITICAL | Add `indexing/README.md`                                               | turso/indexing | 20 min  |
| 3   | 🔴 CRITICAL | Tag v2.2.1 with turso/indexing                                         | repo           | 10 min  |
| 4   | 🟡 HIGH     | Real Turso sync integration tests (build-tagged)                       | turso          | 2 hrs   |
| 5   | 🟡 HIGH     | Index usage statistics (`sqlite_stat1` reader)                         | turso/indexing | 1.5 hrs |
| 6   | 🟡 HIGH     | Cleanup unused indexes helper                                          | turso/indexing | 1 hr    |
| 7   | 🟡 HIGH     | Cover `isUnsupportedPragma` in unit test                               | turso/indexing | 15 min  |
| 8   | 🟡 HIGH     | Fix 5 remaining `perfsprint` lint issues                               | turso/indexing | 20 min  |
| 9   | 🟡 HIGH     | Fix 6 `varnamelen` in tests (rename `db` → `database` where practical) | turso/indexing | 30 min  |
| 10  | 🟢 MEDIUM   | `listing` integration: validate indexes before aggregate reads         | listing        | 2 hrs   |
| 11  | 🟢 MEDIUM   | WAL checkpoint scheduling helper                                       | turso          | 1 hr    |
| 12  | 🟢 MEDIUM   | Add `indexing.BenchmarkAdvisor_MissingIndexes`                         | turso/indexing | 30 min  |
| 13  | 🟢 MEDIUM   | Document index trade-offs (write amplification vs read speed)          | docs           | 45 min  |
| 14  | 🟢 MEDIUM   | `projection` integration: auto-apply CQRS indexes on runner start      | projection     | 1.5 hrs |
| 15  | 🟢 MEDIUM   | `storage/sql` EXPLAIN output parser (extract estimated cost)           | storage/sql    | 2 hrs   |
| 16  | 🟢 MEDIUM   | Add `Index.RecommendationConfidence` score (0-1)                       | turso/indexing | 1 hr    |
| 17  | 🟢 MEDIUM   | `catalog` auto-document indexes in OpenAPI/EventCatalog output         | catalog        | 3 hrs   |
| 18  | 🟢 MEDIUM   | CI job: run `ApplyCQRSIndexes` + verify with `EXPLAIN`                 | .github        | 1 hr    |
| 19  | 🟢 MEDIUM   | `middleware` metrics for index creation events                         | middleware     | 1 hr    |
| 20  | 🟢 MEDIUM   | `example/turso-indexing` usage demo                                    | examples       | 2 hrs   |
| 21  | 🟢 LOW      | Index partial covering (INCLUDE columns) for SQLite 3.35+              | turso/indexing | 2 hrs   |
| 22  | 🟢 LOW      | Multi-column index cardinality estimator                               | turso/indexing | 3 hrs   |
| 23  | 🟢 LOW      | `indexing.DryRun` mode that prints DDL without executing               | turso/indexing | 30 min  |
| 24  | 🟢 LOW      | `indexing.Rollback(ctx, appliedIndexes)` for migrations                | turso/indexing | 1 hr    |
| 25  | 🟢 LOW      | Publish ADR: "Why we ship pre-calculated CQRS indexes"                 | docs/adr       | 1 hr    |

---

## g) Top #1 Question I Cannot Figure Out Myself ❓

**"Should `InitSchema` in the `storage` module itself include the CQRS-optimized indexes, or should index creation remain an explicit opt-in via `turso.InitSchemaWithIndexes` / `turso.ApplyCQRSIndexes`?"**

Arguments for inclusion in base schema:

- CQRS consumers will almost always want these indexes
- One less step for new users
- Prevents performance foot-guns from forgetting to add indexes

Arguments for explicit opt-in:

- Some consumers may have custom index strategies
- Write-heavy workloads may want fewer indexes
- The `storage` module is dialect-agnostic (PG + SQLite); CQRS indexes are SQLite-specific
- Keeps `storage` minimal and compositional (library principle)

**This is a design philosophy question, not a technical one.** My recommendation is to keep indexes opt-in at the `storage` level and provide the convenience wrapper at the `turso` level, but I'd like confirmation before documenting this as a permanent design decision.

---

## Module Health Snapshot

| Module               | Tests | Coverage | Lint        | Status                  |
| -------------------- | ----- | -------- | ----------- | ----------------------- |
| event/v2             | ✅    | 89.4%    | ✅ Zero     | 🟢 Healthy              |
| storage/v2           | ✅    | 86.8%    | ✅ Zero     | 🟢 Healthy              |
| storage/v2/sql       | ✅    | 34.7%    | ✅ Zero     | 🟢 Healthy              |
| turso/v2             | ✅    | 38.1%    | ✅ Zero     | 🟡 OK (sync untestable) |
| turso/v2/indexing    | ✅    | 72.7%    | 🟡 48 minor | 🟡 Good                 |
| All 28 other modules | ✅    | 67-100%  | ✅ Zero     | 🟢 Healthy              |

---

## Files Changed This Session

```
M  turso/README.md              (+112 lines — Auto-Smart Indexing docs)
A  turso/indexing.go            (+45 lines — Convenience re-exports)
A  turso/indexing_test.go       (+96 lines — Integration tests)
A  turso/indexing/doc.go        (+25 lines — Package documentation)
A  turso/indexing/index.go      (+108 lines — Index definition helpers)
A  turso/indexing/index_test.go (+103 lines — Unit tests)
A  turso/indexing/advisor.go    (+378 lines — EXPLAIN PLAN analyzer)
A  turso/indexing/advisor_test.go (+67 lines — Unit tests)
A  turso/indexing/auto.go      (+102 lines — Auto-indexer)
A  turso/indexing/auto_test.go  (+60 lines — Unit tests)
A  turso/indexing/optimizations.go (+109 lines — PRAGMA helpers)
A  turso/indexing/optimizations_test.go (+83 lines — Unit tests)
A  turso/indexing/example_test.go (+85 lines — pkg.go.dev examples)
```

**Total: 12 new files, 1 modified file, ~1,461 lines of new code.**
