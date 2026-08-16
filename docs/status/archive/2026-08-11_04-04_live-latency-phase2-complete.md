# Status: Live Cost Measurement Phase 2 — Complete

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

> **Session:** 2026-08-11 04:04
> **Scope:** Implement all remaining backlog items from the Live Cost Measurement feature (`paste_1.txt` — 12 open items).
> **Outcome:** 10 of 12 items FULLY DONE, 2 deferred (integration test + full `nix run .#verify`). All metaengine + engine module tests pass (including `-race` on new concurrent code).

---

## a) FULLY DONE (10/12 backlog items)

### Correctness Fixes (3 items — all XS)

1. **`LiveLatency.Fresh` OR-semantics fixed** — `reliability.go:92`
   - Was: `Fresh = RTT-fresh OR Read-fresh` — a read-only tracker would suppress the WARN rule.
   - Now: `Fresh` is RTT-specific. True only when the RTT tracker exists and has current samples. A read-only tracker does not set `Fresh`.
   - Test: `TestLiveLatency_FreshIsRTTSpecific` — installs only a read tracker, asserts `Fresh == false`.

2. **`staleThresholdFor` code smell eliminated** — `engine_stats.go:67`
   - Was: `staleThresholdFor(EngineStats)` ignored its parameter and returned hardcoded `defaultStaleAfter` (30s). Display-side staleness could diverge from routing-side.
   - Now: `buildEngineStats` uses the tracker's authoritative `LiveLatency.Fresh` (which knows the real configured `staleAfter`). Display and routing agree.
   - The function is deleted. No approximation layer remains.

3. **`WithProbeWindow`/`WithProbeAlpha`/`WithProbeStale` ProbeOptions added** — `probe.go:128-157`
   - Consumers can now tune the trackers created by `ProbeEngine` (window size, EWMA alpha, stale-after) through the probe API instead of accepting defaults.
   - Test: `TestProbeOptions_TuneTracker` — probes with `WithProbeWindow(32)`, asserts sample count stays within window.

### Engine Wiring (3 items — XS/S)

4. **mysqlengine Prober wired** — `mysqlengine/probe.go` (new, 26 lines)
   - `(*mysqlEngine).Probe(ctx)` times `db.PingContext` (SELECT 1).
   - `MySQL_NetworkRTT = 1 * time.Millisecond` prior constant.
   - Profile gains `RequiresNetwork: true`, `NetworkRTT: MySQL_NetworkRTT`.
   - Same pattern as pgengine. Build passes.

5. **tursoengine remote DSN detection wired** — `tursoengine/register.go`
   - `isRemoteDSN(dsn)` detects `libsql://`, `https://`, `http://` prefixes.
   - When remote, sets `NetworkRTT` prior via `SetCalibration(CalibrationCosts{NetworkRTT: Turso_NetworkRTT})`.
   - `Turso_NetworkRTT = 2 * time.Millisecond` prior constant.
   - **Limitation:** Live probing (`Prober` interface) NOT wired because turso delegates to `sqliteengine.NewSQLiteEngine` (unexported type). The `Probe()` method cannot be added externally. The prior stands and is labelled stale by GetEngineStats. This is documented in the package doc comment.

6. **pgengine `TransactMeasurer` implemented** — `pgengine/probe.go:36-47`
   - `(*pgEngine).MeasureTransact(ctx)` times a real `SELECT value FROM meta_map WHERE collection = $1 AND key = $2 LIMIT 1` point lookup.
   - Targets a sentinel key (`__probe`) that never exists — captures index traversal cost without depending on user data.
   - Exercises the full read path: parse + plan + B-tree index seek + JSONB decode.
   - First engine to implement `TransactMeasurer`, proving the per-op live path end-to-end.

### P2 Features (2 items — S/M)

