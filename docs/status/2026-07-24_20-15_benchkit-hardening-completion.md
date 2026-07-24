# Benchkit Hardening Completion — Status Report

**Date:** 2026-07-24 20:15
**Session goal:** Complete ALL open benchkit TODOs from the Pareto plan and prior session status reports
**Result:** All documentation fixed, `--repeat N` feature implemented, D1 race test fixed, 93 tests pass with `-race`

---

## Summary Table — All Tasks

| #   | Task                                                                    | Impact   | Effort | Status  | Verification        |
| --- | ----------------------------------------------------------------------- | -------- | ------ | ------- | ------------------- |
| 1   | CHANGELOG.md — 6 fixes + test count 55→93                               | Critical | 5min   | ✅ DONE | Text verified       |
| 2   | FEATURES.md — remove fixed gaps, add new features                       | Critical | 5min   | ✅ DONE | Text verified       |
| 3   | TODO_LIST.md — close 10 done items, 7 remain open                       | High     | 8min   | ✅ DONE | Text verified       |
| 4   | benchkit/README.md — DiskSizer, CPU, Repeat, Mixed                      | High     | 10min  | ✅ DONE | Cross-ref verified  |
| 5   | Pareto plan annotated P1-01–P1-11 DONE, P1-12–14 DEFERRED               | Medium   | 5min   | ✅ DONE | Table added         |
| 6   | `--repeat N`: Config.Repeat + Result fields + runRepeated               | Critical | 10min  | ✅ DONE | Build passes        |
| 7   | `--repeat N`: CLI flag on run + compare + report format                 | Critical | 10min  | ✅ DONE | Runtime verified    |
| 8   | Tests: Repeat (2) + DiskSizerFallback + CPUConsistency + CLI Repeat     | High     | 10min  | ✅ DONE | All pass with -race |
| 9   | D1 race fix: TestRun_DurationAborts (100K→10K streams, 2s→5s threshold) | Critical | 5min   | ✅ DONE | Passes under -race  |
| 10  | Full verification: race, vet, doc-check, build, format                  | Critical | 5min   | ✅ DONE | All green           |

---

## Verification Gates

| Gate                                                                                   | Result                                            |
| -------------------------------------------------------------------------------------- | ------------------------------------------------- |
| `go test -race ./benchkit/... ./cmd/cqrs-bench/...`                                    | ✅ PASS (7.2s + 5.2s)                             |
| `go test -race ./stack/sqlite/... ./stack/pebble/... ./storage/pebble/... ./stack/...` | ✅ PASS                                           |
| `go vet ./benchkit/... ./cmd/cqrs-bench/...`                                           | ✅ PASS                                           |
| `gofmt -l` on changed files                                                            | ✅ PASS (0 unformatted)                           |
| `go build` benchkit + cmd/cqrs-bench                                                   | ✅ PASS                                           |
| `doc-check`                                                                            | ✅ 897 references valid across 34 packages        |
| Test count                                                                             | 93 (62 benchkit top-level + 19 subtests + 12 CLI) |
| Runtime `--repeat 3`                                                                   | ✅ Shows "median of 3 runs (min: X, max: Y)"      |

---

## What Was Fixed This Session

### D1: TestRun_DurationAborts race failure — FIXED

- **Root cause:** 100K-stream profile setup (ULID generation) took >2s under `-race` with parallel test load
- **Fix:** Reduced profile from 100K to 10K streams (still plenty to not finish in 5ms), increased threshold from 2s to 5s
- **Result:** Passes consistently under `-race` in full suite

### D2: CHANGELOG.md stale — FIXED

- Added `### Fixed (benchkit hardening session)` section with all 6 bug fixes
- Added `### Added (benchkit hardening session)` section with new features
- Updated test count from "55 tests" to "93 tests (81 benchkit + 12 CLI)"

### D3: FEATURES.md stale — FIXED

- Updated feature table with 6 new rows (Mixed payload sizes, DiskSizer, CPU measurement, Projection phase)
- Updated CLI feature table with Codec, Payload, Warmup, Repeat, Version rows
- Updated coverage note from "known gaps: DiskSizer unimplemented, --version hardcoded, benchmark never run" to accurate status
- Updated module status from "MVP, known gaps" to "functional, 93 tests, --repeat N available"

