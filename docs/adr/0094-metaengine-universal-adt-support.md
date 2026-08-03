# ADR-0094: Metaengine Universal ADT Support

Date: 2026-08-03

## Status

Accepted

## Context

The metaengine planner routes queries to engines based on ADT (Abstract Data
Type) complexity and per-operation cost. Before this ADR, each engine declared
only the ADTs it natively supports in its `Supports` map. When no engine listed
an ADT, the planner returned `errADTNotSupported` — a dead-end with no fallback.

### The fragmentation problem

The routing space was fragmented:

| ADT          | Memory | SQLite  | Pebble  | DuckDB  | Postgres |
| ------------ | :----: | :-----: | :-----: | :-----: | :------: |
| ADTMap       |  O(1)  | O(logN) |  O(1)   | O(logN) | O(logN)  |
| ADTSet       |  O(1)  | O(logN) |  O(1)   |    —    |    —     |
| ADTCounter   |  O(1)  |  O(1)   |  O(N)   |  O(1)   |   O(1)   |
| ADTGraph     | O(deg) |  O(N)   |  O(N)   |    —    |    —     |
| ADTSortedMap |  O(N)  | O(logN) |  O(N)   | O(logN) | O(logN)  |
| ADTLog       |  O(N)  | O(logN) | O(logN) |    —    |    —     |
| ADTMultimap  |  O(1)  | O(logN) | O(logN) |    —    |    —     |
| ADTVector    |  O(N)  |    —    |    —    |    —    |    —     |
| ADTSearch    |  O(N)  |    —    |    —    |    —    |    —     |
| ADTSpatial   |  O(N)  |    —    |    —    |    —    |    —     |
| **Total**    | **10** |  **7**  |  **7**  |  **3**  |  **3**   |

Three consequences:

1. **Dead-ends.** A Vector query with only SQLite/DuckDB/Postgres engines got
   `errADTNotSupported`. No fallback, no degraded path.
2. **Surprising failures.** A consumer registering DuckDB + Memory expecting
   "DuckDB first, Memory fallback" would see the planner silently pick Memory
   for Vector queries with no diagnostic explaining why DuckDB was skipped.
3. **No honest cost signal.** The planner didn't know "DuckDB could do Vector
   via brute-force scan at O(N)" — it just saw "not supported."

## Decision

### 1. Every engine declares every ADT

Each engine's `Supports` map now covers all 10 ADTs. Non-native ADTs get a
**degraded complexity** of `O(N)` reflecting the brute-force fallback cost.

### 2. DegradedADTs field on EngineProfile

A new `DegradedADTs map[ADT]bool` field marks which ADTs are degraded
(non-native fallbacks). This separates "can I do this?" (`Supports`) from
"am I good at this?" (`DegradedADTs`).

```go
type EngineProfile struct {
    // ...existing fields...
    Supports     map[ADT]Complexity  // all 10 ADTs declared
    DegradedADTs map[ADT]bool        // marks non-native fallbacks
}

func (p EngineProfile) IsDegraded(adt ADT) bool
```

### 3. degradedADTRule — SCREAM diagnostics at plan time

When the planner routes a query to an engine via a degraded fallback, the
`degradedADTRule` emits a `DEGRADED` diagnostic:

```
DEGRADED: vector routed to sqlite via O(N) fallback — native engine recommended for production
```

This makes the tradeoff visible at plan time, not at runtime.

### 4. Execution-time type assertion (option b from design doc)

The `Supports` map declares cost, but the executor still type-asserts the
backend interface at runtime. If an engine declares Vector support in its
`Supports` map but does not implement `VectorBackend`, the executor returns
an `errUnsupportedVectorOps` error at runtime. This is acceptable because:

- The SCREAM diagnostic warns at plan time
- Runtime failure when ignoring warnings is expected behavior
- Actual fallback backend implementations are future work

### Coverage after this ADR

| ADT          | Memory | SQLite  | Pebble  | DuckDB  | Postgres |
| ------------ | :----: | :-----: | :-----: | :-----: | :------: |
| ADTMap       |  O(1)  | O(logN) |  O(1)   | O(logN) | O(logN)  |
| ADTSet       |  O(1)  | O(logN) |  O(1)   |  O(N)*  |  O(N)*   |
| ADTCounter   |  O(1)  |  O(1)   |  O(N)   |  O(1)   |   O(1)   |
| ADTGraph     | O(deg) |  O(N)   |  O(N)   |  O(N)*  |  O(N)*   |
| ADTSortedMap |  O(N)  | O(logN) |  O(N)   | O(logN) | O(logN)  |
| ADTLog       |  O(N)  | O(logN) | O(logN) |  O(N)*  |  O(N)*   |
| ADTMultimap  |  O(1)  | O(logN) | O(logN) |  O(N)*  |  O(N)*   |
| ADTVector    |  O(N)  |  O(N)*  |  O(N)*  |  O(N)*  |  O(N)*   |
| ADTSearch    |  O(N)  |  O(N)*  |  O(N)*  |  O(N)*  |  O(N)*   |
| ADTSpatial   |  O(N)  |  O(N)*  |  O(N)*  |  O(N)*  |  O(N)*   |
| **Total**    | **10** | **10**  | **10**  | **10**  |  **10**  |

`*` = degraded (non-native fallback via brute-force scan)

## Consequences

- **No more dead-ends.** `errADTNotSupported` only fires for custom engines
  with partial Supports maps. All built-in engines have 10/10 coverage.
- **Transparent tradeoffs.** SCREAM diagnostics make degraded routing visible
  at plan time, not at runtime.
- **Native preferred.** The cost-based ranker naturally prefers native engines
  (lower complexity) over degraded ones (O(N)) when both are available.
- **Future extensibility.** When DuckDB gains VSS extension (native Vector),
  just move ADTVector out of DegradedADTs and update the complexity. The
  SCREAM diagnostic disappears automatically.

## References

- [Universal ADT design doc](../planning/meta-engine-universal-adt-support.md)
- [Replication model ADR](0093-metaengine-replication-model.md)
- DDIA Ch1 (network RTT), Ch5 (replication)
