# storage/bbolt

Embedded B+tree event store for go-cqrs-lite, backed by [bbolt](https://github.com/etcd-io/bbolt) (the etcd team's pure-Go key-value store).

## Why bbolt?

bbolt uses a single-writer B+tree model. Unlike Pebble (LSM tree, concurrent writes, bloom filters), bbolt provides:

- **Predictable latency** — no compaction spikes, no write amplification
- **Excellent point-read performance** — B+tree leaves are always the live data
- **Pure Go** — no CGo required
- **Atomic multi-bucket writes** — Save's version check + event write happen in one transaction

The single-writer model means writes are fully serialized. This eliminates the need for per-stream locking (Pebble uses a 256-shard mutex pool).

## Stores

| Store           | Bucket                               | Interface                                         |
| --------------- | ------------------------------------ | ------------------------------------------------- |
| EventStore      | `cqrs_events` + `cqrs_journal`       | `event.Store`, `event.SeekableJournal`            |
| SnapshotStore   | `cqrs_snapshots`                     | `snapshot.Store`                                  |
| CheckpointStore | `cqrs_checkpoints`                   | `event.CheckpointStore`                           |
| KVAdapter       | `cqrs_kv`                            | `kv.Store`                                        |
| CommandStore    | `cqrs_commands` + `cqrs_cmd_journal` | `command.Store`, `command.SeekableCommandJournal` |
| QueryStore      | `cqrs_queries`                       | `query.QueryStore`, `query.SeekableQueryJournal`  |

## Usage

```go
import cqrsbbolt "github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4"

backend, err := cqrsbbolt.Open("myapp.db", slog.Default())
defer backend.Close()

eventStore := backend.EventStore()
snapshotStore := backend.SnapshotStore()
checkpointStore := backend.CheckpointStore()
readModels := backend.ReadModels()    // kv.Store
commandStore := backend.CommandStore()
queryStore := backend.QueryStore()
```

### Custom Options

```go
backend, err := cqrsbbolt.OpenWith("myapp.db", &bbolt.Options{
    Timeout: 10 * time.Second,
    NoSync:  false,
}, slog.Default())
```

## Serialization

All stores use CBOR envelopes (via `fxamacker/cbor`) with JSON fallback on read for backward compatibility. Events are self-describing via the `encoding` stamp.

## Durability

bbolt syncs to disk on every transaction commit by default. For relaxed durability (data loss possible on crash), use `OpenWith` with `NoSync: true` and `NoFreelistSync: true`, or use the `stack/bbolt` preset with `WithDurability(stack.DurabilityRelaxed)`.
