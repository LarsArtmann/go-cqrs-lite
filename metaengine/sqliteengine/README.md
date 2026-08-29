# metaengine/sqliteengine — SQLite-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4)

SQLite-backed [metaengine](../README.md) Engine. Pure Go (`modernc.org/sqlite`,
no CGo). The widest-capability disk engine: pushdown scans, layout planning,
raw-value reads, and streaming scans on a single embedded file. This is the
engine `metaengine.PlanFromMemory`-style embedded deployments default to, and
the base the `tursoengine` wraps for remote libSQL.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4
```

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"

engine, err := sqliteengine.NewSQLiteEngineFromDSN("file:app.db")
```

## Backends

MapBackend, MapUpdater, SetBackend, CounterBackend, ScanBackend, PushdownScan,
StreamingScan, LayoutPlanner, LayoutPlanApplier, RawValueReader, RawScanReader.

- **PushdownScan**: filter/sort pushed into SQLite `WHERE`/`ORDER BY` over
  generated columns, avoiding full-table scans.
- **LayoutPlanner / LayoutPlanApplier**: creates expression indexes for
  declared query patterns and applies planned layouts (the reference
  implementation other SQL engines converge on).
- **StreamingScan**: row-at-a-time iteration for large result sets.

## Notes

- Use a unique `file:<name>?mode=memory&cache=shared` DSN for shared in-memory
  databases; plain `:memory:` is per-connection.
- PRAGMAs (journal mode, synchronous, cache size) are accepted at open time
  via `NewSQLiteEngineFromDSN` variadic args and drive the effective
  durability tier.
