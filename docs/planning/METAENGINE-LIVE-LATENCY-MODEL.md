# Modeling Runtime Latency in the Cost Model — a Design Report

> The critique that triggered this report: declaring `NetworkRTT` as a
> compile-time constant on `EngineProfile` is modeling a runtime measurement
> as a static fact. Latency changes _constantly_ — it is an observation, not an
> assertion. And the same applies to the per-op `NsPerOp/NsPerRead/NsPerWrite`
> and `ReadCosts` constants.
>
> This report analyzes the current state, design constraints, and a phased
> path to honest measurement. No code was written; this is a design proposal
> grounded in the actual codebase.
>
> **Status:** P1, P2, P3, Phase 3 improvement backlog, and the GetStats/Doctor/EXPLAIN UX wiring are ALL IMPLEMENTED.
> See "Implementation Status" below for the per-phase mapping to code.
> **Date:** 2026-08-10

---

## 1. TL;DR

You're right, and the codebase half-agrees with you already:

- **`NetworkRTT` today is a compile-time constant** on `EngineProfile` — set by
  each engine, overridable at `Plan` time, frozen for the life of the Store,
  never re-measured. ADR-0093 literally defers the fix ("an auto-calibration
  helper was considered but deferred").
- **The ecosystem already has the right pattern**: `irohengine` measures
  delivery/convergence at runtime with a `LatencyCollector` (windowed
  percentiles) and feeds them back through `Profile()` — but even iroh consumes
  them as a _snapshot_, not a _live value_. The missing piece is everywhere:
  **a runtime path from measurement → cost estimation, that doesn't require a
  re-plan.**
- The good news: the planner **reads `profile.NsForRead(ReadPattern)` and
  `profile.NetworkRTT` through `engine.Profile()` at plan time**. If `Profile()`
  returns a live view, a re-plan (or a re-score against latencies on the fly)
  automatically sees current numbers — no API break, no new `Plan` call shape.

**Proposed model** — the engine declares a _network contact_ and a _measurement
strategy_; the runtime measures true RTT/per-op latency in the background; the
planner consumes a dynamic profile. Three phases, each independently shippable:

| Phase                                   | Ships                                                                                                                                                                                                                                                                           | Into                   |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| **P1 — Probe & measure**                | `Prober`/`TransactMeasurer` interface, `LatencyTracker` (EWMA + percentile), `ProbeEngine()` helper. Engines report `NetworkRTT`/`NsPerRead` from a live tracker.                                                                                                               | `metaengine` core      |
| **P2 — Live planner view**              | Store holds a snapshot of each engine's profile; `Profile()` returns the live view; the Store re-reads profiles at interval/`GetStats()` and re-scores near-cheap alternatives; `GetStats()` exposes measurement artifacts. `WithNetworkRTT` becomes a _prior_, not a constant. | `Store` + planner      |
| **P3 — Latency telemetry into routing** | `OperatingSetReporter` sampled mechanism writes measured per-op/per-RTT (with a neutral ingress via `StatSink`) so external engines can feed real-time costs without a hard interface dependency.                                                                               | external engines, iles |

**Sizing** — P1 is small; P2 is medium; P3 is the largest and can stand alone.

**Why this beats "just put a number in the constructor"?** It separates _deployment
knowledge_ (which the consumer has: topology, expected RTT, allowed latency
budget) from _live observation_ (which only the running system has). The
compile-time profile declares who is remote and a _prior_; the runtime keeps
the number honest.

---

## Implementation Status

All four work items plus the Phase 3 improvement backlog are implemented and covered by tests in `metaengine/`:

| Phase                                    | What shipped                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Code                                                                                                                                                                                                  |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **P1 — Probe & measure**                 | `Prober` / `TransactMeasurer` interfaces, `LatencyTracker` (ring buffer + incremental EWMA + P50/P95/P99), `ProbeEngine()` helper returning `ProbeHandle` (Stop + Failures), `CalibrationCosts.NetworkRTT` prior, `Calibration` hosts live RTT + per-read trackers whose EWMA `ApplyCalibration` merges into `Profile()`. Test-double engine proves a live RTT shift changes `Profile()`. PG `SELECT 1` + `meta_map` point lookup, MySQL `SELECT 1` + `meta_map`, Dgraph healthcheck + predicate index seek, Turso `PingContext` via `sqliteengine.SetProber`. | `latency.go`, `probe.go`, `reliability.go`, `engine.go` (`RequiresNetwork`), `pgengine/probe.go`, `dgraphengine/probe.go`, `mysqlengine/probe.go`, `sqliteengine/probe.go`, `tursoengine/register.go` |
| **P2 — Live planner view + Store stats** | `Store.GetEngineStats()` returns `EngineStats {profile, measured RTT, samples, lastProbe, stale}`; `EXPLAIN`/`Doctor` show `rtt=live … (p95, n)` and stale labelling; `liveLatencyRule` emits a WARN when routing relies on a prior/stale RTT for a remote engine; `WithNetworkRTT` doc now says "prior, not constant." The plan-time `Profile()` read is already live, so a re-plan automatically picks up fresh numbers (gate test: routing flips on an RTT shift).                                                                                          | `engine_stats.go`, `rule_live_latency.go`, `explain.go`, `planner.go`, `rules.go`                                                                                                                     |
| **P3 — Open measurement ingress**        | Exported `StatSink` / `LatencySample` / `SampleKind`; `LatencyTracker` forwards every sample to a configured sink; `ProbeEngine` accepts `WithProbeSink`. External engines can push measurements through a sink without internal helpers. Test: fake prober drives planner decisions with/without live stats.                                                                                                                                                                                                                                                  | `probe.go`, `latency.go` (`WithTrackerSink`)                                                                                                                                                          |
| **UX — GetStats / Doctor**               | `Doctor` adds `--- Latency ---` + `--- Routing ---` sections (plan version, replan count, hysteresis, drift summary); `ExplainPlan` shows the live-latency line per remote engine; `FormatLiveLatency` renders live/stale/local.                                                                                                                                                                                                                                                                                                                               | `explain.go`, `engine_stats.go`                                                                                                                                                                       |
| **Phase 3 — Improvement backlog**        | `Store.Replan(ctx)` three-phase locking; `Store.CheckRouting(ctx)` differential re-scoring with configurable hysteresis (`WithRoutingHysteresis`, `WithRoutingMinDelta`); `Store.StartAutoReplan(ctx, interval)` with parent context; `ProbeHandle` with `Failures()` counter + `WithProbeErrorHandler`; turso live probing via `sqliteengine.SetProber`; RTT amortization for scan-pattern fallback costs; slog-based observability for CheckRouting and Replan; concurrency stress test.                                                                     | `store.go`, `store_routing.go`, `probe.go`, `engine.go`, `planner.go`, `sqliteengine/probe.go`                                                                                                        |

Backward compatibility: the `Engine` interface is unchanged (`Profile()+Close()`); `Profile()` is free to return a live view. Engines without a tracker behave exactly as before. The new profile field `RequiresNetwork` and `CalibrationCosts.NetworkRTT` default to zero (local / no override), so every existing engine compiles and plans unchanged.

---

## 2. What This Is About — a Taxonomy of "the number"

The cost model uses several superficially-similar numbers with different
epistemology:

| Number                              | What it models                                 | Static or dynamic?                                   | Who knows it?                                   |
| ----------------------------------- | ---------------------------------------------- | ---------------------------------------------------- | ----------------------------------------------- |
| `NetworkRTT`                        | the single fixed RTT to a _remote_ engine      | **dynamic** (network health, load, distance)         | only the running system                         |
| `ReplicationLag`                    | staleness of a replicated copy                 | **dynamic** (load, partition healing)                | only the running system                         |
| `NsPerOp/Read/Write`, `ReadCosts.*` | compute cost of an operation on given hardware | **quasi-static** (hardware-fixed, workload-variable) | engine author + calibration, runtime under load |
| `Replication` mode                  | topology (`SingleLeader`, …)                   | **static**                                           | engine author                                   |
| `Persistence`                       | durable vs volatile                            | **static** (except memory-vs-file choice)            | engine author                                   |
| `DegradedADTs`                      | capability fallback                            | **static**                                           | engine author                                   |
| `Complexity` (O(1), O(N))           | asymptotic class                               | **static**                                           | engine author                                   |

The user's critique targets the two dynamic rows (`NetworkRTT` and the
`NsPerOp`/`ReadCosts` family). The current model treats all rows as static
compile-time facts; that is the core modeling error.