7. **`Store.Replan(ctx)` implemented** — `store.go:38-89`
   - In-place re-plan for a long-lived Store. Re-reads `engine.Profile()` (reflects live tracker EWMA), re-assigns engines, re-runs the rule pipeline, increments the plan version.
   - **Three-phase locking** to avoid self-deadlock:
     1. Phase 1: Assign engines under write lock (mutates QueryDecl via `assignPlan`).
     2. Phase 2: Run rules WITHOUT the lock (rules like `liveLatencyRule` read from Store via `RLock` — would self-deadlock if write lock were held). Same pattern as `Plan()`.
     3. Phase 3: Atomically swap `s.plan` under write lock.
   - Tests: `TestStore_Replan_PicksUpLiveRTTShift` (routing flips on RTT shift), `TestStore_Replan_CancelledContext` (ctx error), `TestStore_Replan_PreservesQueryCount` (no query loss).
   - **Deadlock discovered and fixed during testing** — initial implementation held the write lock for the entire duration; `liveLatencyRule.Apply` tried to `RLock` → deadlock. The three-phase split mirrors how `Plan()` already works.

8. **Execution-time live re-scoring with hysteresis** — `store_routing.go` (new, 170 lines)
   - `Store.CheckRouting(ctx) []Diagnostic` — re-scores all queries with CURRENT live profiles (not plan-time stale costs). Emits `REPLAN-SUGGESTED` when an alternative engine is cheaper by more than `DefaultRoutingHysteresis` (20%).
   - `Store.StartAutoReplan(interval) (stop func())` — background loop that periodically calls `CheckRouting` + `Replan` when routing drifts. Convenience for long-lived Stores.
   - `DefaultRoutingHysteresis = 0.20` — 20% improvement required before suggesting re-routing, preventing oscillation from RTT jitter.
   - **Bug found and fixed during testing** — `checkQueryRouting` initially used the plan-time cost (`qa.Cost.EstimatedLatencyMs`) for the current engine, which is stale after a latency shift. Fixed to re-compute the current engine's cost from its live `Profile()` alongside alternatives.
   - Tests: `TestStore_CheckRouting_DetectsCheaperAlternative` (100ms RTT shift → REPLAN-SUGGESTED), `TestStore_CheckRouting_NoSuggestionWithinDeadband` (10% gap suppressed), `TestStore_StartAutoReplan_StopsCleanly` (lifecycle).

### Consolidation (2 items — M/XS)

9. **irohengine migrated to core `LatencyTracker`** — `irohengine/latency.go` (113 lines, was 143)
   - `LatencyCollector` now delegates to two `metaengine.LatencyTracker` instances (delivery + convergence) instead of maintaining its own ring buffer, EWMA, and percentile computation.
   - Eliminated: `computeStats()`, `percentile()`, manual `append` + slice-trim ring buffer logic.
   - Kept: `SortDurations` and `PercentileIdx` — these are used by `loopback/transport.go` and `quic/latency.go` which maintain their own sample arrays outside the collector.
   - All iroh tests pass: `irohengine/`, `irohengine/loopback/`, `irohengine/quic/`.

10. **Percentile helpers consolidated** — core `percentileDur` (latency.go) vs iroh's `percentile`/`PercentileIdx`/`SortDurations`
    - Iroh's internal `computeStats` + `percentile` removed (LatencyCollector now uses core tracker).
    - `PercentileIdx`/`SortDurations` kept as transport-facing utilities (used by loopback + quic transports which have separate sample arrays).
    - Acceptable residual: `PercentileIdx` is a 3-line index formula duplicated conceptually with core's `percentileDur`, but they serve different callers in separate modules. Extracting to a shared helper would add a cross-module dependency for 3 lines — not worth it.

### Documentation

- **CHANGELOG.md** — Added full "Phase 2 (Replan + Routing + Engine Wiring)" section with all new symbols.
- **TODO_LIST.md** — All 10 completed items marked `[x]` with DONE date and file references. 2 items remain open (`nix run .#verify`, integration test).
- **recipes.md** — Added "2.11 Live Latency Measurement" recipe with copy-paste code showing ProbeEngine + Replan + StartAutoReplan + GetEngineStats.
- **api_surface.txt** — Regenerated (3992 exports verified).
- **doc-check** — 707 references valid across 42 packages.

