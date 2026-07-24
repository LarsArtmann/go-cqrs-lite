# deriver — Event-to-Command Derivation

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/deriver/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/deriver/v4)

Transform events into zero or more derived commands with functional composition. A lightweight alternative to sagas: deterministic derivations wired into the event bus via `bus.SubscribeAll`.

```bash
go get github.com/larsartmann/go-cqrs-lite/deriver/v4
```

## Why?

When an event should trigger downstream commands (e.g., `user.created` triggers `send-welcome-email` and `sync-to-crm`), you need a composable, testable way to express that derivation. The `deriver` package provides pure functions that transform events into commands, with built-in support for fan-out, event-type filtering, and idempotent delivery.

## Quick Start

```go
package main

import (
    "context"

    "github.com/larsartmann/go-cqrs-lite/deriver/v4"
    cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v4"
    cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
)

func main() {
    // A Deriver is a pure function: event → commands
    sendWelcomeEmail := func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
        return []cqrscommand.Command{
            command.New("send-welcome-email", evt.AggregateID(), SendWelcomeEmail{UserID: evt.AggregateID()}),
        }, nil
    }

    syncToCrm := func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
        return []cqrscommand.Command{
            command.New("sync-to-crm", evt.AggregateID(), SyncToCrm{UserID: evt.AggregateID()}),
        }, nil
    }

    // Compose: fan-out (Then) + event-type filter (Filter) + idempotent IDs
    d := sendWelcomeEmail.Then(syncToCrm).
        Filter("user.created").
        Idempotent()

    // Wire into the event bus
    bus.SubscribeAll(d.AsHandler(cmdDispatcher))
}
```

## API

| Symbol                     | Kind   | Description                                                              |
| -------------------------- | ------ | ------------------------------------------------------------------------ |
| `Deriver`                  | Type   | `func(ctx, event.Event) ([]command.Command, error)`                      |
| `Deriver.Then(next)`       | Method | Fan-out: runs both derivers on the same event, concatenates commands.    |
| `Deriver.Filter(types...)` | Method | Only processes events of the given types; others produce no commands.    |
| `Deriver.Idempotent()`     | Method | Re-stamps commands with deterministic IDs derived from the source event. |
| `Deriver.AsHandler(disp)`  | Method | Converts to an `event.Handler` that dispatches commands sequentially.    |
| `Noop()`                   | Func   | Terminal Deriver that produces no commands. Placeholder/default.         |
| `SourceEventIDKey`         | Const  | Custom metadata key stamped on derived commands for traceability.        |

## Idempotent Delivery

Chain `.Idempotent()` before `.AsHandler()` to make derivations safe for at-least-once delivery. Each derived command gets a deterministic `CommandID` derived from the source event's ID and its position in the output slice. Re-processing the same event yields the same command IDs, so an idempotency store keyed on the command ID deduplicates automatically:

```go
myDeriver.Idempotent().AsHandler(dispatcher)
```

The source event's ID is also stamped as custom metadata (`SourceEventIDKey`) on each command for end-to-end traceability.

## Design

- **Deterministic contract**: The same event always produces the same commands. This makes derivations safe for at-least-once delivery.
- **Pure functions**: No side effects beyond returning commands. No I/O, no state mutation.
- **Non-terminating errors propagate**: Partial results are NOT dispatched. If a Deriver errors, the error propagates and no commands are sent.
- **Design rationale (ADR-0040)**: The functional/composable API was chosen over a declarative rule registry because go-cqrs-lite is a library, not a database engine.

## Related Modules

- [**event**](../event/README.md) — `bus.SubscribeAll` is the wiring point for derivers
- [**command**](../command/README.md) — Derived commands are dispatched via `command.Dispatcher`
- [**middleware**](../middleware/README.md) — `CommandIdempotency` deduplicates redelivered derived commands