### The "compile-time vs runtime" distinction

- **Compile-time / structural** (`Replication`, `Persistence`, `DegradedADTs`,
  `Complexity`, support map): what the engine _is_. Stay in `EngineProfile`.
- **Runtime / observed** (`NetworkRTT`, per-op latency): what the environment
  _does_. Should come from measurement, not from a constant.

---

## 3. Current State — How the Cost Model Works Today (verified in code)

### 3.1 The profile

`EngineProfile` (in `engine.go`) carries both structural and cost fields
(NsPerOp, NsPerRead, NsPerWrite, ReadCosts, Replication, ReplicationLag,
NetworkRTT, Persistence, DegradedADTs). Doc comments call the cost fields
"calibrated" but they are plain float constants with no runtime update path.

### 3.2 Calibration exists — but is half-finished

`reliability.go` has a full `Calibration`/`Calibratable` machinery intended for
_measured_ costs:

- `CalibrationCosts` { NsPerOp, NsPerRead, NsPerWrite, ReadCosts }
- `Calibration` struct with `SetCalibration` / `ApplyCalibration` (applies only
  non-zero overrides)
- `Calibratable` interface (`SetCalibration(CalibrationCosts)`)
- `CalibrateEngine(eng, iterations)` — a **micro-benchmark** that runs MapSet/
  MapGet/MapDelete in a loop and sets `NsPerOp/Read/Write`.

