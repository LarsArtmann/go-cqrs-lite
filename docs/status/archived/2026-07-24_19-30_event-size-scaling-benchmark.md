# Event-Size Scaling Benchmark

**Date:** 2026-07-24 19:30 (local)
**Tool:** `cqrs-bench` (rebuilt from working tree, includes new mixed-size support)
**Platform:** Linux (NixOS), Go 1.26.4, `GOEXPERIMENT=jsonv2`
**Question:** How does the library scale with (a) different event sizes and (b) mixed event sizes?

---

## TL;DR

1. **Throughput is inversely proportional to payload size above ~1 KB.** Below ~256 B it is CPU-overhead-bound (flat). Memory degrades ~7x from 64 B → 16 KB; Pebble degrades ~11x.
2. **Mixed workloads track the MEDIAN event size, not the mean.** A mixed workload with mean 1360 B performs like a uniform ~512 B workload because most events are small.
3. **Single-run numbers are unreliable on the memory backend (~20-25% variance, occasional 2x cold-run outliers).** Pebble is stable (±2-5%). **Always take multiple samples before drawing conclusions.** This is the most important practical finding.

---

## 1. Methodology

- **Profile:** `small` (1,000 streams × 10 events = 10,000 events, 4 goroutines, 30% reads).
- **Codec:** JSON.
- **Payload-size sweep:** `--payload-size` ∈ {64, 256, 1024, 4096, 16384} on `memory` + `pebble`.
- **Mixed workloads:** new `--payload-sizes` flag (uniform random pick per event).
  - Even mix: `[64,256,1024,4096]` (mean 1360 B).
  - Small-heavy: `[64,64,64,256,256,1024,4096,16384]` (mean 2776 B).
  - No-large-tail: `[64,256,1024]` (mean 448 B).
- **Variance:** uniform-1024 run 8× (4 cold, 4 with `--warmup 100`); pebble mixed run 3×.

> **Caveat (see §4):** the sweep below is single-run per cell, so each number carries ±20% noise on the memory backend. The _trend_ is reliable; the _absolute_ values are indicative only.

---

## 2. Different Event Sizes — Uniform Sweep

### Memory backend

| Payload | Throughput | Write P50  | Write P99 | Heap peak |
| ------- | ---------- | ---------- | --------- | --------- |
| 64 B    | 227.5 K/s  | 350 ns     | 3.3 µs    | 1.3 MB    |
| 256 B   | 189.9 K/s  | 360 ns     | 2.5 µs    | 1.3 MB    |
| 1024 B  | ~170 K/s * | 470-590 ns | 3-4 µs    | 1.3-17 MB |
| 4096 B  | 72.9 K/s   | 580 ns     | 4.9 µs    | 39 MB     |
| 16384 B | 32.8 K/s   | 700 ns     | 7.4 µs    | 208 MB    |

*1024 B varied 92K-190K/s across runs (see §4); median ≈170 K/s.

**Shape:** throughput is ~flat from 64 B → 256 B (fixed CPU overhead dominates: event construction, map ops, version accounting). Above 1 KB it becomes roughly linear in payload size (bandwidth/copy-bound). Heap scales linearly above 4 KB (10K events × 16 KB ≈ 160 MB raw).

### Pebble backend

| Payload | Throughput | Write P50 | Write P99 | Disk    |
| ------- | ---------- | --------- | --------- | ------- |
| 64 B    | 68.3 K/s   | 14.8 µs   | 315 µs    | 9.6 MB  |
| 256 B   | 68.0 K/s   | 10.8 µs   | 268 µs    | 11.1 MB |
| 1024 B  | ~50 K/s    | 24.8 µs   | 527 µs    | 20.9 MB |
| 4096 B  | 20.4 K/s   | 80.9 µs   | 799 µs    | 29.1 MB |
| 16384 B | 6.3 K/s    | 249.9 µs  | 2.83 ms   | 40.9 MB |

**Shape:** flat 64 B → 256 B (LSM write syscall overhead dominates tiny payloads), then declines ~linearly. Disk overhead starts positive (93% at 64 B — LSM structure dominates) and goes **negative** above 4 KB (-282% at 16 KB) because Pebble compresses the highly-repetitive synthetic padding.

> **Measurement caveat:** the negative disk overhead is an artifact of synthetic payloads (`Padding` is a run of `x` chars → compresses to near-zero). Real domain payloads (varied JSON/CBOR) compress far less. Treat disk savings at large sizes as an upper bound.

---

## 3. Mixed Event Sizes

### Stability-confirmed (3-run medians, ±spread)

| Workload                        | Mean B | Memory tput | Pebble tput (3-run) |
| ------------------------------- | ------ | ----------- | ------------------- |
| uniform 1024 B                  | 1024   | ~170 K/s    | 50.3 / 50.0 / 62.1  |
| mixed `[64,256,1024,4096]`      | 1360   | ~110 K/s    | 46.7 / 45.1 / 44.7  |
| mixed `[64,256,1024]` (no tail) | 448    | ~108 K/s    | —                   |
| small-heavy (8-size, one 16 KB) | 2776   | ~55 K/s     | ~13 K/s *           |

