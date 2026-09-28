# metaengine/sqliteengine — SQLite-Backed Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4)

SQLite-backed [metaengine](../README.md) Engine. Pure Go (`modernc.org/sqlite`,
no CGo). The widest-capability disk engine: pushdown scans, layout planning,
raw-value reads, and streaming scans on a single embedded file — the default
choice for embedded deployments that need persistence, and the base the
`tursoengine` wraps for remote libSQL.

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
- **C extensions cannot be loaded — sqlite-vec stays an operator-only option.**
  This engine is pure Go on `modernc.org/sqlite`, whose `database/sql` driver
  exposes no `LoadExtension` (v1.59.0). Native vector SQL (`vector_distance_*`,
  vec0 indexes via sqlite-vec) therefore requires an operator-managed libSQL
  server reached through `tursoengine` — inside this engine, vector search
  always runs the probe-gated libSQL-SQL or Go-scan path (see the metaengine
  vector contract), never an extension.