**What it lacks:** any `NetworkRTT` slot in `CalibrationCosts`; any _live_
(traffic-derived) measurement; any background re-measurement. `CalibrateEngine`
measures synthetic micro-benchmark traffic once, then freezes the result — it
is "cold calibration," not "live observation."

### 3.3 The planner consumes `Profile()` at plan time only

`planner.go:178-201` calls `eng.Profile()` per engine per query and feeds
`NsForRead(pattern)` and `NetworkRTT` into `estimateCost`. `estimateCost`
(`cost.go:104`) computes:

```
latencyMs = (ops × nsPerOp / 1e6) + NetworkRTT_ms
```

The Store's per-query `QueryCost` snapshot (`store.go:54`, `serializable.go`)
is filled at **plan time** and frozen. `planDiagnostics` warns when a volatile
engine is routed, etc.

**Key structural fact:** the planner reads the profile _through the engine_ —
there is no caching of the EngineProfile in the Store. So if `Profile()` itself
returned a _live view_ (e.g., from an internal `LatencyTracker`), a re-plan
would automatically use fresh numbers. **The architecture already supports
live costs if engines provide them.** It just doesn't do anything with them.

Note: `QueryEngine().Profile()` is also read at execution time for
`unsupportedEngine` errors — so dynamic profiles are visible to execution too.

### 3.4 `WithNetworkRTT` is a global override, not a per-engine dynamic

Planner options support `WithNetworkRTT(rtt)` — overrides **all** engines'
RTT at plan time. Useful as an operator knob, but it's a constant again.

### 3.5 `irohengine` already does live latency measurement (the proof it works)

`irohengine/latency.go`:

- `LatencyCollector` keeps a **window** (512 samples) of deliveries and
  convergences, records them from real traffic, and computes `LatencyStats`
  (Mean, P50, P95, P99, Max) on demand.
- `LatencySnapshot` { DeliveryP50, DeliveryP99, ConvergenceP99 } is exposed to
  the engine.
- `irohengine/engine.go:46-60` `Profile()` **reports measured values**:
  `ReplicationLag = ConvergenceP99`, `NetworkRTT = DeliveryP50 × 2`.

