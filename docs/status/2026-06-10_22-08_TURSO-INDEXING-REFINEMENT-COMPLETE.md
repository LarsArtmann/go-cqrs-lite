# Comprehensive Status Report: Turso Indexing Refinement

**Date:** 2026-06-10 22:08 UTC
**Branch:** master
**Commits since last report:** 4 (9eaaf5b5 → 4c99bb81)
**Session Focus:** Post-implementation refinement of turso/indexing sub-package

---

## Work Done Since Last Report

### Commit 1: 9eaaf5b5 — Fix AutoIndexer consistency bug

**Problem:** `AutoIndexer.Apply()` and `ApplyCQRSIndexes()` bypassed `IsEnabled()` check, creating a surprising API where only `ApplyRecommended()` respected the enabled flag.

**Fix:** All three mutation methods now consistently check `IsEnabled()` before executing.

**Convenience exports updated:** `turso.ApplyCQRSIndexes()` now creates an AutoIndexer, calls `Enable()`, then applies — preserving the one-shot "just do it" behavior for consumers.

**Impact:** HIGH — eliminated a foot-gun where `Disable()` didn't actually prevent index creation.

### Commit 2: 2a9dbd70 — Add Index.Partial bool + IndexSet.DropDDL

**Index.Partial:** New boolean flag making partial-index predicates explicit. Previously `Where` alone implied partial, but the intent wasn't discoverable from the type.

**IndexSet.DropDDL():** Symmetric with `DDL()`. Returns `[]string` of `DROP INDEX IF EXISTS ...` statements. Useful for:

- Rollback scripts
- Migration down-paths
- Testing (tear down indexes between benchmarks)

**Impact:** MEDIUM — API completeness and clarity.

### Commit 3: 5b7802b8 — Update turso/doc.go

Package documentation now references `turso/indexing` sub-package and convenience functions (`InitSchemaWithIndexes`, `ApplyCQRSIndexes`), improving discoverability from the root package godoc.

**Impact:** LOW — discoverability.

### Commit 4: 4c99bb81 — Add functional options pattern

**AdvisorOption:**

```go
advisor := indexing.NewAdvisor(db, indexing.WithExcludedTables("sqlite_stat1", "sqlite_stat4"))
```

Prevents the advisor from analyzing specific tables during `MissingIndexes()`.

**AutoIndexerOption:**

```go
auto := indexing.NewAutoIndexer(db, indexing.WithAutoAnalyze())
```

Runs `ANALYZE` automatically after `Apply()` or `ApplyCQRSIndexes()`, ensuring SQLite's query planner has fresh statistics.

Follows the project's existing `SQLEventStoreOption` pattern from `storage/options.go`.

**Impact:** MEDIUM — extensibility and planner quality.

---

## a) FULLY DONE ✅

| #   | Item                                  | Status                |
| --- | ------------------------------------- | --------------------- |
| 1   | AutoIndexer consistency bug fix       | ✅ Committed 9eaaf5b5 |
| 2   | Index.Partial bool                    | ✅ Committed 2a9dbd70 |
| 3   | IndexSet.DropDDL()                    | ✅ Committed 2a9dbd70 |
| 4   | turso/doc.go indexing mention         | ✅ Committed 5b7802b8 |
| 5   | AdvisorOption pattern                 | ✅ Committed 4c99bb81 |
| 6   | AutoIndexerOption pattern             | ✅ Committed 4c99bb81 |
| 7   | WithExcludedTables filter             | ✅ Committed 4c99bb81 |
| 8   | WithAutoAnalyze option                | ✅ Committed 4c99bb81 |
| 9   | Auto-run ANALYZE after index creation | ✅ Committed 4c99bb81 |
| 10  | All tests pass (turso + indexing)     | ✅ 100% pass rate     |

---

## b) PARTIALLY DONE 🟡

### 1. Recommendation type model cleanup

`Recommendation.Reason` still collides semantically with `Index.Reason`. The former is "why we recommend this" (explanation), the latter is "what this index is for" (purpose). Renaming `Recommendation.Reason` → `Explanation` would clarify.