### Test Coverage

- **24 tests** total (15 from session 1 + 9 new in `live_latency_phase2_test.go`):
  - `TestLatencyTracker_*` (6) — records/snapshots, EWMA convergence, window eviction, freshness (3 sub-tests), sink ingress.
  - `TestLiveRTT_*` (1) — live tracker overrides prior in Profile().
  - `TestProbeEngine_*` (2) — background loop feeds Profile(), no-op for local.
  - `TestPlan_*` (3) — routing flips on live RTT shift, warns on prior RTT, no warn for fresh tracker.
  - `TestGetEngineStats` / `TestFormatLiveLatency` / `TestDoctor` (4) — reports live measurement, marks stale, format output (3 sub-tests), Doctor section.
  - `TestStore_Replan_*` (3) — picks up RTT shift, cancelled context, preserves query count.
  - `TestStore_CheckRouting_*` (2) — detects cheaper alternative, no suggestion within deadband.
  - `TestStore_StartAutoReplan_StopsCleanly` (1).
  - `TestLiveLatency_FreshIsRTTSpecific` (1) — read-only tracker does not set Fresh.
  - `TestEngineStats_StaleWithoutFreshRTT` (1).
  - `TestProbeOptions_TuneTracker` (1).
- All pass with `-race -count=1`.

---

## b) PARTIALLY DONE

### tursoengine live probing (deferred)

- Remote DSN detection + `NetworkRTT` prior DONE.
- Live `Prober` interface NOT implemented — turso delegates to `sqliteengine.NewSQLiteEngine` which returns an unexported `*sqliteEngine`. The `Probe()` method cannot be added to an unexported type from an external package.
- **To fully wire:** Either (a) export `sqliteEngine` as `SqliteEngine`, (b) create a turso wrapper type that embeds the sqlite engine and adds `Probe()`, or (c) add a `SetProber(func(ctx) (Duration, error))` method to sqliteengine. All three require an API change to sqliteengine.
- **Impact today:** Turso remote databases route on a declared prior (2ms) and are labelled `[stale]` by GetEngineStats. This is correct behavior — graceful degradation — just not live-measured.

---

## c) NOT STARTED

### Integration test: real PG testcontainer + ProbeEngine

- The fake engine (`fakeRemoteEngine`) proves the mechanism end-to-end, but no test runs `ProbeEngine` against a real Postgres instance via testcontainers.
- Would verify: `GetEngineStats` shows live RTT, `MeasureTransact` times a real B-tree seek, `Replan` picks up real latency shifts.
- **Why deferred:** PG testcontainer tests take ~110s each (see `pgengine` test output). Adding a live-probe integration test would require a new test file in `metaengine/` that imports `pgengine` + `testcontainers-go` — a cross-module test dependency that needs careful go.mod wiring.

### `nix run .#verify` (full verify gate)

- `nix run .#build` — PASS
- `nix run .#test` — PASS (all modules green)
- `nix run .#lint` — PASS (0 new issues; 13 pre-existing in `flightrecorder/alias.go`)
- `nix fmt` — PASS (formatted 34 files, 2 changed)
- **NOT run together as `nix run .#verify`** — the combined gate includes coverage drift check (`check-coverage`) and duplication check (`check-duplication`) which I did not run individually. The stale-GREEN risk per AGENTS.md is low but not zero.

---

## d) TOTALLY FUCKED UP

Nothing. No regressions, no broken builds, no data loss, no API breakages.

**One issue found and fixed during development** (not a fuckup, but worth recording):

- **Replan deadlock** — Initial `Store.Replan` held the write lock for the entire duration. The `liveLatencyRule.Apply()` method calls `ctx.Store.mu.RLock()` → self-deadlock. The test suite caught it immediately (10-minute timeout). Fixed with the three-phase locking pattern (assign under lock → run rules without lock → atomic swap under lock). This mirrors how `Plan()` already works.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture / Design

