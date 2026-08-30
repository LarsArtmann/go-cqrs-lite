# metaengine/bboltengine — Bbolt-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4)

Bbolt-backed [metaengine](../README.md) Engine. Pure Go, single-writer B+tree —
point reads and ordered scans are O(log N). No CGo, no external server; the
tradeoff is that writes serialize on one file lock.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"

// Persistent (disk-backed)
engine, err := bboltengine.NewBboltEngine("/data/myapp.db")

// Wrap an existing *bolt.DB (caller owns lifecycle)
engine, err := bboltengine.NewBboltEngineFromDB(db)
```

## Cost Profile

| ADT       | Complexity                 | Notes                                              |
| --------- | -------------------------- | -------------------------------------------------- |
| Map/Set   | O(log N)                   | B+tree lookup                                      |
| Counter   | O(log N) incr, O(N) get    | Increment is a tx write; CounterGet is prefix scan |
| SortedMap | O(N)                       | Scan + Go-side sort                                |
| Log       | O(log N) append, O(N) tail | Append-only bucket; tail reads are prefix scans    |
| Multimap  | O(log N) add, O(N) get     | Prefix scan for retrieval                          |
| Vector    | O(N·D) search              | Brute-force scan (degraded, no ANN index)          |

**Unsupported ADTs:** Graph, Search, Spatial — the planner will not route
those queries here; declare a capable engine (e.g. `pgengine` for graph)
alongside bbolt if your app needs them.

### Calibrated per-pattern read costs (2026-08-30)

Measured by `calibration_bench_test.go` (medians of 3; baseline:
`docs/benchmarks/calibration-2026-08-30.md`). The planner prices each query
by its read pattern:

| Pattern | Cost | Bench |
| ------- | ---- | ----- |
| Point lookup | ~750 ns/query | `BenchmarkCalibration_BboltGet` |
| Filtered scan | ~620 ns/row | `BenchmarkCalibration_Bbolt_FilteredScan` (full scan + Go-side predicate — no SQL pushdown on a KV engine) |
| Aggregate | ~100 ns/row | `BenchmarkCalibration_Bbolt_CounterScan` (`CounterGet` prefix scan — the `ReadAggregate` path, ADR-0133) |
| Full scan | ~660 ns/row | `BenchmarkCalibration_Bbolt_FullScan` |

## Backends

MapBackend, ScanBackend, SetBackend, CounterBackend, MultimapBackend,
LogBackend, StreamLogBackend (`StreamAppend`/`StreamRead`/`StreamVersion`,
sequence-carrying journal reads), and VectorBackend (degraded brute-force).

## API

| Symbol                     | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `NewBboltEngine(path)`     | Opens/creates the DB file. Self-registers the driver. |
| `NewBboltEngineFromDB(db)` | Wraps an existing `*bolt.DB` (caller owns lifecycle). |
| `Profile()`                | Returns `EngineProfile` with the bbolt cost model.    |
| `HealthCheck(ctx)`         | Verifies the DB is usable.                            |
