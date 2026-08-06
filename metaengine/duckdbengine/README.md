# metaengine/duckdbengine — DuckDB-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4)

DuckDB-backed [metaengine](../README.md) Engine for analytical (OLAP) workloads. DuckDB's vectorized columnar engine makes Counter reads O(1) (vectorized GROUP BY) and enables efficient filtered scans via `json_extract` pushdown.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4
```

> **CGo required.** This module statically links the DuckDB C++ engine.
> Isolated in its own Go module so consumers who don't import it never need CGo.

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"

// Persistent file
engine, err := duckdbengine.New("analytics.db")

// In-memory (ephemeral)
engine, err := duckdbengine.New("")
```

## Backends

MapBackend, CounterBackend, ScanBackend, PushdownScan, and LayoutPlanner.

- **PushdownScan**: Filter/sort pushed into DuckDB `WHERE`/`ORDER BY` via `json_extract()`.
- **LayoutPlanner**: Creates dedicated planned tables with extracted typed columns
  (float64→DOUBLE, int→INTEGER) and ART indexes. After `ApplyLayout`, queries use
  direct column references instead of `json_extract`, enabling zone-map pruning.

## Cost Profile

| ADT     | Complexity | Notes                                           |
| ------- | ---------- | ----------------------------------------------- |
| Map     | O(1)       | Point lookup via primary key                    |
| Counter | O(1)       | Vectorized GROUP BY — faster than SQLite O(N)   |
| Scan    | O(N)       | Columnar scan with filter/sort pushdown         |

## API

| Symbol              | Description                                          |
| ------------------- | ---------------------------------------------------- |
| `New(dsn)`          | Creates engine from DSN. Empty = in-memory.         |
| `NewFromDB(db)`     | Wraps an existing `*sql.DB` (caller owns lifecycle). |
| `Profile()`         | Returns `EngineProfile` with DuckDB cost model.     |

## Design

- **Columnar-native storage** ([ADR-0092](../../docs/adr/0092-duckdb-columnar-native-storage.md)):
  `WithColumnarLayout()` extracts all exported fields of R into native SQL columns.
- **Persistence**: `New("")` is volatile (`:memory:`); `New("file.db")` is persistent.
- **Cross-engine parity**: Verified via `adttest.RunMatrix`.

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**metaengine/adttest**](../adttest/) — Cross-engine ADT test harness
- [**stack/duckdb**](../../stack/duckdb/README.md) — DuckDB stack preset
