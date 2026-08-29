# Status Report: Live Cost Measurement (Dynamic NetworkRTT / Per-Op Latency)

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

**Date:** 2026-08-10 18:49
**Session focus:** Implement the Live Cost Measurement task list from the backlog paste (P1–P3 + UX wiring).
**Design doc:** `docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md`
**Verdict:** P1, P3, and UX are solid; P2 is ~80% (missing in-place re-plan and Store profile snapshot); several engine-wiring gaps remain.

---

## a) FULLY DONE (verified: builds, tests pass, golden updated)

### P1 — Prober + LatencyTracker (core)

| Deliverable                                                                                            | File                                                                            | Status               |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------- | -------------------- |
| `LatencyTracker` (ring buffer + incremental EWMA + P50/P95/P99 + Mean/Max)                             | `metaengine/latency.go`                                                         | ✅ 6 unit tests pass |
| `LatencyStats` snapshot, `Fresh()`, `Live()` freshness gating                                          | `metaengine/latency.go`                                                         | ✅                   |
| `Prober` / `TransactMeasurer` optional interfaces                                                      | `metaengine/probe.go`                                                           | ✅                   |
| `ProbeEngine()` helper (interval, jitter, timeout, stop func, no-op for local engines)                 | `metaengine/probe.go`                                                           | ✅                   |
| `CalibrationCosts.NetworkRTT` prior field                                                              | `metaengine/reliability.go`                                                     | ✅                   |
| `Calibration` hosts live RTT + per-read trackers; `ApplyCalibration` merges live EWMA into `Profile()` | `metaengine/reliability.go`                                                     | ✅                   |
| `EngineProfile.RequiresNetwork` structural flag                                                        | `metaengine/engine.go`                                                          | ✅                   |
| `EngineProfile.IsRemote()` helper                                                                      | `metaengine/replication.go`                                                     | ✅                   |
| PG `Prober` (`SELECT 1` timing) + `PG_NetworkRTT` prior + `RequiresNetwork`                            | `metaengine/pgengine/probe.go`, `engine.go`                                     | ✅ builds            |
| Dgraph `Prober` (healthcheck timing) + `DG_NetworkRTT` prior + `RequiresNetwork`                       | `metaengine/dgraphengine/probe.go`, `engine.go`                                 | ✅ builds            |
| **Gate test:** test-double engine proves a live RTT shift changes `Profile().NetworkRTT`               | `metaengine/probe_live_test.go` `TestLiveRTT_OverridesPriorInProfile`           | ✅                   |
| **Gate test:** background probe loop feeds `Profile()` via embedded Calibration                        | `metaengine/probe_live_test.go` `TestProbeEngine_FeedsProfileViaBackgroundLoop` | ✅                   |
| **Gate test:** `ProbeEngine` no-ops safely for local engines                                           | `metaengine/probe_live_test.go` `TestProbeEngine_NoopForLocalEngine`            | ✅                   |

### P3 — Open measurement ingress (StatSink)

| Deliverable                                                 | File                                      | Status                                        |
| ----------------------------------------------------------- | ----------------------------------------- | --------------------------------------------- |
| `StatSink` interface, `LatencySample`, `SampleKind`         | `metaengine/probe.go`                     | ✅                                            |
| `LatencyTracker` forwards every sample to a configured sink | `metaengine/latency.go` `WithTrackerSink` | ✅ test: `TestLatencyTracker_StatSinkIngress` |
| `ProbeEngine` accepts `WithProbeSink`                       | `metaengine/probe.go`                     | ✅                                            |

### P2 — Live planner view + diagnostics (partial — see section b)

| Deliverable                                                                       | File                                                                  | Status                      |
| --------------------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------- |
| `liveLatencyRule` — WARN when routing on prior/stale RTT for a remote engine      | `metaengine/rule_live_latency.go`                                     | ✅ registered in `rules.go` |
| `WithNetworkRTT` doc updated to "prior, not constant"                             | `metaengine/planner.go`                                               | ✅                          |
| **Gate test:** routing flips on a live RTT shift (re-plan picks up fresh numbers) | `metaengine/probe_live_test.go` `TestPlan_RoutingFlipsOnLiveRTTShift` | ✅                          |
| **Gate test:** WARN fires on prior RTT; clears with fresh tracker                 | `metaengine/probe_live_test.go`                                       | ✅                          |

### UX — GetStats / Doctor / EXPLAIN

