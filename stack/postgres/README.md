# stack/postgres — PostgreSQL Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/postgres/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/postgres/v4)

PostgreSQL preset for multi-process and production deployments. Supports distributed event delivery via Postgres `LISTEN/NOTIFY`.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/postgres/v4
```

## Quick Start

```go
import postgres "github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"

bundle, err := postgres.New("postgres://user:pass@localhost:5432/myapp?sslmode=disable")
if err != nil { log.Fatal(err) }
defer bundle.Close()
```

### Distributed Event Bus (cross-process pub/sub)

```go
listener, _ := postgres.NewPgxListenerFromDSN(ctx, dsn)
bundle, err := postgres.New(dsn,
    postgres.WithDistributedBus(listener),
)
```

With `WithDistributedBus`, the event bus uses Postgres `LISTEN/NOTIFY` for cross-process event delivery. Without it, the bus is GoChannel (single-process only).

### Multi-Database Topology

```go
bundle, err := postgres.New(primaryDSN,
    postgres.WithDSN(
        sqlopt.WithEventDB("postgres://host/events_db"),
        sqlopt.WithQueryDB("postgres://host/queries_db"),
        sqlopt.WithViewDB("postgres://host/views_db"),
    ),
)
```

## API

### Constructor

| Symbol              | Description                                |
| ------------------- | ------------------------------------------ |
| `New(dsn, opts...)` | Creates a Postgres-backed `*stack.Bundle`. |

### Options

| Option                                     | Description                                        |
| ------------------------------------------ | -------------------------------------------------- |
| `WithDSN(opts...)`                         | Configures multi-database topology.                |
| `WithDistributedBus(listener, busOpts...)` | Enables `LISTEN/NOTIFY` for cross-process pub/sub. |

## Design

- **Default event bus**: GoChannel (single-process). Use `WithDistributedBus` for multi-process.
- **Multi-DB split**: Events, audit logs, and read models can use separate Postgres databases.
- **Read models**: Via `storage.SQLKVStore` (KV documents) or `storage.SQLViewStore` (column-mapped views).
- **Auto-migration**: Schema tables created automatically on first run.

## View Models

```go
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{ ... },
    ScanRow: func(scan func(...any) error) (*TodoView, error) { ... },
}
store, _ := postgres.SQLViewModel[TodoView, TodoID](bundle, mapper)
```

## When to Use

- Multi-process or multi-instance deployments
- Production services requiring managed database infrastructure
- Teams already running Postgres

## Related Modules

- [**stack**](../README.md) — The `Bundle` type
- [**storage**](../../storage/README.md) — SQL event store, `PostgresListenNotifyBus`
- [**stack/sqlite**](../sqlite/README.md) — SQLite alternative for single-node
