# scheduling — Durable Deadline Timers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/scheduling/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/scheduling/v4)

Durable deadline timers for event-sourced systems: "cancel order after 30 minutes unpaid", "send reminder email 24 hours after signup", "expire session after 15 minutes idle".

```bash
go get github.com/larsartmann/go-cqrs-lite/scheduling/v4
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

type CancelOrderCmd struct {
    OrderID string
}

func main() {
    ctx := context.Background()

    store := scheduling.NewMemoryTimerStore[CancelOrderCmd]()
    sched := scheduling.New[CancelOrderCmd](store, func(ctx context.Context, t scheduling.Timer[CancelOrderCmd]) error {
        return commandBus.Dispatch(ctx, t.Payload)
    })

    // Schedule a timer
    _ = store.Schedule(ctx, scheduling.Timer[CancelOrderCmd]{
        ID:      "order-cancel-123",
        FireAt:  time.Now().Add(30 * time.Minute),
        Payload: CancelOrderCmd{OrderID: "123"},
    })

    // Cancel it if the order is paid
    _ = store.Cancel(ctx, "order-cancel-123")

    go sched.Start(ctx) // polls for due timers, dispatches, marks fired
}
```

## API

### Core Types

| Symbol                    | Kind      | Description                                                         |
| ------------------------- | --------- | ------------------------------------------------------------------- |
| `Timer[P]`                | Struct    | A scheduled timer: `ID`, `FireAt`, `Payload P`. Generic payload.    |
| `TimerID`                 | Type      | `= string`                                                          |
| `TimerStore[P]`           | Interface | `Schedule`, `Due`, `MarkFired`, `Cancel`. Persistence boundary.    |
| `DispatchFunc[P]`         | Type      | `func(ctx, Timer[P]) error`. Called when a timer fires.             |

### Scheduler

| Symbol              | Kind   | Description                                                         |
| ------------------- | ------ | ------------------------------------------------------------------- |
| `Scheduler[P]`      | Struct | The poller. Polls the store for due timers and dispatches them.     |
| `New[P](store, fn)` | Func   | Creates a Scheduler with functional options.                        |
| `Start(ctx)`        | Method | Blocks until context is canceled. Polls, dispatches, retries.       |

### Options

| Option                  | Default | Description                                                       |
| ----------------------- | ------- | ----------------------------------------------------------------- |
| `WithPollInterval(d)`   | 1s      | How often to poll for due timers.                                 |
| `WithMaxRetries(n)`     | 3       | Max dispatch retries before leaving the timer due for next poll.  |
| `WithRetryDelay(d)`     | 100ms   | Base for exponential backoff between retries.                     |
| `WithLogger(l)`         | `slog.Default()` | Structured logger.                                      |

### MemoryTimerStore

| Symbol                    | Kind   | Description                                              |
| ------------------------- | ------ | -------------------------------------------------------- |
| `MemoryTimerStore[P]`     | Struct | In-memory `TimerStore` for development and testing.      |
| `NewMemoryTimerStore[P]()`| Func   | Constructor.                                             |

## Design

- **Generic payload `P`**: Compile-time safety and clean JSON round-tripping. Pick a concrete command type instead of untyped `any`.
- **Idempotent scheduling**: `Schedule` with an existing un-fired `TimerID` is a no-op. Safe to call multiple times.
- **Equal-jitter exponential backoff**: `delay = base * 2^attempt / 2 + rand(0..half)`. Guarantees a minimum delay so the downstream system has a recovery window.
- **At-least-once delivery**: Failed dispatch after retries leaves the timer due for the next poll. `MarkFired` only happens after successful dispatch.
- **Context-aware**: Cancellation is honored during both retry loops and backoff sleeps.

## Related Modules

- [**command**](../command/README.md) — Timer payloads are typically commands dispatched via `command.Dispatcher`
- [**projectionhost**](../projectionhost/README.md) — Managed projection lifecycle (scheduling is a separate concern)