**What this shows:** (a) the engine already _measures_ live latency; (b) it
feeds it back into the profile; (c) **however** — it's still a snapshot: the
network transport delivers per-op, the collector holds a window, but nothing
_reacts_ to a worsening RTT, and the planner sees a fixed profile copy unless
re-planned. Iroh is the philosophical proof-of-concept; the missing step is
making the _consumer_ of the profile live too — plus measuring point-to-point
RTT of actual I/O (not just replication control traffic).

### 3.6 `ReplicationLag`: declared, not measured

`ReplicationLag` is a declared expectation used only for diagnostics
(`rule_replication.go` INFO), never for routing. Iroh reports it live; nothing
else does.

### 3.7 The relevant code, mapped

| Concern                 | File / symbol                                                                                                                                           |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Profile struct          | `metaengine/engine.go` (`EngineProfile`)                                                                                                                |
| Cost estimator          | `metaengine/cost.go` (`estimateCost`)                                                                                                                   |
| Planner profile reading | `metaengine/planner.go:178-201`                                                                                                                         |
| Store profile copies    | `metaengine/store.go:54,78`, `serializable.go:95`                                                                                                       |
| Calibration machinery   | `metaengine/reliability.go` (`Calibration`, `Calibratable`, `CalibrateEngine`)                                                                          |
| Iroh live latency       | `metaengine/irohengine/latency.go`, `engine.go:46-60`                                                                                                   |
| ADR defining NetworkRTT | `docs/adr/0093-metaengine-replication-model.md` (says "auto-calibration considered but deferred")                                                       |
| ReadCosts variance doc  | `docs/planning/2026-08-04_07-00_READ-COSTS-PER-OPERATION-VARIANCE.md` ("values are compile-time… do not evolve at runtime. This is acceptable for now") |

---

## 4. Design Proposal

### 4.1 Core principle

> **Static facts belong in the profile; measured facts belong in the runtime.**
> The engine declares its topology (`Replication`, `Persistence`,
> `DegradedADTs`, `Complexity`) and a _prior_ for costs. The runtime measures
> `NetworkRTT` (point-to-point, from real I/O), per-op latency, and staleness.
> The planner consumes a **live profile** — same shape, dynamic numbers.

No ADT/capability-interface break: `Engine` stays `Profile() + Close()`;
`Profile()` is free to return a live view.

### 4.2 New components

#### a) `Probe` / `TransactMeasurer` (interface)

An optional capability engines implement when they can measure
point-to-point I/O latency:

```go
// Probe measures the current network round-trip to the engine's data.
// Implemented by remote engines (FDB/PG/Dgraph); local engines return
// zero/not-supported.
type Prober interface {
    Probe(ctx context.Context) (time.Duration, error) // current RTT sample
}

// TransactMeasurer measures the current round-trip of an actual I/O
// operation (e.g. a read transaction). Used to build per-operation
// latency from live traffic instead of micro-benchmarks.
type TransactMeasurer interface {
    MeasureTransact(ctx context.Context) (time.Duration, error)
}
```

`Probe` is for engines that can answer "how far away are you right now?"
(e.g., FDB: a `txn.Get(readVersion only)`; PG: `SELECT 1`; Dgraph: a
`healthcheck`/query). This is the _honest_ `NetworkRTT`.

#### b) `LatencyTracker` (live, EWMA + percentiles)

```go
// LatencyTracker maintains a sliding window of latency samples with
// exponential-decay weighting, so estimates respond to recent conditions
// while dampening jitter. Stats are computed on demand (O(1) with a
// ring buffer + running moments).
type LatencyTracker struct { /* window, EWMA, min/max */ }

func (t *LatencyTracker) Record(sample time.Duration)
func (t *LatencyTracker) P50/P95/P99() time.Duration
func (t *LatencyTracker) EWMA() time.Duration   // recent-dominant average
func (t *LatencyTracker) Snapshot() LatencyStats
```

