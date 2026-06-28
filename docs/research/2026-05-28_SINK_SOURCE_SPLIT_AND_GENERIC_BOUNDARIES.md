# Sink/Source Split, Generic Boundaries, and Interface Design
\n> **Status:** IMPLEMENTED — ISP split shipped (EventSink/EventSource)

**Date:** 2026-05-28
**Sources:**

- [Zig's New Async I/O](https://andrewkelley.me/post/zig-new-async-io-text-version.html) — Andrew Kelley, Zigtoberfest 2025
- [Watermill CQRS Component](https://watermill.io/docs/cqrs/)
- [Watermill Pub/Sub Interface](https://watermill.io/development/pub-sub-implementing/)
- Internal review of `go-cqrs-lite` core interfaces (`event.Store`, `event.Bus`, `decider.Repository`)

---

## 1. Zig's Async I/O: Core Lessons

### 1.1 Explicit I/O Interface (like Allocator)

Zig passes `Io` explicitly through the application (`io: Io`), not hidden in a global event loop. Reusable code accepts `Io` if it needs I/O, just like it accepts `Allocator` if it needs allocation.

> "Setting up a std.Io implementation is a lot like setting up an allocator. You typically do it once, in main(), and then pass the instance throughout the application."

**Implication for us:** Dependencies (storage, event bus) should be explicit parameters rather than hidden globals. Our `Repository` already does this with `Store` and `Publisher` — the lesson is to keep doing it and extend the pattern.

### 1.2 Async ≠ Concurrency

Zig makes this explicit with two different APIs:

| Primitive                 | Semantics                 | Use Case                        |
| ------------------------- | ------------------------- | ------------------------------- |
| `io.async(fn, args)`      | Decouple call from return | Non-blocking I/O expression     |
| `io.concurrent(fn, args)` | Actually run in parallel  | Producer-consumer with blocking |

Using `async` with an unbuffered queue on a single-threaded pool deadlocks — you asked for asynchrony, but what you needed was concurrency. The runtime can optimize async (e.g., two sleeps in one second), but it cannot create parallelism without explicit `concurrent`.

**Implication for us:** Our `Store` interface conflates read and write capabilities. A projection only needs reads; a repository needs both. The split makes this explicit.

### 1.3 Cancellation as a Core Primitive

`cancel` has identical semantics to `await` plus a cancellation request. Combined with `defer`, it eliminates the "early return skips cleanup" bug:

```zig
var a = io.async(doWork, .{...});
defer a.cancel(io) catch {};  // always cleaned up
try a.await(io);
```

Both `cancel` and `await` are idempotent with respect to themselves and each other.

**Implication for us:** Go's `context.Context` provides cancellation, but our interfaces don't expose it well. `Source` should accept `ctx` on every call (already does). The deeper lesson: cleanup must be structured and automatic, not opt-in.

### 1.4 Start Simple, Optimize Later

Zig ships a threaded implementation first (no io_uring, no kqueue), then adds stackful coroutines and io_uring as drop-in replacements. Same interface, better performance.

**Implication for us:** The `Sink`/`Source` split enables exactly this. Start with `memory.MemoryStore` (simple), swap to `storage.SQLEventStore` (durable), then add `storage.PebbleEventStore` (fast). All satisfy the same interface.

---

## 2. Store → Sink + Source Split

### 2.1 Current Problem

`event.Store` has 8 methods. Consumers that only read must depend on the full interface:

```go
type Store interface {
    io.Closer
    Save(ctx, aggType, aggID, events, expectedVersion) error
    AppendBatch(ctx, aggType, aggID, events) error
    Load(ctx, aggType, aggID) ([]Event, error)
    LoadFromVersion(ctx, aggType, aggID, version) ([]Event, error)
    LoadToVersion(ctx, aggType, aggID, maxVersion) ([]Event, error)
    LoadToTimestamp(ctx, aggType, aggID, maxTime) ([]Event, error)
    Delete(ctx, aggType, aggID) error
}
```

A projection runner only calls `Load*` methods. A repository only calls `Save`/`AppendBatch`/`Delete` on the write path. Yet both depend on the entire interface.

### 2.2 Proposed Split

Exactly how `Bus` already splits into `Publisher` + `Subscriber`:

```go
// Sink is the write-only side of event persistence.
type Sink interface {
    io.Closer
    Save(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID,
        events []Event, expectedVersion Version) error
    AppendBatch(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID,
        events []Event) error
    Delete(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) error
}

// Source is the read-only side of event retrieval.
type Source interface {
    io.Closer
    Load(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID) ([]Event, error)
    LoadFromVersion(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID,
        version Version) ([]Event, error)
    LoadToVersion(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID,
        maxVersion Version) ([]Event, error)
    LoadToTimestamp(ctx context.Context, aggregateType AggregateType, aggregateID id.AggregateID,
        maxTime time.Time) ([]Event, error)
}

// Store composes Sink and Source for consumers that need both.
type Store interface {
    Sink
    Source
}
```

### 2.3 Deployment Boundary

| Capability | Characteristic                               | Deployment    |
| ---------- | -------------------------------------------- | ------------- |
| `Sink`     | Sequential, consistency-bound, single-writer | Write master  |
| `Source`   | Parallel, cacheable, replay-friendly         | Read replicas |

By splitting the interface, you make this scaling boundary **visible in the type system**. A function accepting `Source` can safely be routed to a read replica. A function accepting `Sink` must go to the write master.

### 2.4 Wrapper Types Should Depend on Minimal Interface

| Wrapper              | Current Dependency | Should Be                                    |
| -------------------- | ------------------ | -------------------------------------------- |
| `VersionedStore`     | `Store`            | `Source` (only overrides Load methods)       |
| `StoreStreamAdapter` | `Store`            | `Source` (only calls Load)                   |
| `projection.Runner`  | `Store`            | `Source` + `PositionalLoader` (never writes) |

### 2.5 Repository Can Separate Concerns

```go
// Before: Repository forces both paths through one Store
type Repository[State any] struct {
    store     event.Store   // write path only uses Save, read path only uses Load
    publisher event.Publisher
    outbox    event.Outbox
    // ...
}

// After: Repository internally separates sink from source
type Repository[State any] struct {
    sink          event.Sink
    source        event.Source
    publisher     event.Publisher
    outbox        event.Outbox
    snapshotStore event.SnapshotStore
    // ...
}

// Convenience constructor for the common case where sink == source:
func NewRepositoryWithStore[State any](
    store event.Store,   // Store = Sink + Source
    publisher event.Publisher,
    decider Decider[State],
    opts ...RepositoryOption[State],
) (*Repository[State], error) {
    return NewRepository(sink: store, source: store, publisher, decider, opts...)
}
```

### 2.6 Test Helpers Split for Lighter Mocks

`FakeStore` currently implements 8 methods. Most tests only need a subset:

```go
// New test helpers
type FakeSink struct { /* Save, AppendBatch, Delete */ }
type FakeSource struct { /* Load, LoadFromVersion, LoadToVersion, LoadToTimestamp */ }

// FakeStore becomes the composition, reusing both:
type FakeStore struct {
    *FakeSink
    *FakeSource
}

// A test for a read-only projection now only needs:
proj := NewRunner(testhelpers.NewFakeSource(), /* ... */)
```

### 2.7 Backward Compatibility

Because Go uses structural typing, existing implementations need **zero changes**:

```go
// MemoryStore already satisfies Store, Sink, Source, GlobalLoader, etc.
var _ event.Store = (*memory.MemoryStore)(nil)
var _ event.Sink = (*memory.MemoryStore)(nil)   // automatic
var _ event.Source = (*memory.MemoryStore)(nil)  // automatic
```

Existing code using `event.Store` continues to work. The win is that **new code and refactored consumers** can now depend on the minimal interface they need.

---

## 3. Save vs Flush: Different Contracts

|                  | `Save(aggType, aggID, events, expectedVersion)` | `Flush()`                           |
| ---------------- | ----------------------------------------------- | ----------------------------------- |
| **Scope**        | Single aggregate                                | Entire buffered batch               |
| **Concurrency**  | Optimistic (per-aggregate version check)        | Atomic (all-or-nothing)             |
| **Timing**       | Immediate                                       | Deferred                            |
| **Error domain** | `ErrVersionConflict` on _this_ aggregate        | `ErrFlushFailed` on the whole batch |
| **State held**   | None (stateless)                                | Buffered events (stateful)          |

Event sourcing's core invariant is **per-aggregate optimistic concurrency**. `Save` encodes that directly: "append these events to _this_ stream if and only if its version is still `expectedVersion`." A generic `Flush()` loses that granularity — it becomes a bulk operation that can't reason about individual aggregate versions.

### 3.1 Where Flush Makes Sense

Flush shines when you want **multi-aggregate transactions** — a pattern the current `TransactionalStore` already supports, but only for the outbox:

```go
// Current: atomic save+outbox for ONE aggregate
TransactionalStore.SaveWithOutbox(ctx, aggType, aggID, events, ver)

// Flush-based: buffer multiple aggregates, flush as one transaction
sink.Append("User", userID, userEvents)    // buffered
sink.Append("Order", orderID, orderEvents) // buffered
sink.Flush(ctx)                            // one tx, all aggregates
```

This is useful for process managers or sagas that mutate multiple aggregates atomically. But it requires the Sink to hold state, which introduces:

- Lifecycle management (when does the buffer reset?)
- Error handling (what happens to buffered events on panic?)
- Scope ambiguity (is `Flush()` per-call-site or global?)

### 3.2 Sink/Source is Orthogonal to Flush

The Zig lesson is **"depend on the smallest capability you need."** `Flush` doesn't replace this — it adds a _different_ capability layer:

```
Source (read)          Sink (write)              BatchSink (write+buffer)
    │                      │                            │
    │  Load()              │  Save()                    │  Append()
    │  LoadFromVersion()   │  AppendBatch()             │  Flush()
    │  LoadToVersion()     │  Delete()                  │
    └──────────────────────┴────────────────────────────┘
                           Store (Sink + Source)
```

A `BatchSink` could embed `Sink` and add `Flush()`:

```go
type BatchSink interface {
    Sink
    Append(ctx context.Context, aggType AggregateType, aggID id.AggregateID, events []Event) error
    Flush(ctx context.Context) error
}
```

But most consumers (Repositories, projectors, read models) don't need batching. They need the **capability guarantee**: "I only read" or "I only write." That's what Sink/Source gives you.

### 3.3 Hybrid: Sink with Explicit Flush

Make the transaction boundary explicit in the Sink:

```go
// Optional: for consumers that need multi-aggregate atomicity
type AtomicSink interface {
    Sink
    Begin() TxSink
}

type TxSink interface {
    Sink
    Flush(ctx context.Context) error  // commit
    Cancel() error                    // rollback
}
```

This mirrors how databases work: you have a connection (`Sink`) and you can start transactions (`Begin()`) that get flushed. But the base `Sink` stays stateless and simple.

The current `TransactionalStore` is actually a manually-coupled version of `AtomicSink` (Save + Outbox flush in one transaction). Making that explicit with a `Begin()...Flush()` API would be cleaner — but that's a layer _on top of_ Sink, not a replacement for it.

---

## 4. Watermill Interface Design Lessons

### 4.1 Transport vs Application Layers

Watermill's architecture separates **untyped transport** (`message.Publisher` / `message.Subscriber` with `*message.Message`) from **typed application** (`EventBus` / `CommandBus` with generics):

```go
// Transport layer — untyped, wire format
type Publisher interface {
    Publish(topic string, messages ...*Message) error
}

// Application layer — typed, domain objects
func NewEventHandler[T any](handleFunc func(ctx context.Context, event *T) error) EventHandler
```

This maps directly to our design: **Sink/Source are the transport layer** (untyped `Event`), while **handlers and repositories are the application layer** where generics shine.

### 4.2 Interface Split Enables Different Topologies

Watermill's `Publisher` + `Subscriber` split allows:

- One publisher, many subscribers
- Different backends per direction (Kafka pub, Redis sub)
- Fan-out without fan-in coupling

Our `Sink` + `Source` split enables the same:

- Write master + read replicas
- Different storage backends (Pebble sink, SQL source)
- Archival: old events in S3 source, new events in hot sink

### 4.3 Message Ack/Nack Pattern

Watermill subscribers return `<-chan *Message` and require explicit `Ack()` / `Nack()` on each message. This enables:

- At-least-once delivery semantics
- Dead letter queues on persistent Nack
- Backpressure (consumer controls pace)

Our `EventStream` (`Next()`) is similar but lacks explicit ack. Consider whether projections should `Ack()` after processing or if the checkpoint serves that role.

---

## 5. Generics Strategy: Where They Help

### 5.1 Don't: Generic Store/Sink/Source

This is a trap:

```go
// WRONG — a store serves many event types
type Sink[T any] interface {
    Save(ctx context.Context, aggType AggregateType, aggID id.AggregateID, event T, ver Version) error
}
```

You'd need `Sink[UserCreated]`, `Sink[UserUpdated]`, `Sink[OrderPlaced]` — one per event type. That's unworkable. The transport layer must stay untyped (`Event`).

### 5.2 Do: Generic Handler Boundaries (Like Watermill)

Where Watermill uses `NewEventHandler[T any]`, we already have `projection.On[T]`. Extend this pattern:

```go
// What we already have (projection layer)
func On[T any](b *Builder, eventType event.Type, handler func(context.Context, T) error)

// What we could add (event bus layer — typed handlers without manual type assertions)
func SubscribeTyped[T any](
    bus event.Subscriber,
    eventType event.Type,
    codec event.Codec,
    handler func(context.Context, T) error,
) error
```

### 5.3 Do: Generic Aggregate Store

This is the biggest opportunity. Currently `Repository[State]` takes `Store` (untyped) + `Decider[State]`. What if the store itself understood the aggregate type?

```go
// Typed aggregate store — combines Source + Sink + Decider for ONE aggregate type
type AggregateStore[State any] interface {
    Load(ctx context.Context, aggID id.AggregateID) (State, event.Version, error)
    Save(ctx context.Context, aggID id.AggregateID, state State, newEvents []event.Event) error
}

// Built from untyped Sink + Source + Decider[State]
func NewAggregateStore[State any](
    sink event.Sink,
    source event.Source,
    decider decider.Decider[State],
    opts ...AggregateStoreOption[State],
) (AggregateStore[State], error)
```

This pushes the `aggType` parameter into the constructor, eliminating it from every call site:

```go
// Before
repo := decider.NewRepository(store, bus, decider, opts...)
state, ver, err := repo.Load(ctx, aggID, "User")
err = repo.Execute(ctx, aggID, "User", decideFn)

// After — aggType is bound at construction
userStore := event.NewAggregateStore[UserState](sink, source, decider, WithAggregateType("User"))
state, ver, err := userStore.Load(ctx, aggID)
err = userStore.Execute(ctx, aggID, decideFn)
```

### 5.4 Do: Generic Source with Auto-Fold

```go
// Typed source that folds events into State automatically
type TypedSource[State any] interface {
    event.Source
    LoadState(ctx context.Context, aggID id.AggregateID) (State, event.Version, error)
    LoadStateAtVersion(ctx context.Context, aggID id.AggregateID, maxVer event.Version) (State, error)
}

func NewTypedSource[State any](
    source event.Source,
    decider decider.Decider[State],
    aggType event.AggregateType,
) TypedSource[State]
```

This removes the boilerplate of `load → fold` from every consumer:

```go
// Before
events, _ := store.Load(ctx, "User", aggID)
state := UserState{}
for _, e := range events {
    state, _ = fold(state, e)
}

// After
state, ver, _ := typedSource.LoadState(ctx, aggID)
```

---

## 6. The Full Architecture Picture

| Layer           | Role                    | Types                                    | Example                                                      |
| --------------- | ----------------------- | ---------------------------------------- | ------------------------------------------------------------ |
| **Transport**   | Persistence + messaging | Untyped `Event`, `Sink`, `Source`, `Bus` | `event.Store`, `memory.MemoryStore`, `storage.SQLEventStore` |
| **Aggregate**   | Per-type state machine  | Generic `State`                          | `decider.Decider[State]`, `AggregateStore[State]`            |
| **Application** | Handlers + projections  | Generic `T` (payload)                    | `projection.On[T]`, `SubscribeTyped[T]`                      |

This mirrors Watermill exactly:

- Watermill's `Publisher/Subscriber` = our `Sink/Source` (transport, untyped)
- Watermill's `EventBus` = our `AggregateStore` + `projection.Builder` (application, typed)
- Watermill's `NewEventHandler[T]` = our `projection.On[T]` (handler boundary, generic)

---

## 7. What to Implement

### Immediate

1. **`event.Sink` + `event.Source` interfaces** — split from `event.Store`
2. **Update `VersionedStore` → `VersionedSource`** — wrap `Source`, not `Store`
3. **Update `StoreStreamAdapter`** — accept `Source`, not `Store`
4. **Update `projection.Runner`** — accept `Source` + `PositionalLoader` instead of `Store`
5. **Extract `FakeSink` + `FakeSource`** from `testhelpers.FakeStore`

### Medium-term

6. **`AggregateStore[State]`** — binds `aggType` at construction, eliminates it from every method call
7. **`TypedSource[State]`** — auto-folds events into state, removing `load → fold` boilerplate
8. **`SubscribeTyped[T]` / `PublishTyped[T]`** — type-safe bus handlers with automatic codec integration
9. **`AtomicSink` / `TxSink`** — explicit multi-aggregate transaction boundaries

### Long-term

10. **Separate read/write deployment paths** — route `Source` consumers to read replicas, `Sink` consumers to write master
11. **Consider Watermill-style Ack/Nack** — for `EventStream` to enable at-least-once delivery semantics

---

## 8. Open Questions

1. **Should `GlobalLoader` / `PositionalLoader` be separate from `Source`?**
   - They have different cardinality (all aggregates vs one aggregate).
   - Current design: `GlobalLoader` is a standalone interface, `PositionalLoader` extends it.
   - Alternative: `Source` = per-aggregate, `GlobalLoader` = cross-aggregate, keep separate.

2. **Should `StreamLoader` be part of `Source`?**
   - `StreamLoader.LoadStream` is a memory-efficient variant of `Load`.
   - Could be a separate capability or embedded in `Source`.

3. **How does `TransactionalStore` fit post-split?**
   - Currently extends `Store` with `SaveWithOutbox`.
   - Post-split: could extend `Sink` with `SaveWithOutbox`, or become `AtomicSink`.

4. **Should the bus also split into `TypedPublisher` / `TypedSubscriber`?**
   - Watermill has untyped transport + typed application.
   - We could add `PublishTyped[T]` and `SubscribeTyped[T]` as application-layer wrappers.

---

## 9. Key Principles Summary

1. **Capability separation** — Split interfaces by what they do (read vs write), not by what they are.
2. **Minimal dependency** — Accept the smallest interface that satisfies the need.
3. **Transport stays untyped** — Generics belong at application boundaries, not in persistence.
4. **Explicit over implicit** — Pass dependencies (Store, Bus, Codec) as parameters, not globals.
5. **Composable over monolithic** — Build `Store` from `Sink` + `Source`, not the other way around.
6. **Start simple, optimize later** — Same interface, different implementations (memory → SQL → Pebble).
7. **Type safety at human boundaries** — `On[T]`, `AggregateStore[State]`, `TypedSource[State]` are where generics add value.
