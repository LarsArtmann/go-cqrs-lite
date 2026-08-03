# ADR-0093: Metaengine Replication Model

Date: 2026-08-03

## Status

Accepted

## Context

The metaengine planner routes queries across 5 single-node engines (Memory,
SQLite, Pebble, DuckDB, Postgres) based on ADT complexity and per-operation
cost. As the project moves toward distributed engines (Iroh CRDT, CockroachDB,
Cloud Spanner), the planner needs to reason about three orthogonal cost
dimensions that single-node engines conflate:

1. **Compute cost** — how many operations, at what per-op latency (scales with volume)
2. **Network cost** — round-trip time to reach the data (additive fixed, does NOT scale with volume)
3. **Replication staleness** — how fresh is the data (not latency at all — a freshness property)

### What went wrong first

The initial implementation (commit `31f26b8c`) proposed a `VisibilityModel`
with `VisibilityLocal` / `VisibilityGlobal` and placed a `visibility` field on
`QueryConfig`. Through five rounds of review this was proven wrong on three
counts:

1. "Visibility" is a temporal concept in distributed systems literature (MVCC,
   visibility lag), not a spatial/topological one
2. Replication topology is an **engine property**, not a **query property** —
   queries declare what to compute, not where data lives
3. The naming conflated three orthogonal dimensions (topology, latency, staleness)

The canonical terms come from DDIA: Replication (Ch5), Replication Lag (Ch5),
Network RTT (Ch1).

## Decision

Add three fields to `EngineProfile` using DDIA-canonical naming:

```go
type EngineProfile struct {
    // ...existing fields...
    Replication    Replication     // DDIA Ch5: topology
    ReplicationLag time.Duration   // DDIA Ch5: staleness
    NetworkRTT     time.Duration   // DDIA Ch1: network distance
}
```

### Replication type (4 modes)

```go
type Replication string
const (
    ReplicationNone         Replication = ""               // zero value
    ReplicationSingleLeader Replication = "single-leader"
    ReplicationMultiLeader  Replication = "multi-leader"
    ReplicationLeaderless   Replication = "leaderless"
)
```

### Three locked design decisions

1. **`ReplicationNone = ""`** — the Go zero value IS none. Every existing
   engine profile defaults to single-node automatically without code changes.
   This matches the `io.SeekStart = 0` pattern.

2. **Replication is engine-only.** `QueryConfig` has ZERO new fields. Queries
   declare what to compute (folds, ADTs). Engines declare how data is stored.

3. **NetworkRTT is additive in the cost formula; ReplicationLag is NOT.**

```
estimated_latency = (ops × nsPerRead / 1e6) + NetworkRTT
```

Staleness (`ReplicationLag`) is a freshness property, not a performance cost.
It surfaces as a planner diagnostic, not as latency inflation.

### Cost model: why RTT must be separate from NsPerRead

Baking RTT into `NsPerRead` (e.g., 500,000 ns/op for a remote engine) causes
the cost estimator to wildly overestimate scans:

| Approach                             | Point lookup (1 op) | Scan (10K ops) | Problem                    |
| ------------------------------------ | ------------------- | -------------- | -------------------------- |
| RTT baked into NsPerRead (500,000ns) | 0.5ms               | 5000s          | Wildly overestimates scans |
| RTT additive (separate)              | 0.5ms               | 30.5ms         | Correct in both regimes    |

### Planner integration

A `replicationRule` in the planner pipeline emits an INFO diagnostic when a
query is routed to a replicated engine with non-zero lag:

```
[INFO] find_task: routed to replicated engine "pg-replica" (single-leader, lag=50ms) — reads may be stale
```

The rule also appends a `RuleTraceEntry` for EXPLAIN debuggability.

## Consequences

### Positive

- **Foundation for distributed engines.** Iroh (CRDT/leaderless), CockroachDB
  (multi-leader), and streaming Postgres replicas (single-leader) can declare
  their replication mode without any planner changes.
- **Honest cost model.** Remote engines are penalized correctly for point
  lookups (RTT-dominated) but not over-penalized for scans (compute-dominated).
- **Zero backward-compatibility cost.** All existing engines are
  `ReplicationNone` with zero RTT — no profile constructors change.
- **CALM theorem alignment.** The metaengine's fold operations are monotonic
  for 5 of 10 ADTs (Map, Set, Counter, Multimap, Log), meaning CRDT
  convergence is mathematically guaranteed for leaderless replication.

### Negative

- **RTT is deployment-dependent, not engine-dependent.** The same Postgres
  binary could be localhost (RTT=0.1ms) or across the country (RTT=50ms). The
  consumer must set `NetworkRTT` at construction time. An auto-calibration
  helper was considered (Q1 in the plan) but deferred — the consumer knows
  their topology better than the engine does.
- **ReplicationLag is advisory.** It's a declared expectation, not a measured
  value. Real-world lag varies with load, network conditions, and partition
  healing. The planner uses it for diagnostics, not for routing decisions.

### Neutral

- `EngineProfile.String()` now includes a `(replication=X, lag=Y, rtt=Z)`
  suffix when any replication field is non-default. Existing tests use
  `ContainSubstring`, so the suffix is invisible to them.

## References

- [DDIA Ch1](https://dataintensive.net/) — performance, RTT, latency
- [DDIA Ch5](https://dataintensive.net/) — replication modes, replication lag
- [CALM Theorem](https://link.springer.com/chapter/10.1007/978-3-642-04243-6_27) — monotonicity → eventual consistency
- [`replication.go`](../../metaengine/replication.go) — implementation
- [`rule_replication.go`](../../metaengine/rule_replication.go) — planner rule
- [Plan doc](../planning/2026-08-03_00-51_SUPERB-METAENGINE-REPLICATION-MODEL-CORRECTION.md) — full execution plan
- [Design exploration](../planning/meta-engine-eventual-consistency-and-iroh.md) — Iroh + eventual consistency
- [Naming crisis detail](../status/2026-08-03_00-46_metaengine-iroh-eventual-consistency-design.md) — the 5-round review journey
