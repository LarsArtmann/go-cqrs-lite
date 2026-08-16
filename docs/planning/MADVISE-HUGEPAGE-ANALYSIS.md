# madvise(MADV_HUGEPAGE) — Analysis

> Date: 2026-08-16
> Question: Could this repo benefit from `madvise(MADV_HUGEPAGE)`? If so, where, how, and why?
> Status: **Analysis only — no code change recommended.** Deployment/benchmark lever, not a library feature.

## TL;DR

**Not as code in this library — but as a deployment/benchmark lever for the big embedded-store workloads, yes, worth an experiment.**

- The repo owns **zero mmaps**; there is nothing for a `MADV_HUGEPAGE` call to attach to.
- The Go 1.26.5 runtime never opts its heap arenas into THP, and we cannot change that from library code.
- Hot paths are **decode-CPU-bound** (JSON/CBOR deserialize), not TLB-bound — THP does not speed up `json.Unmarshal`.
- Flipping process-global VM policy from inside a library violates the "operators decide at deployment time" invariant.

## Verified ground truth (as of this analysis)

| Claim | Status | Evidence |
| --- | --- | --- |
| No `madvise`/THP usage anywhere in this repo | Verified | `rg "madvise|MADV_|hugepage"` → no matches |
| Go 1.26.5 runtime defines but never applies `MADV_HUGEPAGE` | Verified | `runtime/defs_linux_*.go` constants only; no call sites. Heap stays 4KB-backed. See golang/go#8832 (open) |
| Pebble v1.1.5 makes no madvise calls | Verified | `rg MADV` over module cache source → no hits |
| bbolt v1.5.0 madvises its mmap with `MADV_RANDOM` only | Verified | `bolt_unix.go:63` (`unix.Madvise(b, syscall.MADV_RANDOM)`) |
| bbolt's mapping pointer is unexported | Verified | Our `storage/bbolt` wrapper cannot reach it directly |
| `golang.org/x/sys` already available (indirect) in `storage/bbolt` | Verified | `storage/bbolt/go.mod` |

## Why it would help (mechanism)

- **TLB reach**: ~2K dTLB entries cover ~8MB at 4KB pages vs ~4GB at 2MB pages. Pointer-chasing over hundreds of MB is exactly the TLB-miss-shaped workload.
- **512× fewer soft faults** on first touch of large buffers.
- **~512× less page-table memory** per GB touched.

Candidate in-repo workloads with that access shape:

- In-memory projections during 10M-event soak / catch-up replays
- `graph/` MemoryDriver
- `storage/memory` LogStore
- Pebble block cache / memtables

## Why it mostly doesn't apply here