Provides the `P50/P95/P99/EWMA` shape iroh proved works, but reusable
in core, and _updatable live_.

#### c) `ProbeEngine(eng, opts)` (helper)

Runs `Probe`/`MeasureTransact` at a configurable interval (default ~1s,
jittered), records into the engine's `LatencyTracker`, and stops on
`Store.Close` or an explicit stop func. Because the engine's `Profile()`
reads the tracker, any re-plan picks up live numbers automatically.

```go
// Prior (compile-time) stays: engines declare a reasonable default RTT
// (e.g. same-DC 1ms) so planning works before the first probe; the live
// value REPLACES the prior once measured. If the engine can't measure,
// the prior remains — but it's now explicitly a prior, not a fact.
```

#### d) Making `Profile()` live

Engines that measure should return:
`p.NetworkRTT = tracker.EWMA()` (or `P50`, configurable), and
`p.NsPerRead = tracker.NsPerReadEWMA()` if per-op timing is tracked.

For **remote engines in separate modules**, we must avoid a hard dependency on
the core probe helper (they may only embed `Calibration` today). Proposal:
the core exposes small exported helpers on `Calibration` so external engines
can embed the same tracker:

```go
// on Calibration:
func (c *Calibration) SetProber(p Prober, interval time.Duration) (stop func())
func (c *Calibration) SetOperatorStatsSink(sink OperatingSetReporter /* see P3 */)
func (c *Calibration) ApplyCalibration(p *EngineProfile) // already reads tracker fields
```

No interface break: engines that don't call these behave exactly as today.

#### e) Store: live profile snapshot + `GetStats()`

The Store keeps a _runtime snapshot_ per engine (profile + measured stats)
that it refreshes:

- at plan time (as today),
- on `GetStats()` / explicit `store.RefreshProfile()` (for the diagnostics/UX
  path, no background goroutine requirement),
- optionally on a background interval when `ProbeEngine` is active.

Expose the mechanism honestly:

```go
type EngineStats struct {
    Name            string
    Profile         EngineProfile   // what the planner uses
    MeasuredRTT     time.Duration   // live P50/P95/EWMA from tracker
    MeasuredNsRead  float64
    MeasuredNsWrite float64
    Samples         int             // how many observations
    LastProbe       time.Time       // freshness of measurement
    Stale           bool            // no recent samples
}
func (s *Store) GetStats(ctx context.Context) []EngineStats
```

If a remote engine's measurement goes stale (no traffic, no probe success),
`GetStats` marks `Stale: true` and the planner can fall back to the prior —
and emit a diagnostic instead of silently trusting an old number.

#### f) Diagnostics

Add a planner diagnostic whenever routing relies on a _stale or prior-only_
RTT for a remote engine: `[WARN] routing on prior RTT (no live samples)`.
This keeps the "graceful degradation, never silent lies" spirit.

### 4.3 How it changes planning (the important part)

Today: `estimateCost` receives `profile.NetworkRTT` once at plan time. After a
re-plan, `Profile()` is re-read — so live values flow in. Two options:

1. **Re-plan on demand** — `store.Replan(ctx)` re-runs the planner and
   consumers see updated assignments; `PROBE_INTERVAL` defaults to ~1s and
   `ReplanInterval` ~30s. Simple, keeps the current architecture, but a
   re-plan can churn assignments.
2. **Live re-scoring without re-plan** — at execution time, when a query is
   routed, the executor re-reads `engine.Profile().NetworkRTT` and compares
   against the plan-time cost for the second-best candidate that was _close_;
   if the current measured RTT makes the alternative strictly cheaper (with a
   hysteresis deadband), route to it (or emit a `REPLAN-SUGGESTED`
   diagnostic). **This makes routing react to RTT shifts without a plan
   recomputation.** It is optional in P2, and P1 can ship without it.

Both need `estimateCost` to accept a possibly-live RTT — which it already does
(the parameter is just `time.Duration`; the only change is that the caller
supplies measured instead of constant values).

### 4.4 Concrete mapping to user's complaint

