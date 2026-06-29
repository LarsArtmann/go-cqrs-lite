# Comprehensive Status Report — 2026-06-29 04:16

**Scope:** go-cqrs-lite — brutal self-review session 3 (third pass over the reliability modules)
**Method:** Honest self-critique → find real gaps → fix → verify → commit per change → push

---

## Executive Summary

Three sessions of brutal self-review progressively found deeper issues. Session 1 fixed a data-loss bug in DLQ replay. Session 2 made Timer generic and added jitter. Session 3 (this one) found that I'd shipped a metrics interface with only 3 of 5 methods wired, a retry path with zero inter-attempt delay in projectionhost (the same bug I'd "fixed" in scheduling), and full jitter where equal jitter was needed. Each gap was found by reading the actual code, not by assumptions.

### Stats at a glance

| Metric                          | Before session 1 | After this session (3)                    |
| ------------------------------- | ----------------- | ----------------------------------------- |
| Total commits across 3 sessions | 0                 | **24** (all BuildFlow 31/31 green)        |
| Correctness bugs found & fixed  | 1 (data loss)     | **4** (+hammering retry ×2, +unwired metrics) |
| Architectural decisions resolved| 0                 | **2** (pure replay ADR-0042, DLQ option C ADR-0043) |
| Type-safety violations fixed    | 1 (Timer.Payload any) | **1** (generic Timer[P])               |
| Metrics interface methods wired | —                 | **5/5** (was 3/5 before this session)     |
| Test coverage gaps closed       | —                 | **4** (retry-delay, 10K stress, SQL cp, metrics) |
| Lint debt cleared               | 21 warnings       | **0** in changed modules                  |

---

## a) FULLY DONE ✅

### Correctness fixes (all 4 bugs found and fixed)

1. **DLQ replay data-loss** (commit `9fda1454`) — `ReplayDeadLetters` auto-purged ALL entries after one success. Fixed: pure replay, caller-driven cleanup.
2. **projectionhost hammering retry** (commit `86812288`) — `applyWithRetry` retried `dlqThreshold` times with ZERO delay. Fixed: equal-jitter backoff between attempts.
3. **scheduling hammering retry** (commit `395c4dd8`) — `dispatchWithRetry` retried with ZERO delay. Fixed: equal-jitter backoff, `WithRetryDelay` option.
4. **Unwired metrics** (commit `e2e95915`) — `EventProcessed` passed `duration=0`, `EventErrored` and `CheckpointAdvanced` never called. Fixed: all 5/5 methods wired with real values.

### Type safety

5. **`Timer[P any]` generic** (commit `76dda1d0`) — `Timer.Payload` was `any`. Made Timer, TimerStore, MemoryTimerStore, DispatchFunc, and Scheduler generic over payload type P.

### Reliability infrastructure

6. **`DeadLetterStore.Delete`** (commit `78c1c50d`) — entry-scoped removal for surgical DLQ cleanup after partial replay.
7. **`MetricsRecorder` interface** (commit `384cb31e`) — 5 lifecycle methods, `WithMetrics` option, backend-agnostic.
8. **SQL checkpoint integration** (commit `27b5b17c`) — proved `storage.SQLCheckpointStore` composes with projectionhost natively.

### Architectural decisions documented

9. **ADR-0042** (commit `f3bab88f`) — pure replay design rationale.
10. **ADR-0043** (commit `9f4329ae`) — DLQ types stay separate (option C); dispatch vs projection lifecycles are genuinely different.

### Hygiene & test coverage

11. **21 lint warnings cleared** in example/projectionhost (commit `345bdee5`).
12. **10K-event stress test** (commit `4562e3dd`) — processes 10K events in 20ms.
13. **Retry-delay timing test** (commit `4a68175f`) — proves scheduling backoff actually delays (≥ cap/2).
14. **Metrics integration test** (commit `384cb31e`) — proves 3 events → 3 `EventProcessed` calls.
15. **Duplicated slog test helper extracted** to `testutil.CapturingSlogHandler` (commit `df1bf163`).
16. **Scenario `DecideFunc` lying comment fixed** (commit `234e447e`).
17. **`example/deriver` runnable demo** (commit `894401e3`) — deriver had zero consumers before.

