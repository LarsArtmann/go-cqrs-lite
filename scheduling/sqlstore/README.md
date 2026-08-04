# scheduling/sqlstore — SQL-Backed Durable Timer Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4)

Adapts any `database/sql`-compatible database (SQLite, PostgreSQL, MySQL) into a `scheduling.TimerStore[P]` for durable deadline timers that survive process restarts.

```bash
go get github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4
```

## Why?

The base `scheduling/` package ships an in-memory `MemoryTimerStore` for
development. For production sagas ("cancel order after 30 min unpaid"), the
timer MUST survive process crashes — if the process dies before the timer
fires, the timer must still be present on restart. This subpackage uses a SQL
database as the backing store so timers persist across restarts.

## Quick Start

```go
package main

import (
    "context"
    "database/sql"

    "github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
    "github.com/larsartmann/go-cqrs-lite/scheduling/v4"
    _ "modernc.org/sqlite"
)

type CancelOrderCmd struct {
    OrderID string
}

func main() {
    db, _ := sql.Open("sqlite", "file:app.db?_pragma=busy_timeout(5000)")
    defer db.Close()

    store, _ := sqlstore.NewSQLiteStore[CancelOrderCmd](context.Background(), db)

    sched := scheduling.New(store, dispatch, scheduling.WithPollInterval(time.Second))
    sched.Start(ctx)
}
```

## Durability

Timers are persisted to the `timers` table on `Schedule`. The schema matches
the `timers` table in the storage module's embedded migrations, so consumers
using both see no conflicts. The caller owns the `*sql.DB`; `Close` is a no-op.

## API

| Method     | Description                                       |
| ---------- | ------------------------------------------------- |
| `Schedule` | Insert timer (no-op if ID exists — idempotent)    |
| `Due`      | Return timers where `fire_at <= now`, ordered ASC |
| `MarkFired`| Delete timer after dispatch                       |
| `Cancel`   | Delete timer before it fires                      |
| `Close`    | No-op (caller owns the `*sql.DB`)                 |

## Constructors

- `NewSQLiteStore[P](ctx, db)` — `?` placeholders, RFC3339 text timestamps
- `NewPostgresStore[P](ctx, db)` — `$N` placeholders, native `TIMESTAMP WITH TIME ZONE`
- `NewMySQLStore[P](ctx, db)` — `?` placeholders, `DATETIME(3)`, `ON DUPLICATE KEY UPDATE`
