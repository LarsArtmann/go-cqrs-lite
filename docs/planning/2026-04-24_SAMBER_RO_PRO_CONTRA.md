# samber/ro Integration — PRO / CONTRA Analysis

**Date:** 2026-04-23
**Status:** Decision pending
**Related:** `event/bus.go`, `event/store.go`, `event/memory_bus.go`

## What is samber/ro?

A Go implementation of the **ReactiveX specification** — reactive/stream programming with composable operators for handling asynchronous data streams. Think RxJS/RxJava but for Go.

| Aspect       | Detail                                                    |
| ------------ | --------------------------------------------------------- |
| License      | Apache 2.0 (core), custom license (enterprise `ee/`)     |
| Version      | v0.3.0 (pre-1.0, breaking changes possible)              |
| Stars        | ~641                                                      |
| Dependencies | Minimal beyond stdlib                                     |
| API          | Observable, Observer, Operators, Subjects                 |

### Core Concepts

| Concept       | Description                                               |
| ------------- | --------------------------------------------------------- |
| Observable    | Stream of values emitted over time                        |
| Observer      | Consumes values (`onNext`, `onError`, `onComplete`)       |
| Subject types | `Behavior` (current value), `Replay` (replay history), `Publish` (broadcast), `Unicast` (single observer) |
| Operators     | Map, Filter, Merge, Zip, Buffer, Retry, Timeout, etc.    |

## Current State

No references to `samber/ro` or any reactive/stream programming library exist in the codebase. The event bus is a simple `Publish()`/`Subscribe()` in-memory model.

## PRO — Adding samber/ro

1. **Natural event stream model** — Events are inherently streams over time. Observable/Observer maps cleanly to the event sourcing paradigm where consumers react to new events.
2. **Event replay built in** — `ReplaySubject` mirrors event store behavior (replaying past events to new subscribers), potentially simplifying the `MemoryStore` + `MemoryBus` split.
3. **Composable event pipelines** — Operators like `Filter`, `Map`, `Buffer`, `Debounce`, `Throttle` could replace hand-written event processing logic in consumer code.
4. **Backpressure handling** — ReactiveX has built-in strategies for slow consumers / fast producers, relevant for high-throughput event streams.
5. **Error handling in streams** — `Catch`, `Retry`, `RetryWhen` operators provide declarative error recovery for event processing.
6. **Combining streams** — `Merge`, `Zip`, `CombineLatest` could be useful for projecting across multiple aggregate event streams.
7. **Minimal core dependencies** — The core library has few external deps, somewhat aligned with the zero-dep philosophy.

## CONTRA — Adding samber/ro

1. **Breaks zero-dependency principle** — Even with minimal deps, it's still an external library. The project's identity is built on stdlib + uuid + errors.
2. **Different paradigm** — go-cqrs-lite uses simple, explicit interfaces (`Publish()`, `Subscribe()`, `Handle()`). Reactive programming introduces a fundamentally different mental model (observable chains, subscription lifecycle, disposal). This would be a paradigm shift for users.
3. **Pre-1.0 stability** — v0.3.0 means breaking changes are expected. Not suitable for a library that others depend on.
4. **Over-engineering for current scope** — The project is a lightweight CQRS library. ReactiveX is designed for complex stream processing (real-time systems, sensor data, WebSocket gateways). Most CQRS users don't need `Buffer`, `Debounce`, or `CombineLatest` on their event bus.
5. **Overlap with existing interfaces** — `event.Bus` already has `Publish`/`Subscribe`. Wrapping it in Observable adds a layer of indirection without clear benefit for the core use case.
6. **Dual licensing risk** — Enterprise features under `ee/` with a custom license could create confusion about what's safe to use.
7. **Go idioms** — Reactive programming is more natural in languages with async/await or callback cultures. Go favors goroutines and channels for concurrency. samber/ro works against the grain of idiomatic Go.
8. **Community size** — 641 stars suggests a niche library. Risk of abandonment or slow maintenance.

## Comparison with Alternatives

| Option                   | Fits Zero-Dep | Maturity | Idiomatic Go | Complexity  |
| ------------------------ | ------------- | -------- | ------------ | ----------- |
| Status quo (in-memory)   | ✅             | ✅        | ✅            | Low         |
| samber/ro                | ❌             | ⚠️ v0.3  | ❌            | High        |
| Go channels (std lib)    | ✅             | ✅        | ✅            | Medium      |
| Watermill (for brokers)  | ❌             | ✅        | ✅            | High        |

## Recommendation

**Do not integrate samber/ro.** The reactive paradigm doesn't align with go-cqrs-lite's design philosophy of simple, explicit, idiomatic Go interfaces. If stream processing capabilities are needed in the future, consider:

1. **Go channels** — Already provide composable, idiomatic stream processing. A channel-based `event.Bus` implementation would give many of the same benefits without external dependencies.
2. **Optional adapter module** — If users want reactive streams, they can wrap the `event.Bus` interface in an Observable themselves with minimal code. No library support needed.

The project should stay focused on being a lightweight, idiomatic Go CQRS library — not a reactive framework.
