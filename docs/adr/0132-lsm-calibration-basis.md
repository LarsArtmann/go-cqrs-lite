# ADR-0132: LSM Storage Calibration Measures Post-Flush, Pre-Compact

- **Status:** Accepted (implemented; this ADR documents the basis decision)
- **Date:** 2026-08-29
- **Context:** plan V3 T40; deep-review question "Pebble calibration basis:
  flush-only vs post-Compact ratio" (status 2026-08-16_14-44).

## Context

`metaengine/bench`'s disk layout calibration
(`BenchmarkDiskLayoutCalibration_Storage`) compares normalized vs embedded
storage layouts on real LSM engines (Pebble, bbolt). The measured ratio feeds
`StorageSpace` cost cells that the layout planner scores — so the basis must
be deterministic, or plans flip between runs.

An LSM engine's on-disk footprint changes over time even with no writes:
memtables hold unflushed data, and background compaction continuously
rewrites SSTables. Closing the engine is not enough — Pebble's `Close` does
NOT flush the memtable, leaving data in the WAL where SSTable compression
never applies.

## Decision

The calibration basis is **post-Flush, pre-Compact, with identical input**:

1. Both layouts receive the same seeded writes.
2. The bench calls `db.Flush()` explicitly before reading `FileSize()`
   (`Close` alone would measure a WAL-resident footprint; the bench comments
   document this).
3. No manual compaction is requested, and ambient background compaction is
   why calibration runs claim `StorageSpace` cells only from quiet-window
   runs recorded in `docs/BENCHMARKS.md`.

## Rationale

- **Flush-only is deterministic**: identical seeds + flush → identical
  SSTable content, so planner comparisons are reproducible.
- **Post-Compact is realistic but unstable**: compaction timing depends on
  ambient load and level thresholds; ratios drift run-to-run, which would
  make plan selection flaky.
- The known bias — flush-only slightly overstates pre-compaction size for
  compressed blocks — is bounded and noted next to the recorded cells.

## Consequences

- The per-child key overhead measured this way is **43–46 bytes**
  (earlier notes said ~41; the measured value corrected 2026-08-29).
- Any engine adding an LSM backend should implement the same
  flush-then-measure pattern; `enginetest`'s storage-calibration helper is the
  shared entry point.
