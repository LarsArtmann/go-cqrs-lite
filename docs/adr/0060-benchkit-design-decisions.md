# ADR-0060: Benchkit Design Decisions

**Status:** Accepted
**Date:** 2026-07-24

## Context

The `benchkit` module provides synthetic benchmark workloads for go-cqrs-lite
backends. It was built over multiple sessions with several non-obvious design
decisions that needed codification. This ADR records the five key decisions and
their rationale so future contributors don't accidentally revert them.

## Decisions

### 1. Codec-Aware Payload Padding

`NewGenerator(seed, size, codec)` accepts the codec as a constructor argument.
The generator generates payload data, then encodes it with the chosen codec to
measure the actual encoded size. This ensures `PayloadSize` reflects real wire
bytes, not arbitrary data that would be a different size after encoding.

**Alternative rejected:** Post-hoc size estimation (`estimateJSONSize`).
This was fragile and produced inaccurate results for CBOR.

### 2. Warmup Isolation via Separate Bundle

Warmup uses a **separate Bundle** (the factory is called a second time) so
warmup events never pollute the measurement store's journal or metrics. This
is verified by `TestRun_Memory` which checks the factory is called exactly
twice when `Warmup > 0`.

**Alternative rejected:** Warmup in the same store, then clear. This risks
leftover state and conflates warmup pollution with measurement.

### 3. ReadRatio-as-Passes

`ReadRatio` is translated into a number of read passes via `readPassesFor(ratio)`:
`ratio * 10`, clamped to `[1, 10]`. A ratio of 0.3 means 3 passes, 0.8 means 8
passes. Each pass loads every stream once. This makes read workload proportional
to write count rather than independent.

**Alternative rejected:** Fixed read count. A fixed count decouples reads from
writes, making it impossible to express "read 80% as much as I write."

### 4. SkipPhases for Targeted Benchmarking

`Config.SkipReads`, `SkipReadModels`, `SkipProjections` allow skipping specific
phases. This enables targeted benchmarks (e.g., write-only, read-only) without
creating separate profiles or runner code paths.

### 5. DiskSizer Interface with -1 Sentinel

`DiskSizer.DiskSize()` returns `-1` when no disk-size reporter is registered
(e.g., memory backend, SQLite without `WithDiskSize`). The runner checks for
`>= 0` before using the value, falling back to `Config.DiskPath` filesystem walk.

This avoids a nil-interface check (since `*stack.Bundle` always implements
`DiskSizer` via the concrete `DiskSize()` method) while still signaling
"not available" cleanly.

**Alternative rejected:** Separate `HasDiskSize()` method. More API surface
for no benefit; the sentinel pattern is idiomatic Go (cf. `syscall`, `time`).

## Consequences

- The benchmark accurately measures what it claims to measure (real encoded sizes, no warmup pollution).
- Read workload is proportional to writes, not independent.
- Disk measurement works across all backends (DiskSizer for Pebble, filesystem walk fallback for others).
- The `--codec` flag affects payload sizing, not just encoding speed.