| Deliverable                                                                                    | File                         | Status                                       |
| ---------------------------------------------------------------------------------------------- | ---------------------------- | -------------------------------------------- |
| `Store.GetEngineStats(ctx) []EngineStats`                                                      | `metaengine/engine_stats.go` | ✅                                           |
| `EngineStats {profile, measured RTT, samples, lastProbe, stale}`                               | `metaengine/engine_stats.go` | ✅                                           |
| `FormatLiveLatency()` renders `rtt=live … (p95, n)` / `rtt=prior … [stale]` / `rtt=0s (local)` | `metaengine/engine_stats.go` | ✅                                           |
| `ExplainPlan()` shows live-latency line per remote engine                                      | `metaengine/explain.go`      | ✅                                           |
| `Doctor()` adds `--- Latency ---` section                                                      | `metaengine/explain.go`      | ✅ test: `TestDoctor_IncludesLatencySection` |

### Docs / verification gates

| Gate                                                                                                         | Status |
| ------------------------------------------------------------------------------------------------------------ | ------ |
| `docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md` status updated to IMPLEMENTED + implementation-status table | ✅     |
| `docs/api_surface.txt` golden regenerated (3981 exports, 11 new symbols)                                     | ✅     |
| `cmd/api-stability` verify passes                                                                            | ✅     |
| `cmd/doc-check` passes (695 references, 42 packages)                                                         | ✅     |
| Full `metaengine/` test suite passes (15s)                                                                   | ✅     |
| Race test on all new concurrent code passes                                                                  | ✅     |
| gofumpt formatting clean on all changed files                                                                | ✅     |

---

## b) PARTIALLY DONE

### P2 gaps

1. **No `Store.Replan(ctx)`** — The design doc proposes an in-place re-plan method so consumers refresh assignments without constructing a new Store. My routing-flip test works around this by calling `Plan()` again (new Store), which is the pragmatic path today. But there is no public `Store.Replan()` for a long-lived Store to pick up fresh profiles. The plan-time read IS already live (planner calls `engine.Profile()` directly), so a re-plan would see fresh numbers — the method just doesn't exist yet.

2. **No Store-side runtime profile snapshot** — The design describes the Store keeping a cached profile snapshot refreshed on plan / `GetStats()` / background interval. Currently `GetEngineStats()` reads `Profile()` live (which is correct and always current), but there's no cached snapshot or background refresh ticker. This means `GetEngineStats` calls `Profile()` on each invocation — fine for diagnostics, but the design envisioned a periodic refresh.

3. **No execution-time live re-scoring** — The design's P2 optional item (re-score near-tied queries at execution time with a hysteresis deadband, without a full re-plan) was not implemented. This is explicitly optional in the design.

4. **`staleThresholdFor(EngineStats)` ignores its parameter** — The display-side staleness check in `engine_stats.go` falls back to `defaultStaleAfter` (30s) because `LatencyStats` doesn't carry the configured stale-after duration. The routing-side check (inside `LatencyTracker.Live()`) IS precise. These two checks could diverge — a tracker configured with `WithStaleAfter(5*time.Second)` would route on stale at 5s but display as fresh until 30s. This is a real (minor) correctness gap.

5. **`LiveLatency.Fresh` uses OR semantics** — `Calibration.LiveLatency()` sets `Fresh = RTT-fresh OR Read-fresh`. A remote engine with only a read tracker (no RTT tracker) would report `Fresh=true`, suppressing the WARN rule — even though the RTT is still prior. In practice `ProbeEngine` always installs RTT when `Prober` is implemented, so this doesn't bite today, but the semantics are imprecise.

### Engine wiring gaps

