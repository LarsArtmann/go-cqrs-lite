# samber/ro Integration — PRO / CONTRA Analysis

**Date:** 2026-04-23
**Status:** Decision pending
**Related:** `event/bus.go`, `event/store.go`, `event/memory_bus.go`

## What is samber/ro?

A Go implementation of the **ReactiveX specification** by [Samuel Berthe](https://github.com/samber) — the same author behind **samber/lo** (~18k stars, one of the most popular Go libraries). Reactive/stream programming with composable operators for handling asynchronous data streams.

| Aspect        | Detail                                                       |
| ------------- | ------------------------------------------------------------ |
| Author        | samber (samber/lo ~18k★, samber/do, samber/mo)              |
| License       | Apache 2.0 (core), custom license (enterprise `ee/`)        |
| Version       | v0.3.0 (pre-1.0, follows SemVer strictly)                   |
| Stars         | ~641                                                         |
| Dependencies  | Minimal beyond stdlib                                        |
| Plugin system | HTTP, WebSocket, Redis, Cron, JSON, CSV, Sentry, Zap, etc.  |

### Core Concepts

| Concept       | Description                                                    |
| ------------- | -------------------------------------------------------------- |
| Observable    | Stream of values emitted over time                             |
| Observer      | Consumes values (`onNext`, `onError`, `onComplete`)            |
| BehaviorSubject | Maintains current value — new subscribers get latest immediately |
| ReplaySubject  | Replays N past values to new subscribers — mirrors event replay |
| PublishSubject | Broadcasts to all current subscribers — fire-and-forget        |
| Operators     | Map, Filter, Merge, Zip, BufferWhen, GroupBy, Retry, etc.     |

### Why it's more than "RxJava for Go"

- **Plugin ecosystem** — Redis pub/sub, WebSocket client/server, HTTP, cron, file watching, structured logging, Sentry, rate limiting. These are production-grade integrations, not toy examples.
- **Real distributed examples** — Distributed WebSocket gateway with Redis, stock price enrichment with live market data, SQL-to-CSV streaming.
- **Generic** — Built on Go 1.18+ generics, type-safe throughout. No `any` escape hatches.
- **Same author as samber/lo** — Track record of maintaining widely-adopted, well-designed Go libraries. Not a weekend experiment.

## Current State

No references to `samber/ro` or any reactive/stream programming library exist in the codebase. The event bus is a simple `Publish()`/`Subscribe()` in-memory model.

## PRO — Adding samber/ro

### Conceptual fit with CQRS/Event Sourcing

1. **Events ARE streams** — Event sourcing is literally "an observable of facts over time." Observable/Observer is the natural abstraction, not a forced fit.
2. **ReplaySubject = event replay** — Core ES concept. New subscribers can replay past events, exactly what `event.Store.Load()` does. Could unify the Store/Bus split.
3. **BehaviorSubject = aggregate state** — Maintains current value derived from history. Maps directly to aggregate root state after applying events.
4. **Composable projections** — `Filter` (events of type X), `Map` (transform events), `GroupBy` (group by aggregate ID), `BufferWhen` (time-windowed aggregation), `MergeAll` (combine multiple aggregate streams) — these are exactly what projection handlers need.
5. **Backpressure** — Built-in strategies for slow consumers. Relevant when event consumers can't keep up with producers.

### Practical benefits

6. **Declarative error handling** — `Catch`, `Retry`, `RetryWhen` for resilient event processing. No hand-written retry loops.
7. **Combining event streams** — `Merge`, `Zip`, `CombineLatest` for projecting across multiple aggregates or event types.
8. **Redis plugin for distribution** — The distributed WebSocket gateway example already demonstrates Redis pub/sub as a transport. Could serve as the multi-instance event bus go-cqrs-lite currently lacks.
9. **Testing support** — `plugins/testify` provides test utilities for observables. Useful for testing event handlers.
10. **Observability built in** — Plugins for Zap, Slog, Zerolog, Sentry. Event processing is inherently observable.

### Author credibility

11. **samber/lo is ubiquitous** — ~18k stars, used by thousands of Go projects. samber has a proven track record of API design, backward compatibility, and long-term maintenance.
12. **Ecosystem coherence** — samber/lo (sync), samber/ro (async), samber/mo (monads), samber/do (DI) cover complementary concerns. A project using lo might naturally reach for ro.

## CONTRA — Adding samber/ro

### Dependency & stability

1. **Pre-1.0** — v0.3.0 means breaking API changes are possible. Risky for a library others depend on. Though samber/lo maintained excellent backward compatibility even pre-1.0.
2. **External dependency** — Goes beyond the current stdlib + uuid + errors policy. However, ro itself has minimal deps (unlike Watermill which pulls in broker clients).

### Paradigm & design

3. **Different mental model** — go-cqrs-lite uses simple, explicit interfaces (`Publish()`, `Subscribe()`, `Handle()`). Reactive programming introduces observable chains, subscription lifecycle, disposal patterns. A shift for users.
4. **Go idiom tension** — Go favors goroutines and channels. ReactiveX is from the callback/async world. That said, samber/ro adapts the paradigm well to Go with generics and clean API design.
5. **Overlap with existing interfaces** — `event.Bus` already has `Publish`/`Subscribe`. Wrapping it in Observable adds indirection. The question is whether the indirection pays for itself.
6. **Enterprise license** — `ee/` directory (OpenTelemetry, Prometheus) under custom license. Core is Apache 2.0 but some observability features require checking the ee terms.

### Scope

7. **Not a message broker** — samber/ro handles in-process stream processing. It doesn't replace Watermill for Kafka/RabbitMQ integration. They solve different problems and could coexist.
8. **May be application-level** — Reactive stream processing might belong in the application layer (how users consume events from go-cqrs-lite), not in the library itself.

## Comparison with Alternatives

| Option                   | Event Replay | Projections | Distribution | Maturity | Complexity |
| ------------------------ | ------------ | ----------- | ------------ | -------- | ---------- |
| Status quo (in-memory)   | Manual       | Manual      | ❌            | ✅        | Low        |
| samber/ro                | ReplaySubject | Operators   | Via Redis plugin | ⚠️ v0.3 | Medium     |
| Go channels              | Manual       | Manual      | ❌            | ✅        | Medium     |
| Watermill                | Manual       | Manual      | ✅ Kafka/NATS/etc | ✅   | High       |

samber/ro and Watermill are **complements, not competitors**:
- **samber/ro** — In-process reactive stream processing (how you transform/react to events)
- **Watermill** — Inter-process message transport (how events move between services)

## Integration Options

### Option A: samber/ro as optional adapter module

```
go-cqrs-lite/                    ← core (zero deps, stays as-is)
go-cqrs-lite/ro/                 ← opt-in module (Observable-backed Bus + Store)
```

An `event.Bus` implementation backed by `PublishSubject`, an `event.Store` using `ReplaySubject` for replay. Users who want reactive operators can `go get` the adapter.

### Option B: Document as recommended companion

Keep go-cqrs-lite dependency-free but provide examples/docs showing how to wrap `event.Bus` subscriptions in samber/ro observables for projection pipelines. ~20 lines of adapter code.

### Option C: Don't recommend

Users who want reactive streams can figure it out themselves. Go channels suffice for most use cases.

## Decision: Implementation detail, not a standalone module

samber/ro is **not** a user-facing module. It's an internal engine for modules that need reactive stream processing.
Users don't wake up wanting "reactive streams" — they want working projections, streaming, and event replay.
samber/ro is *how* we build those, not *what* users import.

## Integration Plan

### Why not a standalone `ro/` module

A standalone `ro/` module would expose `Observable`, `Subject`, and reactive operators as the public API.
This forces users to learn ReactiveX concepts to use go-cqrs-lite — a paradigm shift that doesn't belong at the CQRS layer.

Instead, samber/ro is a **dependency of modules that need it**, encapsulated behind existing CQRS interfaces:

```
core/           # interfaces (Store, Bus, Streamer, Projection) — no ro dep
memory/         # maps + mutexes — no ro dep
projection/     # depends on samber/ro internally — users just call projector.On()
storage/        # SQL-backed store — no ro dep
watermill/      # broker-backed bus — no ro dep
```

Users never see an `Observable` type. If samber/ro is ever swapped out, it's an internal change with no public API breakage.

### Where samber/ro lives

#### `projection/` — Primary consumer

```
projection/
├── go.mod              # deps: core, storage, samber/ro
├── runner.go           # subscribes to events, dispatches to handlers
├── handler.go          # Handler interface, checkpoint tracking
├── checkpoint.go       # stores projection position (SQL-backed)
└── internal/
    └── stream/
        ├── pipeline.go     # ro.Pipe wrappers for event stream processing
        ├── filters.go      # FilterType, FilterAggregate via ro.Filter
        └── windows.go      # time-windowed aggregation via ro.BufferWhen
```

Users call:
```go
projector.On("user.created", func(ctx context.Context, evt event.Event) error {
    return updateUserReadModel(ctx, evt)
})
```

Internally, the Runner uses samber/ro to:
- `Filter` events by type before dispatching to handlers
- `GroupBy` aggregate ID for partitioned processing
- `BufferWhen` for batched projection writes
- `Retry` / `RetryWhen` for resilient event processing (replaces `middleware/retry.go`)
- `Catch` for declarative error handling in projection pipelines

#### Potential future consumers

| Module | What ro would power | Priority |
| ------ | ------------------- | -------- |
| `projection/` | Event stream → read model pipeline | **Now** |
| `core/event/` Streamer impl | `Stream() (<-chan Event)` via Observable → channel | Later |
| `memory/` bus upgrade | ReplaySubject for in-memory event replay | Low priority |

### What this replaces from the current codebase

| Current code                       | What ro replaces internally            | Savings          |
| ---------------------------------- | ------------------------------------ | ---------------- |
| `middleware/retry.go` (~83 lines)  | `ro.Retry` / `ro.RetryWhen` operator | Delete           |
| Hand-written projection loops      | `ro.Pipe(filter, map, groupBy, ...)` | Per projection   |
| `Streamer` (unimplemented)         | Observable → channel adapter         | ~60 lines        |

### Dependency impact

- `core/` — zero-dep (unchanged)
- `memory/` — zero-dep (unchanged, maps + mutexes are fine for testing)
- `projection/` — pulls in `samber/ro` (minimal transitive deps)
- Users who don't use projections never see samber/ro

### If samber/ro is ever replaced

All ro code lives inside `projection/internal/stream/`. Swapping it for Go channels, a different reactive library, or hand-rolled processing is a single-package change. No public API changes. No user code breaks.

### Prerequisite

Depends on the multi-module monorepo migration (Phases 1–5 of `2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md`) — we need `core/`, `memory/`, and `storage/` as separate modules first. The `projection/` module with ro is Phase 7.
