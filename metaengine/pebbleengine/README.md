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

| ADT          | Complexity      | Notes                                            |
| ------------ | --------------- | ------------------------------------------------ |
| Map/Set      | O(1)            | LSM point read — faster than SQLite B-tree       |
| Counter      | O(1) incr, O(N) get | Increment is atomic; CounterGet is prefix scan |
| Multimap     | O(1) add, O(N) get | Prefix scan for retrieval                      |
| Log          | O(1) append, O(N) tail | Prefix scan for tail reads                 |
| Graph        | O(N^d)          | BFS via prefix scan (degraded, no indexes)       |

## Backends

All 7 metaengine ADT backends are implemented: MapBackend, ScanBackend,
SetBackend, CounterBackend, GraphBackend, MultimapBackend, LogBackend.

Plus RawValueReader and RawScanReader for zero-JSON-decode point lookups
and filtered scans — eliminates the JSON decode tax on hot paths.

## API

| Symbol                       | Description                                          |
| ---------------------------- | ---------------------------------------------------- |
| `NewPebbleEngine(dir)`       | Creates engine. Empty dir = in-memory (`vfs.NewMem`). |
| `NewPebbleEngineFromDB(db)`  | Wraps an existing `*pebble.DB` (caller owns lifecycle). |
| `Profile()`                  | Returns `EngineProfile` with Pebble cost model.     |

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
