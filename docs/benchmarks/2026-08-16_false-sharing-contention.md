# False-Sharing Contention Baseline — 2026-08-16

> F47-F49 measure-then-pad campaign from the perf Pareto plan
> (`docs/planning/2026-08-16_03-18_PERF-PARETO-SAFETY-FIRST-EXECUTION.md`).
> Protocol: bench adjacent (production) layout against a padded mirror at
> `-cpu 16,32`, `count=10`; pad ONLY if the padded variant wins by more than
> 10%. Decisions recorded either way.

## Machine context

AMD Ryzen AI MAX+ 395, 32 logical cores, linux/amd64, NixOS 26.11, Go 1.26.x,
`goexperiment.jsonv2`. All commands run exclusively (no concurrent load).

```bash
go test -tags "goexperiment.jsonv2" -run '^$' -bench '<Pattern>' -cpu 16,32 -count=10
```

## 1. workloadMeter (metaengine) — already padded, baseline extended

`BenchmarkWorkloadMeterContention` (`metaengine/workload_meter_bench_test.go`),
write/read role split across parallel goroutines:

| GOMAXPROCS | sec/op        |
| ---------- | ------------- |
| 16         | 4.381n ± 7%   |
| 32         | 4.829n ± 5%   |

The shipped 128-byte pad between `writeCount` and `readCount`
(`store_collaborators.go`) holds at high core counts — no regression vs the
recorded @4/@8 numbers (3.4n/3.2n).

## 2. multiSeqCounter (sqliteengine) — PADDED (the win)

Per-multimap-collection sequence counters
(`sqliteengine/backends.go`). Go's allocator packs 32-byte objects 16-per-512B
span, so two hot collections' counters can share a cache line. The `[2]T`
array benches force the worst-case adjacency.

Decision: **pad** — padded wins 2.2-2.8x, far beyond the 10% threshold.
`multiSeqCounter` now carries a trailing `_ [96]byte` (128-byte size class,
covers 128-byte-line ARM cores), same convention as `workloadMeter`.

| Benchmark (10 runs)        | @16 cpu       | @32 cpu       |
| -------------------------- | ------------- | ------------- |
| Unpadded control (pre-pad) | 19.74n ± 27%  | 15.42n ± 22%  |
| Padded mirror (campaign)   | 7.106n ± 4%   | 8.295n ± 8%   |
| Unpadded control (verify)  | 19.64n ± 17%  | 18.80n ± 5%   |
| **Padded production**      | **6.889n ± 4%** | **7.383n ± 20%** |

The post-pad production-struct run reproduces the mirror numbers: −65% @16,
−61% @32 vs the unpadded control. `BenchmarkMultiSeqCounterUnpadded` stays in
the tree as the negative control — if the pad is ever removed,
`BenchmarkMultiSeqCounterPadded` degrades to those numbers.

## 3. projectionhost worker counters — NO PAD (documented decision)

`BenchmarkWorkerCountersAdjacent`/`Padded`
(`projectionhost/worker_falsesharing_bench_test.go`): one writer goroutine
(the worker event loop's `processed` + `lastProcessedNs` traffic) against
GOMAXPROCS−1 spinning `snapshot()` readers — the pessimistic bound of a
metrics scrape, which in production is rare.

| Benchmark (10 runs) | @16 cpu      | @32 cpu      |
| ------------------- | ------------ | ------------ |
| Adjacent (current)  | 217.8n ± 9%  | 289.2n ± 53% |
| Padded mirror       | 344.5n ± 5%  | 400.4n ± 51% |

Decision: **no pad** — the padded layout is ~58% SLOWER for the writer under
reader load (robust at @16, ±5%). This matches the prior analysis: the four
counters are single-writer within a worker (same core writes them all, so they
cannot false-share with each other), and the mutex/state read path is rare.
Worker counters keep their current layout.

## 4. SSEReplay.seq (metaengine) — NO PAD (documented decision)

`BenchmarkSSEReplaySeqAdjacent`/`Padded`
(`metaengine/sse_replay_falsesharing_bench_test.go`): parallel `record()`
calls — `seq.Add(1)` before the lock, ring write under `mu`.

| Benchmark (10 runs) | @16 cpu      | @32 cpu      |
| ------------------- | ------------ | ------------ |
| Adjacent (current)  | 54.66n ± 46% | 70.62n ± 31% |
| Padded mirror       | 61.45n ± 30% | 49.29n ± 17% |

Decision: **no pad** — deltas are contradictory across core counts (+12% @16,
−30% @32) with high variance. `record()` touches BOTH `seq` and the
mutex-guarded fields on every call, so concurrent recorders pull both cache
lines regardless of layout; separating them cannot pay. `SSEReplay` keeps its
current layout.

## Ledger updates

These results are reflected in `docs/BENCHMARKS.md` (CPU / memory micro-paths).
Re-run this file's commands and compare with benchstat before touching any of
the three measured layouts.
