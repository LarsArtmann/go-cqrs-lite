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

## Actor Attribution

`Timer.Actor` (typed `id.ActorID`; wire form is the "kind:raw" audit-trail
attribution, e.g. `user:01JXYZ...`) survives SQL persistence via a
versioned payload envelope (`{"v":1,"actor":...,"payload":...}`). The
column stays a plain string — conversion happens at the store boundary.
Rows written by pre-actor versions (bare payload JSON) still decode, with
an empty actor.

## API

| Method      | Description                                       |
| ----------- | ------------------------------------------------- |
| `Schedule`  | Insert timer (no-op if ID exists — idempotent)    |
| `Due`       | Return timers where `fire_at <= now`, ordered ASC |
| `MarkFired` | Delete timer after dispatch                       |
| `Cancel`    | Delete timer before it fires                      |
| `Close`     | No-op (caller owns the `*sql.DB`)                 |

## Constructors

- `NewSQLiteStore[P](ctx, db)` — `?` placeholders, RFC3339 text timestamps
- `NewPostgresStore[P](ctx, db)` — `$N` placeholders, native `TIMESTAMP WITH TIME ZONE`
- `NewMySQLStore[P](ctx, db)` — `?` placeholders, `DATETIME(3)`, `ON DUPLICATE KEY UPDATE`

## Claiming (multi-dispatcher safety)

With a single dispatcher, `Due` + `MarkFired` is race-free. Running MULTIPLE
dispatchers against one store requires claiming: `Due` atomically takes a
timed lease (owner + deadline) so each timer is handed to exactly one
dispatcher even under concurrent polling. Claiming stores add `RenewLease`
for handlers that may outlive their lease — renewal extends a live claim
(other claimers see nothing) and fails with `ErrLeaseNotHeld` once the lease
has lapsed (the timer went back to the pollable pool).

Claiming store support matrix:

| Constructor                          | Claim mechanism                          | Verified on                                            |
| ------------------------------------ | ---------------------------------------- | ------------------------------------------------------ |
| `NewClaimingPostgresStore[P](ctx, db, lease)` | `FOR UPDATE SKIP LOCKED` + `UPDATE ... RETURNING` | Postgres 16 (testcontainers)                    |
| `NewClaimingSQLiteStore[P](ctx, db, lease)`   | Single `UPDATE ... RETURNING` (SQLite 3.35+) | modernc.org/sqlite                                |
| `NewClaimingMySQLStore[P](ctx, db, lease)`    | `FOR UPDATE SKIP LOCKED` + `UPDATE` by IDs (two statements, one tx) | MariaDB 11.4 (live) |

MySQL/MariaDB version floor: the claim transaction uses
`FOR UPDATE SKIP LOCKED` — MySQL 8.0.1+ and MariaDB 10.6.0+ (InnoDB;
MDEV-13115). There is NO construction-time version probe: older servers
accept the store and fail loudly at the first `Due` call with a syntax
error. This is the documented contract — probe your server version at
deployment time if you need an earlier, clearer signal. SKIP LOCKED
semantics (not just syntax) were additionally verified live on MariaDB
11.4 (2026-09-06): a transaction holding row locks does not block a
concurrent SKIP LOCKED claim of the remaining rows.
`ErrClaimingUnsupported` is a plain sentinel (`errors.Is` works) returned
only for unknown SQL dialects.

Integration pins: `pg_integration_test.go` (Postgres) and
`mysql_claiming_integration_test.go` (build tag `integration`,
`MYSQL_TEST_DSN`) cover two-claimer no-double-fire, lease expiry reclaim,
and lease renewal on live servers.

## Related Modules

- [**scheduling**](../README.md) — `TimerStore[P]` interface and `Scheduler`
- [**storage**](../../storage/README.md) — Embedded migrations include a compatible `timers` table
- [**command**](../../command/README.md) — Timer payloads are typically commands dispatched on fire
