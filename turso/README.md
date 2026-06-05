# turso — Turso Database Connectors

CQRS storage adapters for [Turso](https://turso.tech/) databases (embedded LibSQL/SQLite).

```bash
go get github.com/larsartmann/go-cqrs-lite/turso/v2
```

## Constructors

| Function                            | Returns               | Description                      |
| ----------------------------------- | --------------------- | -------------------------------- |
| `Open(dbPath)`                      | `*sql.DB`             | Open a local Turso database file |
| `OpenInMemory()`                    | `*sql.DB`             | In-memory database for testing   |
| `OpenSync(ctx, dbPath, url, token)` | `*SyncDB`             | Open with remote sync            |
| `InitSchema(ctx, db)`               | —                     | Create all tables                |
| `NewEventStore(db)`                 | `*SQLEventStore`      | Event store backed by Turso      |
| `NewSnapshotStore(db)`              | `*SQLSnapshotStore`   | Snapshot store                   |
| `NewCheckpointStore(db)`            | `*SQLCheckpointStore` | Checkpoint store                 |

## Quick Start

```go
db, _ := turso.OpenInMemory()
turso.InitSchema(ctx, db)

store, _ := turso.NewEventStore(db)
store.Save(ctx, ref, events, 0)
```

## Sync

For offline-first with remote sync:

```go
syncDB, _ := turso.OpenSync(ctx, "local.db", "libsql://my-db.turso.io", "token")
syncDB.Push(ctx)  // send local changes
syncDB.Pull(ctx)  // receive remote changes
```

All constructors delegate to the equivalent SQLite implementations in the `storage` module.
