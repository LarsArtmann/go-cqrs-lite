# Benchmarks — Performance Ledger

> Single source of truth for measured performance characteristics. Every entry
> names a **runnable benchmark**, the machine context, and the last measured
> result. Update an entry when you touch its code path — wins that aren't
> recorded here can silently regress.
>
> Machine context for all 2026-08-16 entries: AMD Ryzen AI MAX+ 395, 32
> logical cores, linux/amd64, NixOS 26.11, Go 1.26.x, `goexperiment.jsonv2`.

## Storage write paths

| Path                                     | Benchmark                                                                    | How to run                                                                              | Baseline (before)                                 | Last measured (after)                                                                                                                       |
| ---------------------------------------- | ---------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| SQL event batch insert (PG/MySQL/DuckDB) | `TestMaxParametersForDialect`, `TestSharedBatchInsertChunks` (`storage/sql`) | `go test ./storage/sql/ -run 'Batch\|Chunk'`                                            | 99 rows/statement (SQLite limit for all dialects) | 3276 rows/statement (33x fewer round-trips), dual-capped by `MaxStatementBytes` 8 MiB packet guard; verified MariaDB-in-VM 2000×8 KiB       |
| SQL view batch upsert                    | `TestViewBatchSet_LargeValuesChunkSafely` (`storage/view`)                   | `go test ./storage/view/ -run Large`                                                    | 99 rows/statement                                 | 8191 rows/statement + byte cap (6×2 MiB values chunk safely)                                                                                |
| pgengine stream-log bulk append          | `BenchmarkStreamAppend` (`metaengine/pgengine`)                              | `go test ./metaengine/pgengine/ -bench StreamAppend -benchtime 10x -run XXX` (needs PG) | per-row INSERT in a loop                          | default: chunked multi-VALUES (10k rows/stmt); opt-in `WithCopyAppend`: COPY FROM, 1.41x @10k rows (39.3→27.9 ms), 1.49x @100k (368→248 ms) |
| bbolt concurrent writers                 | `TestBatchCommit_ConcurrentWritersIdenticalJournal` (`storage/bbolt`)        | `go test ./storage/bbolt/ -run BatchCommit -race`                                       | `db.Update` per call (one fsync each)             | opt-in `WithBatchCommit()`: `db.Batch` group commit, one fsync per writer group; journal byte-identical under 8 concurrent writers          |

## Storage read paths

| Path                          | Benchmark                                     | How to run                                                  | Baseline                                                               | Last measured                                                                                                                                                                                        |
| ----------------------------- | --------------------------------------------- | ----------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Pebble event deserialize      | `BenchmarkDeserialize*` (`storage/pebble`)    | `go test ./storage/pebble/ -bench Deserialize`              | JSON metadata round-trip per read: 5000 ns/op, 2247 B/op, 43 allocs/op | `ReconstructEventWithMetadata` direct pass: 2680 ns/op, 1205 B/op, 20 allocs/op (−46% ns, −53% allocs); + payload adopt via `ReconstructEventWithAdoptedPayload` (2026-08-16): 2872 ns/op, 1178 B/op |
| bbolt event deserialize       | `BenchmarkEventDeserialize` (`storage/bbolt`) | `go test ./storage/bbolt/ -bench EventDeserialize -run XXX` | JSON round-trip                                                        | same reconstruct shape as pebble; 2026-08-16: 2815 ns/op, 1210 B/op pre-adopt → 2521 ns/op, 1178 B/op with `ReconstructEventWithAdoptedPayload` (−10% ns)                                            |
| SQL journal ReadFrom (keyset) | storage integration suite                     | `nix run .#integration-pg`                                  | O(N²) OFFSET-style self-join cursor                                    | keyset `(occurred_at, id)` seek — ~285x faster full drains (browser-history restart case: ~4.5 min CPU → seconds)                                                                                    |

## CPU / memory micro-paths

| Path                                 | Benchmark                                              | How to run                                                               | Baseline                                                      | Last measured                                                                                                       |
| ------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| metaengine workloadMeter (contended) | `BenchmarkWorkloadMeter*` (`metaengine`)               | `go test ./metaengine/ -bench WorkloadMeter -cpu 4,8`                    | adjacent hot counters share cache lines: 6.3 ns/op @4, 6.6 @8 | 128-byte pad: 3.4 ns/op @4, 3.2 @8 (−46..51%)                                                                       |
| metaengine row layout scoring        | `BenchmarkRowLayoutCalibration_*` (`metaengine/bench`) | `go test ./metaengine/bench/ -bench RowLayout -run XXX` (engines needed) | assumed normalize≪embed                                       | measurement-derived: geomean reads 1.27x, writes 0.52x, storage 0.35x; sign-flip corrected (JOIN reads NOT cheaper) |

## Read-model / projection paths

| Path                           | Benchmark                                 | How to run                                                 | Baseline                            | Last measured                                                                                                    |
| ------------------------------ | ----------------------------------------- | ---------------------------------------------------------- | ----------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| projectionhost live checkpoint | `TestLiveCheckpoint_*` (`projectionhost`) | `go test ./projectionhost/ -run Checkpoint -count=3 -race` | save-per-event (default, unchanged) | opt-in `WithCheckpointEvery`/`WithCheckpointInterval` batch saves; crash window ≤ n−1 reprocess, flushed on Stop |

## How to add an entry

1. Name the runnable benchmark (tests count when they assert perf budgets).
2. Record baseline AND after numbers with the date.
3. Note the machine context if it differs from the header.
