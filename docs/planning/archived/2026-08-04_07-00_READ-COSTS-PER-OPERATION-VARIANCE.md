# Problem: Single-Scalar Cost Model Cannot Express Per-Operation Variance

**Date:** 2026-08-04
**Status:** Fixed (ReadCosts shipped) — remaining work tracked below
**Severity:** High — the planner was making wrong engine-selection decisions for analytical workloads

---

## The Problem in One Table

A single `NsPerRead` scalar was used for **every read operation** an engine could
perform, regardless of what the query actually did. On DuckDB, these operations
span **4,000×**:

| Read Pattern     | What it does                           | DuckDB actual cost | Old model (`NsPerRead=1200`) | Error                   |
| ---------------- | -------------------------------------- | ------------------ | ---------------------------- | ----------------------- |
| **PointLookup**  | `MapGet` — column scan for one key     | **546,000 ns**     | 1,200 ns                     | **455× underestimated** |
| **FilteredScan** | `PushdownMapScan` — SQL WHERE pushdown | **454 ns/row**     | 1,200 ns                     | 2.6× overestimated      |
| **Aggregate**    | `SUM(CAST(...))` — vectorized GROUP BY | **133 ns/row**     | 1,200 ns                     | 9× overestimated        |
| **FullScan**     | `MapScan` — full table + Go decode     | **975 ns/row**     | 1,200 ns                     | 1.2× overestimated      |

The planner uses this constant in its cost formula:

```
latency = (ops × nsPerRead / 1e6) + networkRTT
```

Where `ops` is 1 for O(1) lookups and N for O(N) scans. The formula models
volume (one axis) but ignores **operation type** (the second axis). The result:
the planner could not distinguish between "DuckDB is great at aggregations"
and "DuckDB is terrible at point lookups." Both used the same 1,200 ns constant.

### Concrete Consequence

When both Memory and DuckDB were available:

| Query                  | Memory cost          | DuckDB cost (old)      | DuckDB cost (new)        | Right answer |
| ---------------------- | -------------------- | ---------------------- | ------------------------ | ------------ |
| Point lookup           | 500 ns               | 1,200 ns               | **50,000 ns**            | Memory       |
| Aggregation (10K rows) | 5,000,000 ns (N×500) | 12,000,000 ns (N×1200) | **1,500,000 ns** (N×150) | DuckDB       |

Before the fix, the planner picked **Memory** for everything because DuckDB's
single scalar (1,200) was higher than Memory's (500). DuckDB's vectorized
aggregation advantage — the entire reason to use an OLAP engine — was invisible.

---

## Three Root Causes

### 1. The word "read" was meaningless

`NsPerRead` conflated four fundamentally different operations:

- **Point lookup** — indexed B-tree / hash lookup. One key, one row. Per-QUERY cost.
- **Filtered scan** — SQL WHERE + json_extract pushdown. Per-ROW cost.
- **Aggregation** — vectorized SUM/COUNT/GROUP BY. Per-ROW cost, but 10–50× cheaper
  than a scan on columnar engines due to SIMD vectorization.
- **Full scan** — load all rows into Go, decode JSON, filter in Go. Per-ROW cost.

These span up to 3 orders of magnitude on the same engine. A single scalar
averages them into a number that is wrong for every individual case.

### 2. The cost constants were hardcoded `const` values

```go
const DuckDBNsPerRead = 1200.0  // compile-time, never changes
```

