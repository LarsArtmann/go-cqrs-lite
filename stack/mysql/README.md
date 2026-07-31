# stack/mysql — MySQL / MariaDB Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/mysql/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/mysql/v4)

MySQL and MariaDB preset for production deployments. Uses the pure-Go
`go-sql-driver/mysql` driver (no CGo required). Fully compatible with both MySQL
8.x and MariaDB 10.x+.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/mysql/v4
```

## Quick Start

```go
import mysql "github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"

// DSN format: user:password@tcp(host:port)/dbname?parseTime=true
// parseTime=true is REQUIRED for correct time.Time handling.
bundle, err := mysql.New("root:password@tcp(localhost:3306)/myapp?parseTime=true")
if err != nil { log.Fatal(err) }
defer bundle.Close()
```

### Multi-Database Topology

Split events, audit logs, and read models across separate MySQL databases on the
same server:

```go
bundle, err := mysql.New(primaryDSN,
    mysql.WithDSN(
        sqlopt.WithEventDB("root:pass@tcp(host:3306)/events_db?parseTime=true"),
        sqlopt.WithQueryDB("root:pass@tcp(host:3306)/queries_db?parseTime=true"),
        sqlopt.WithViewDB("root:pass@tcp(host:3306)/views_db?parseTime=true"),
    ),
)
```

## DSN Format

The DSN follows the [go-sql-driver/mysql format](https://github.com/go-sql-driver/mysql#dsn-data-source-name):

```
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```

**Required parameter**: `parseTime=true` — without it, `DATETIME` columns are
returned as `[]byte` instead of `time.Time`, breaking event timestamps.

Common parameters:

| Parameter         | Default | Recommendation         |
| ----------------- | ------- | ---------------------- |
| `parseTime`       | `false` | **`true`** (required)  |
| `charset`         | none    | `utf8mb4`              |
| `loc`             | `UTC`   | `UTC` (or `Local`)     |
| `timeTruncate`    | none    | `AUTO` (MySQL 8.0.28+) |

## API

### Constructor

| Symbol              | Description                                                        |
| ------------------- | ------------------------------------------------------------------ |
| `New(dsn, opts...)` | Creates a MySQL/MariaDB-backed `*stack.Bundle` with auto-migration. |

### Options

| Option              | Description                                          |
| ------------------- | ---------------------------------------------------- |
| `WithDSN(opts...)`  | Configures multi-database topology and auto-migrate. |
| `WithStack(opts...)`| Passes through additional `stack.Option` values.     |

## Design

- **Event bus**: GoChannel (single-process). For multi-process, use a Watermill
  adapter with an external message broker.
- **Multi-DB split**: Events, audit logs, and read models can use separate MySQL
  databases on the same server.
- **Read models**: Via `storage.SQLKVStore` (KV documents) or
  `storage.SQLViewStore` (column-mapped views).
- **Auto-migration**: All CQRS schema tables are created automatically on first
  run. Disable with `sqlopt.WithoutAutoMigrate()`.
- **Schema**: Uses MySQL-native types (`LONGBLOB` for payloads, `JSON` for
  metadata, `DATETIME(3)` for millisecond timestamps).
- **Upsert strategy**: `ON DUPLICATE KEY UPDATE col = VALUES(col)` (works on
  MySQL 5.7+, 8.x, and MariaDB).

## MariaDB Notes

MariaDB is fully supported — the same DSN and API work without changes. The
`ON DUPLICATE KEY UPDATE` syntax and `VALUES()` function are standard SQL
features supported by both MySQL and MariaDB.

## View Models

```go
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{ ... },
    ScanRow: func(scan func(...any) error) (*TodoView, error) { ... },
}
store, _ := mysql.SQLViewModel[TodoView, TodoID](bundle, mapper)
```

## When to Use

- Existing MySQL/MariaDB infrastructure
- Teams already operating MySQL at scale
- Applications requiring MySQL-specific tooling (replication, proxySQL, etc.)

For greenfield projects, consider `stack/postgres` (distributed bus via
`LISTEN/NOTIFY`) or `stack/sqlite` (single-node simplicity).

## Related Modules

- [**stack**](../README.md) — The `Bundle` type
- [**storage**](../../storage/README.md) — SQL event store, `SQLBackend`
- [**stack/postgres**](../postgres/README.md) — PostgreSQL alternative with `LISTEN/NOTIFY`
- [**stack/sqlite**](../sqlite/README.md) — SQLite alternative for single-node
