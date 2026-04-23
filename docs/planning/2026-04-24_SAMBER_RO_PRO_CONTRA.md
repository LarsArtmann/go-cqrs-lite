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

## Recommendation

samber/ro is a legitimate, well-designed library from a proven author that maps well to event-driven patterns. The question isn't "is it good?" — it's "where does it belong?"

**Recommended: Option B (document as companion).** Don't add it as a dependency, but provide a documented example showing how `event.Bus` subscriptions can be lifted into samber/ro observables. This gives users the reactive toolkit without committing to it as a dependency — and the adapter code is trivial (~20 lines).

## Decision: Integrate as a separate module in the multi-module monorepo

samber/ro will be added as a new `ro/` module alongside `storage/`, `nats/`, etc. in the existing multi-module plan.

## Integration Plan

### Where it fits in the monorepo

```
go-cqrs-lite/
├── core/              # interfaces only (Store, Bus — no MemoryStore)
├── memory/            # MemoryStore, MemoryBus (current behavior)
├── ro/                # NEW — samber/ro-backed implementations
│   └── go.mod         # deps: core, samber/ro
│   ├── bus.go         # ObservableBus  — implements event.Bus
│   ├── store.go       # ReplayStore    — implements event.Store
│   ├── streamer.go    # ObservableStreamer — implements event.Streamer
│   └── projection.go  # Helper functions for building projections with ro operators
├── storage/           # SQL-backed event store
├── nats/              # NATS-backed event bus
└── ...
```

### What each file implements

#### `ro/bus.go` — ObservableBus

Implements `event.Bus` backed by `PublishSubject[event.Event]`:
- `Publish()` → calls `subject.Next(event)` for each event
- `Subscribe(type, handler)` → creates a filtered observable (`ro.Filter` by event type), subscribes handler as observer
- `SubscribeAll(handler)` → subscribes handler directly to subject
- `Observable()` → returns the underlying `ro.Observable[event.Event]` for users who want composable operators
- `Use(middleware)` — still supports existing `event.Middleware` chain

#### `ro/store.go` — ReplayStore

Implements `event.Store` backed by per-aggregate `ReplaySubject[event.Event]`:
- `Save()` → validates expected version, calls `subject.Next(event)` for each new event
- `Load()` → collects all events from the ReplaySubject (replays history)
- `LoadFromVersion()` → filters replayed events by version
- `AppendBatch()` → appends without OCC check
- `Delete()` → completes and removes the subject
- ReplaySubject gives us event replay for free — new subscribers get all past events

#### `ro/streamer.go` — ObservableStreamer

Implements `event.Streamer` (currently defined but unimplemented):
- `Stream()` → returns `<-chan Event` derived from a filtered observable
- Uses `ro.Filter` for `StreamOptions.AggregateType`
- Uses `ro.Skip` for `StreamOptions.FromVersion`
- Uses `ro.BufferWhen` for `StreamOptions.BatchSize`

#### `ro/projection.go` — Projection helpers

Utility functions for building projections with ro operators:
- `FilterType(observable, eventType)` — filter events by type
- `FilterAggregate(observable, aggType, aggID)` — filter by aggregate
- `Project(observable, fn)` — map events to a read model, maintaining state
- `Window(observable, duration)` — time-windowed aggregation
- These are thin wrappers around ro operators with CQRS-friendly signatures

### What this replaces from the current codebase

| Current code                       | What ro module replaces              | Savings          |
| ---------------------------------- | ------------------------------------ | ---------------- |
| `middleware/retry.go` (~83 lines)  | `ro.Retry` / `ro.RetryWhen` operator | Delete           |
| `MemoryBus` replay (doesn't exist) | `ReplaySubject` built-in             | Don't have to write |
| `Streamer` (unimplemented)         | Observable → channel adapter         | ~60 lines        |
| Hand-written projection loops      | `ro.Pipe(filter, map, groupBy, ...)` | Per projection   |

### Dependency impact

Only users who `go get github.com/larsartmann/go-cqrs-lite/ro` pull in samber/ro.
Core remains zero-dep. Memory module remains as-is for simple use cases.

### Migration path for existing users

No breaking changes. Users can:
1. Keep using `memory/` — nothing changes
2. Switch to `ro/` for reactive capabilities
3. Mix: `memory/` for Store, `ro/` for Bus (they're independent)

### Phased implementation

1. **Phase 1** — `ro/bus.go` (ObservableBus) + tests
2. **Phase 2** — `ro/store.go` (ReplayStore) + tests
3. **Phase 3** — `ro/streamer.go` (ObservableStreamer) + tests
4. **Phase 4** — `ro/projection.go` helpers + tests
5. **Phase 5** — Evaluate replacing `middleware/retry.go` with ro operators
6. **Phase 6** — Update examples to show ro usage alongside memory

### Prerequisite

Depends on the multi-module monorepo migration (Phase 1–2 of `2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md`) being at least partially complete — we need `core/` and `memory/` as separate modules first. The `ro/` module then follows the same pattern as `storage/` and `nats/`.
