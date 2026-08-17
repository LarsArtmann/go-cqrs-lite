# Vector Search at Scale — Quantization / HNSW Spike

**Date:** 2026-08-16
**Status:** Spike complete; Phase 0 (binary float32 encoding) implemented 2026-08-17 — see §4 Option A and §7 Q1. Phases 1-2 remain proposed.
**TODO item:** "Vector search at scale — quantization/HNSW spike for LSM engines when collections exceed ~100K vectors (brute-force scan is O(N))"

## 1. Problem

Every LSM-family engine (pebble, bbolt, badger) serves `VectorSearch` as a brute-force prefix scan: read every embedding in the collection, compute the distance in Go, keep top-k. The profile honestly declares `ADTVector = ComplexityON` + `DegradedADTs[ADTVector] = true`. At small N this is fine; at ~100K+ vectors per-query latency crosses interactive budgets.

Filtered k-NN (`VectorSearchFiltered`, shipped same day) is also linear — it filters before scoring, so excluded vectors skip the distance math, but the scan itself is still O(N) I/O.

## 2. Measured baseline (2026-08-16, 32-core host, Go 1.26.5, D=128, k=10, cosine, random uniform vectors)

| Collection size | MemoryVectorIndex (in-RAM ceiling) | Pebble (LSM, JSON payloads)                                                                           |
| --------------- | ---------------------------------- | ----------------------------------------------------------------------------------------------------- |
| 1K              | 90.8 µs/query                      | 15.9 ms/query                                                                                         |
| 10K             | 910.6 µs/query                     | 172.2 ms/query                                                                                        |
| 100K            | 9.57 ms/query                      | ~1.7 s/query (extrapolated; insert-path fsync makes a 100K setup pass impractical in a bench harness) |

Per-vector cost: **~90 ns** (memory) vs **~17 µs** (pebble) — a ~190x constant-factor gap.

After Phase 0 shipped (2026-08-17, binary payloads, same host/params):

| Collection size | MemoryVectorIndex (in-RAM ceiling) | Pebble (LSM, binary payloads) |
| --------------- | ---------------------------------- | ------------------------------ |
| 1K              | 79.8 µs/query                      | 459.9 µs/query                 |
| 10K             | 825.9 µs/query                     | 5.63 ms/query                  |

Per-vector cost drops from ~17 µs to **~460-560 ns** (~31-35x); the LSM scan
sits within ~6x of the in-RAM ceiling instead of ~190x. The remaining gap is
scan I/O plus the fixed-width decode, not text parsing.

Follow-up (2026-08-17, same host/params; bench twins added for the other two
LSM engines):

| Collection size | MemoryVectorIndex | Pebble   | bbolt    | badger   |
| --------------- | ----------------- | -------- | -------- | -------- |
| 1K              | 79.8 µs           | 457.4 µs | 425.7 µs | 646.7 µs |
| 10K             | 825.9 µs          | 5.23 ms  | 5.79 ms  | 5.85 ms  |

All three LSM engines land in the same ~430-650 ns/vector band — the win is
the shared format, not a pebble artifact. Filtered k-NN on pebble (half the
1K collection matching the filter): 1034.5 µs/query vs 457.4 µs unfiltered —
the per-row metadata read + filter evaluation roughly doubles the scan cost.
Codec-level micro-bench, pure decode cost per vector: binary 196 ns (D=128)
/ 1.81 µs (D=1536) with 1 alloc, vs JSON 8.51 µs / 110.1 µs with 8-13
allocs — 43-61x, the constant the engine numbers converge to once scan I/O
dominates.

Benchmarks live in the repo and can be re-run as things change:

- `metaengine/vector_scale_bench_test.go` — memory ceiling (1K/10K/100K)
- `metaengine/{pebble,bbolt,badger}engine/vector_bench_test.go` — LSM validation points (1K/10K; pebble adds a filtered 1K)
- `metaengine/vector_binary_bench_test.go` — codec micro-bench (decode/encode, binary vs JSON, D=128/1536)

Command: `GOWORK=off go test -tags "goexperiment.jsonv2" -bench 'VectorSearch' -benchtime=20x -run XXX .`

## 3. Where the time actually goes

The memory ceiling shows pure distance math at D=128 is ~90 ns/vector — 100K vectors cost ~10 ms of compute. Pebble's 190x overhead is NOT the scan or the distance math; it is **JSON decode of every vector on every query** (`DecodeVectorJSON` per row: ~128 float parse operations per vector per query).

Implication: an approximate index is NOT the first lever. The constant factor is.

## 4. Options analysis

### Option A — binary float32 payload encoding (not approximation, just stop using JSON)

Replace the JSON vector payload with fixed-width little-endian float32 bytes (`math.Float32frombits` / manual `encoding/binary` decode). Zero API change, zero recall impact, trivially CGo-free.

- Expected gain: decode becomes a bounds-checked memcpy — historically 10-30x cheaper than JSON float parsing. At 100K that takes the LSM path from ~1.7 s to roughly 100-170 ms (approaching the memory ceiling plus scan I/O).
- Cost: a key-family version marker so old JSON data still decodes (self-describing envelope precedent: `vec\x00` bytes can gain a 1-byte format tag, or a new `vecb\x00` family read-fallback ordered JSON-first).
- Risk: low. Reversible by re-encoding.

**Verdict: do this regardless of anything else.** It is the 80/20 of this whole spike.

