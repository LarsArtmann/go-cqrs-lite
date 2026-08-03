# Meta-Engine Universal ADT Support

> **Date:** 2026-08-02
> **Status:** Design exploration — NOT STARTED (implementation deferred)
> **Related:**
>
> - Replication model: [`2026-08-03_00-51_SUPERB-METAENGINE-REPLICATION-MODEL-CORRECTION.md`](2026-08-03_00-51_SUPERB-METAENGINE-REPLICATION-MODEL-CORRECTION.md)
> - Canonical design: [`meta-engine-design.md`](meta-engine-design.md)
> - Eventual consistency + Iroh: [`meta-engine-eventual-consistency-and-iroh.md`](meta-engine-eventual-consistency-and-iroh.md)

---

## TL;DR

Every engine should declare complexity for **all 10 ADTs** instead of silently skipping the ones it doesn't natively implement. The planner's routing space becomes continuous (every engine is a candidate for every query), with **SCREAM diagnostics** surfacing non-native fallbacks at plan time. This eliminates the `errADTNotSupported` dead-end and enables gradual adoption.

---

## Problem: Fragmented Routing Space

The planner currently skips engines that don't list an ADT in their `Supports` map. If no engine supports the ADT, `planQuery` returns `errADTNotSupported`. The routing space is fragmented:

| ADT          | Memory | SQLite | Pebble | DuckDB | Postgres |
| ------------ | :----: | :----: | :----: | :----: | :-------: |
| ADTMap       | O(1)   | O(logN)| O(1)   | O(logN)| O(logN)   |
| ADTSet       | O(1)   | O(logN)| O(1)   |   —    |    —      |
| ADTCounter   | O(1)   | O(1)   | O(N)   | O(1)   | O(1)      |
| ADTGraph     | O(deg) | O(N)   | O(N)   |   —    |    —      |
| ADTSortedMap | O(N)   | O(logN)| O(N)   | O(logN)| O(logN)   |
| ADTLog       | O(N)   | O(logN)| O(logN)|   —    |    —      |
| ADTMultimap  | O(1)   | O(logN)| O(logN)|   —    |    —      |
| ADTVector    | O(N)   |   —    |   —    |   —    |    —      |
| ADTSearch    | O(N)   |   —    |   —    |   —    |    —      |
| ADTSpatial   | O(N)   |   —    |   —    |   —    |    —      |
| **Total**    | **10** | **7**  | **7**  | **3**  | **3**     |

**Three consequences:**

1. **Dead-ends.** A Vector query with only SQLite/DuckDB/Postgres engines → `errADTNotSupported`. No fallback, no degraded path.
2. **Surprising failures.** A consumer registers DuckDB (fast columnar) + Memory (fallback) expecting "DuckDB first, Memory fallback". Instead, for Vector queries, the planner picks Memory silently — no diagnostic explaining WHY DuckDB was skipped.
3. **No honest cost signal.** When DuckDB doesn't list Vector, the planner doesn't know "DuckDB could do this via brute-force scan at O(N)". It just sees "not supported". The consumer has no way to evaluate the tradeoff.

---

## Proposed Design: Universal Supports + SCREAM Diagnostics

### Step 1: Every engine declares every ADT

Each engine's `Supports` map grows to cover all 10 ADTs. Non-native ADTs get a **degraded complexity** reflecting the brute-force cost:

| Engine   | Native ADTs                | Fallback strategy                         | Fallback complexity |
| -------- | -------------------------- | ----------------------------------------- | -------------------- |
| Memory   | All 10 (brute-force native)| N/A                                       | N/A                  |
| SQLite   | Map, Set, Counter, Graph, SortedMap, Log, Multimap | Full-table scan + in-memory computation | O(N) for Vector/Search/Spatial |
| Pebble   | Same 7 as SQLite           | Prefix scan + in-memory computation       | O(N) for Vector/Search/Spatial |
| DuckDB   | Map, Counter, SortedMap    | Full-table scan + SQL aggregation for Set/Log/Multimap; in-memory for Vector/Search/Spatial | O(N) for all fallbacks |
| Postgres | Map, Counter, SortedMap    | Same as DuckDB + `pg_trgm` for Search     | O(N) for all fallbacks |

