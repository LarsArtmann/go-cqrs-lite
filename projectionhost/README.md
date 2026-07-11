# projectionhost/v3

Managed lifecycle for projection workers — the "last loop every consumer rewrites."

## What It Does

Wraps journal reads, projection handlers, checkpoint tracking, and failure
handling into a single embeddable component:

- **Per-projection goroutines** — each projection runs independently
- **Crash auto-restart** with exponential backoff
- **Checkpoint persistence** — survives restarts, no event loss
- **Dead-letter queue** — poison messages captured, checkpoint advances
- **Health/liveness** — `Status()` reports per-worker state with lag
- **Per-projection lag** — `LagPerProjection()` for dashboards
- **Graceful drain** — `Stop()` waits for in-flight events

## Quick Start

```go
host, _ := projectionhost.New(journal, checkpointStore,
    projectionhost.WithBatchSize(100),
    projectionhost.WithDeadLetterStore(dlqStore, 3),
)
host.Register(&UserProjection{})
host.Register(&OrderProjection{})

go host.Start(ctx)
// ... run your application ...
host.Stop() // graceful drain
```

## Key Types

| Type                    | Purpose                                           |
| ----------------------- | ------------------------------------------------- |
| `Host`                  | Manages projection workers, lifecycle, and health |
| `WorkerState`           | Point-in-time snapshot of a worker's state        |
| `DeadLetterStore`       | Interface for poison-message capture              |
| `MemoryDeadLetterStore` | In-memory DLQ for dev/test                        |

## Configuration Options

| Option                              | Default  | Description                              |
| ----------------------------------- | -------- | ---------------------------------------- |
| `WithMaxRestarts(n)`                | 5        | Max restarts per worker (-1 = unlimited) |
| `WithBackoff(initial, max)`         | 1s, 30s  | Exponential backoff between restarts     |
| `WithBatchSize(n)`                  | 100      | Events read per journal batch            |
| `WithDeadLetterStore(s, threshold)` | disabled | Poison-message capture after N retries   |

## Status & Lag

```go
for _, s := range host.Status() {
    fmt.Printf("%s: %s (processed=%d, errors=%d, lag=%s)\n",
        s.Name, s.Status, s.Processed, s.Errors, s.Lag)
}

// Per-projection lag for dashboards:
for name, lag := range host.LagPerProjection() {
    gauge.WithLabelValues(name).Set(float64(lag.Milliseconds()))
}

// Aggregate lag (max across all workers):
gauge.Set(float64(host.LagDuration().Milliseconds()))
```

Worker states: `idle`, `running`, `live`, `backoff`, `draining`, `stopped`, `failed`.

## Reset

Rebuild a projection from scratch after fixing a handler bug:

```go
host.Stop()
// Drop checkpoint + call Resettable.Reset + optionally purge DLQ:
host.Reset(ctx, "users", projectionhost.WithPurgeDeadLetters())
host.Start(ctx) // replays from zero
```

## Design

The Host reads directly from `event.SeekableJournal` — it does NOT depend on
Watermill or any message bus. This keeps it a pure library component. For live
streaming (push-based event delivery), use the `watermill/CatchUpSubscriber`
alongside the Host.
