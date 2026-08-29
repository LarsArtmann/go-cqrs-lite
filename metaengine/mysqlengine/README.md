# metaengine/mysqlengine — MySQL/MariaDB-Backed Engine

[![Go Reference](https://pkg.go.dev/banner.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4)

MySQL- and MariaDB-backed [metaengine](../README.md) Engine (verified against
MySQL 8.4 and MariaDB 11.8). Pure Go (`go-sql-driver/mysql`, no CGo). JSON
documents live in a `JSON` column with generated-column expression indexes for
declared query patterns.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/mysqlengine/v4"

engine, err := mysqlengine.New("user:pass@tcp(localhost:3306)/myapp")
```

## Backends

MapBackend, CounterBackend, ScanBackend, PushdownScan, LayoutPlanner.

- **PushdownScan**: filter/sort pushed into `WHERE`/`ORDER BY` using the
  universal `JSON_UNQUOTE(JSON_EXTRACT(value,'$.field'))` form — the
  MariaDB-safe dialect that avoids `->` and `CAST(? AS JSON)` (Error 1064)
  and keeps numeric sorts correct via a dual
  `CAST(... AS DECIMAL(65,10))` + text key.
- **LayoutPlanner**: planned-column and expression-index support for declared
  query patterns.

## Notes

- MariaDB's `JSON_EXTRACT` returns LONGTEXT; numeric ordering must go through
  the dual-key form above — the engine's dialect does this for you (see
  `dialect.go`).
- Health: `db.PingContext`.