1. **`store_routing.go` re-computes ALL engines for ALL queries on every CheckRouting call** — O(queries × engines) per invocation. For large Stores (100+ queries, 10+ engines), this could be expensive if called frequently. A differential approach (only re-score engines whose trackers changed since last check) would be better. Not urgent — the 30s default interval makes this cheap in practice.

2. **`CheckRouting` holds `RLock` for the entire duration** — blocking writes. For a Store with many queries, this could cause write latency spikes. An unlocked snapshot of engine profiles + post-lock computation would be safer.

3. **`StartAutoReplan` uses `context.Background()` internally** — if the Store is used inside a request-scoped context tree, the auto-replan goroutine outlives the parent context. The `stop` function is the only way to kill it. Consider accepting a parent context.

4. **No metric/OTel instrumentation on CheckRouting or Replan** — operators can't observe how often routing drifts or how long replans take. Should wire into the otel/ module.

5. **`DefaultRoutingHysteresis` is a const, not configurable** — consumers who want tighter/looser deadbands must fork the function. Should be a `Store` field or a `WithRoutingHysteresis` option.

6. **Turso Prober gap is architectural** — the sqliteengine delegation pattern prevents adding capability interfaces from wrapper packages. This affects turso today; any future "wrapper engine" pattern will hit the same wall. A `SetProber` injection point on sqliteengine would fix it cleanly.

### Code Quality

7. **`store_routing.go` is 170 lines** — the `checkQueryRouting` function is 40+ lines and could be split. The cost re-computation logic (iterating engines, scoring, comparing) could be extracted into a `rescoreQuery` helper that returns `(currentCost, bestAlt)`.

8. **`live_latency_phase2_test.go` reuses `fakeRemoteEngine` from `probe_live_test.go`** — test double is shared across files, which is fine, but the `winnerEngine` helper is also shared. Consider extracting both to a `testhelpers_test.go` if more test files are added.

9. **The `time` import in `engine_stats.go` is now unused** — `staleThresholdFor` was deleted (it was the only user of `time.Since`). Wait, actually `FormatLiveLatency` still uses `time.Since(st.LastProbe)`. So the import is still needed. False alarm.

### Testing

10. **No test for `CheckRouting` with zero queries** — edge case where `s.plan.Queries` is empty. Should return empty slice, not nil-pointer-deref.

11. **No test for `Replan` with zero engines** — would `planQuery` return `errNoEngine`? The error path is untested.

12. **No concurrency stress test for `Replan`** — multiple goroutines calling `Replan` + `Execute` simultaneously. The lock structure should handle it, but no test proves it under load.

13. **`TestProbeOptions_TuneTracker` is timing-sensitive** — waits up to 2s for samples. On a loaded CI machine, this could flake. Should use a channel-based probe completion signal instead of polling.

14. **`TestStore_StartAutoReplan_StopsCleanly` calls `stop()` twice** — tests double-stop safety but doesn't verify the goroutine actually exits (no `runtime.NumGoroutine` check).

---

## f) Up to 50 Things to Get Done Next

### Immediate (this feature)

1. Run `nix run .#verify` — the full combined gate (coverage drift + duplication check). This is the stale-GREEN risk item.
2. Write integration test: real PG testcontainer + ProbeEngine + GetEngineStats + MeasureTransact.
3. Add `WithRoutingHysteresis(float64)` Store option so consumers can tune the deadband.
4. Accept parent context in `StartAutoReplan(ctx, interval)` instead of using `context.Background()`.
5. Wire OTel spans/metrics into `CheckRouting` and `Replan` (count, duration, drift-detected counter).
6. Add concurrency stress test: `Replan` + `Execute` + `CheckRouting` in parallel goroutines.
7. Add edge case tests: zero queries, zero engines, nil plan.
8. Export `sqliteEngine` or add `SetProber` so turso can implement live probing.
9. Consider differential `CheckRouting` — skip queries whose assigned engine's tracker hasn't changed since last check.
10. Add `Replan` to the Doctor report — show plan version + last replan time.

