# stack/sqlite — SQLite Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4)

Recommended persistent preset for single-node deployments. Uses a single SQLite database file with WAL mode and auto-migration.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4
```

## Quick Start

```go
import sqlite "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

bundle, err := sqlite.New("app.db")
if err != nil { log.Fatal(err) }
defer bundle.Close()
```

### Multi-Database Topology

Split events, queries, and views across separate SQLite files:

```go
bundle, err := sqlite.New("",
    sqlite.WithDSN(
        sqlopt.WithEventDB("events.db"),
        sqlopt.WithQueryDB("queries.db"),
        sqlopt.WithViewDB("views.db"),
    ),
)
```

### Custom Pragmas

```go
bundle, err := sqlite.New("app.db",
    sqlite.WithPragmas(
        sqlopt.WithJournalMode("WAL"),
        sqlopt.WithBusyTimeout(10000),
    ),
)
```

## API

| Symbol                  | Description                                              |
| ----------------------- | -------------------------------------------------------- |
| `New(dsn, opts...)`     | Creates a `*stack.Bundle` from a SQLite DSN.             |
| `WithDSN(opts...)`      | Configures multi-database topology.                      |
| `WithPragmas(opts...)`  | Sets SQLite PRAGMA values.                               |

## Design

- **Pure-Go driver**: Uses `modernc.org/sqlite` (no CGo required).
- **WAL mode by default**: Write-Ahead Logging for concurrent read/write access.
- **Auto-migration**: Schema tables created automatically on first run.
- **GoChannel event bus**: In-process pub/sub (SQLite has no native pub/sub).
- **Busy timeout**: Enabled by default (5000ms) to eliminate "database is locked" errors.
- **Filesystem**: ext4/xfs/ZFS are fine; NFS/SMB are broken with WAL mode.

## View Models

```go
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{ ... },
    ScanRow: func(scan func(...any) error) (*TodoView, error) { ... },
}
store, _ := sqlite.SQLViewModel[TodoView, TodoID](bundle, mapper)
```

## Related Modules

- [**stack**](../README.md) — The `Bundle` type
- [**storage**](../../storage/README.md) — SQL event store, command store, view store
- [**stack/postgres**](../postgres/README.md) — PostgreSQL alternative for multi-process