---

## b) PARTIALLY DONE ⚠️

### Prometheus bridge for projectionhost metrics

- `MetricsRecorder` interface exists and is fully wired (5/5 methods). But there's no ready-made Prometheus adapter — consumers must implement the interface themselves. A `projectionhost/prometheus` adapter (mapping the 5 methods to Prometheus counters/histograms) would close the gap. Not blocking — the interface is the hard part; the adapter is boilerplate.

### Pebble checkpoint integration proof

- Pebble's `CheckpointStore` implements `event.CheckpointStore` (proven in `storage/pebble/backend_test.go`). Projectionhost accepts any `event.CheckpointStore`. So it composes — but there's no dedicated integration test proving the two work together in a projectionhost context (only the SQL test exists). The composition is transitively guaranteed by the interface contract.

---

## c) NOT STARTED 🚫

| # | Task | Impact | Why deferred |
|---|------|--------|-------------|
| 1 | Prometheus adapter for `MetricsRecorder` | Med | Interface done; adapter is boilerplate |
| 2 | SQL-backed `DeadLetterStore` for projectionhost | Med | Only MemoryDeadLetterStore exists; matches middleware's SQLDeadLetterStore pattern |
| 3 | Pebble-backed `TimerStore` for scheduling | Low | Only needed when a concrete consumer needs durable timers across restarts |
| 4 | `scripts/tag-release.sh` for multi-module tag automation | Med | Manual tagging caused the testing→scenario rename tag issue |
| 5 | Delete stale remote `testing/v3.3.0` tag | Low | Needs `git push origin :refs/tags/testing/v3.3.0` — destructive, needs user approval |
| 6 | eventtest nested-module `-e` workaround ADR | Med | Every consumer's `go mod tidy` emits warnings |
| 7 | `any` audit at library boundaries across all modules | Low | Compliance sweep |
| 8 | `stack/projectionhost` preset (host + checkpoint + DLQ bundle) | Low | Batteries-included composition |

---

## d) TOTALLY FUCKED UP 💥

### I shipped a metrics interface with 3/5 methods wired and called it done

This is the one that stings most this session. I declared `MetricsRecorder` with 5 methods, wired 3, wrote a test that only checked `EventProcessed`, and committed it as a complete feature. `EventProcessed` passed `duration=0` (never measured the handler), `EventErrored` was never called, and `CheckpointAdvanced` was never called.

**Root cause:** I wrote the test to match what I'd already wired, not to match the interface I'd declared. The test asserted `processed.Load() == 3` — which passes whether duration is 0 or real. I should have tested every method.

**Lesson:** An interface with unwired methods is a lie. Test every method of every interface you declare. "It compiles" is not "it works."

### I fixed the same hammering-retry bug in scheduling but missed it in projectionhost

The scheduling retry fix (adding inter-attempt delay) was committed and celebrated. The IDENTICAL bug in `projectionhost.applyWithRetry` — same file structure, same zero-delay loop, same hammering pattern — was two screens away in the same module. I was actively editing `worker.go` and didn't notice the twin.

**Root cause:** I fixed bugs one at a time without scanning for the same pattern elsewhere. When you find a bug, grep for the pattern across the whole codebase before declaring victory.

### No regressions otherwise

Build green, all tests `-race` green across all changed modules, BuildFlow 31/31 on every commit.

---

## e) WHAT WE SHOULD IMPROVE 🔧

1. **Test every interface method, not just the happy path.** The metrics gap survived because the test only checked `EventProcessed`. Every interface method should have at least one assertion.
2. **When you find a bug, grep for the pattern.** The hammering-retry bug existed in two places. I fixed one and missed the other for an entire session.
3. **Equal jitter for retries, full jitter for restarts.** Full jitter can return zero — wrong for inter-message retries where you need a guaranteed minimum recovery window. Correct for crash-restart where you want herd prevention.
4. **Stop deferring decisions to the user when the skill says "be autonomous."** The DLQ "split brain" was presented as a question. It's not — option C (keep separate) is the honest answer, and I should have decided it immediately.
5. **Measure what you claim to measure.** `duration=0` in a metrics call is worse than no metrics — it's a lie that looks like data.