| User's call                                                    | Design answer                                                                                                                                                                                                                  |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| "We should declare NETWORK CALL NEEDED at compile time"        | ✅ **Yes** — that's a _structural_ fact: add/keep a boolean-ish `RequiresNetwork` (or `Replication != None` implies it, but explicit is better). The profile declares "this engine does network I/O"; the _value_ is measured. |
| "RTT is always a runtime measurement and CAN CHANGE"           | ✅ **Yes** — `Prober`/`LatencyTracker` measure live RTT; `Profile()` returns it; planner/executor consume it; `GetStats` reports it with freshness.                                                                            |
| "NsPerOp / NsPerRead / NsPerWrite / ReadCosts are runtime too" | ✅ **Yes** — same mechanism: live `TransactMeasurer` per-op samples → `Calibration`'s read/write fields → profile, rather than only frozen micro-benchmark constants.                                                          |
| "We need to model that better"                                 | ✅ This design — structural facts static, observed facts live, freshness tracked, diagnostics honest.                                                                                                                          |

---

## 5. Phased Implementation Plan

> Each phase is independently valuable, shippable, and backwards compatible.
> No interface break: existing engines keep `Profile()+Close()`; new optional
> interfaces are type-asserted.

### Phase 1 — Probers + LatencyTracker (core, small)

**Goal:** make measurement possible and feed `Profile()` live.

- Add `Prober` + `TransactMeasurer` optional interfaces in `metaengine`.
- Add `LatencyTracker` (window + EWMA + percentiles) in core (inspired by,
  and later reusable by, iroh's collector).
- Extend `Calibration`/`CalibrationCosts` with optional `NetworkRTT` + measured
  per-read/write fields, and make `ApplyCalibration` merge them.
- Add `ProbeEngine(eng, opts)` helper (interval, jitter, stop func).
- Engines that can measure implement `Prober` (PG `SELECT 1`; Dgraph
  healthcheck; **FDB probe = read-version-only transaction**).
- Local engines (Memory/SQLite/Pebble/DuckDB) implement nothing — they remain
  `NetworkRTT=0` / structurally local. `ProbeEngine` no-ops when the engine
  has no `Prober`.

**Gate:** `Profile().NetworkRTT` reflects the last live probe; `GetStats()`
shows samples + freshness. Decay/expiry of stale samples handled.

### Phase 2 — Live planner view + Store stats (core, medium)

**Goal:** make the planner and operators _see_ live numbers without an API
break.

- Store keeps a runtime profile snapshot; refresh on plan, on `GetStats()`,
  and on a background `ReplanInterval` when live measurement active.
- `EngineStats` {profile, measured RTT, samples, lastProbe, stale} surfaced via
  `Store.GetStats(ctx)` (+ EXPLAIN shows `[rtt=live 2.1ms (p95 4.0ms, n=512)]`).
- Diagnostics: WARN when routing relies on prior/stale RTT for a remote engine;
  mention `Replan(ctx)`.
- Optional: live re-scoring of near-tied queries at execution time with a
  hysteresis deadband, instead of waiting for a re-plan.

**Gate:** a simulated RTT shift (tests) changes visible `GetStats`,
planner diagnostics, and — with re-planning — the assignment.

### Phase 3 — Open telemetry ingress for external engines (largest, independent)

**Goal:** let _any_ engine (especially future FDB engine and existing
remote engines) feed live measurements without a hard core dependency.

- Optional `StatSink` interface (exported): `Report(EngineStats)` — engines
  push measurements; core aggregates.
- External engines embed `Calibration` (as today), and use optional
  `SetStatsSink()` to wire measured values in — no import of internal helpers.
- The planner reads the same profile; freshness/staleness handling shared.

**Gate:** a test double engine (fake prober) drives planner decisions with and
without live stats.

---

## 6. What Does NOT Change (and why)

- **`Engine` interface** (`Profile()+Close()`) stays — `Profile()` is already
  the single source; a live `Profile()` needs no new method. ✅ backward-compatible.
- **`estimateCost` formula** stays `(ops×nsPerOp/1e6) + RTT` — the user is right
  that RTT is additive; the fix is _whose_ RTT, not _how it combines_.
- **Capability/ADT scheduling logic** — unchanged.
- **`WithNetworkRTT` override** stays, but its doc changes to "a prior for the
  initial plan, replaced by live measurement when available."
- **Everything the existing `Calibration` already does** — `CalibrateEngine`
  still provides cold micro-benchmark priors; live measurement _updates_ them.

---

## 7. Risks, Caveats, Honest Limits

1. **Measurement is not free.** Probing every second adds a tiny load; keep
   probe interval low (default 1s), jittered, and skip probes when the engine
   is already processing traffic (piggyback on real I/O where possible).
2. **RTT is only one axis.** The bigger variance is per-op compute cost under
   load (and queueing). Live per-op measurement (P2/P3) is the honest fix for
   that too; RTT is just the most obvious offender.
3. **`Probe` measures reachability, not necessarily query path** — an FDB
   read-version probe measures client→GRV-proxy→logs, but storage-read latency
   is different. Docs: "read <1ms, commit 1.5-2.5ms" — a fitness measurement
   must choose a probe that matches the _query_ it will serve (or use
   `TransactMeasurer` on real ops).
4. **P50 vs P95 for routing.** Averages hide tail latency; routing on EWMA can
   oscillate. Use EWMA (or P50) for steady-state routing, P95/P99 for
   budgets/diagnostics.
5. **Don't over-react.** Hysteresis + minimum cooldown between re-plans (e.g.
   30s) prevents routing flapping on a flapping network.
