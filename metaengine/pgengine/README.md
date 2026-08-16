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

MapBackend, CounterBackend, ScanBackend, PushdownScan, LayoutPlanner,
GraphBackend, and StreamLogBackend.

- **PushdownScan**: Filter/sort pushed into Postgres `WHERE`/`ORDER BY` using
  JSONB operators (`value->>'field'`), avoiding full-table scans.
- **LayoutPlanner**: Creates expression indexes on JSONB paths for B-tree
  performance on declared query patterns.
- **GraphBackend**: Native graph traversal on a dedicated `meta_graph_edges`
  table. `GraphNeighbors` resolves the whole depth-limited neighborhood in
  one `WITH RECURSIVE` query (cycle-safe, deduplicated); each hop is an
  index lookup on `(collection, from_node)`. No capability probe is needed:
  Postgres has shipped recursive CTEs since 8.4.
- **StreamLogBackend**: Append-only `meta_stream_log` journal with
  position-based reads.

Vector search runs as a degraded O(N) scan (no native vector index).

## Cost Profile

| ADT     | Complexity | Notes                                        |
| ------- | ---------- | -------------------------------------------- |
| Map     | O(log N)   | B-tree point lookup via primary key          |
| Counter | O(N)       | Native GROUP BY aggregation                  |
| Scan    | O(N)       | Sequential scan with filter/sort pushdown    |
| Graph   | O(degree)  | Native `WITH RECURSIVE` per-hop index lookup |
| Vector  | O(N)       | Degraded: brute-force scan, no ANN index     |

Calibrated from benchmark measurements (see `calibration_bench_test.go`).

## API

| Symbol          | Description                                           |
| --------------- | ----------------------------------------------------- |
| `New(dsn)`      | Creates engine from Postgres DSN. Opens its own pool. |
| `NewFromDB(db)` | Wraps an existing `*sql.DB` (caller owns lifecycle).  |
| `Profile()`     | Returns `EngineProfile` with Postgres cost model.     |

## Design

- **Always persistent**: Postgres data always survives process restarts.
- **JSONB storage**: Efficient binary JSON with GIN/B-tree index support.
- **Cross-engine parity**: Verified via `adttest.RunMatrix`.
- **Pure Go**: Uses `pgx` driver via `database/sql` — no CGo required.

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**metaengine/adttest**](../adttest/) — Cross-engine ADT test harness
- [**stack/postgres**](../../stack/postgres/README.md) — Postgres stack preset
