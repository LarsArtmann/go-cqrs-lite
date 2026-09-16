# Vector search paths — per-engine k-NN benchmarks (2026-09-16)

Feature: TODO_LIST "Vector-search verification tail" item (c) — measured the
same k-NN workload on the three engines that ship a real `VectorBackend`
implementation with a search benchmark next to the engine. Semantics per
ADR-0140 (cosine = 1−cosSim, ascending = nearest via `TopKNearest`).

Harnesses:

- `metaengine/sqliteengine/vector_bench_test.go` — `BenchmarkVectorSearch_GoScan`
  (modernc.org/sqlite, pure Go; rows cross `database/sql` as JSON text and are
  scored in Go)
- `metaengine/tursoengine/vector_bench_test.go` — `BenchmarkVectorSearch_LibSQLPushdown`
  (embedded libSQL; `vector_distance_cos` + `ORDER BY ... LIMIT k` pushed into SQL)
- `metaengine/duckdbengine/vector_bench_cgo_test.go` — `BenchmarkVectorSearch_SQLPushdown`
  (CGo; `array_cosine_distance` over `FLOAT[64]` casts, parameters cross as JSON text)

## Environment

| Item    | Value                                                        |
| ------- | ------------------------------------------------------------ |
| Date    | 2026-09-16                                                   |
| CPU     | AMD Ryzen AI MAX+ 395 (32 threads)                           |
| RAM     | 124 GB                                                       |
| Go      | 1.26.x, `-tags "goexperiment.jsonv2"` (+ `cgo` for DuckDB)   |
| Load    | quiet machine, benchmarks run solo (no concurrent test suite) |

Workload: 1000 vectors × 64 dims (`float32`), metadata present, k = 10,
metric = cosine, single collection. 3 runs × 2 s each; medians below.

## Results (lower is better)

| Engine                     | Path                                    | Median ns/op | B/op   | allocs/op |
| -------------------------- | --------------------------------------- | ------------ | ------ | --------- |
| turso (embedded libSQL)    | SQL pushdown (`vector_distance_cos`)    | **759 423**  | 8 404  | 343       |
| sqlite (modernc, pure Go)  | Go scan (fetch all, score in Go)        | 967 391      | 944 122| 10 036    |
| duckdb (CGo)               | SQL pushdown (`array_cosine_distance`)  | 1 217 911    | 4 114  | 127       |

## Raw output

```text
BenchmarkVectorSearch_LibSQLPushdown-32    3009    761584 ns/op    8405 B/op    343 allocs/op
BenchmarkVectorSearch_LibSQLPushdown-32    3150    757233 ns/op    8404 B/op    343 allocs/op
BenchmarkVectorSearch_LibSQLPushdown-32    3024    759423 ns/op    8404 B/op    343 allocs/op
BenchmarkVectorSearch_GoScan-32            2497    980846 ns/op  944123 B/op  10036 allocs/op
BenchmarkVectorSearch_GoScan-32            2488    967391 ns/op  944122 B/op  10036 allocs/op
BenchmarkVectorSearch_GoScan-32            2440    956627 ns/op  944122 B/op  10036 allocs/op
BenchmarkVectorSearch_SQLPushdown-32       2012   1217895 ns/op   4117 B/op    127 allocs/op
BenchmarkVectorSearch_SQLPushdown-32       2032   1362503 ns/op   4109 B/op    127 allocs/op
BenchmarkVectorSearch_SQLPushdown-32       1881   1217911 ns/op   4114 B/op    127 allocs/op
```

## Reading

- **Pushdown wins on the boundary, not the math.** libSQL transfers only the
  k winners across the driver boundary (8.4 KB, 343 allocs); the sqlite Go
  scan hauls all 1000 rows through `database/sql` + JSON decode (944 KB,
  10 036 allocs — ~112× more bytes, ~29× more allocs) to score them in Go.
  Net: pushdown ≈1.3× faster end-to-end at this corpus size.
- **DuckDB is the leanest per query (127 allocs, 4.1 KB) but the slowest
  wall-clock.** Per-query `CAST(vec AS FLOAT[64])` on both sides plus JSON
  text parameters dominate; its vectorized C++ scan does not offset that at
  N=1000. libSQL's native vector functions need no cast.
- **All three are O(N) brute force** — no ANN index ships (DuckDB's VSS/HNSW
  is a tracked ROADMAP item). Numbers describe the boundary cost shape, not
  asymptotics; do not extrapolate linearly to N=1M.

## Gate policy

These benchmarks are deliberately NOT wired into
`scripts/benchmark-regression.sh` (25%-threshold CI gate): the O(N) paths are
corpus-size-sensitive and would false-positive under normal CI noise. Re-run
manually with the commands above when touching a vector path.
