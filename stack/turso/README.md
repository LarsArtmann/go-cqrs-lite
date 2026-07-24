# stack/turso — Turso Database Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/turso/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/turso/v4)

Embedded Turso database preset with optional remote sync. Offline-first architecture: work locally with a SQLite-compatible embedded database, sync to a remote Turso server when connectivity allows.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/turso/v4
```

## Quick Start

### Local Embedded Mode

```go
import turso "github.com/larsartmann/go-cqrs-lite/stack/turso/v4"

bundle, err := turso.New("data/myapp.db")
if err != nil { log.Fatal(err) }
defer bundle.Close()
```

### Remote Sync Mode

```go
bundle, err := turso.NewSync(ctx, "data/myapp.db",
    "libsql://my-app.turso.io",
    "your-auth-token",
)
if err != nil { log.Fatal(err) }
defer bundle.Close()

// Push local changes to remote:
sync := bundle.Sync()
_ = sync.Push(ctx)

// Pull remote changes:
_ = sync.Pull(ctx)
```

## API

### Constructors

| Symbol                                      | Description                          |
| ------------------------------------------- | ------------------------------------ |
| `New(dbPath, opts...)`                      | Local embedded mode. No remote sync. |
| `NewSync(ctx, dbPath, url, token, opts...)` | Local + remote sync mode.            |

### Options

| Option                     | Description                                       |
| -------------------------- | ------------------------------------------------- |
| `WithPragmas(opts...)`     | Sets SQLite PRAGMA values.                        |
| `WithDSN(opts...)`         | Configures multi-database topology (local only).  |
| `WithSyncOptions(opts...)` | Configures sync behavior (interval, retry, etc.). |

### Bundle Extensions

The Turso `Bundle` embeds `*stack.Bundle` and adds:

| Method   | Description                                                                       |
| -------- | --------------------------------------------------------------------------------- |
| `Sync()` | Returns `*cqrsturso.SyncDB` for `Push`/`Pull`/`Checkpoint`/`Stats`/`HealthCheck`. |

## Design

- **WAL + synchronous=NORMAL by default**: Balances durability and performance.
- **Multi-DB topology is local-only**: Sync requires a single database file.
- **GoChannel event bus**: In-process pub/sub (no native pub/sub in embedded mode).
- **Offline-first**: All operations work without network connectivity. Sync is explicit.

## Sync Operations

```go
sync := bundle.Sync()

// Push local writes to remote Turso server
_ = sync.Push(ctx)

// Pull remote changes to local
_ = sync.Pull(ctx)

// Checkpoint: optimize the WAL
_ = sync.Checkpoint(ctx)

// Health check
stats, _ := sync.Stats()
err := sync.HealthCheck(ctx)
```

## When to Use

- Edge / IoT deployments needing offline-first architecture
- Applications that sync to a central server intermittently
- Teams wanting SQLite locally with managed cloud backup

## Related Modules

- [**stack**](../README.md) — The `Bundle` type
- [**storage/turso**](../../storage/turso/README.md) — Turso connector and indexing advisor
- [**stack/sqlite**](../sqlite/README.md) — Plain SQLite (no sync) alternative