`Recommendation.EstimatedCost` field exists but is never populated. Should either be removed or implemented by parsing cost from EXPLAIN output.

### 2. Coverage (turso/v2/indexing: 70.4%)

Good but not great. Untested paths:

- `Advisor` with `WithExcludedTables` option
- `AutoIndexer` with `WithAutoAnalyze` option
- `maybeAnalyze()` when `autoAnalyze` is false (default path covered)
- `isUnsupportedPragma()` with actual unsupported pragma errors

### 3. Lint (48 → ~45 remaining minor issues)

The functional options added new `varnamelen` instances in test code. No functional lint issues remain.

---

## c) NOT STARTED 🔵

### 1. OTel tracing for indexing operations

The rest of the project traces everything (event store, command dispatch, query execution). Indexing operations are a blind spot:

- `Advisor.AnalyzeQuery` could trace EXPLAIN execution time
- `AutoIndexer.ApplyCQRSIndexes` could trace DDL execution time
- `ApplyOptimizations` could trace PRAGMA execution time

### 2. Performance benchmarks

No before/after benchmarks proving that CQRS indexes improve query performance. Need:

- `BenchmarkReadFrom_WithIndexes` vs `BenchmarkReadFrom_WithoutIndexes`
- `BenchmarkLoadFromVersion_WithIndexes` vs `BenchmarkLoadFromVersion_WithoutIndexes`

### 3. Index usage statistics helper

`indexing.IndexUsageStats(ctx, db)` — query `sqlite_stat1` or `PRAGMA index_list` + `PRAGMA index_info` to report which indexes are actually being used by the query planner.

### 4. Index dropping / cleanup

`AutoIndexer.CleanupUnused(ctx)` to remove indexes with zero query hits.

### 5. Real Turso sync integration tests

`sync.go` `OpenSync`, `Push`, `Pull`, `Checkpoint`, `Stats` still only tested for rejection path.

---

## d) TOTALLY FUCKED UP! 🔴

**Nothing.** All 5 commits build, test, and run clean. Working tree is clean.

---

## e) WHAT WE SHOULD IMPROVE! 🟢

### Immediate (Next Session)

1. **Rename `Recommendation.Reason` → `Explanation`** — Remove semantic collision with `Index.Reason`
2. **Remove dead `Recommendation.EstimatedCost`** — Or implement it by parsing EXPLAIN cost
3. **Add benchmark for indexed vs unindexed queries** — Prove the value proposition
4. **Test `WithExcludedTables` and `WithAutoAnalyze` options** — Coverage gaps

### Short-Term (Next 3 Sessions)

5. **Add OTel tracing to indexing operations** — Observability parity with rest of project
6. **Add `indexing.IndexUsageStats(ctx, db)`** — Query planner feedback loop
7. **Add `AutoIndexer.CleanupUnused(ctx)`** — Index lifecycle management
8. **Add `InitSchemaWithIndexesAndOptimizations` convenience** — One-shot everything

### Medium-Term (Next 2 Weeks)

9. **Integration with `listing` module** — Validate expected indexes before aggregate reads
10. **Schema evolution index migration** — When upcasters change event types, detect and update indexes
11. **WAL checkpoint scheduling** — `turso.ScheduleCheckpoint(interval)` for long-running sync DBs
12. **CI job: verify indexes with EXPLAIN** — Fail build if CQRS queries scan tables

---

## f) Top #15 Things We Should Get Done Next!

