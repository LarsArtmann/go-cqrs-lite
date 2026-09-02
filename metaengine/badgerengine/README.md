# metaengine/badgerengine — BadgerDB-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4)

BadgerDB-backed [metaengine](../README.md) Engine. Pure Go. An LSM key-value
engine with native multimap and log-ADT support: keyspace-prefix-encoded
storage gives O(1) point reads and appends, with counter and scan paths
degraded to prefix scans (no secondary indexes).

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"

engine, err := badgerengine.NewBadgerEngine("path/to/dir")
```

`NewBadgerEngineFromDB` wraps an existing `*badger.DB` (ownership transfers:
`Close` closes it).

## Cost Profile

### Calibrated per-pattern read costs (2026-08-30)

Measured by `calibration_bench_test.go` (medians of 3; baseline:
`docs/benchmarks/calibration-2026-08-30.md`). The planner prices each query
by its read pattern:

| Pattern       | Cost            | Bench                                                                                                       |
| ------------- | --------------- | ----------------------------------------------------------------------------------------------------------- |
| Point lookup  | ~1,100 ns/query | `BenchmarkCalibration_BadgerGet` (LSM point lookup + JSON decode)                                           |
| Filtered scan | ~650 ns/row     | `BenchmarkCalibration_Badger_FilteredScan` (full scan + Go-side predicate — no SQL pushdown on a KV engine) |
| Aggregate     | ~165 ns/row     | `BenchmarkCalibration_Badger_CounterScan` (`CounterGet` prefix scan — the `ReadAggregate` path, ADR-0133)   |
| Full scan     | ~630 ns/row     | `BenchmarkCalibration_Badger_FullScan`                                                                      |

## Backends

MapBackend, MapUpdater, SetBackend, CounterBackend, MultimapBackend,
ScanBackend, LogBackend, StreamLogBackend, SeqSeekableStreamLog,
AtomicAppender, plus Calibratable/TrackerHost for the live-latency model.

- **Multimap + Log ADTs are first-class**: encoded child keys make appends and
  per-key iteration native, unlike pure-LSM encodings that re-scan a
  collection prefix.
- **Counter/scan paths are prefix scans** (degraded vs. SQLite) — declare
  counters with pushdown-capable engines if those collections dominate.

## Notes

- Health: opens a read-only transaction round-trip.
- Conforms to the shared `metaengine/adttest` and `enginetest` harnesses; the
  badger/bbolt `StreamLog` tail similarity is an intentional
  `//art-dupl:accept` (dep-isolated engines, same contract).