6. **Stale priors are still better than nothing.** When no samples exist,
   fall back to the prior and _label_ it stale — don't refuse to route.
7. **Scope honesty.** This report is a design; the FDB-engine cost profile from
   the earlier report is a _prior_ and must be re-derived from live
   measurement once a probe exists.

---

## 8. What I Recommend Doing First (Actionable)

1. **Land P1** — `Prober` + `LatencyTracker` + `ProbeEngine` helper + live
   `Profile()` composition in core, with a test-double engine proving a live
   RTT shift changes `Profile()`.
2. **Wire PG (and Dgraph) probes** — the two existing remote engines get
   honest live RTT instead of a hardcoded estimate.
3. **Extend `GetStats`** (P2 minimal) so `EXPLAIN`/Doctor show `rtt=live …`
   and stale labeling; run a simulated-shift test against the planner.
4. **Then design FDB engine** on top of the probe mechanism (read-version-only
   probe → live RTT), not on a hardcoded `1ms` constant.
5. **P3 (open sink)** only when an external engine actually needs it — avoid
   speculative interfaces.

---

## 9. Measured Dialect Costs (2026-08-16, mysqlengine benchmarks)

Benchmarks live in `metaengine/mysqlengine/graph_bench_test.go` and
`sort_bench_test.go` (run with `MYSQL_TEST_DSN` against each server;
`-benchtime 10x/20x`). Environment: localhost TCP, AMD Ryzen AI MAX+ 395,
MariaDB 11.4.12 (userspace) and MySQL 8.4.11 (Docker) — so absolute numbers
include ~60-100µs loopback RTT; ratios are the durable signal.

### Graph traversal: recursive CTE vs iterative BFS (crossover table)

Synthetic graph, out-degree 2 (chain + scattered edge), start node 0,
`GraphNeighbors(depth)`; median of forced-mode runs (ns/op, 10x):

| depth | MariaDB CTE | MariaDB iter | MySQL CTE | MySQL iter | Winner |
| ----- | ----------- | ------------ | --------- | ---------- | ------ |
| 1     | 160-253µs   | **65-109µs** | 137-253µs | **69-98µs** | iterative (2-4x) |
| 2     | 111-167µs   | 96-168µs     | 169-181µs | 137-138µs   | ~parity |
| 3     | 128-143µs   | 176-232µs    | 117-175µs | 220-241µs   | CTE (1.3-1.9x) |
| 4     | 144-189µs   | 308-330µs    | 117-142µs | 380-415µs   | CTE (2.2-2.9x) |
| 5     | 156-275µs   | 470-571µs    | 134-239µs | 620-735µs   | CTE (3.0-4.3x) |
| 6     | 169-245µs   | 887-963µs    | 176-221µs | 1013-1339µs | CTE (4.4-6.1x) |

