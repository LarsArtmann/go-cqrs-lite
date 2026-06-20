# example/deployer-first

A minimal CQRS/Event-Sourcing app built with the **deployer-first** composition
model: the person deploying the app decides where data lives; the application
code is identical regardless of the choice.

## The split

| Role     | Responsibility                                          | Code            |
| -------- | ------------------------------------------------------- | --------------- |
| Deployer | Pick infrastructure (memory / SQLite / Pebble)          | `main.go`       |
| Consumer | Define the domain + how events build a read model       | `domain.go`, `view.go` |

Swapping memory for SQLite is a one-line change in `main.go`.

## Pipeline

```
Decider → EventBus → CatchUpSubscriber → Materialize
```

- **CatchUpSubscriber** replays the journal from the last checkpoint (ordered),
  then switches to live delivery with EventID-based deduplication.
- **Materialize** turns the ordered event stream into a typed, tombstone-aware,
  queryable view (`mat.View`, `mat.List`).

## Run

```bash
go run .
# todo 01KV…: "Buy milk" completed=true tombstoned=true
```

## Ordering: why no Watermill Router

The CatchUpSubscriber's output channel is a FIFO Go channel. Consuming it from
a **single goroutine** guarantees event ordering (create before update before
delete). Watermill's Router processes messages in **parallel** (one goroutine
per message — documented in `message/router.go:30`), which breaks ordering for
multi-event aggregates. For ordered projections, consume the subscriber channel
directly, as this example does. The Router is appropriate for unordered,
multi-handler, multi-topic routing.

## Startup pattern

Commands execute **before** the projection starts. All events land in the
journal. The CatchUpSubscriber replays them (ordered), then enters live mode
for subsequent events. This is the standard CQRS startup sequence and avoids
the startup race between replay and live subscription.