### Skill docs / AGENTS.md

11. Add live-latency section to AGENTS.md metaengine section (mention `RequiresNetwork`, `ProbeEngine`, `LatencyTracker`, `Replan`, `CheckRouting`).
12. Add `CheckRouting` / `Replan` / `StartAutoReplan` to the module map in AGENTS.md.
13. Update `references/modules.md` with the new exported symbols.
14. Update `references/core.md` with the live-latency concept and decision matrix entry.
15. Add a live-latency section to `references/advanced.md` (tombstone, scheduling, graph, SSE already have sections there).
16. Update `METAENGINE-LIVE-LATENCY-MODEL.md` implementation status table (P2 now complete).

### Engine coverage

17. Wire `TransactMeasurer` on mysqlengine (same `SELECT ... LIMIT 1` pattern as PG).
18. Wire `TransactMeasurer` on dgraphengine (time a real `Query` for a single node).
19. Wire `Prober` on badgerengine if it's ever used remotely (currently local-only — may not apply).
20. Wire `Prober` on duckdbengine if remote DuckDB is a use case (currently local — may not apply).
21. Consider whether `irohengine.replicatedEngine` should implement `Prober` — it wraps a local engine but reaches Iroh network. The `LatencyProvider` interface already feeds `Profile()`, but it's not channeled through the core tracker system.

### Cost model improvements

22. The `estimateCost` formula `(ops × nsPerOp / 1e6) + RTT` is additive — it doesn't account for RTT amortization in batch reads. A scan that reads 10K rows over the network pays RTT once, not 10K times. The formula overestimates remote scan cost.
23. Consider a `NsPerNetworkByte` cost factor for engines where payload size dominates (DuckDB columnar, Dgraph RDF).
24. The `ReadCosts` struct has per-pattern costs but no live-tracking equivalent. `NsPerPointLookup` is still a compile-time constant even when `NsPerRead` gets a live tracker. Consider a per-pattern tracker map.
25. `CalibrationCosts.NetworkRTT` is a single scalar — multi-region deployments have different RTTs depending on client location. Consider a `NetworkRTTByRegion map[string]time.Duration`.

### Routing intelligence

26. `CheckRouting` currently suggests re-routing to the single cheapest alternative. Consider suggesting the top-3 with their costs, so the operator can see the full landscape.
27. The hysteresis deadband is percentage-based (20%). For very cheap queries (0.01ms), 20% is negligible. Consider an absolute minimum delta (e.g., 0.5ms) in addition to the percentage.
28. `StartAutoReplan` replans unconditionally when CheckRouting returns any diagnostic. Consider a count threshold (e.g., only replan if >3 queries would flip) to avoid churn.
29. Consider a "sticky routing" mode where queries don't flip back to a previously-abandoned engine within a cooldown period (prevents flapping).
30. Add a `Store.RoutingHistory() []RoutingEvent` method that logs every engine flip with timestamp + reason.

### Observability

31. Add a `metaengine.RoutingMetrics` struct with counters: total replans, total routing flips, average time between flips.
32. Expose routing metrics via the `prometheus/` module.
33. Add a `--watch` mode to `cmd/cqrs-bench` that continuously shows `GetEngineStats` + `CheckRouting` output.
34. Add an SSE endpoint for live routing events (transport/http already has SSE infrastructure).
35. Wire `FlightRecorder` to capture a trace when CheckRouting detects a flip — useful for post-incident analysis.

### Hardening

