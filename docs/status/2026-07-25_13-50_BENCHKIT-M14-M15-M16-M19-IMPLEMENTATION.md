# Status Report: Benchkit M14/M15/M16/M19 Implementation

**Date:** 2026-07-25 13:50
**Session:** Implement benchkit milestones M14, M15, M16, M19

---

## Executive Summary

All four benchkit milestones are **fully implemented, tested, and lint-clean**. The benchkit library now benchmarks the complete CQRS read path (publish→projection→query journey, typed query dispatch, snapshot/cache hit-rate) plus a soak test mode for leak/degradation detection.

- **Build:** PASS (benchkit + cmd/cqrs-bench)
- **Tests:** 16 new tests, all PASS (full suite: 30s)
- **Lint:** PASS (zero findings on benchkit + cqrs-bench)
- **CLI:** `--soak` flag works end-to-end

---

## Fully Done

### M14 — Journey Phase (publish→projection→query)
- **Files:** `phases_journey.go` (136 lines), `phases_journey_test.go` (125 lines)
- **What it does:** For each sample: writes a single event → synchronously projects it into a kv.Store read model (Get+Set) → dispatches a typed query that reads the materialized counter.
- **Metrics produced:** `JourneyLatency` (full round trip), `JourneyProjectionLatency` (projection.Handle leg), `JourneyQueryLatency` (query dispatch leg), `JourneySamples`, `QueryCorrectnessErrors`.
- **Design decision:** Uses synchronous `projection.Handle` instead of `projectionhost` because the host is a batch-drainer (catches up and exits), not a live tail. Polling for per-event materialization through the host would add poll-interval artifacts. The existing `projectionPhase` already benchmarks the batch-drain path.
- **Correctness assertion:** Each journey stream has exactly one event → materialized count must equal 1.

### M15 — Query Dispatch Phase
- **Files:** `phases_query.go` (134 lines), `phases_query_test.go` (84 lines)
- **What it does:** Pre-populates per-stream counters in the read model, then benchmarks three query dispatch paths through `query.Dispatcher`.
- **Metrics produced:** `QueryHitLatency` (registered handler, reads real data), `QueryMissLatency` (unregistered type → handler-not-found error path), `QueryPaginatedLatency` (paginated result construction), `QueryCorrectnessErrors`.
- **Correctness assertions:** Hit path returns expected count (i+1); paginated path returns correct TotalCount.

### M16 — Snapshot/Cache Hit-Rate Phase
- **Files:** `phases_snapshot.go` (287 lines, refactored into 5 helpers), `phases_snapshot_test.go` (141 lines)
- **What it does:** Measures decider Load performance under three strategies, with state/version equality checks across all of them:
  1. **Cold replay:** plain repository, full event replay.
  2. **Snapshot load:** snapshot store + EveryNEvents(1), snapshot + delta fold.
  3. **Cache miss/hit:** first Load (full replay, populates cache) vs second Load (LoadFromVersion of 0 delta events).
- **Metrics produced:** `SnapshotColdLatency`, `SnapshotLoadLatency`, `CacheMissLatency`, `CacheHitLatency`, `SnapshotCorrectnessErrors`.
- **Refactoring:** Extracted into `populateSnapshots`, `benchmarkColdLoad`, `benchmarkSnapshotLoad`, `benchmarkCache` helpers to stay under gocognit/nestif limits.

### M19 — Soak Test Mode
- **Files:** `soak.go` (233 lines), `soak_test.go` (159 lines), CLI `--soak` flag
- **What it does:** Runs the benchmark repeatedly for a fixed duration, forcing GC between iterations and recording heap usage, throughput, and latency per iteration. Computes drift metrics (heap growth, throughput drift, P99 drift).
- **API:** `RunSoak(ctx, SoakConfig, factory) → *SoakResult`, `PrintSoakReport(w, r)`, `WriteSoakJSON(w, r)`.
- **CLI:** `cqrs-bench run --backend memory --profile dev --soak 5m` — prints intermediate progress to stderr, final report to stdout (text or JSON).
- **Drift metrics:** `HeapGrowthBytes`, `HeapLeakRate` (bytes/iteration), `ThroughputDriftPct`, `WriteP99DriftPct`.

