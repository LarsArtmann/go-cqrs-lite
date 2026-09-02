# metaengine/pebbleengine — Pebble-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4)

Pebble-backed [metaengine](../README.md) Engine. Excels at LSM point reads — Map/Set lookups are O(1), significantly faster than SQLite's O(log N) B-tree for high-throughput key-value workloads.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"

// Persistent (disk-backed)
engine, err := pebbleengine.NewPebbleEngine("/data/myapp")

// In-memory (for tests)
engine, err := pebbleengine.NewPebbleEngine("")
```

## Cost Profile

| ADT      | Complexity             | Notes                                          |
| -------- | ---------------------- | ---------------------------------------------- |
| Map/Set  | O(1)                   | LSM point read — faster than SQLite B-tree     |
| Counter  | O(1) incr, O(N) get    | Increment is atomic; CounterGet is prefix scan |
| Multimap | O(1) add, O(N) get     | Prefix scan for retrieval                      |
| Log      | O(1) append, O(N) tail | Prefix scan for tail reads                     |
| Vector   | O(N·D) search          | Brute-force scan (degraded, no ANN index)      |

### Calibrated per-pattern read costs (2026-08-30)

Measured by `calibration_bench_test.go` (medians of 3; baseline:
`docs/benchmarks/calibration-2026-08-30.md`). The planner prices each query
by its read pattern:

| Pattern       | Cost          | Bench                                                                                                     |
| ------------- | ------------- | --------------------------------------------------------------------------------------------------------- |
| Point lookup  | ~700 ns/query | `BenchmarkCalibration_PebbleGet`                                                                          |
| Filtered scan | ~830 ns/row   | `BenchmarkCalibration_Pebble_FilteredScan` (`ScanRawValues` + Go-side filter — no SQL pushdown)           |
| Aggregate     | ~125 ns/row   | `BenchmarkCalibration_Pebble_CounterScan` (`CounterGet` prefix scan — the `ReadAggregate` path, ADR-0133) |
| Full scan     | ~700 ns/row   | `BenchmarkCalibration_Pebble_FullScan` (`ScanRawValues`, JSON decode per row)                             |

## Backends

MapBackend, ScanBackend, SetBackend, CounterBackend, MultimapBackend,
LogBackend, StreamLogBackend, and VectorBackend (degraded brute-force).
Search and Spatial are served via degraded brute-force scans. **Graph is
unsupported** — Pebble implements no GraphBackend and its profile declares no
graph ADT, so the planner will not route graph traversals here; declare a
graph-capable engine (e.g. `pgengine`/`mysqlengine` via recursive CTE, or
`dgraphengine`) alongside Pebble if your app needs graph queries.

Plus RawValueReader and RawScanReader for zero-JSON-decode point lookups
and filtered scans — eliminates the JSON decode tax on hot paths.

## API

| Symbol                      | Description                                             |
| --------------------------- | ------------------------------------------------------- |
| `NewPebbleEngine(dir)`      | Creates engine. Empty dir = in-memory (`vfs.NewMem`).   |
| `NewPebbleEngineFromDB(db)` | Wraps an existing `*pebble.DB` (caller owns lifecycle). |
| `Profile()`                 | Returns `EngineProfile` with Pebble cost model.         |

## Design

- **Separate module**: Exists outside the zero-dependency metaengine core
  (ADR-0062) because it requires `cockroachdb/pebble`.
- **RawValueReader/RawScanReader**: Eliminates JSON decode overhead on point
  lookups by reading raw stored bytes directly.
- **Persistence**: `NewPebbleEngine("")` is volatile (in-memory);
  `NewPebbleEngine("/path")` is persistent (disk LSM).

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**metaengine/adttest**](../adttest/) — Cross-engine ADT test harness
- [**storage/pebble**](../../storage/pebble/README.md) — Pebble-backed event/snapshot stores