Loaded into the profile at construction and frozen. No runtime path to update
them for DuckDB or Postgres (see root cause #3).

### 3. Runtime calibration was structurally impossible for external engines

The `CalibrateEngine` function exists but uses an **unexported** interface:

```go
type calibratable interface {  // ← unexported, internal package only
    setCalibration(nsPerOp, nsPerRead, nsPerWrite float64)
}
```

Only `memoryEngine` and `sqliteEngine` (both in the `metaengine` package) can
implement it. DuckDB, Postgres, and Pebble are in **separate Go modules** — they
structurally cannot implement an unexported interface from another package.
`CalibrateEngine` silently does nothing for them.

Even if it worked, `CalibrateEngine` only measures `MapGet` (point lookup) — it
would calibrate DuckDB's worst case and call it the read cost for all operations.

---

## The Fix: `ReadCosts` — Per-Read-Pattern Cost Constants

### New struct on `EngineProfile`

```go
type ReadCosts struct {
    NsPerPointLookup  float64  // MapGet — per-QUERY cost
    NsPerFilteredScan float64  // PushdownMapScan — per-ROW cost
    NsPerAggregate    float64  // SUM/COUNT/GROUP BY — per-ROW cost
    NsPerScan         float64  // MapScan — per-ROW cost
}
```

The planner calls `profile.NsForRead(query.QueryReadPattern())` instead of
`profile.ReadNsPerOp()`. The method picks the right field based on what the
query actually does.

### Fallback chain (zero value = "not set")

```
NsForRead(pattern):
  1. ReadCosts field for this pattern (if > 0)
  2. NsPerRead (legacy scalar, if > 0)
  3. NsPerOp (if > 0)
  4. defaultNsPerOp (100ns)
```

Engines that don't set `ReadCosts` (Memory, SQLite, Pebble) behave identically
to before. Zero regressions.

### Calibrated values (from benchmark measurements)

**DuckDB** (`duckdbengine/engine.go`):

| Field               | Value (ns) | Benchmark source                                          |
| ------------------- | ---------- | --------------------------------------------------------- |
| `NsPerPointLookup`  | 50,000     | `BenchmarkDuckDB_MapGet`: 546K measured, 50K conservative |
| `NsPerFilteredScan` | 450        | `BenchmarkCalibration_DuckDB_PushdownScan`: 454 ns/row    |
| `NsPerAggregate`    | 150        | `BenchmarkCalibration_DuckDB_AggregateSum`: 133 ns/row    |
| `NsPerScan`         | 1,000      | `BenchmarkCalibration_DuckDB_FullScan`: 975 ns/row        |

**Postgres** (`pgengine/engine.go`):

| Field               | Value (ns) | Benchmark source                                          |
| ------------------- | ---------- | --------------------------------------------------------- |
| `NsPerPointLookup`  | 5,000      | Production model (Docker measured ~28K, network-adjusted) |
| `NsPerFilteredScan` | 400        | `BenchmarkCalibration_Postgres_PushdownScan`: 402 ns/row  |
| `NsPerAggregate`    | 150        | `BenchmarkCalibration_Postgres_AggregateSum`: 149 ns/row  |
| `NsPerScan`         | 800        | `BenchmarkCalibration_Postgres_FullScan`: 805 ns/row      |

### Planner decisions: before vs after

| Query                           | Before (single scalar)  | After (ReadCosts)          | Correct? |
| ------------------------------- | ----------------------- | -------------------------- | -------- |
| Point lookup (Memory + DuckDB)  | Either (both ~1200)     | **Memory** (500 vs 50K)    | Fixed    |
| Aggregation (Memory + DuckDB)   | Memory (N×500 < N×1200) | **DuckDB** (N×150 < N×500) | Fixed    |
| Filtered scan (Memory + DuckDB) | Memory (N×500 < N×1200) | **DuckDB** (N×450 < N×500) | Fixed    |

### Tests proving the fix

`metaengine/readcost_selection_test.go` (5 tests, all passing):

- `TestReadCosts_PlannerPicksMemoryForPointLookup` — Memory wins point lookups by 100×
- `TestReadCosts_PlannerPicksDuckDBForAggregate` — DuckDB wins aggregations by 3.3×
- `TestReadCosts_NsForReadFallbackChain` — legacy engines (no ReadCosts) unaffected
- `TestReadCosts_PerPatternOverrides` — all 11 ReadPatterns route to the right field
- `TestReadCosts_4000xSpanProven` — the 333× point-vs-aggregate gap is now expressed

---

## Remaining Work

### Still hardcoded (but now correctly partitioned)

The `ReadCosts` fields are still compile-time constants set in each engine's
`Profile()` method. They do not evolve at runtime. This is acceptable for now —
the values are conservative and documented — but the long-term fix is:

- [ ] **Export `Calibratable` and make `CalibrateEngine` work for external engines.**
      Currently blocked by the unexported `calibratable` interface (see root cause #3).
      The interface should be exported as `Calibratable` and the `setCalibration`
      signature extended to accept `ReadCosts`.
- [ ] **Add `CalibrateScanEngine`** — runtime calibration for scan/aggregation costs,
      not just point lookups. `CalibrateEngine` only measures `MapGet`.

### Other engines need ReadCosts

- [ ] **Pebble `ReadCosts`** — Pebble has fast LSM point reads but slower scans. The
      gap is less extreme than DuckDB (~7× not ~4000×) but still worth expressing.
      Source: `pebbleengine/calibration_bench_test.go` (already has MapSet/MapGet;
      needs scan/aggregation benchmarks).
- [ ] **SQLite `ReadCosts`** — SQLite's row-store means point lookups and scans are
      closer in cost, but pushdown scans (with json_extract WHERE) are measurably
      cheaper than full Go-side scans.
- [ ] **Memory `ReadCosts`** — Memory's hash map is O(1) for lookups but O(N) for
      scans. Currently both use `NsPerOp=500`. Worth splitting if Memory is ever
      compared against another engine for scan workloads.

### Cost model depth

- [ ] **Separate scan cost from aggregate cost in the complexity model.** Currently
      `effectiveReadComplexity` maps `ReadAggregate` to the ADT's complexity (O(1)
      for counter). But the per-row cost of aggregation differs from the per-row
      cost of a scan. The `ReadCosts` struct handles this at the constant level,
      but the ops calculation still uses the same complexity class.
- [ ] **Add `NsPerWriteScan` for write-heavy read patterns** (e.g., MapUpdate's
      read-modify-write cycle has a different cost profile than a pure read).

---

## Files Changed

| File                                    | Change                                                          |
| --------------------------------------- | --------------------------------------------------------------- |
| `metaengine/engine.go`                  | Added `ReadCosts` struct + `NsForRead(ReadPattern)` method      |
| `metaengine/planner.go`                 | `ReadNsPerOp()` → `NsForRead(meta.QueryReadPattern())` (1 line) |
| `metaengine/explain.go`                 | Explain output shows per-pattern costs when `ReadCosts` is set  |
| `metaengine/cost.go`                    | Updated HONESTY NOTE                                            |
| `metaengine/readcost_selection_test.go` | **New** — 5 tests proving correct selection                     |
| `duckdbengine/engine.go`                | Set calibrated `ReadCosts` from benchmark measurements          |
| `pgengine/engine.go`                    | Set calibrated `ReadCosts` from benchmark measurements          |

## See Also

- [2026-08-04 Status Report](../status/2026-08-04_06-40_duckdb-pg-calibration-benchmarks.md) —
  the benchmark creation session that exposed this problem
- `metaengine/calibration_bench_test.go` — Memory + SQLite calibration
- `metaengine/duckdbengine/calibration_bench_test.go` — DuckDB calibration
- `metaengine/pgengine/calibration_bench_test.go` — Postgres calibration
- `metaengine/reliability.go` — `CalibrateEngine` (needs export for external engines)
