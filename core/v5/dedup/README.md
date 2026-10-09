# dedup — Bounded Ring Buffer for ID Deduplication

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/dedup/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/dedup/v4)

A fixed-capacity ring buffer for deduplicating short-lived IDs at stream boundaries, such as the replay-to-live handoff in projection catch-up or SSE reconnection.

```bash
go get github.com/larsartmann/go-cqrs-lite/dedup/v4
```

## Why?

When a projection replays historical events from a journal and then switches to a live stream, the same events may appear in both phases. The ring retains only the most recently seen IDs, evicting the oldest when full. Both `Add` and `Has` are O(1), and memory is bounded at capacity regardless of how many IDs flow through over the ring's lifetime.

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/dedup/v4"
)

func main() {
    ring := dedup.NewRing(dedup.DefaultCapacity)

    ring.Add("event-001")
    ring.Add("event-002")

    ring.Has("event-001") // true
    ring.Has("event-999") // false

    // Once the ring fills, the oldest ID is evicted:
    for i := 0; i < dedup.DefaultCapacity; i++ {
        ring.Add(fmt.Sprintf("event-%03d", i))
    }
    ring.Has("event-001") // false — evicted
}
```

## API

| Symbol              | Kind   | Description                                                         |
| ------------------- | ------ | ------------------------------------------------------------------- |
| `Ring`              | Struct | Fixed-capacity set of string IDs. Add/Has are O(1).                 |
| `NewRing(capacity)` | Func   | Creates a Ring. Falls back to `DefaultCapacity` if `capacity <= 0`. |
| `Ring.Add(id)`      | Method | Inserts an ID. No-op if already present. Evicts oldest when full.   |
| `Ring.Has(id)`      | Method | Reports whether the ID is in the ring. Nil-safe (returns false).    |
| `Ring.Len()`        | Method | Number of IDs currently held. Nil-safe (returns 0).                 |
| `Ring.Capacity()`   | Method | Maximum number of IDs the ring can hold. Nil-safe (returns 0).      |
| `DefaultCapacity`   | Const  | 1024 — a sensible ring size for replay-to-live dedup (~90 KB).      |

## Design

- **Not safe for concurrent use.** Callers sharing a ring across goroutines must synchronize externally.
- **Nil-safe receiver.** A nil `*Ring` returns `false` from `Has` and `0` from `Len`/`Capacity`, so callers can pass nil when no replay occurred.
- **DefaultCapacity rationale:** Overlapping events always cluster at the tail of the replay sequence. The live channel buffer is typically bounded at 100-256, so a 1024-entry ring gives a 4-10x safety margin while bounding memory to ~90 KB.

## Related Modules

- [**projectionhost**](../projectionhost/README.md) — Uses the ring for replay-to-live deduplication
- [**watermill**](../watermill/README.md) — `CatchUpSubscriber` uses the ring at the replay-to-live boundary
- [**transport/http**](../transport/http/README.md) — `SSEBroker` uses the ring for SSE reconnection dedup