6. **irohengine NOT migrated** — Iroh already has its own `LatencyCollector` (the design's proof-of-concept). It still uses `Profile().NetworkRTT = DeliveryP50 × 2` hardcoded, not the core `LatencyTracker`. The design explicitly envisioned consolidating iroh onto the core tracker. Not done.

7. **mysqlengine NOT wired** — MySQL is a remote SQL engine like PG, but has no `Prober`, no `RequiresNetwork`, no `NetworkRTT` prior.

8. **tursoengine NOT wired** — Turso (libSQL) is remote; same gap as MySQL.

9. **No engine implements `TransactMeasurer`** — The per-read live measurement interface exists and `ProbeEngine` handles it, but zero engines implement it. Only `Prober` (RTT) is wired for PG + Dgraph. The per-op latency path is structurally complete but dormant.

---

## c) NOT STARTED

1. **Store.Replan(ctx)** — in-place re-plan for a long-lived Store.
2. **Store.RefreshProfile()** — explicit profile-snapshot refresh.
3. **Background refresh ticker** — periodic profile re-read on an interval.
4. **Execution-time re-scoring with hysteresis** — the P2 optional deadband.
5. **irohengine migration** — consolidate onto core `LatencyTracker`, deprecate local `LatencyCollector`.
6. **mysqlengine Prober + RequiresNetwork.**
7. **tursoengine Prober + RequiresNetwork.**
8. **Any engine implementing `TransactMeasurer`** (per-read live latency).
9. **Cookbook / recipe doc** — no `metaengine/COOKBOOK.md` or skill `recipes.md` entry for the live-latency usage pattern.
10. **AGENTS.md update** — metaengine section doesn't mention the live-latency feature.
11. **CHANGELOG entry.**
12. **Integration test with real PG** — pgengine Prober only tested via fake engine; no testcontainer-based test that times a real `SELECT 1`.
13. **`nix run .#check-duplication`** — not run; potential duplication between core `percentileDur` and iroh's `percentile`/`SortDurations`.
14. **`nix run .#lint`** — full lint suite not run (only gofumpt on changed files).
15. **`nix run .#verify`** — comprehensive gate not run.
16. **`nix run .#check-coverage`** — coverage drift not checked.
17. **`nix run .#check-arch`** — dependency budget not verified (though no new deps were added — all core code is pure Go stdlib).

---

## d) TOTALLY FUCKED UP

Nothing is catastrophically broken. Everything that was committed builds, passes tests, and is race-clean. The closest things to "fucked up":

1. **`staleThresholdFor` is a code smell** — takes a parameter it ignores, returns a constant. Should either read the tracker's configured stale-after or be a package constant. Currently a half-abstraction.

2. **`LiveLatency.Fresh` OR-logic is semantically imprecise** for the WARN rule (see b.5 above). Not a crash, but the diagnostic suppression logic could mask a missing-RTT-tracker case.

3. **I left a stray `init() {}` placeholder in `probe.go`** during development — caught and removed before the first build, but it's a sign of writing-while-thinking rather than writing-clean-once.

---

## e) WHAT WE SHOULD IMPROVE

### Correctness

1. **Fix `staleThresholdFor`** — either carry the configured stale-after in `LatencyStats` (so display and routing agree), or make `EngineStats.Stale` read directly from `LiveLatency.Fresh` (single source of truth).
2. **Fix `LiveLatency.Fresh` semantics** — split into `RTTFresh` and `ReadFresh`, or make the WARN rule check RTT-freshness specifically.
3. **Add a ProbeOption for window/alpha/stale** — currently `ProbeEngine` hardcodes tracker defaults; a consumer can't tune EWMA responsiveness through the probe API.

### Completeness

4. **Wire mysqlengine + tursoengine** — same Prober + RequiresNetwork pattern as PG/Dgraph.
5. **Migrate irohengine** onto the core `LatencyTracker` — eliminate the parallel `LatencyCollector` implementation.
6. **Implement `TransactMeasurer` on at least one engine** (PG: time a real `SELECT ... LIMIT 1` read) — prove the per-read live path end-to-end.
7. **Add `Store.Replan(ctx)`** — the missing in-place refresh for long-lived Stores.

### Quality

8. **Run `nix run .#verify`** before claiming done — the session did build + test + gofumpt but NOT the full verify gate (lint, vet, coverage, duplication, doc-check together). This is a "stale GREEN" risk per AGENTS.md.
9. **Consolidate percentile helpers** — core `percentileDur` vs iroh's `percentile` + `PercentileIdx` + `SortDurations`. DRY violation.
10. **Update AGENTS.md metaengine section** — note the live-latency feature, the `RequiresNetwork` flag, and the `ProbeEngine` usage pattern.

---

## f) Up to 50 things to do next

### Immediate (correctness + close P2 gaps)

1. Fix `staleThresholdFor` to use the tracker's actual stale-after (carry it in `LatencyStats` or read from `LiveLatency`).
2. Split `LiveLatency.Fresh` into RTT-specific and Read-specific freshness; fix the WARN rule to check RTT-freshness.
3. Add `WithProbeWindow`, `WithProbeAlpha`, `WithProbeStale` `ProbeOption`s.
4. Implement `Store.Replan(ctx) (*PlanResult, error)`.
5. Implement `Store.RefreshProfile()` (or fold into Replan).

### Engine wiring (close the remote-engine gap)

6. Wire mysqlengine: `Prober` (`SELECT 1`), `RequiresNetwork`, `NetworkRTT` prior.
7. Wire tursoengine: `Prober`, `RequiresNetwork`, `NetworkRTT` prior.
8. Implement `TransactMeasurer` on pgengine (time a real point-lookup read).
9. Implement `TransactMeasurer` on dgraphengine (time a `MapGet`).
10. Migrate irohengine `LatencyCollector` → core `LatencyTracker`.
11. Update irohengine `Profile()` to read RTT from the core tracker.
12. Deprecate/remove irohengine's local `latency.go` `LatencyCollector` (or keep as a thin wrapper).

### Verification gates (the "stale GREEN" debt)

13. Run `nix run .#lint` and fix any findings.
14. Run `nix run .#check-duplication` and update `.art-dupl-baseline.json` if needed.
15. Run `nix run .#check-coverage`.
16. Run `nix run .#check-arch` (dependency budget).
17. Run `nix run .#verify` end-to-end.
18. Run `nix run .#verify-fast` as a quicker sanity check.

### Testing

19. Add an integration test: real PG testcontainer + `ProbeEngine` → verify `GetEngineStats` shows live RTT.
20. Add a test: `TransactMeasurer` path (once wired on an engine).
21. Add a test: `ProbeEngine` with erroring `Prober` — verify failed probes are dropped, not recorded.
22. Add a test: `ProbeEngine` stop func is idempotent (call twice, no panic).
23. Add a property test: `LatencyTracker` EWMA stays within [min, max] of the window.
24. Add a test: multiple engines with a shared `StatSink` — verify sample demultiplexing by name.
25. Add a bench: `LatencyTracker.Record` allocation count (should be 0).
26. Add a bench: `LatencyTracker.Snapshot` with full window (512 samples).

### Docs / skill

27. Update `AGENTS.md` metaengine section with live-latency feature.
28. Add a recipe to `.agents/skills/go-cqrs-lite/references/recipes.md` for `ProbeEngine` usage.
29. Add a section to `.agents/skills/go-cqrs-lite/references/advanced.md` for live-latency model.
30. Update `metaengine/COOKBOOK.md` with a live-probing example.
31. Add a CHANGELOG entry.
32. Update `METAENGINE_DOMAIN_LANGUAGE.md` with `Prober`, `LatencyTracker`, `StatSink`, `RequiresNetwork`, `EngineStats` terms.
33. Add an ADR for the live-latency model (ADR-0093 follow-up).

### Design refinements

34. Consider: should `RequiresNetwork` be auto-inferred from `Replication != None`? (Design says explicit is better — confirm.)
35. Consider: should `ProbeEngine` be auto-wired by `stack/*` presets? (A `stack.WithProbing` option?)
36. Consider: a `Store.StartProbing(opts...)` convenience that wires `ProbeEngine` for all remote engines and stops on `Store.Close`.
37. Consider: `EngineStats` in the `system.DeployerConfig` for operator-visible deployment-time RTT.
38. Consider: OTel metrics for live RTT (a `metrics.WithLiveLatency` middleware).

### Code quality

39. Consolidate `percentileDur` (core) with iroh's `percentile`/`PercentileIdx` — extract to a shared helper or delete the core one in favor of iroh's exported helpers.
40. Remove the `staleThresholdFor` parameter smell.
41. Add doc examples (`ExampleProbeEngine`, `ExampleLatencyTracker`) to the `metaengine` package.
42. Verify `Calibration` value-embedding works for ALL engine modules (not just PG/Dgraph/memory) — the pointer-receiver `SetRTTTracker` promotion depends on engines being `*T`, not `T`.
43. Audit: does `bboltengine` / `pebbleengine` / `badgerengine` need `RequiresNetwork`? (They're local — confirm `IsRemote()` returns false.)

### Future (P2 optional / beyond scope)

44. Execution-time live re-scoring with hysteresis deadband.
45. Background profile-refresh ticker on Store.
46. `REPLAN-SUGGESTED` diagnostic when near-tied queries would flip on current RTT.
47. Probe piggybacking on real I/O (skip synthetic probe when traffic is flowing).
48. P95 vs EWMA routing policy option (`WithRTTPolicy(P50|P95|EWMA)`).
49. Minimum cooldown between re-plans (prevent flapping).
50. Operator API: `Store.LiveProfile() map[string]EngineProfile` for dashboards.

---

## g) Questions I cannot answer myself

1. **Should `ProbeEngine` be auto-wired by the `stack/*` presets (e.g. `stack/postgres.New()` calls `ProbeEngine` internally and stops on `Store.Close`), or should the consumer always opt in explicitly?** The design says "optional," but auto-wiring would serve the "zero storage-layer thinking" north star. This is a product decision about the consumer experience that depends on your intended deployment story.

2. **Should irohengine's local `LatencyCollector` be deleted (consolidate onto core `LatencyTracker`) or kept as a transport-level collector that feeds the core tracker?** Iroh's collector measures delivery + convergence (two axes), while the core tracker is single-axis (RTT). Deleting it loses the convergence axis; keeping it duplicates the percentile machinery. This depends on whether convergence latency should flow into the cost model or stay as iroh-internal telemetry.

3. **Is the pre-existing `record → id.ActorID` version-sequence break (commit `7e374b753`) something I should fix as part of this work, or is it tracked elsewhere?** It blocks `GOWORK=off go build` for the metaengine module standalone (workspace build works fine). It's unrelated to live latency but it prevents `nix run .#verify` from passing the per-module standalone build check. I need to know if this is your WIP or an external dependency.
