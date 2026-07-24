# idempotency/sqlstore — SQL-Backed Idempotency Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4)

Adapts any `database/sql`-compatible database (SQLite, PostgreSQL) into an `idempotency.Store` with TTL-based key expiry. Enables multi-process idempotency for horizontally-scaled command handlers.

```bash
go get github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4
```

## Why?

The base `idempotency/` package ships an in-memory `MemoryStore` for development.
For production with multiple processes (e.g., command handlers behind a load
balancer), each process has its own in-memory store — deduplication fails
across processes. This subpackage uses a shared SQL database as the backing
store, so all processes see the same idempotency keys.

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "time"

    "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
    _ "modernc.org/sqlite"
)

func main() {
    db, _ := sql.Open("sqlite", "file:app.db?_pragma=busy_timeout(5000)")
    defer db.Close()

    store, _ := sqlstore.NewSQLiteStore(context.Background(), db)
    defer store.Close()

    // Atomic check-and-record: exactly one process wins per key.
    err := store.CheckAndRecord(context.Background(), "order-123", 10*time.Minute)
    if err != nil {
        // Duplicate command — skip processing.
        return
    }

    // Process the command...
}
```

## Atomicity

`CheckAndRecord` uses `INSERT ... ON CONFLICT DO UPDATE WHERE` — a single
atomic statement that handles three cases:

| Scenario                  | SQL outcome                  | Result          |
| ------------------------- | ---------------------------- | --------------- |
| Key does not exist        | INSERT succeeds              | nil (claimed)   |
| Key exists but expired    | UPDATE overwrites expiry     | nil (claimed)   |
| Key exists and not expired| No rows affected             | ErrDuplicate    |

Both SQLite and PostgreSQL evaluate the `WHERE` clause within the same
statement, so concurrent callers are serialized at the row level by the
database engine. No application-level locking is needed.

## API

| Method           | Description                                         |
| ---------------- | --------------------------------------------------- |
| `CheckAndRecord` | Atomic claim — returns nil or `ErrDuplicate`        |
| `Seen`           | Check if key is recorded and not expired            |
| `Record`         | Insert key with TTL (no-op if key already exists)   |
| `Sweep`          | Delete all expired entries (call periodically)      |
| `Close`          | No-op (caller owns the `*sql.DB`)                   |

## Constructors

- `NewSQLiteStore(ctx, db)` — creates table, uses `?` placeholders
- `NewPostgresStore(ctx, db)` — creates table, uses `$N` placeholders