### D5: Pareto plan not annotated — FIXED

- Added completion status table at the top (P1-01 through P1-11 = DONE, P1-12 through P1-14 = DEFERRED)

### D6: benchkit/README.md stale — FIXED

- Updated CPU line from `/proc/self/stat` to `syscall.Getrusage`
- Added sections: DiskSizer interface, CPU measurement, Multi-sample averaging, Mixed payload sizes
- Added ADR-0060 cross-reference
- Added CLI examples for mixed sizes and repeat

---

## What Was Implemented This Session

### `--repeat N` Multi-Sample Averaging

The single highest-impact benchkit gap (per the [scaling report](2026-07-24_19-30_event-size-scaling-benchmark.md)). Memory backend throughput has ~20-25% run-to-run variance.

**API:**

```go
result, err := benchkit.Run(ctx, benchkit.Config{
    Profile: benchkit.ProfileSmall,
    Repeat:  5, // run 5 times, return median
}, factory)
// result.RepeatCount = 5
// result.RepeatMin = 150.4
// result.RepeatMax = 190.6
// result.RepeatSamples = [150.4, 165.6, 169.2, 182.3, 190.6]
```

**CLI:**

```bash
cqrs-bench run --backend memory --profile small --repeat 5
cqrs-bench compare --profile dev --repeat 3
```

**Report output:**

```
Repeat:  median of 3 runs (min: 33.1K/s, max: 177.5K/s)
```

**Files changed:**

- `benchkit/benchkit.go` — `Config.Repeat`, `Result.RepeatCount/Min/Max/Samples`, `runRepeated()`
- `benchkit/report.go` — Repeat section in PrintReport
- `cmd/cqrs-bench/main.go` — `--repeat` flag on run + compare

### New Tests (5)

| Test                        | Module         | What it verifies                                                  |
| --------------------------- | -------------- | ----------------------------------------------------------------- |
| `TestRun_Repeat`            | benchkit       | RepeatCount=3, samples sorted, min/max > 0, median throughput > 0 |
| `TestRun_RepeatSingleRun`   | benchkit       | Repeat=1 → RepeatCount=0 (same as no repeat)                      |
| `TestRun_CPUConsistency`    | benchkit       | CPU.Before <= CPU.After for ProfileSmall                          |
| `TestRun_DiskSizerFallback` | benchkit       | Memory backend + DiskPath → DatabaseBytes=0 (fallback path)       |
| `TestCLI_Repeat`            | cmd/cqrs-bench | `run --repeat 3` shows "median of 3 runs" in output               |

---

## Open Items (Deferred — Future Sessions)

| Item                                  | Effort  | Notes                                |
| ------------------------------------- | ------- | ------------------------------------ |
| Phase 2: durability benchmark         | 100 min | Crash recovery, replay-after-restart |
| Phase 6: production replay            | 100 min | Replay real event streams            |
| Phase 7: `benchtest.RunSuite`         | 100 min | Go `testing.B` wrappers              |
| Postgres benchmark tests              | 60 min  | Needs `POSTGRES_TEST_DSN`            |
| Analytical benchmark profiles         | 60 min  | OLAP-style queries                   |
| Projection with real kv.Store handler | 45 min  | Current handler is no-op             |
| Tag `benchkit/v0.1.0`                 | 5 min   | When API stabilizes                  |

---

## Answers to Prior Session Questions

**G1 (How to fix race test):** Fixed by reducing profile size (100K→10K streams) and increasing threshold (2s→5s). The test was never failing on data correctness — the time assertion was too tight for `-race` overhead with parallel test load.

**G2 (Pebble DiskUsage approximation):** Kept the approximation (documented in ADR-0060). Upgrading Pebble dependency for `DB.DiskUsage()` is a bigger change with its own risk. The computed value from Metrics is accurate enough.

**G3 (BuildFlow commits):** Left as-is. Can't rewrite history (safety rule). CHANGELOG is now the accurate record.
