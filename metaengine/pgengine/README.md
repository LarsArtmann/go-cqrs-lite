# metaengine/pgengine — Postgres-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4)

Postgres-backed [metaengine](../README.md) Engine. Uses JSONB columns for efficient JSON storage, UPSERT via `ON CONFLICT`, and native `GROUP BY` for counter aggregation. Pure Go (pgx driver, no CGo).

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"

engine, err := pgengine.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
```

## Backends

MapBackend, CounterBackend, ScanBackend, PushdownScan, and LayoutPlanner.

- **PushdownScan**: Filter/sort pushed into Postgres `WHERE`/`ORDER BY` using
  JSONB operators (`value->>'field'`), avoiding full-table scans.
- **LayoutPlanner**: Creates expression indexes on JSONB paths for B-tree
  performance on declared query patterns.

## Cost Profile

| ADT     | Complexity | Notes                                          |
| ------- | ---------- | ---------------------------------------------- |
| Map     | O(log N)   | B-tree point lookup via primary key            |
| Counter | O(N)       | Native GROUP BY aggregation                    |
| Scan    | O(N)       | Sequential scan with filter/sort pushdown      |

Calibrated from benchmark measurements (see `calibration_bench_test.go`).

## API

| Symbol              | Description                                           |
| ------------------- | ----------------------------------------------------- |
| `New(dsn)`          | Creates engine from Postgres DSN. Opens its own pool. |
| `NewFromDB(db)`     | Wraps an existing `*sql.DB` (caller owns lifecycle).  |
| `Profile()`         | Returns `EngineProfile` with Postgres cost model.     |

## Design

- **Always persistent**: Postgres data always survives process restarts.
- **JSONB storage**: Efficient binary JSON with GIN/B-tree index support.
- **Cross-engine parity**: Verified via `adttest.RunMatrix`.
- **Pure Go**: Uses `pgx` driver via `database/sql` — no CGo required.

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**metaengine/adttest**](../adttest/) — Cross-engine ADT test harness
- [**stack/postgres**](../../stack/postgres/README.md) — Postgres stack preset
