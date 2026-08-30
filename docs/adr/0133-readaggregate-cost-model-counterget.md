# ADR-0133: ReadAggregate Cost Prices the CounterGet Path

- **Status:** Accepted (decision G1, status 2026-08-30_16-13; alignment in progress)
- **Date:** 2026-08-30
- **Context:** Session-6 question G1 — `EngineProfile.ReadCosts.NsPerAggregate` had
  three different meanings across engines: KV engines (badger/bbolt/pebble,
  2026-08-30 wave) calibrated it as a `CounterGet` prefix-scan per-row cost;
  pg/mysql/duckdb calibrated it as SQL SUM-over-map-rows (AggregateReader
  workloads, 10K rows); dgraph calibrated it against GraphNeighbors depth-3.

## Context

There are two aggregate execution paths in metaengine, and they are disjoint:

1. **Store-level `ReadAggregate` read pattern** — declared ONLY for
   `ADTCounter` queries (`query.go` `infer()`), executed by
   `Store.executeQueryInner` → `CounterBackend.CounterGet` on every engine
   (`execute.go`). This is the path the cost-based planner prices: a query
   whose `QueryReadPattern()` is `ReadAggregate` gets its latency estimate
   from `EngineProfile.NsForRead(ReadAggregate)`.
2. **Typed-reader aggregation** — `TypedReader.Sum/Avg/Count` dispatch
   directly to `AggregateReader.Aggregate` when the collection's engine
   implements it (SQL engines; `typed_reader.go aggregatePushdown`), with an
   in-Go Scan fallback otherwise. This path NEVER consults the planner and
   has no read pattern.

So `NsForRead(ReadAggregate)` prices exactly one workload: CounterGet over a
counter collection (bounded by the ADTCounter scale threshold, ~1K distinct
keys). The SQL-SUM and GraphNeighbors calibrations priced workloads the
pattern cannot reach, skewing counter-query routing: a remote engine's
counter reads looked as cheap as its vectorized SUM.

## Decision

`ReadCosts.NsPerAggregate` is DEFINED as the per-row cost of
`CounterBackend.CounterGet` on that engine (total query time divided by the
counter-map size; benches use 1,000 counters, matching the scale threshold).

- KV engines: already conform (2026-08-30 wave).
- duckdb, pg: recalibrated in the 2026-08-30/31 live windows
  (`BenchmarkCalibration_<Eng>_CounterGet`).
- mysql, dgraph: constants retain their legacy numbers (SQL SUM /
  GraphNeighbors) until a live calibration window; their engine.go comments
  carry an explicit divergence note referencing this ADR so no reader
  mistakes them for CounterGet prices.
- The typed AggregateReader SUM costs are deliberately NOT stored in
  ReadCosts: that path bypasses the planner, so a cost constant for it would
  be dead weight. If planner-priced typed aggregation ever lands, it needs a
  new read pattern (e.g. ReadTypedAggregate), not a reinterpretation of this
  field.

## Consequences

- Counter-query routing estimates are consistent across all engines: an
  engine's aggregate price is what serving an ADTCounter query actually
  costs on it.
- The pre-existing SQL-SUM calibration benches (pg/duckdb
  `BenchmarkCalibration_*_AggregateSum`) remain valid as documentation of
  the AggregateReader path; their numbers just no longer feed
  NsPerAggregate.
- Routing behavior may shift for ADTCounter queries on remote engines
  (their aggregate price rises toward honest CounterGet costs; Memory and
  embedded KV remain the cheap choices, preserving the ranking).