**Implemented 2026-08-17** as `metaengine.EncodeVectorBinary` /
`DecodeVectorBinary` / `DecodeVectorAuto` (`metaengine/vector_binary.go`):
wire format `'b' | dim uint32 LE | dim × float32 LE`; pebble, bbolt, and
badger write binary and read through the sniffing decoder, so legacy JSON
rows keep working in place (mixed-format collections verified by per-engine
tests). Measured ~31-35x faster VectorSearch than the JSON baseline (§2).
pgengine stays JSON — its vector column is typed JSONB, so binary needs a
BYTEA DDL migration; deferred until the KV win proves insufficient there.

### Option B — scalar quantization (int8)

Store an int8 quantized copy alongside (or instead of) float32. Scan computes int8 dot products (pure Go: widen to int32, no SIMD dependency needed), then re-ranks the top ~2k candidates exactly from the float32 originals.

- Expected gain: 4x less scan I/O and cheaper arithmetic; combined with Option A the compute part drops further.
- Recall: bounded loss (typical recall@10 > 0.95 with per-dimension scale factors); the exact re-rank step hides most of it for top-k use.
- Filtered k-NN: composes perfectly — it is still a linear scan, filters still apply pre-score.
- Cost: per-dimension min/max bookkeeping (a small sidecar per collection, updatable on upsert), an encode/decode path, and a re-rank step.

**Verdict: the right Phase 1 if numbers keep growing after Option A.**

### Option C — product quantization (PQ)

16-32 byte codes per vector, distance via codebook lookups. Bigger compression than scalar (16-32x vs 4x), but: codebook training needs batch rebuilds, distance math is lookup-bound and slower per element than int8 arithmetic in pure Go, and recall tuning adds operational surface.

**Verdict: skip until int8 demonstrably insufficient. The complexity/return ratio is poor for a library that must stay boring to trust.**

### Option D — HNSW graph index in the LSM

Store the HNSW adjacency as keys (`vecn\x00<col>\x00<level>\x00<node>` → neighbor list), entry point in a meta key. Search = O(log N) seek + greedy graph walk. Pure-Go HNSW over a KV store is well-trodden.

- Expected gain: sublinear — 100K+ vectors queried in low single-digit ms even on LSM.
- **The filtered k-NN problem (critical):** HNSW with a pre-filter starves — the greedy walk can exhaust its candidate frontier inside filtered-out regions and return fewer than k results. Standard mitigations: (a) post-filter with over-fetch factor (k*10 or dynamic expansion), (b) hybrid switch to linear scan when filter selectivity is low, (c) labeled subgraphs per filter value (only viable for a small fixed set of equality filters). This is real design surface, not a bolt-on.
- Write amplification: every insert updates O(M log N) neighbor lists under a sync-per-write engine — needs batched/buffered inserts or an eventual-consistency index build.
- Staleness: a crash between vector write and index write leaves the graph inconsistent unless index entries are the source of truth (then search must fall back to scan for unindexed stragglers).

**Verdict: Phase 2, behind an optional capability interface, only after A+B prove insufficient — and it must ship with the filter-fallback strategy, not just the happy path.**

### Option E — route to a native ANN engine at deployment time

The metaengine north star: "developer declares, operator deploys." The planner already treats `ADTVector` as a routable ADT; an engine with native ANN (a future turso/sqlite-vec or a dedicated vector engine) declares `ComplexityOLogN` and wins routing automatically. `DegradedADTs` + `Doctor` already tell operators the LSM path is brute-force.

**Verdict: already architecturally supported; zero code today. Keep as the escape hatch that makes D optional.**

## 5. Recommended phasing

| Phase | Action                                                                | Trigger                                                                                                                                             |
| ----- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0     | ✅ Done 2026-08-17: binary float32 encoding for vector payloads (pebble, bbolt, badger; legacy JSON rows still readable) | Shipped — ~31-35x measured scan win |
| 1     | int8 scalar quantization + exact re-rank                              | Post-Phase-0 p99 still above budget at real N                                                                                                       |
| 2     | Optional ANN capability (HNSW or IVF) with filter-aware fallback      | Sustained collections > ~500K or latency-critical vector queries                                                                                    |
| -     | Size-triggered advisory                                               | Cheap to add anytime: a `VectorCount` optional capability lets `Doctor`/`EXPLAIN` say "collection X has N vectors on a degraded engine" with a WARN |

## 6. Non-goals / explicitly deferred

- GPU or SIMD assembly kernels (cgo-free constraint rules out most; revisit if a pure-Go SIMD package matures).
- Distributed/sharded vector search (single-engine scope).
- PQ (Option C) unless int8 fails.

## 7. Open questions for implementation time

1. ~~Phase 0 format marker~~ **Answered 2026-08-17:** value-level 1-byte marker
   (`'b'`) plus a uint32 LE dimension header on the existing `vec\x00` payloads —
   not a new key family. One key family means one prefix scan (no dual-family
   reads or migration backfill), a JSON text can never start with `'b'` so the
   sniff is unambiguous, and the dim header makes torn writes detectable. The
   read path prefers the exact binary decode and falls back to JSON.
2. Does Phase 1's quantized copy live in the same key (value = header + int8 + float32) or a sidecar family? Same key halves the seeks but doubles the write width.
3. For Phase 2, is the HNSW build synchronous-on-insert (simple, slow writes) or background-compaction style (fast writes, eventual consistency)? The event-sourced replay path (`Apply` of historical events) favors bulk-build.
