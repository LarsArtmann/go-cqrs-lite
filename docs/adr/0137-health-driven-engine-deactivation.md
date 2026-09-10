# ADR-0137: Health-Driven Engine Deactivation

**Date:** 2026-09-10
**Status:** Accepted
**Related:** ADR-0112 (ES-native planner), ADR-0124 (operator-driven layout planning), `METAENGINE-LIVE-LATENCY-MODEL.md`, ADR-0136 (temporal contract — recovery is a first-class concern)

## Context

A multi-engine `Store` routes each query to the cost-optimal engine at plan time.
The live-latency model (ADR-0124 addenda) reacts to engines getting *slow* —
probes and EWMA trackers feed `CheckRouting` replan suggestions — but nothing
reacts to an engine going *dead*. A network-partitioned remote engine today
produces an error storm: every query assigned to it fails until an operator
notices, calls `RemoveEngine` (destructive — it drops the engine's state
role), or replans manually.

Existing mechanisms do not fit:

- **Poison tracker** (`IsPoisoned`) is per-*query* (a fold that keeps failing),
  not per-*engine* — it cannot express "this backend is down".
- **Circuit breaker middleware** (`middleware/circuit_breaker.go`) guards
  command dispatch, not the metaengine's internal read routing.
- **Routing hysteresis** (`WithRoutingHysteresis`) is a *cost* deadband for
  replan suggestions; a dead engine's cost does not change — its errors do.

## Decision

The Store gains a per-engine **health tracker** that quarantines failing
engines, reroutes execution around them, and re-probes them for
reactivation:

1. **Failure classification.** After a routed operation fails, the error is
   classified with `errorfamily.Classify`. Only **Infrastructure** and
   **Transient** failures count toward an engine's health: these are the
   "backend unavailable/overloaded" families. **Rejection**, **Conflict**, and
   **Corruption** never do — client mistakes and data corruption must fail
   loudly, not trigger failover.
2. **Quarantine.** Consecutive classified failures on one engine above a
   threshold (default 3, `WithEngineFailureThreshold`) quarantine it: the
   engine is excluded from execution-time routing while keeping its plan
   assignments and state. Quarantine is logged (`slog.Warn`) and stamped.
3. **Reroute.** `Execute` against a quarantined engine's assignment reroutes
   to the cheapest non-quarantined engine that natively serves the query's
   ADT (same capability-aware partition rule as the planner, so a reroute
   target is one the planner itself could have chosen). With no candidate,
   the original error is returned — fail loud, never silently wrong.
   Reroute is execution-scoped (a `shadowQuery` engine override); the plan
   is untouched, so recovery is a probe away rather than a replan away.
4. **Auto-reprobe.** `Store.StartAutoReprobe(ctx, interval)` probes
   quarantined engines that implement `Prober`; a successful probe
   reactivates the engine (health reset, logged). Engines without `Prober`
   stay quarantined until an operator calls `ReactivateEngine` explicitly —
   an engine we cannot probe is an engine we cannot vouch for.
5. **Observability.** `GetEngineStats` and `Doctor` expose per-engine health
   (state, consecutive failures, quarantined-at, last classified error) so
   operators see deactivation, not just its symptoms.

### Interaction with routing hysteresis

Cost-based replanning keeps its deadband (hysteresis + min-delta) because
flapping *data placement* costs a rebuild. Health-based quarantine is
immediate (a dead engine has no useful deadband) and its recovery is
deliberately conservative: reactivation requires a successful probe, not a
timeout, so a flapping engine re-enters rotation only when it answers.

## Consequences

- A dead engine degrades the Store from "error storm" to "logged failover" —
  queries answer from the next-best engine while the operator is informed.
- Additive in v4: no existing call changes behavior while engines are
  healthy; a failing engine previously errored, now it fails over. No
  signature changes; new options and methods only.
- Engine wrapper authors should classify their errors with errorfamily
  families; unclassified errors do not quarantine (conservative default).
- Follow-up (out of scope): fold-write failover for quarantined engines
  (currently reroute covers reads; writes to a quarantined engine's
  collections error loudly until reactivation or replan).