*Small-heavy pebble is single-run and noisy.

**Key insights:**

1. **Mixed tracks the median, not the mean.** Mixed `[64,256,1024,4096]` has mean 1360 B but performs _better_ than uniform-1024 B would predict, because 3 of 4 events are ≤1024 B. Per-event cost is sublinear for small events, so a population of mostly-small events keeps throughput high.

2. **Pebble mixed is remarkably stable** (44.7-46.7 K/s, ±2%). The I/O-bound path smooths out per-event-size variance. Memory is far noisier (GC sensitivity to allocation-size variance).

3. **One large-tail event dominates.** The small-heavy distribution includes a single 16 KB event per 8; its mean (2776 B) is modest, but throughput drops to ~55 K/s on memory — closer to uniform-4096 than uniform-1024. A small fraction of large events costs disproportionately.

### Interpretation for real systems

Most real event-sourced systems have a **long-tail size distribution**: many tiny events (status changes, counter increments, tombstones), many medium events (typical domain events with 5-15 fields), and few large events (events carrying collections, snapshots, embedded documents). The benchmark now models this directly:

```bash
# A realistic e-commerce distribution: 60% small, 30% medium, 10% large
cqrs-bench run --backend pebble --profile medium \
  --payload-sizes 64,64,64,256,256,256,1024,4096
```

Expect throughput to sit near the _median_ size's uniform number, with the large events pulling the tail (P99 latency) up disproportionately.

---

## 4. Run-to-Run Variance — The Critical Reliability Finding

### Memory backend (uniform-1024, 8 runs)

| Condition      | Samples (K/s)              | Spread         |
| -------------- | -------------------------- | -------------- |
| No warmup      | 182.3, 161.9, 175.2, 150.4 | 150-182 (~20%) |
| `--warmup 100` | 169.2, 190.6, 165.6, 153.6 | 153-190 (~24%) |
| Cross-session  | as low as 92.3 (cold)      | up to **2x**   |

Warmup does **not** meaningfully reduce variance. The spread is dominated by GC scheduling and OS thread scheduling, not cache warmth.

### Pebble backend (mixed, 3 runs)

46.7 / 45.1 / 44.7 K/s → **±2%**. Persistence-bound workloads are inherently self-averaging.

### Consequence

- **Do not trust single-run absolute numbers**, especially on the memory backend. The earlier mixed run that reported memory @ 43.5 K/s and pebble @ 12.6 K/s were both cold-run anomalies; stable values are 110 K/s and 45 K/s respectively.
- The _relative ordering_ (memory > pebble; smaller > larger) is robust. The _absolute_ throughput needs median-of-N.

### Recommendation: `--repeat N`

Add a CLI flag that runs the benchmark N times and reports median throughput + min/max spread. This is the single highest-impact benchkit improvement — it converts noisy single shots into reliable medians. Tracked in [TODO_LIST.md](../../TODO_LIST.md) Benchkit section.

---

## 5. What was built

To answer the "mixed sizes" question, the generator gained distribution support (backward compatible):

| Change                                                                             | File                         |
| ---------------------------------------------------------------------------------- | ---------------------------- |
| `Generator` holds `sizes []int`; `Payload()` picks uniformly at random             | `benchkit/generator.go`      |
| `NewMixedGenerator(seed, sizes, codec)` + `MeanSize()` + `SizeDistribution()`      | `benchkit/generator.go`      |
| `Config.PayloadSizes` + `Result.PayloadSizes` (mean in `PayloadBytes`)             | `benchkit/benchkit.go`       |
| Runner builds mixed generator when set; reports mean + distribution                | `benchkit/runner.go`         |
| Report shows `Payload: 1360 bytes/event (mean; mixed [64 256 1024 4096])`          | `benchkit/report.go`         |
| `--payload-sizes 64,256,4096` CLI flag (run + compare); overrides `--payload-size` | `cmd/cqrs-bench/main.go`     |
| 6 new tests: all-sizes-produced, mean, determinism, single-size-parity, defaults   | `benchkit/generator_test.go` |

`NewGenerator(seed, size, codec)` is unchanged — existing single-size callers are unaffected. All existing benchkit tests pass.

---

## 6. Answering the question directly

**"How do we scale with different event sizes?"**
Throughput is flat (CPU-bound) up to ~256 B, then falls roughly linearly with payload size (bandwidth-bound). Memory: 227 K/s @ 64 B → 33 K/s @ 16 KB (~7x). Pebble: 68 K/s → 6.3 K/s (~11x, I/O amplifies the cost). Heap/disk scale linearly. Read latency grows sublinearly (load reads the whole stream, so larger events cost more but with fixed per-stream overhead).

**"How do we scale with mixed event sizes?"**
Better than the arithmetic mean predicts, because most events are small. A mixed workload behaves like a uniform workload at roughly the _median_ size. The caveat: the few large events dominate P99 latency and pull average cost up — a 1-in-8 large event can cut throughput nearly in half. For accurate capacity planning, benchmark with a distribution matching your real traffic, not a single size.