| #   | Priority    | Task                                                 | Module         | Effort  |
| --- | ----------- | ---------------------------------------------------- | -------------- | ------- |
| 1   | 🔴 CRITICAL | Rename `Recommendation.Reason` → `Explanation`       | turso/indexing | 10 min  |
| 2   | 🔴 CRITICAL | Remove dead `Recommendation.EstimatedCost`           | turso/indexing | 5 min   |
| 3   | 🔴 CRITICAL | Benchmark indexed vs unindexed `ReadFrom`            | turso          | 30 min  |
| 4   | 🟡 HIGH     | Test `WithExcludedTables` and `WithAutoAnalyze`      | turso/indexing | 20 min  |
| 5   | 🟡 HIGH     | Add OTel tracing to indexing operations              | turso/indexing | 25 min  |
| 6   | 🟡 HIGH     | Add `indexing.IndexUsageStats(ctx, db)`              | turso/indexing | 1.5 hrs |
| 7   | 🟡 HIGH     | Real Turso sync integration tests (build-tagged)     | turso          | 2 hrs   |
| 8   | 🟢 MEDIUM   | Add `AutoIndexer.CleanupUnused(ctx)`                 | turso/indexing | 1 hr    |
| 9   | 🟢 MEDIUM   | Add `InitSchemaWithIndexesAndOptimizations`          | turso          | 15 min  |
| 10  | 🟢 MEDIUM   | `listing` integration: validate indexes before reads | listing        | 2 hrs   |
| 11  | 🟢 MEDIUM   | WAL checkpoint scheduling helper                     | turso          | 1 hr    |
| 12  | 🟢 MEDIUM   | Document index trade-offs (write amplification)      | docs           | 45 min  |
| 13  | 🟢 LOW      | `AutoIndexer.Close()` + lifecycle cleanup            | turso/indexing | 15 min  |
| 14  | 🟢 LOW      | `indexing.DryRun` mode (print DDL, don't execute)    | turso/indexing | 30 min  |
| 15  | 🟢 LOW      | Publish ADR: "Why pre-calculated CQRS indexes"       | docs/adr       | 1 hr    |

---

## g) Top #1 Question I Cannot Figure Out Myself ❓

**"Should the `Index` struct expose a `Cost` or `Impact` field that quantifies the expected query-plan improvement, or is that premature optimization without real workload data?"**

SQLite's `EXPLAIN QUERY PLAN` gives estimated costs, but these are unitless planner internals (loop counts, not milliseconds). A naive "cost reduction = 500" metric would be meaningless to consumers.

Alternatives:

- **A) Keep it qualitative** — `Index.Reason` + `Recommendation.Explanation` tell the story
- **B) Add benchmark-driven metrics** — Run actual queries before/after and report latency delta
- **C) Add `Index.Priority` enum** — `Critical` / `Recommended` / `Optional` based on scan frequency

My gut says (B) is the only honest approach, but it requires infrastructure (benchmark suite + baseline tracking). (C) is a pragmatic middle ground. I'd like guidance on which direction to take.

---

## Module Health Snapshot

| Module               | Tests | Coverage | Lint        | Status     |
| -------------------- | ----- | -------- | ----------- | ---------- |
| event/v2             | ✅    | 89.4%    | ✅ Zero     | 🟢 Healthy |
| storage/v2           | ✅    | 86.8%    | ✅ Zero     | 🟢 Healthy |
| storage/v2/sql       | ✅    | 34.7%    | ✅ Zero     | 🟢 Healthy |
| turso/v2             | ✅    | 40.9%    | ✅ Zero     | 🟡 OK      |
| turso/v2/indexing    | ✅    | 70.4%    | 🟡 45 minor | 🟡 Good    |
| All 28 other modules | ✅    | 67-100%  | ✅ Zero     | 🟢 Healthy |

---

## Commits This Session (5 total)

```
59e1578b feat(turso/indexing): add auto-smart index management sub-package
9eaaf5b5 fix(turso/indexing): make AutoIndexer.Apply and ApplyCQRSIndexes respect IsEnabled
2a9dbd70 feat(turso/indexing): add Index.Partial bool and IndexSet.DropDDL
5b7802b8 docs(turso): mention indexing sub-package in doc.go
4c99bb81 feat(turso/indexing): add AdvisorOption and AutoIndexerOption patterns
```

**Total new code:** 1,844 + 143 = 1,987 lines across 14 files.
