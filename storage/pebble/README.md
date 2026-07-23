# storage/pebble — Embedded Key-Value Persistence (CockroachDB Pebble)

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/storage/pebble/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/storage/pebble/v4)

Full CQRS persistence stack backed by CockroachDB Pebble: event store + journal, snapshot store, checkpoint store, command store, query store, and KV adapter. All sharing one `*pebble.DB` via disjoint key prefixes.

```bash
go get github.com/larsartmann/go-cqrs-lite/storage/pebble/v4
```

## Why?

Pebble is a high-performance embedded LSM key-value store (the engine behind CockroachDB). It provides excellent write throughput, efficient point reads, and built-in compression. This module packages the entire CQRS persistence layer on a single Pebble database, making it ideal for single-node event-sourced services that need disk persistence without a separate database server.

## Quick Start

### Backend Facade (preferred)

```go
backend, err := pebble.Open("data/myapp", pebble.DefaultOptions(), slog.Default())
if err != nil { log.Fatal(err) }
defer backend.Close() // closes DB AND all stores

eventStore := backend.EventStore()
snapStore  := backend.SnapshotStore()
cpStore    := backend.CheckpointStore()
```

### Manual Wiring (advanced)

```go
db, err := pebble.Open("data/myapp", &pebble.Options{})
if err != nil { log.Fatal(err) }

eventStore, _ := pebble.NewStore(db, logger)
snapStore, _  := pebble.NewSnapshotStore(db, logger)
cpStore, _    := pebble.NewCheckpointStore(db, logger)
```

### Pebble as kv.Store

```go
kvStore, _ := pebble.NewKVStore(db, pebble.WithSyncWrites())
defer kvStore.Close()

kvStore.Set([]byte("key"), []byte("value"))
val, _ := kvStore.Get([]byte("key"))
```

## API

### Backend Facade

| Symbol                              | Description                                          |
| ----------------------------------- | ---------------------------------------------------- |
| `Open(dir, opts, logger)`           | Creates and owns the DB. Recommended entry point.    |
| `NewBackend(db, logger)`            | Wraps an external DB (caller owns lifecycle).        |
| `Backend.EventStore()`              | Event store (implements `event.Store`, `Journal`).   |
| `Backend.SnapshotStore()`           | Snapshot store.                                      |
| `Backend.CheckpointStore()`         | Checkpoint store.                                    |
| `Backend.Close()`                   | Closes DB AND all stores.                            |
| `Backend.Metrics()`                 | LSM health metrics (block cache, compactions).       |
| `Backend.Checkpoint(dir)`           | Point-in-time physical backup snapshot.              |
| `Backend.NewSnapshot()`             | Consistent read view (close when done).              |

### Options

| Symbol                          | Description                                          |
| ------------------------------- | ---------------------------------------------------- |
| `DefaultOptions()`              | Production-grade: bloom filter, concurrent compactions. |
| `DefaultOptionsWithLogging(l)`  | Same as default + operational logging.               |

### KV Adapter

| Symbol                  | Description                                          |
| ----------------------- | ---------------------------------------------------- |
| `NewKVStore(db, opts)`  | Adapts Pebble as a `kv.Store`.                        |
| `WithSyncWrites()`      | Force synchronous writes.                            |
| `WithBorrowedDB()`      | Adapter does not close the DB (shared via Backend).  |

## Design

- **Disjoint key prefixes**: `cqrs_event:`, `cqrs_journal:`, `cqrs_snapshot:`, `cqrs_checkpoint:` — no prefix is a substring of another, guaranteeing safe key-space isolation.
- **CBOR envelope**: Canonical/deterministic encoding (RFC 7049). Payloads stored as raw bytes (no base64 overhead). Legacy JSON envelopes supported via format sniffing.
- **Optimistic concurrency**: Per-aggregate sharded mutex pool for write isolation.
- **Backend owns the DB**: Unlike the SQL backend (which borrows `*sql.DB`), `Backend.Close()` closes the Pebble DB itself. After `Close()`, all store ops return `ErrClosed`.
- **Nil-safe logger**: Pass `nil` for `*slog.Logger` to disable logging.

## Operations

### Backup

```go
err := backend.Checkpoint("backups/2026-01-15")
```

### LSM Health Check

```go
m := backend.Metrics()
hitRate := float64(m.BlockCacheHits) /
    float64(m.BlockCacheHits + m.BlockCacheMisses)
```

## Related Modules

- [**storage**](../README.md) — SQL-based alternative (SQLite/Postgres)
- [**kv**](../../kv/README.md) — `kv.Store` interface that the KV adapter implements
- [**stack/pebble**](../../stack/pebble/README.md) — All-in-one Pebble stack preset
- [**event**](../../event/README.md) — Store, Journal interfaces
