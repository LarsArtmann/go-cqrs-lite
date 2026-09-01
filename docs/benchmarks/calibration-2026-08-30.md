# Engine Calibration Baseline — 2026-08-30

> Hardware: AMD Ryzen AI MAX+ 395, 32 threads, shared multi-user host
> Ambient load at measurement: load avg ~3.8 (22 users, no compile storms)
> Toolchain: Go 1.26.7, `-tags "goexperiment.jsonv2"`, GOWORK=off per module
> Protocol: this file is the baseline the CI drift job and future
> recalibrations diff against. See §Protocol.

## Constants shipped 2026-08-30

| Engine | PointLookup (ns/query) | FilteredScan (ns/row) | Aggregate (ns/row) | Scan (ns/row) | Source |
|---|---|---|---|---|---|
| badger | 1100 | 650 | 165 | 630 | `BenchmarkCalibration_Badger_{Get,_FilteredScan,_CounterScan,_FullScan}` |
| bbolt | 750 | 620 | 100 | 660 | `BenchmarkCalibration_Bbolt_*` (re-verified below) |
| pebble | 700 | 830 | 125 | 700 | `BenchmarkCalibration_Pebble_*` (scan paths via ScanRawValues) |
| sqlite | 3100 | 1080 | 530 | 1240 | `BenchmarkCalibration_SQLite_*` (in-memory modernc) |
| duckdb (aggregate) | 50_000 (prior) | 450 (prior) | **420** | 1000 (prior) | `BenchmarkCalibration_DuckDB_CounterGet` (ADR-0133) |
| pg (aggregate) | 5_000 (prior) | 400 (prior) | **250** | 800 (prior) | `BenchmarkCalibration_Postgres_CounterGet` (ADR-0133, 2026-09-01) |
| mysql (aggregate) | 5_000 (prior) | 400 (prior) | **320** | 800 (prior) | `BenchmarkCalibration_MySQL_CounterGet` (ADR-0133, 2026-09-01) |
| dgraph (aggregate) | 350_000 (prior) | 900_000 (prior) | **2_700** | 450_000 (prior) | `BenchmarkDgraph_CounterGet` (ADR-0133, 2026-09-01) |

Aggregate semantics on every engine: per-row cost of `CounterGet` over a
1K-key counter map (ADR-0133 — the `ReadAggregate` pattern executes
CounterGet; typed `Sum/Avg` bypasses the planner and is deliberately not
priced). The G1 reconciliation completed 2026-09-01: pg/mysql/dgraph were
recalibrated onto CounterGet in live windows (raw runs below) — no engine
carries a legacy aggregate number or DIVERGENCE marker anymore.

## Raw runs (ns/op; medians underlined in committed constant comments)

### badger (count=3, 2026-08-30 ~15:2x)
```
Get            1150 / 1085 / 1076        → median 1085  → constant 1100
FilteredScan   6391387 / 6372411 / 6410470 (10K rows)  → median 6.37ms → 639 ns/row → 650
CounterScan    165666 / 163416 / 163465 (1K rows)      → median 163.4µs → 164 ns/row → 165
FullScan       6381567 / 6284795 / 6291774 (10K rows)  → median 6.29ms → 629 ns/row → 630
```

### bbolt (count=3, 2026-08-30 ~15:3x — first campaign; superseded by re-run below)
```
Get            761 / 742 / 737            → median 742   → constant 750
FilteredScan   8477921 / 6139867 / 5938259 → first-run cold outlier ~30% high; median 6.14ms → 614 ns/row → 620
CounterScan    87690 / 98155 / 99031      → median 98.2µs → 98 ns/row → 100
FullScan       6561501 / 6536647 / 7283958 → median 6.54ms → 654 ns/row → 660
```

### pebble (count=3, 2026-08-30 ~15:4x)
```
Get            666 / 684 / 685            → median 684   → constant 700
FilteredScan   8951820 / 8329294 / 8111506 (10K rows, ScanRawValues) → median 8.33ms → 833 ns/row → 830
CounterScan    121250 / 127971 / 125457 (1K rows)      → median 125.5µs → 125 ns/row → 125
FullScan       6950028 / 6622967 / 7178342 (10K rows)  → median 6.95ms → 695 ns/row → 700
```

### sqlite (count=3, 2026-08-30 ~19:4x)
```
PointLookup    3129 / 3057 / 3102         → median 3102  → constant 3100
FilteredScan   10697334 / 11145647 / 10753330 (10K rows, json_extract pushdown) → median 10.75ms → 1075 ns/row → 1080
CounterGet     515350 / 537145 / 531244 (1K rows)      → median 531.2µs → 531 ns/row → 530
FullScan       12379020 / 11910681 / 13002722 (10K rows) → median 12.38ms → 1238 ns/row → 1240
```