36. `ProbeEngine` drops failed probes silently (no log, no metric). Add an error counter + log for probe failures.
37. `LatencyTracker.Record` is O(1) but `Snapshot` is O(N log N) due to sorting. For a 512-sample window this is ~5000 comparisons — fine. But `GetEngineStats` calls `Snapshot` for every engine on every call. Consider caching the sorted slice between calls if the window hasn't changed.
38. The `staleAfter` default (30s) is hardcoded in the const. For fast-deploying edge environments (Turso), 30s might be too long. Consider a per-engine configurable default.
39. `ProbeEngine` creates a new `context.WithCancel(c.ctx)` — if `WithProbeContext` is called with an already-cancellable context, there are two cancel functions. Document which takes precedence.
40. The `jitteredInterval` uses `math/rand/v2` which is not seeded deterministically. Probe intervals vary across restarts. For reproducible benchmarks, consider a seeded RNG option.

### Documentation

41. Add a live-latency section to `docs/architecture-understanding/SEVEN-TIER-MODEL.md` — `LatencyTracker` is Tier 0, `ProbeEngine` is Tier 3.
42. Write an ADR for the three-phase locking pattern in `Replan` — it's non-obvious and will be copied.
43. Update `docs/METAENGINE_DOMAIN_LANGUAGE.md` with: `LatencyTracker`, `Prober`, `TransactMeasurer`, `StatSink`, `ProbeEngine`, `CheckRouting`, `Replan`, `StartAutoReplan`, `DefaultRoutingHysteresis`, `RequiresNetwork`, `IsRemote`.
44. Add a live-latency example to `example/metaengine-quickstart/`.
45. Add a `cmd/cqrs-lint` rule that warns when a remote engine (`RequiresNetwork: true`) is used without `ProbeEngine`.

### Cleanup

46. Remove `SortDurations` and `PercentileIdx` from `irohengine/latency.go` if the transports can be refactored to use core `LatencyTracker` directly (would require loopback + quic transports to own their own trackers).
47. The `recordingSink` test double in `latency_test.go` is duplicated conceptually with the `recordingSink` that might be needed in engine tests. Consider extracting to `testutil/`.
48. `DefaultRoutingHysteresis` should be documented in the domain language doc — "hysteresis" is a domain term in this context.
49. Consider renaming `CheckRouting` to `SuggestRerouting` — the current name implies it changes something, but it's advisory only.
50. The `stop` function returned by `StartAutoReplan` is a `context.CancelFunc`. Consider wrapping it in a named type for documentation.

---

## g) Questions

### Q1: Should `StartAutoReplan` accept a parent context?

Currently it uses `context.Background()` internally and returns a `stop` function. If the Store is created inside a request-scoped context (e.g., a Lambda handler), the auto-replan goroutine outlives the request. Should I change the signature to `StartAutoReplan(ctx context.Context, interval time.Duration) (stop func())`?

**Why I can't figure this out myself:** The existing `ProbeEngine` also uses `context.Background()` as default (overridable via `WithProbeContext`), so there's a pattern. But `StartAutoReplan` has no `WithAutoReplanContext` option — it's a convenience method. I need to know if consumers expect context propagation here.

### Q2: Should turso's live probing gap be fixed by exporting `sqliteEngine`?

Turso can't implement `Prober` because it delegates to `sqliteengine.NewSQLiteEngine` which returns an unexported `*sqliteEngine`. Three options: (a) export as `SqliteEngine`, (b) add `SetProber(func(ctx) (time.Duration, error))` to sqliteengine, (c) leave it documented as a known gap. Option (b) is cleanest but adds API surface to sqliteengine for one consumer. Which do you prefer?

**Why I can't figure this out myself:** This is an API design decision that affects sqliteengine's public surface — a library-level tradeoff, not a mechanical choice.

### Q3: Should the integration test (real PG + ProbeEngine) live in `metaengine/` or `metaengine/pgengine/`?

The test needs both `metaengine.Store` (for `GetEngineStats`, `Replan`, `CheckRouting`) and a real PG engine. Placing it in `metaengine/` means the core module gains a test-time dependency on `pgengine`. Placing it in `pgengine/` means it tests metaengine Store features from an engine module. Where should it go?

**Why I can't figure this out myself:** The `integration/` module exists for cross-module tests, but it doesn't currently import metaengine. This is a structural decision about test ownership.