1. **No attach point.** We own no mmap anywhere in the repo; the Go runtime never opts its heap arenas into THP (open since golang/go#8832; verified absent in our toolchain).
2. **Hot paths are decode-CPU-bound**, not TLB-bound. THP doesn't accelerate `json.Unmarshal` / CBOR decode, which dominate our profiles.
3. **This is a library.** Flipping process-global VM policy from inside consumer processes violates the "operators decide at deployment time" invariant.
4. **Well-documented downsides**: RSS bloat (2MB granularity per region) and khugepaged compaction stalls causing p99 spikes — the reason Redis and CockroachDB tell you to avoid THP `always` for latency-sensitive databases. Go heaps with GC churn (`MADV_DONTNEED` → khugepaged re-collapse) fight each other particularly badly.

## Where/how it could plausibly pay off

| Target | How | Honest odds |
| --- | --- | --- |
| Go heap (in-memory projections, graph drivers, Pebble memtables) | System mode `always` on a **dedicated bench/soak box** — the only lever that touches the heap | The one good case: large, stable, long-lived heaps during catch-up/replay. Watch RSS + p99, not just ops/s |
| bbolt large-DB scans (`ReadStreamFrom` over the journal mmap) | We know the DB path → parse `/proc/self/maps`, find that file's VMA, `unix.Madvise(range, unix.MADV_HUGEPAGE)`. Composes fine with bbolt's own `MADV_RANDOM` (readahead vs page granularity are independent advice) | Most credible in-repo win, but it's **file-backed THP**: needs kernel 6.x ext4/XFS large-folio support, or tmpfs (which is what `t.TempDir()` on `/tmp` gives you — soak tests would benefit) |
| SQLite engine reads | We own pragmas in `metaengine/sqliteengine`: `PRAGMA mmap_size` + system `always` mode | Indirect; same file-backed caveat |
| DuckDB (CGo buffer manager) | Ops-level only | Not ours to touch |

Precondition for any experiment: `/sys/kernel/mm/transparent_hugepage/enabled` must include `madvise` — on `madvise`-less systems the flag silently does nothing.

## Recommendation

1. **Do not ship madvise code in the library modules.** The decode paths dominate and the library-shape argument is decisive.
2. If pursued, two bounded follow-ups:
   - A deployment-tuning note for operators in the skill references (`.agents/skills/go-cqrs-lite/references/`).
   - A Linux-only, bench-gated opt-in in `testutil`/`benchkit` (smaps-walk + madvise) to A/B THP on the soak and `stack/bench` suites with `benchstat` — decided by data, not vibes.

Any experiment must measure with `-count=10` + benchstat (per the benchmark methodology in the how-to-golang performance-tuning reference) and watch RSS and p99 alongside throughput.

## Experiment results (2026-08-16)

Standalone A/B harness (kept out of the repo at `~/thpbench/`): builds the workload in-process, walks `/proc/self/maps`/`smaps`, flips the target VMAs to the requested THP state (`off` = `MADV_NOHUGEPAGE`, `on` = `MADV_HUGEPAGE` + `MADV_COLLAPSE`), verifies via smaps that hugepages actually materialized, then times 5 iterations per pattern. Arms run in **separate processes, alternating off/on across 5 rounds** (MADV_NOHUGEPAGE cannot split existing PMDs) to control for machine noise.

### Environment

- Kernel 7.1.8, 32 cores, 93GB RAM (load avg 20-30 during runs — medians + alternation handle drift)
- THP: `enabled=[madvise]`, `shmem_enabled=[never]` → tmpfs ruled out; root fs is **btrfs**
- zram swap (28.2G zstd) present; `VmSwap=0` in every measured process → no contamination
- Both arms fully RAM-resident (page cache / heap); disk never measured — intentional: THP only changes TLB/page-table costs

### Arm 1: Go heap (in-memory projection / graph-driver shape) — **+11-12%**

1.5M records × 1KB payloads (~1.6GB stable heap, GC settled before measurement). `on` collapsed 1000MB of 1604MB eligible (62%; `AnonHugePages=1000` verified, off-arm verified 0).

| Pattern | off (median) | on (median) | delta |
| --- | --- | --- | --- |
| random chase (1.5M shuffled touches) | 29.52 ms | 25.95 ms | **−12.1%** |
| sequential sweep | 11.12 ms | 9.86 ms | **−11.3%** |

All 5 `off` rounds slower than all 5 `on` rounds in both patterns (nonparametric p ≈ 0.008). Sequential benefits too — cheaper page walks, not just TLB coverage. Effect is if anything *diluted* by the 38% of the heap that would not collapse.

### Arm 2: bbolt DB mmap on btrfs — **untestable: THP unavailable for file VMAs**

2.9GB bbolt DB (1.5M keys × 1KB), full VMA matched (3072MB). `MADV_COLLAPSE` failed `EINVAL` on **all 1536** aligned 2MB extents; smaps shows `THPeligible=0` for the VMA, `FilePmdMapped=0`. btrfs on this kernel does not support PMD-mapped file folios for this mapping, so the "most credible in-repo win" cannot exist on this filesystem — the analysis caveat ("needs kernel/fs large-folio support") is the operative constraint, now empirically confirmed. Reference numbers (no THP possible): seq scan 16-18ms, 300K random Seeks 390-443ms.

### Verdict (updated)

- The **Go-heap case is real and repeatable**: ~12% on hot random access over a ~1.6GB stable heap. That maps to the deployment-time `always`-mode-on-a-bench-box recommendation for large in-memory projections/catch-up replays (graph MemoryDriver, storage/memory LogStore, soak replays).
- The **bbolt-mmap idea is dead on btrfs**; it would only matter on filesystems/kernels with PMD-order file folio support (and tmpfs only where `shmem_enabled` permits).
- Library recommendation unchanged: no madvise in module code. The winning lever is the deployment knob (`transparent_hugepage=madvise` + workload opt-in, or `always` on a dedicated box), now backed by a measured ~11-12% rather than theory.
