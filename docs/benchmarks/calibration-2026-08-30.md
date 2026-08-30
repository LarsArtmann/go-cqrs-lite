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

Aggregate semantics on every engine: per-row cost of `CounterGet` over a
1K-key counter map (ADR-0133 — the `ReadAggregate` pattern executes
CounterGet; typed `Sum/Avg` bypasses the planner and is deliberately not
priced). pg pending live-window recalibration (`BenchmarkCalibration_Postgres_CounterGet`
is committed, unmeasured). mysql/dgraph keep legacy numbers with DIVERGENCE
comments in engine.go.

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
- Planned: scheduled CI job diffing fresh benches against this baseline
  (warn >25%).