### duckdb CounterGet (count=3, 2026-08-30 ~19:3x, CGo embedded)
```
CounterGet     470897 / 407230 / 418305 (1K rows) → median 418.3µs → 418 ns/row → 420
```

### bbolt re-run (count=5, 2026-08-30 ~19:5x, steady-state check)
```
Get            (1029)/757/763/836/886      → discard first; median ~800  vs constant 750 (+7%)
FilteredScan   (5951097)/5980036/6543350/6568908/6634477 → discard first; median ~6.55ms → ~655 vs constant 620 (+5.6%)
CounterScan    (88385)/90100/91156/93161/94821 → discard first; median ~92µs → ~92 vs constant 100 (−8%)
FullScan       (5938257)/6129576/6164832/6179682/7134950 → discard first; median ~6.17ms → ~617 vs constant 660 (−6.5%)
```
Interpretation: ambient load was RISING during the re-run (concurrent
planned-tables compile wave on the same host). All deltas are within ±8%
— below any recalibration threshold, and the ±8% spread across patterns
in both directions is the load signature, not a constant error.
Constants UNCHANGED. A re-run on a genuinely quiet window (no concurrent
builds) remains worthwhile before tagging the engine modules.

### G1 reconciliation: pg/mysql/dgraph CounterGet (count=5, 2026-09-01 21:49–21:52)

Closes decision G1 (ADR-0133) end-to-end: every engine's `NsPerAggregate`
now prices the actual `ReadAggregate` execution path. Windows: ephemeral
nixpkgs Postgres (scripts/ephemeral-pg.sh), userspace MariaDB 11.4.12 on
:33061, ephemeral Dgraph 25.4.0 (nix run .#integration-dgraph).
Ambient load ~4.8–7.0 (17-18 users, no compile storms); these are ms-scale
remote-engine benches, so spread stayed ±2% (pg/mysql) and ±4% (dgraph,
excluding the cold first run).

```
pg      CounterGet  240130 / 242070 / 245802 / 246217 / 248165 (1K rows) → discard first; median 245.9µs → 246 ns/row → 250
mysql   CounterGet  325880 / 322848 / 323088 / 318659 / 315368 (1K rows) → discard first; median 320.8µs → 321 ns/row → 320
dgraph  CounterGet  3837756 / 2666601 / 2744322 / 2549433 / 2659298 (1K rows) → discard cold 3.8ms; median 2.663ms → 2663 ns/row → 2_700
```

Superseded: pg/mysql 150 (SQL-SUM-era, AggregateSum benches — those now
document the planner-bypassing typed path only) and dgraph 950_000
(GraphNeighbors depth-3 per-op misused as a per-row constant, ~350x high).

Known remaining divergence (OUT of G1 scope, flagged not silently changed):
dgraph `NsPerFilteredScan` (900_000) and `NsPerScan` (450_000) carry
per-OP measurements (SearchQuery anyofterms / GraphNeighbors depth-1) in
per-ROW fields — a k-row result is one RPC, so per-row pricing overstates
by ~k×. Honest recalibration needs result-size-scaled benches (cost as a
function of k); the point-lookup and aggregate fields are already correct.

## Protocol

1. Run per module, `GOWORK=off`, tags `goexperiment.jsonv2` (add `cgo` for
   duckdb/quic), `-run '^$' -bench '<calibration pattern>' -count=N`.
2. Discard the first count (cold caches); take the median of the rest.
3. Record ambient load (`uptime`) next to the numbers; do not calibrate
   during compile storms or concurrent `#verify` runs (AGENTS: load skews).
4. A constant change ships in the SAME commit as its bench run + this
   baseline update + the `TestRealProfiles_ReadCostsPinned` update
   (metaengine/bench/routing_regression_test.go).
5. check-changelog-symbols gates any `pkg.Symbol` cited in CHANGELOG.

## Cross-checks

- `TestRealProfiles_ReadCostsPinned` (metaengine/bench) pins bbolt/pebble
  constants end-to-end against real engines.
- `TestEngineProfilesSetReadCosts` (cmd/api-stability) enforces the roster.
- Shipped: nightly `benchmarks.yml` calibration-drift job diffs fresh benches
  against this baseline (warn >25%, fail >100%) — local engines only;
  pg/mysql/dgraph need live-DSN windows and are re-anchored by hand.