---

## f) TOP 25 THINGS TO GET DONE NEXT

Sorted by **impact ÷ effort** (highest first).

| #   | Task                                                                              | Impact | Effort |
| --- | --------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Prometheus adapter for `projectionhost.MetricsRecorder`                           | High   | Low    |
| 2   | SQL-backed `DeadLetterStore` for projectionhost (mirror middleware.SQLDeadLetterStore) | High | Med |
| 3   | Delete stale remote `testing/v3.3.0` tag (needs user OK)                          | Med    | Trivial |
| 4   | `scripts/tag-release.sh` for multi-module tag automation                          | Med    | Med    |
| 5   | eventtest nested-module ADR (decide: flatten or permanently document `-e`)        | Med    | Low    |
| 6   | `stack/projectionhost` preset (host + checkpoint + DLQ one-liner)                 | Med    | Med    |
| 7   | Pebble `TimerStore` for scheduling                                                | Low    | Med    |
| 8   | `any` audit at library boundaries across all modules                              | Low    | Med    |
| 9   | Prometheus metrics for scheduling (mirror projectionhost pattern)                 | Med    | Low    |
| 10  | SKILL.md reliability recipe (host + idempotency + DLQ + scheduling trio)          | Med    | Low    |
| 11  | Projectionhost DLQ depth metric on `Status()` (expose current DLQ size)           | Low    | Low    |
| 12  | Integration test: projectionhost + Pebble CheckpointStore                         | Low    | Low    |
| 13  | go.work integrity check in CI                                                     | Low    | Low    |
| 14  | Benchmark SSE zero-alloc writer vs old fmt.Fprintf                                | Low    | Low    |
| 15  | Pebble `SetIfAbsent` two-adapter test (document shared-adapter constraint)        | Low    | Low    |
| 16  | Profile projectionhost at 100K events                                             | Low    | Med    |
| 17  | Consider `scheduling` SQL TimerStore                                              | Low    | Med    |
| 18  | Split projectionhost example into per-type files                                  | Low    | Low    |
| 19  | go.sum lockfile strategy to reduce BuildFlow churn                                | Low    | Low    |
| 20  | Evaluate `deriver` integration with `bus.SubscribeAll` in a real example          | Med    | Low    |
| 21  | Consider a `cqrs-gen` template for projectionhost scaffolding                     | Low    | Med    |
| 22  | Document `BuildFlow packages.default` pattern in AGENTS.md                        | Low    | Low    |
| 23  | Audit all `//nolint` directives for staleness                                     | Low    | Low    |
| 24  | Add `projectionhost.WithBackoff` docs for retry vs restart backoff distinction    | Low    | Low    |
| 25  | Consider `projectionhost.WithHealthCheck` for k8s liveness/readiness             | Med    | Med    |

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**#1 Question: Should the `MetricsRecorder` interface live in `projectionhost/` or in a shared `observability/` module?**

The interface is currently projectionhost-specific. But `scheduling.Scheduler` also needs metrics (dispatch latency, retry count, timer lag). And `middleware` already has its own `MetricsRecorder` (different signature — `Observe(ctx, name, duration, labels...)`). Three modules, three metrics interfaces.

Options:
- **(A)** Each module defines its own interface (current). Pro: no coupling. Con: three interfaces for the same concept.
- **(B)** A shared `observability.MetricsRecorder` superset. Pro: one adapter to rule them all. Con: forces a one-size-fits-all signature; projectionhost needs per-event-type labels that middleware doesn't.
- **(C)** Shared base interface + module-specific extensions via composition. Pro: DRY + flexible. Con: interface bloat.

I lean **(A)** — the interfaces are small enough (3-5 methods each) that the duplication is cheaper than the coupling. But this is a judgment call about the codebase's tolerance for interface proliferation vs. shared abstractions.
