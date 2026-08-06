# ADR-0099: Per-Read-Pattern Cost Model (ReadCosts)

## Status

Accepted

## Context

The original metaengine cost model used a single `NsPerRead` scalar per engine
profile. This worked adequately for engines where all read patterns have similar
costs (e.g., Memory engine: ~100ns for everything).

However, columnar engines like DuckDB have **4000x cost variation** across read
patterns:

- Point lookup (random access): ~4000 ns/row
- Filtered scan (columnar WHERE): ~1 ns/row
- Aggregate (vectorized GROUP BY): ~0.5 ns/row

A single `NsPerRead` value forces the planner to either:

1. Overestimate scan costs (using the point-lookup rate) → wrongly rejects columnar engines
2. Underestimate point-lookup costs (using the scan rate) → wrongly selects columnar engines for O(1) work

Neither choice produces correct plan decisions.

## Decision

Introduce `ReadCosts` — a struct with per-read-pattern calibrated costs:

```go
type ReadCosts struct {
    NsPerPointLookup  float64  // MapGet, membership, log-tail
    NsPerFilteredScan float64  // SQL WHERE pushdown
    NsPerAggregate    float64  // GROUP BY/SUM/AVG
    NsPerScan         float64  // Full table scan
}
```

The planner picks the appropriate cost based on the query's `ReadPattern`.
Engines without `ReadCosts` fall back to `NsPerRead` for all patterns (backward
compatible).

`CalibrateEngine` populates `ReadCosts` automatically by running calibration
benchmarks that exercise each read pattern.

## Consequences

- `EngineProfile.ReadCosts` is now part of the plan decision — serialized into
  `SerializableQuery.ReadCosts` for plan diff/pin.
- `ExplainPlan` shows the active `ReadCosts` per engine line.
- Existing engines that set only `NsPerRead` continue working unchanged.
- `CalibrateEngine` must run benchmarks for each pattern to populate all four
  fields. Partial calibration (only some patterns) leaves the rest at zero,
  which the planner treats as "use NsPerRead fallback".

## References

- [DDIA Ch1](https://dataintensive.net/) — read patterns and their cost characteristics
- `metaengine/reliability.go` — CalibrateEngine implementation
- `metaengine/serializable.go` — SerializableReadCosts for plan serialization
- `metaengine/explain.go` — ExplainPlan ReadCosts display
