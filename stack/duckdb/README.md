# stack/duckdb

DuckDB-backed preset for [go-cqrs-lite](../../README.md) — an embedded analytical (OLAP) SQL engine.

## When to Use

| Workload | Recommended Preset |
|----------|-------------------|
| Analytical queries, dashboards, reporting | **duckdb** |
| OLTP, high-write, single-node | sqlite |
| Distributed, high-throughput | postgres |

DuckDB excels at:
- Columnar scans over large datasets
- GROUP BY aggregations and window functions
- Complex analytical SQL (CTEs, pivots, statistical functions)
- Read-heavy materialized views and projections

## CGo Requirement

DuckDB statically links a C++ engine. This module requires:

- `CGO_ENABLED=1`
- A C/C++ compiler (gcc or clang)

It is isolated in its own Go module. Consumers who do **not** import `stack/duckdb` never need CGo — the rest of go-cqrs-lite remains pure-Go.

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"

// Persistent database
b, err := duckdb.New("analytics.db")
defer b.Close()

// In-memory (ephemeral)
b, err := duckdb.New("")
defer b.Close()
```

## Performance Tuning

```go
b, err := duckdb.New("analytics.db",
    duckdb.WithThreads(4),           // limit worker threads
    duckdb.WithMemoryLimit("1GB"),   // cap memory usage
)
```

## Multi-Database Topology

```go
b, err := duckdb.New("primary.db",
    duckdb.WithDSN(
        sqlopt.WithEventDB("events.db"),
        sqlopt.WithQueryDB("queries.db"),
        sqlopt.WithViewDB("views.db"),
    ),
)
```

## Analytical Read Models

```go
bundle, _ := duckdb.New("analytics.db")
mapper := storage.ViewMapper[OrderView]{
    Table: "orders_view",
    Columns: []storage.ViewColumn[OrderView]{
        {Name: "total", Type: "DOUBLE", Extract: func(v *OrderView) any { return v.Total }},
        {Name: "status", Type: "VARCHAR", Extract: func(v *OrderView) any { return v.Status }},
    },
    ScanRow: func(scan func(dest ...any) error) (*OrderView, error) { ... },
}
store, _ := duckdb.SQLViewModel[OrderView, OrderID](bundle, mapper)
```

DuckDB's columnar engine makes these view tables especially powerful for analytical SQL queries directly on the materialized data.