Findings (identical shape on both servers, independent of graph size 1k-100k):

1. **CTE cost is depth-flat; iterative cost is depth-linear** (one RTT per
   frontier node per level). Crossover sits between depth 1 and 2.
2. **Depth-1 walks are 2-4x faster via the direct adjacency query** — the
   WITH RECURSIVE machinery (UNION dedup + DISTINCT) is pure overhead there.
   Optimization candidate: short-circuit `depth == 1` to
   `mysqlGraphNeighborsDirect` (+ `AND to_node <> ?` to preserve the
   start-node exclusion). Unimplemented; see TODO_LIST.
3. For the cost model: model CTE graph reads as ~flat in depth (one RTT +
   small per-level CPU), iterative as `RTT × frontier_size × depth`.

### Sort pushdown: MariaDB dual-key vs MySQL JSON-typed ORDER BY

50k shuffled numeric-priority rows in `meta_map`, `ORDER BY ... LIMIT 100`
(20x runs):

| Form | MariaDB 11.4 | MySQL 8.4 |
| ---- | ------------ | --------- |
| dual-key `CAST(... AS DECIMAL), JSON_UNQUOTE(...)` (engine's MariaDB form) | 47.0ms | 48.1ms |
| single `JSON_UNQUOTE(JSON_EXTRACT(...))` (text-order control) | 37.4ms | 37.7ms |
| single `value->'$.p'` (JSON-typed; engine's MySQL form) | n/a (1064) | **19.0ms** |

Findings:

1. The DECIMAL dual-key costs **+26%** over the single-expression form on
   both servers — the price of numeric text-sort correctness on MariaDB.
2. MySQL's JSON-typed arrow form is **2.5x faster** than MariaDB's dual-key
   (19.0 vs 47.0ms) and even 2x faster than the unquoted-text control on
   MySQL itself: typed JSON comparison skips per-row string materialization.
3. For the cost model: numeric sort pushdown on MariaDB ≈ 2.5x the MySQL
   cost for the same row count — relevant when both engines are candidates
   for a sort-heavy read model.

---

## 10. References

- [ADR-0093 — Metaengine Replication Model](../adr/0093-metaengine-replication-model.md) — defines
  NetworkRTT, explicitly defers auto-calibration
- [Read Costs per Operation Variance](../planning/2026-08-04_07-00_READ-COSTS-PER-OPERATION-VARIANCE.md) —
  fixes ReadCosts, marks them "compile-time, do not evolve at runtime… acceptable for now"
- [irohengine/latency.go](../../metaengine/irohengine/latency.go) — live `LatencyCollector`
  (window + stats), the seed of this design
- [metaengine/reliability.go](../../metaengine/reliability.go) — `Calibration`/`Calibratable`/
  `CalibrateEngine` (cold calibration)
- [metaengine/cost.go](../../metaengine/cost.go) — `estimateCost` (additive RTT)
- [metaengine/planner.go](../../metaengine/planner.go) — profile consumption at plan time
- [metaengine/store.go](../../metaengine/store.go) — per-query `QueryCost` snapshot
- [metaengine/irohengine/engine.go](../../metaengine/irohengine/engine.go) — measured profile
  (ReplicationLag=ConvergenceP99, NetworkRTT=DeliveryP50*2)
- [Operator-Driven Layout Planning](METAENGINE-LAYOUT-PLANNING-MODEL.md) — extends this model:
  operator priorities weight the cost model's embed-vs-normalize scoring. The
  adaptive planner mode (§6.2) reuses `Store.Replan` / `CheckRouting` from this
  design. [ADR-0124](../adr/0124-operator-driven-layout-planning.md) defines the
  full layout planning model.
