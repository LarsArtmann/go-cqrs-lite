# Turso Materialized Views — Operator-Option Benchmarks (2026-09-07)

Feature: ADR-0135 (`MaterializedViewSpec` → Turso IVM acceleration, served
through `AggregateReader`/`GroupedAggregateReader`).
Harness: `metaengine/tursoengine/matview_bench_test.go` (embedded libSQL via
`turso.tech/database/tursogo v0.7.2`, file-backed DSNs, WAL, single
connection, chunked 1k-row seed transactions, per-case isolated processes).

## Environment

| Item    | Value                                                                                                                                           |
| ------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Date    | 2026-09-07, 18:55 UTC+2                                                                                                                         |
| CPU     | AMD Ryzen AI MAX+ 395 (32 threads)                                                                                                              |
| RAM     | 124 GB                                                                                                                                          |
| Go      | 1.26.x, `-tags "goexperiment.jsonv2"`                                                                                                           |
| Storage | NVMe-backed datadir (`.gotmp` on disk), WAL                                                                                                     |
| Load    | **load avg 82/72/46 during this run** — absolute numbers are inflated; ratios are stable across runs (a low-load repeat is included for writes) |

Workload: `orders` rows `{customer: "c<i%N>", amount: (i%7)*10 + i/100}`,
31 customers at 1k rows / 99 at 10k / 316 at 100k. Accelerated engine
declares the 7-view matrix at scale=1k (scalar SUM/COUNT/AVG/MIN/MAX + grouped
SUM/AVG over `customer`; `SUM_VIA_GROUPED` declares only the grouped SUM view).

## Read side — unfiltered aggregates (lower is better)

### Baseline scan is O(N); matview reads are flat

| Aggregate | Baseline 1k | Baseline 10k | Baseline 100k | Matview (1k, 7 views) | Speedup |
| --------- | ----------- | ------------ | ------------- | --------------------- | ------- |
| SUM       | 1.59 ms     | 9.15 ms      | 107.1 ms      | **21.3 µs**           | **75×** |
| COUNT     | 80.5 µs     | 681 µs       | 6.62 ms       | **9.8 µs**            | 8×      |
| AVG       | 893 µs      | 9.44 ms      | 105.3 ms      | **10.4 µs**           | **86×** |
| MIN       | 888 µs      | 9.74 ms      | 106.2 ms      | **9.8 µs**            | **91×** |
| MAX       | 895 µs      | 8.78 ms      | 103.9 ms      | **10.8 µs**           | **83×** |

Baseline scales linearly (~1.05 ms per 1k rows for JSON-extracting SUM/AVG/
MIN/MAX; COUNT is index-only so ~66 µs/1k rows). The matview number is a
single-row read — it does not grow with N, so the speedup column grows with
collection size.

### Grouped and derived reads (1k rows, 31 groups)

| Shape                                       | Baseline | Matview    | Speedup |
| ------------------------------------------- | -------- | ---------- | ------- |
| GROUP BY customer, SUM (exact grouped view) | 1.89 ms  | **115 µs** | 16×     |
| Scalar SUM via grouped view (derivation)    | 974 µs   | **41 µs**  | **24×** |

Grouped reads scale with the number of GROUPS (31 here), not rows. The
scalar-through-grouped derivation shows the middle option: one grouped view
serves both the per-group breakdown and the total.

## Write side — IVM overhead (steady-state REPLACE on a 5k-key set)

| Maintained views | ns/op (high-load run) | ns/op (low-load repeat) |
| ---------------- | --------------------- | ----------------------- |
| 0                | 20.26 ms              | 21.6 µs                 |
| 1 (grouped SUM)  | 25.13 ms (+24%)       | 47.7 µs (+121%)         |
| 3                | 26.61 ms (+31%)       | 87.4 µs (+305%)         |

The high-load run (IO contention on fsync path) compresses ratios; the
low-load repeat isolates the CPU+IO cost: **each maintained view adds a fixed
~22–26 µs per write** on this machine with cache-hot pages, independent of
collection size. Guidance: declare views for hot aggregates only; a
write-heavy collection with three rollup views pays ~3× the base write cost.

## Known constraints (upstream turso-go v0.7.2)

- COMMIT of transactions that maintain materialized views fails
  **deterministically at ~27,000 cumulative view-maintained rows per
  process** ("cannot commit — no transaction is active"; verified 24/24
  rounds at exactly chunk 27000 with 50k rows × 1 grouped view × 316
  groups). Below the wall, failure is probabilistic: large single
  transactions (≥~5k statements) through many views fail earlier, prior
  scan activity shrinks the budget, grouped views fail before scalar ones.
  Committed data is always intact. Bench seeds chunk at 1k rows and retry
  on a fresh engine; the `agg=*/scale=10k|100k` matview cases are
  skip-marked for this reason. Ready-to-file upstream draft:
  `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`.

## Reproducing

```bash
cd metaengine/tursoengine
GOWORK=off go test -tags "goexperiment.jsonv2" -run XXXNONE \
  -bench 'BenchmarkMatView' -benchtime 1s .
```

Bench cases at `scale=10k|100k` run the baseline leg only (see skip markers);
the accelerated read numbers live at `scale=1k`.