### Shared Infrastructure
- **`benchmodel.go`** (198 lines): Shared types — `CounterState`, `counterDecider()`, query types (`getCountQuery`, `listCountsQuery`, `missQuery`), `CountResult`, `newJourneyProjection()`, `newBenchQueryDispatcher()`, `readCount`/`writeCount` helpers.
- **Result struct:** 13 new fields added with JSON tags and documentation.
- **Config struct:** 3 new skip flags (`SkipJourney`, `SkipQuery`, `SkipSnapshot`).
- **Runner:** Extracted `runPhases()` helper to reduce `run()` cyclomatic complexity from 26→under limit.
- **PrintReport:** Updated with Journey, Query Dispatch, and Snapshot/Cache sections.
- **go.mod:** Added `decider/v4` as direct dep (query/v4 and snapshot/v4 promoted from indirect).
- **ReplayOnly guard:** Journey and snapshot phases skip when `ReplayOnly=true` (they write events).

### Test Regressions Fixed
- `TestRun_ReplayOnly_SQLite`: Added `SkipJourney` + `SkipSnapshot` to write phase (these phases write extra events that inflate journal count).
- `TestRun_Recovery_SQLite` / `TestRun_Recovery_Pebble`: Added `SkipSnapshot` (snapshot populate writes extra events to existing streams).

---

## Partially Done

### CLI Skip Flags
The `Config` struct exposes `SkipJourney`, `SkipQuery`, `SkipSnapshot` for library users, but the `cqrs-bench` CLI does not expose `--skip-journey`, `--skip-query`, `--skip-snapshot` flags. Only `--soak` was added. CLI users cannot selectively disable the new phases.

### Documentation
- `doc.go` metric boundaries section not updated with JourneyLatency, QueryHitLatency, SnapshotColdLatency descriptions.
- `CHANGELOG.md` `[Unreleased]` section not updated with the new features.
- `README.md` not updated.

---

## Not Started

- CLI `--skip-journey`, `--skip-query`, `--skip-snapshot` flags.
- `doc.go` metric boundary documentation for new fields.
- `CHANGELOG.md` entries.
- Soak JSON round-trip test (marshal → unmarshal → verify).
- Snapshot phase on Pebble backend (tested on memory + SQLite only).
- Journey phase with projectionhost batch-drain variant (current design uses synchronous projection; documented why).

---

## What to Improve

1. **Lint friction:** The `nilerr` linter flags intentional `ctx.Err() != nil → return nil` patterns as false positives. Required 5 `//nolint:nilerr` directives in `phases_snapshot.go` alone. A repo-wide `nolintlint` policy or a shared helper (`ctxSkip(ctx)`) could eliminate this friction.
2. **Journey phase event pollution:** The journey and snapshot phases write extra events to the store. This is correct behavior (they benchmark write paths) but means `TotalEvents` no longer reflects ONLY the write phase. The doc comments explain this, but it surprised two existing tests.
3. **Snapshot phase writes events to populate snapshots:** This is unavoidable (you need snapshots to benchmark snapshot loads), but it means the phase is not purely read-only. Documented in comments.
4. **wsl_v5 formatting rules** are aggressive about blank lines between declarations and statements. Required manual intervention on several files where `nix fmt` didn't auto-fix.

---

## Next Tasks (Priority Order)

1. Add `--skip-journey`, `--skip-query`, `--skip-snapshot` CLI flags to `cqrs-bench`
2. Update `CHANGELOG.md` with M14/M15/M16/M19 features
3. Update `doc.go` metric boundaries section
4. Update benchkit `README.md` with new phases and `--soak` examples
5. Add soak JSON round-trip test
6. Test snapshot phase on Pebble backend
7. Consider extracting `ctxSkip(ctx) bool` helper to eliminate nilerr friction
8. Update `cmd/cqrs-bench/main.go` usage text with `--soak` description in the flags list
9. Add `ProfileSnapshot` or `ProfileJourney` named profiles optimized for the new phases
10. Benchmark the new phases on PostgreSQL (requires `POSTGRES_TEST_DSN`)
11. Add `SoakResult` to the `WriteComparisonJSON` output for multi-backend soak comparisons
12. Consider a `--soak-report-interval` CLI flag (currently hardcoded to 10s)
13. Add OTel span recording to soak iterations for distributed tracing
14. Document the synchronous-projection design decision in an ADR
15. Add a `RunSoakCompare` API for cross-backend soak comparisons

---

## Verdict

All four milestones are production-ready. The implementation follows existing benchkit patterns (factory-driven, phase-based, latency-collector, graceful skip on missing capabilities). 16 new tests cover happy paths, skip flags, skip-without-capabilities, correctness assertions, and report output. The soak mode provides actionable leak/degradation detection with per-iteration samples and computed drift metrics.
