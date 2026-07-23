# stack/pebble — PebbleDB Stack Preset

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/pebble/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/pebble/v4)

Embedded PebbleDB stack preset. All capabilities share one LSM tree via disjoint key prefixes. Adds backup, snapshot, and metrics capabilities on top of the base `Bundle`.

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/pebble/v4
```

## Quick Start

```go
import pebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"

bundle, err := pebble.New("data/myapp")
if err != nil { log.Fatal(err) }
defer bundle.GracefulClose(ctx)
```

### With Custom Options

```go
bundle, err := pebble.New("data/myapp",
    pebble.WithPebbleOptions(pebble.DefaultOptionsWithLogging(slog.Default())),
    pebble.WithLogger(slog.Default()),
)
```

## API

### Constructor

| Symbol          | Description                                          |
| --------------- | ---------------------------------------------------- |
| `New(dir, opts...)` | Creates a Pebble-backed `*Bundle`.               |

### Options

| Option                  | Description                                    |
| ----------------------- | ---------------------------------------------- |
| `WithPebbleOptions(o)`  | Sets Pebble DB options (bloom filter, etc.).   |
| `WithLogger(l)`         | Sets the structured logger.                    |

### Bundle Extensions

The Pebble `Bundle` embeds `*stack.Bundle` and adds Pebble-specific methods:

| Method                       | Description                                             |
| ---------------------------- | ------------------------------------------------------- |
| `Checkpoint(dir)`            | Point-in-time physical backup snapshot.                 |
| `NewSnapshot()`              | Consistent read view (close when done).                 |
| `Flush()`                    | Flushes buffered writes to disk.                        |
| `Metrics()`                  | LSM health metrics (block cache hit rate, compactions). |
| `GracefulClose(ctx)`         | Context-bounded close (prevents hang on slow flush).    |

## Operations

### Backup

```go
err := bundle.Checkpoint("backups/2026-01-15")
```

### Health Check

```go
m := bundle.Metrics()
hitRate := float64(m.BlockCacheHits) /
    float64(m.BlockCacheHits + m.BlockCacheMisses)
```

## Design

- **Persistent on disk**: Data survives process restarts.
- **Single LSM tree**: All stores (events, snapshots, checkpoints, KV) share one Pebble DB via disjoint key prefixes.
- **GoChannel event bus**: In-process pub/sub (Pebble has no native pub/sub).
- **GracefulClose**: Bounds `Close()` with a context timeout to prevent indefinite hangs during shutdown.

## When to Use

- Single-node services needing high write throughput
- Embedded deployments (no external database server)
- Workloads with heavy point reads and writes

## Related Modules

- [**stack**](../README.md) — The `Bundle` type
- [**storage/pebble**](../../storage/pebble/README.md) — Pebble store implementations
- [**stack/sqlite**](../sqlite/README.md) — SQLite alternative with SQL query power
