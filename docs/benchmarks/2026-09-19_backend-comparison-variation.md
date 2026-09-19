# Backend Comparison with Cross-Run Variation — 2026-09-19

> **PROVENANCE — READ BEFORE CITING NUMBERS.** Captured on the shared dev
> host while it was OVERSUBSCRIBED (load average ~34-40 on 32 CPUs,
> `uptime` at capture start: `15:02, load 34.39 38.78 48.89`). That violates
> the quiet-machine rule this capture format exists to enforce: the
> `Variation` section below flags 24-30 of ~34 metrics NOISY per backend,
> including the write_throughput headline on memory (CoV 11.7%).
>
> **Per-backend deltas in this capture are NOT decision-grade.** The file
> exists to (a) replace the pre-variation 2026-07-31 format example and
> (b) show what a noisy capture looks like — the Variation section naming
> and ranking every noisy metric IS the deliverable.
>
> Supersede on a quiet window: `scripts/calibration-gate.sh` PASS, then
>
>     cqrs-bench compare --profile small --repeat 5 --format markdown \
>       --quiet --output docs/benchmarks/<date>_backend-comparison-variation.md
>
> Command: `cqrs-bench compare --profile small --repeat 5 --format markdown`.
> Backends: memory, pebble, sqlite (optimized pragmas). Machine: AMD Ryzen AI
> MAX+ 395 (32 CPUs), Go recorded per-run in each result.

 Backend | Write P50 | Write P99 | Load P50 | Load P99 | Cold P50 | GC Max Pause | Tail Ratio | Allocs/Op | Write Amp | CoV % | Noisy | RAM    | Heap    | Disk   |
|--------|----------|----------|---------|---------|---------|-------------|-----------|----------|----------|------|------|-------|--------|-------|
| memory  | 550ns     | 2.6µs    | 750ns    | 3.4µs   | 900ns    | 280.5µs     | 4.5x       | 132       | -         | 11.7% | 29    | 21 MiB | 78 MiB  | 0 B    |
| pebble  | 4.98ms    | 51.598ms  | 44.3µs  | 193.7µs | 43.5µs  | 967.5µs     | 4.4x       | 30927     | 25.6x     | 33.5% | 30    | 0 B    | 152 MiB | 63 MiB |
| sqlite  | 256.1µs  | 12.446ms  | 200.1µs | 950µs   | 199.7µs | 760.3µs     | 4.7x       | 1216      | 40.4x     | 40.5% | 24    | 0 B    | 152 MiB | 99 MiB |

Variation (CoV >= 10% = noisy, not decision-grade):
  memory     5/34 stable  noisy: gc_max_pause_ns (144.4%), gc_total_pause_ns (143.2%), gc_percent (140.4%), query_miss_p99_ns (115.5%), heap_bytes (57.0%), query_hit_p99_ns (55.6%), load_max_ns (54.6%), journey_projection_p99_ns (46.2%), journey_query_p99_ns (45.4%), journey_p99_ns (43.6%), snapshot_cold_p99_ns (42.1%), rawsink_p99_ns (41.1%), cache_hit_p99_ns (38.3%), query_paginated_p99_ns (33.5%), query_hit_p50_ns (31.7%), load_p99_ns (26.4%), cold_read_p99_ns (23.5%), snapshot_cold_p50_ns (22.2%), write_p99_ns (21.1%), write_max_ns (19.4%), cache_miss_p99_ns (19.1%), write_tail_ratio (18.6%), tail_ratio (18.4%), snapshot_load_p99_ns (18.3%), journey_p50_ns (18.3%), load_p50_ns (14.4%), rawsink_throughput (14.3%), rawsink_p50_ns (12.7%), write_throughput (11.7%)
  pebble     5/35 stable  noisy: journey_projection_p99_ns (117.1%), snapshot_cold_p99_ns (105.4%), journey_p99_ns (90.9%), write_tail_ratio (90.5%), write_p99_ns (89.2%), gc_total_pause_ns (78.3%), gc_max_pause_ns (76.8%), cache_hit_p99_ns (64.9%), gc_percent (62.0%), query_paginated_p99_ns (61.0%), cold_read_p99_ns (59.7%), allocs_per_op (56.5%), alloc_count (56.5%), bytes_per_op (56.4%), rawsink_p99_ns (54.9%), journey_query_p99_ns (47.2%), rawsink_throughput (36.6%), snapshot_cold_p50_ns (35.1%), write_throughput (33.5%), query_hit_p50_ns (32.5%), write_amplification (32.0%), heap_bytes (31.8%), write_max_ns (31.4%), cache_miss_p99_ns (31.4%), snapshot_load_p99_ns (27.5%), load_max_ns (24.4%), load_p99_ns (20.2%), tail_ratio (17.3%), query_miss_p99_ns (16.9%), query_hit_p99_ns (12.6%)
  sqlite     11/35 stable  noisy: write_max_ns (123.4%), cache_miss_p99_ns (91.7%), gc_max_pause_ns (64.1%), query_hit_p99_ns (57.5%), snapshot_load_p99_ns (55.2%), query_paginated_p99_ns (52.3%), write_amplification (44.7%), write_p99_ns (42.4%), journey_query_p99_ns (41.9%), snapshot_cold_p99_ns (40.6%), heap_bytes (40.5%), write_throughput (40.5%), write_tail_ratio (36.5%), gc_percent (35.0%), rawsink_throughput (33.3%), query_miss_p99_ns (31.9%), query_hit_p50_ns (26.0%), bytes_per_op (16.1%), allocs_per_op (15.8%), alloc_count (15.8%), gc_total_pause_ns (15.7%), cache_hit_p99_ns (13.9%), journey_p99_ns (12.3%), load_max_ns (10.4%)

