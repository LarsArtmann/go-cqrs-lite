# projectionhost/v3

Managed lifecycle for projection workers — the "last loop every consumer rewrites."

## What It Does

Wraps journal reads, projection handlers, checkpoint tracking, and failure
handling into a single embeddable component:

- **Per-projection goroutines** — each projection runs independently
- **Crash auto-restart** with exponential backoff
- **Checkpoint persistence** — survives restarts, no event loss
- **Dead-letter queue** — poison messages captured, checkpoint advances
- **Health/liveness** — `Status()` reports per-worker state
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

## Status

```go
for _, s := range host.Status() {
    fmt.Printf("%s: %s (processed=%d, errors=%d, restarts=%d)\n",
        s.Name, s.Status, s.Processed, s.Errors, s.Restarts)
}
```

Worker states: `idle`, `running`, `backoff`, `draining`, `stopped`, `failed`.

## Design

The Host reads directly from `event.SeekableJournal` — it does NOT depend on
Watermill or any message bus. This keeps it a pure library component. For live
streaming (push-based event delivery), use the `watermill/CatchUpSubscriber`
alongside the Host.