### Step 2: SCREAM diagnostics at plan time

When the planner routes a query to an engine via a **non-native fallback** (degraded complexity), it emits a `DEGRADED` diagnostic:

```
DEGRADED: vector_search routed to sqlite via O(N) brute-force scan
  Estimated 800ms at 10K embeddings. DuckDB VSS extension would be sub-ms.
  Affected query: semantic_search

DEGRADED: graph_traversal routed to duckdb via O(N) full-table scan
  No native graph support on DuckDB. Recursive CTE would be O(depth × degree).
  Affected query: message_replies
```

### Step 3: Execution-time type assertion (Q2 answer: option b)

The `Supports` map declares cost, but the executor still type-asserts the backend interface at runtime. If the engine doesn't implement the backend:

```
ERROR: vector_search routed to sqlite (DEGRADED at plan time) but
  sqlite does not implement VectorBackend. Query fails at runtime.
  The plan-time DEGRADED diagnostic warned about this.
```

This is acceptable because:
- The SCREAM diagnostic warned at plan time (the consumer SAW the tradeoff)
- The consumer chose to proceed (ignored or accepted the warning)
- Runtime failure is the honest outcome of ignoring a degraded plan

**Future enhancement (option a):** engines implement actual fallback backends (brute-force Vector in SQLite via scan + Go-side cosine similarity). High effort, deferred.

---

## Open Questions

### Q1: Degraded complexity marker

**Question:** Should non-native ADTs use a special `Complexity` value (e.g., `ComplexityDegraded`) or just `ComplexityON`?

**Analysis:** A special marker is more honest — it distinguishes "O(N) because this engine scans" (native, e.g., Memory's SortedMap) from "O(N) because this engine brute-forces a non-native operation" (degraded, e.g., SQLite's Vector).

**Recommendation:** Add a `Supports` map entry with `ComplexityON` AND a parallel `DegradedADTs` set on `EngineProfile`. The planner checks both: if the ADT is in `DegradedADTs`, emit SCREAM even if the cost is competitive. This avoids overloading the `Complexity` type.

### Q2: Execution fallback vs declared cost

**Question:** When the planner routes a degraded query to an engine, does the engine need an actual fallback implementation?

**Answer (from plan Q2):** Option (b) first — the `Supports` map declares cost, the executor type-asserts at runtime, and the SCREAM diagnostic warns at plan time. Runtime failure is acceptable when the warning was ignored. Option (a) (actual fallback implementations) is a future increment.

---

## Implementation Phases

### Phase A: Audit + design (this doc)

- [x] Audit engine ADT coverage (matrix above)
- [x] Design SCREAM diagnostic format
- [x] Answer Q1 (degraded marker) and Q2 (execution fallback)
- [ ] Review with stakeholder

### Phase B: Universal Supports entries

- [ ] Add `DegradedADTs` field to `EngineProfile`
- [ ] Extend each engine's `Supports` map to all 10 ADTs
- [ ] Mark non-native ADTs in `DegradedADTs`
- [ ] Add `degradedADTRule` to planner rule pipeline

### Phase C: SCREAM diagnostics

- [ ] Implement `degradedADTRule` — emit DEGRADED diagnostic when ADT is in `DegradedADTs`
- [ ] Include estimated cost at scale in the message
- [ ] Test: planner picks cheapest engine, emits SCREAM for degraded routing

### Phase D: Eliminate errADTNotSupported

- [ ] Change `planQuery` to never return `errADTNotSupported` when any engine is available
- [ ] The only failure case: zero engines registered
- [ ] Test: every ADT routes to some engine, with or without SCREAM

---

## References

- [`AllADTs()`](../../metaengine/enum_validation.go:9) — canonical 10-ADT registry
- [`EngineProfile.Supports`](../../metaengine/engine.go:13) — the map to extend
- [DDIA Ch1](https://dataintensive.net/) — performance, latency, brute-force vs indexed
- [`meta-engine-design.md`](meta-engine-design.md) — canonical planner design
