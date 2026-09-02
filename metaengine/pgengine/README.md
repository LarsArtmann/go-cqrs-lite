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

### Planned vs meta_map (measured, 2K rows, ephemeral PG testcontainer, 30-iter medians, 2026-08-30)

| Operation                  | meta_map (JSONB) | planned (native columns) | Verdict                                                            |
| -------------------------- | ---------------: | -----------------------: | ------------------------------------------------------------------ |
| Filtered scan (`status =`) |         873.8 µs |                 778.9 µs | planned wins; the gap grows with N                                 |
| CounterGet                 |         287.4 µs |                 260.0 µs | equal within noise — counters STAY on meta_map (ADR-0124 addendum) |

Source: `planned_vs_metamap_bench_test.go`. The planned-table decision rule
lives in the ADR-0124 addendum; the observability surface (registration +
live row counts) is `Doctor`'s `--- Planned tables ---` section.

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

## Planned tables: ApplyLayout vs ApplyLayoutPlan

Two layout mechanisms exist; they are NOT interchangeable:

- **`ApplyLayout(collection, filterFields, sortFields)`** (metaengine.LayoutPlanner)
  keeps all rows in `meta_map` and creates PARTIAL EXPRESSION INDEXES on
  `value->'field'` paths. Cheapest option: no second table, no backfill — but
  reads still extract JSONB at query time and only declared fields are indexed.
- **`ApplyLayoutPlan(metaengine.LayoutPlan)`** (metaengine.LayoutPlanApplier) creates a
  dedicated per-collection extracted-column table (`meta_planned_<collection>`:
  JSONB value + DOUBLE PRECISION/BIGINT/TEXT columns + declared indexes) and routes
  `MapSet`/`MapGet`/`MapDelete`, `MapUpdate`, `PushdownMapScan`, and `MapScan` through it.
  No backfill: rows written before registration stay in `meta_map` and are NOT visible
  to planned reads (opt-in per collection, choose at deployment time).

Mis-typed filter/sort/cursor values are validated against the declared column
types BEFORE SQL executes and fail as
`metaengine.ErrPlannedColumnTypeMismatch` (Rejection family — fix the query or
the plan; retrying cannot succeed). The write path keeps its fail-loud
driver-level Infrastructure behavior (Postgres rejects a row whose extracted
value contradicts the column type).

Routing decision: counters, graph, and aggregate operations stay on
`meta_map` even for planned collections — planned tables optimize map reads;
the other ADTs keep their native paths. `MapUpdate` serializes concurrent
read-modify-writes with `SELECT ... FOR UPDATE` (PG is multi-writer).

## Related Modules

- [**metaengine**](../README.md) — Core planner and `Engine` interface
- [**metaengine/adttest**](../adttest/) — Cross-engine ADT test harness
- [**stack/postgres**](../../stack/postgres/README.md) — Postgres stack preset
