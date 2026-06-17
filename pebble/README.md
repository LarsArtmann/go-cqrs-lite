# pebble — Embedded Key-Value CQRS Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/pebble/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/pebble/v2)

Event, snapshot, and checkpoint stores backed by [CockroachDB Pebble](https://github.com/cockroachdb/pebble).
A single `*pebble.DB` can back the full CQRS stack via disjoint key prefixes.

```bash
go get github.com/larsartmann/go-cqrs-lite/pebble/v2
```

## Stores

| Store             | Implements                                                              | Key prefix         |
| ----------------- | ----------------------------------------------------------------------- | ------------------ |
| `EventStore`      | `event.Store` + `event.Journal` + `event.SeekableJournal`               | `cqrs_event:`      |
| `SnapshotStore`   | `snapshot.SnapshotSink` + `SnapshotSource` (= `snapshot.SnapshotStore`) | `cqrs_snapshot:`   |
| `CheckpointStore` | `event.CheckpointSink` + `CheckpointSource` (= `event.CheckpointStore`) | `cqrs_checkpoint:` |

The `cqrs_journal:` prefix holds the global event log used by `ReadAll` / `ReadFrom`.

## Quick Start

```go
db, _ := pebble.Open("data", &pebble.Options{})

// One DB can back the full stack — the three stores share it via disjoint key prefixes.
eventStore := pebble.NewStore(db, slog.Default())
snapStore  := pebble.NewSnapshotStore(db, slog.Default())
cpStore    := pebble.NewCheckpointStore(db, slog.Default())

// Events: optimistic concurrency via per-aggregate sharded mutexes.
_ = eventStore.Save(ctx, ref, events, expectedVersion)

// Snapshots: one per aggregate; older versions silently ignored on Save.
_ = snapStore.Save(ctx, snapshot.Snapshot{
    AggregateID: aggID, AggregateType: "User",
    Version: event.Version(10), State: stateBytes, CreatedAt: time.Now(),
})

// Checkpoints: zero-value returned for missing projections (first-run detection).
cp, _ := cpStore.Load(ctx, "user-projection")
if cp.IsZero() {
    // No prior progress — start from the beginning of the journal.
}
```

## Storage Format

All envelopes use canonical CBOR via `fxamacker/cbor` (sorted map keys, shortest
floats). The reader falls back to JSON if the leading byte is not a CBOR map
header, so snapshots/checkpoints/events written before the CBOR migration are
still readable.

| Field         | Encoding                         |
| ------------- | -------------------------------- |
| Event payload | Raw bytes (no base64)            |
| Timestamps    | `int64` UnixNano                 |
| Event version | Zero-padded `(%010d)` key suffix |

## Async Writes

Trade durability for throughput when data is replayable (caches, projections):

```go
store    := pebble.NewStore(db, logger, pebble.WithAsyncWrites())
snap     := pebble.NewSnapshotStore(db, logger, pebble.WithSnapshotAsyncWrites())
checkpoint := pebble.NewCheckpointStore(db, logger, pebble.WithCheckpointAsyncWrites())
```

## Snapshot Semantics

`SnapshotStore` matches the in-memory reference implementation:

- `Save` ignores snapshots whose version is **older** than the currently stored
  one (prevents state regressions).
- `Load` returns `snapshot.ErrSnapshotNotFound` when no snapshot exists.
- `LoadAtVersion(ref, maxVersion)` returns the snapshot **iff** its version is
  `<=` the requested version (honors the "at or before" contract).
- `Delete` is idempotent.

## Checkpoint Semantics

- `Save` overwrites unconditionally (matches the SQL store).
- `Load` returns a zero-value `event.Checkpoint` for missing projections so
  callers can distinguish "first run" without error inspection.
- Empty projection names are rejected with a classified `Rejection` error.

## Lifecycle

`SnapshotStore.Close` and `CheckpointStore.Close` are no-ops — the caller owns
the `*pebble.DB` lifetime. This matches the SQL stores' "no-op Close" convention
and lets multiple stores share one DB without use-after-close hazards.

`EventStore.Close` **does** close the underlying DB; the caller must not share
that DB with other stores afterward. If you need shared-DB usage, do not call
`EventStore.Close` and let the parent process release the DB instead.

## Related Modules

- [**event/v2**](../event/README.md) — `Store`, `Journal`, `SeekableJournal`, `CheckpointStore` interfaces implemented here
- [**snapshot/v2**](../snapshot/README.md) — `SnapshotStore` interface implemented here
- [**decider/v2**](../decider/README.md) — Provides `EventStore` + `SnapshotStore` to the aggregate repository
- [**projection/v2**](../projection/README.md) — Provides `EventStore` + `CheckpointStore` to the projection runner
- [**kv/v2**](../kv/README.md) — `pebble.NewKVStore` implements the generic `kv.Store` interface
- [**storage/v2**](../storage/README.md) — Sibling backend (PostgreSQL/SQLite)
- [**memory/v2**](../memory/README.md) — In-memory reference implementations
- [**codec/v2**](../codec/README.md) — CBOR codec used for the on-disk envelope format
