# ADR-0084: Metaengine Layered Architecture (StorageLayout, Cost Matrix, Materialize-vs-Replay)

**Date:** 2026-08-01
**Status:** ACCEPTED

## Context

After the rule pipeline extraction (ADR-0083), the planner could support
new capabilities as composable rules. The plan identified three
high-value additions:

1. **Materialize-vs-replay** — THE event-sourcing-specific killer feature.
   No relational query engine can ask "should this table exist?" because
   tables are deployment facts. In ES, projections are planning decisions.

2. **StorageLayout + cost matrix** — Make the planner reason about WHY one
   engine beats another (e.g., columnar beats B-Tree for Counter) rather
   than relying on opaque NsPerOp values.

3. **Plan serialization** — Persist/diff/pin plan decisions for auditability.
   "Why did the planner pick this engine?" → inspect serialized plan.

## Decision

### Materialize-vs-Replay

Added `WorkloadStats` struct and `WithWorkloadStats()` plan option. The
planner computes two costs:

```
replay_cost      = read_rate * avg_stream_length * fold_cost_per_event
materialize_cost = write_rate * fold_cost_per_event + read_rate * query_cost_per_lookup
```

When `materialize_cost < replay_cost`, emits INFO diagnostic recommending
materialization. Otherwise emits WARN suggesting replay. Pure advisory —
no hard overrides.

Added `Store.ObservedWorkloadStats()` for automatic rate derivation from
internal write/read counters + uptime.

### StorageLayout Type + Cost Matrix

Added `StorageLayout` constants: `LayoutRow`, `LayoutColumnar`, `LayoutLSM`,
`LayoutKV`. Added `Layouts map[ADT]StorageLayout` field to `EngineProfile`
(additive — zero existing fields changed, default = current behavior).

Built universal cost matrix `(ADT × StorageLayout) → Complexity` in
`layout_type.go:layoutComplexity()`. Example:

- `(Counter, Columnar)` → O(1) via native aggregation
- `(Counter, Row)` → O(N) via full scan
- `(Map, KV)` → O(1) via hash lookup

### Plan Serialization

Added `SerializablePlan` type — JSON-serializable representation of
`PlanResult`. Captures engines, queries, rules, layouts. Supports
roundtrip via `SerializeToJSON()` / `DeserializePlan()`.

### Enriched EXPLAIN

Added `RuleTraceEntry` to `PlanResult`. Each rule appends its name, the
query it acted on, and a brief reason. `Report()` now includes a
`--- Rule Trace ---` section showing the full rule chain.

## Consequences

- EngineProfile gained one additive field (`Layouts`) — backward compatible
- The Store tracks write/read counts via atomic counters for WorkloadStats
- Rules now record trace entries for enriched EXPLAIN output
- Plans can be serialized, diffed, and pinned for audit/debugging
- The materialize-vs-replay formula is advisory — wrong advice is annoying, not destructive
