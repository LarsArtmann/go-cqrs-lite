# samber/ro Integration Plan — Projection Module

**Date:** 2026-04-30
**Status:** Ready for Execution
**Prerequisite:** Fix CI first (golden files + fuzz test)

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                     core/event (interfaces)                       │
│  Projection, CheckpointStore, Store, Bus, Handler, Event         │
│  InMemoryRunner (stays — simple, for testing)                    │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    ▼                     ▼
┌─────────────────────────┐  ┌────────────────────────────────────┐
│  memory (testing)       │  │  projection/ (NEW MODULE)          │
│  MemoryCheckpointStore  │  │  depends on: core, samber/ro       │
│                         │  │                                    │
│                         │  │  runner.go    — RORunner            │
│                         │  │  handler.go   — HandlerRegistry    │
│                         │  │  pipeline.go  — Pipeline (ro.Pipe) │
│                         │  │  options.go   — RunnerOption        │
│                         │  │                                    │
│                         │  │  internal/stream/                  │
│                         │  │    filters.go  — FilterByType etc  │
│                         │  │    windows.go  — Batch, Buffer     │
│                         │  │    partition.go — GroupBy AggID    │
│                         │  └────────────────────────────────────┘
└─────────────────────────┘
```

Users never see `ro.Observable` or `ro.Pipe`. They see:

```go
runner := projection.NewRunner(store, bus, checkpointStore,
    projection.WithRetry(3),
    projection.WithBatchSize(50),
)
runner.On("user.created", handleUserCreated)
runner.On("user.email_changed", handleEmailChanged)
go runner.Run(ctx)
```

---

## Key Design Decision: Two Runners, Not One

**`InMemoryRunner` stays in `core/event/`** — Simple loop, no dependencies, perfect for tests.

**`RORunner` in `projection/`** — Production-grade, samber/ro-backed, handles partitioning, batching, retry, backpressure.

Same `event.Handler` signature. Users pick based on needs:

```go
// testing
runner := event.NewInMemoryRunner(checkpoint)
bus.SubscribeAll(runner.Handle)

// production
runner := projection.NewRunner(store, bus, checkpoint, opts...)
go runner.Run(ctx)
```

---

## Current State Inventory

### What Exists

| File                       | Lines | Purpose                                                             |
| -------------------------- | ----- | ------------------------------------------------------------------- |
| `core/event/projection.go` | 53    | `Projection` interface, `ProjectionFunc` convenience type           |
| `core/event/runner.go`     | 166   | `InMemoryRunner` with sequential + parallel dispatch                |
| `core/event/checkpoint.go` | 23    | `CheckpointStore` interface                                         |
| `core/event/codec.go`      | 51    | `Codec`, `JSONCodec`, `DecodePayload[T]`                            |
| `core/event/upcaster.go`   | 55    | `Upcaster` interface, `UpcasterFunc`                                |
| `memory/checkpoint.go`     | 50    | `MemoryCheckpointStore`                                             |
| `core/event/bus.go`        | 27    | `Bus` interface: Publish, Subscribe, SubscribeAll                   |
| `core/event/store.go`      | 54    | `Store` interface: Save, Load, LoadFromVersion, AppendBatch, Delete |

### What's Missing (the gap `projection/` fills)

1. **Replay from store** — `InMemoryRunner` only handles live events. No way to catch up from `event.Store`.
2. **Filtering** — `subscribesTo()` is a linear scan. No composable filter pipeline.
3. **Partitioning** — No `GroupBy(aggregateID)` for ordered processing per aggregate.
4. **Batching** — Events processed one-by-one. No `BufferWithTime` for batched writes.
5. **Retry** — Projection errors are fail-fast. No retry with backoff.
6. **Backpressure** — Slow projections block the bus. No buffer or drop strategy.
7. **Checkpointing on replay** — No resume-from-checkpoint when loading from store.

---

## Module Structure

```
projection/
├── go.mod                          # github.com/larsartmann/go-cqrs-lite/projection
│                                   # deps: core, samber/ro
├── runner.go                       # RORunner — main orchestrator (~120 lines)
├── handler.go                      # HandlerRegistry — On() registration (~80 lines)
├── pipeline.go                     # Pipeline — ro.Pipe composition (~100 lines)
├── options.go                      # RunnerOption functional options (~60 lines)
├── errors.go                       # Sentinel errors (~15 lines)
│
├── internal/stream/
│   ├── filters.go                  # FilterByType, FilterByAggregate (~60 lines)
│   ├── windows.go                  # BatchEvents, BufferWithTime (~70 lines)
│   └── partition.go                # PartitionByAggregate (~50 lines)
│
├── runner_test.go                  # Core runner tests
├── handler_test.go                 # HandlerRegistry tests
├── pipeline_test.go                # Pipeline tests
├── options_test.go                 # Option validation tests
├── integration_test.go             # Full replay + live pipeline test
└── benchmark_test.go               # Throughput benchmarks
```

**Total estimate:** ~550 lines production code, ~400 lines tests

---

## Detailed Implementation Plan

### Phase 1: Module Scaffold (15 min)

**1.1 Create `projection/go.mod`**

```
module github.com/larsartmann/go-cqrs-lite/projection

go 1.26.2

require (
    github.com/larsartmann/go-cqrs-lite/core v0.0.0
    github.com/samber/ro v0.3.0
)
```

**1.2 Add to `go.work`**

```go
use (
    // ... existing ...
    ./projection
)
```

**1.3 Create `errors.go`**

```go
var (
    ErrRunnerStopped    = errors.New("projection: runner stopped")
    ErrNilHandler       = errors.New("projection: nil handler")
    ErrDuplicateHandler = errors.New("projection: duplicate handler for event type")
)
```

### Phase 2: Handler Registry (30 min)

**File: `handler.go`**

```go
// HandlerRegistry maps event types to handler functions.
// Thread-safe. Users call On() to register handlers before starting the runner.
type HandlerRegistry struct {
    mu       sync.RWMutex
    handlers map[event.Type][]event.Handler
    wildcard []event.Handler  // handlers for all event types
}

func NewHandlerRegistry() *HandlerRegistry
func (r *HandlerRegistry) On(eventType string, handler event.Handler) error
func (r *HandlerRegistry) OnAll(handler event.Handler) error
func (r *HandlerRegistry) Lookup(eventType event.Type) []event.Handler
func (r *HandlerRegistry) EventTypes() []event.Type
```

Design notes:

- `On("user.created", handler)` — register for specific type
- `OnAll(handler)` — register for all types (wildcard)
- `Lookup()` returns handlers for a given type (specific + wildcard)
- `EventTypes()` returns registered types (used for filtering the stream)

**Tests:** Register, duplicate detection, wildcard, concurrent access, lookup.

### Phase 3: Stream Operators (45 min)

**File: `internal/stream/filters.go`**

This is where samber/ro shines. Creates `ro.Operator[event.Event, event.Event]` filters:

```go
// FilterByType creates an operator that only passes events matching the given types.
func FilterByType(types ...event.Type) ro.Operator[event.Event, event.Event]

// FilterByAggregate creates an operator that only passes events for the given aggregate.
func FilterByAggregate(aggType event.AggregateType, aggID id.AggregateID) ro.Operator[event.Event, event.Event]

// FilterFromCheckpoint creates an operator that drops events until the checkpoint event ID is seen.
func FilterFromCheckpoint(checkpointID id.EventID) ro.Operator[event.Event, event.Event]
```

Each returns a `ro.Operator[event.Event, event.Event]` for use in `ro.Pipe`.

Under the hood, `FilterByType` is:

```go
func FilterByType(types ...event.Type) ro.Operator[event.Event, event.Event] {
    set := make(map[event.Type]bool, len(types))
    for _, t := range types {
        set[t] = true
    }
    return ro.Filter(func(evt event.Event) bool {
        return set[evt.Type()]
    })
}
```

**File: `internal/stream/windows.go`**

```go
// BatchEvents batches events into slices of up to maxSize.
// Uses ro.Map with a stateful accumulator.
func BatchEvents(maxSize int) ro.Operator[event.Event, []event.Event]

// BufferTime collects events within a time window into a batch.
// Uses ro.BufferWithTime.
func BufferTime(window time.Duration) ro.Operator[event.Event, []event.Event]
```

**File: `internal/stream/partition.go`**

```go
// PartitionByAggregate splits the event stream into per-aggregate substreams.
// Events for the same aggregate are always processed in order.
// Uses ro.GroupBy with aggregate ID as key.
func PartitionByAggregate(
    ctx context.Context,
    handler func(id.AggregateID, ro.Observable[event.Event]),
) ro.Operator[event.Event, event.Event]
```

Note: `GroupBy` emits `Observable[Observable[T]]`. Each inner observable is a
per-aggregate event stream. The handler subscribes to each and processes
events with ordering guarantee per aggregate.

**Tests:** Each operator in isolation with synthetic events. Verify filtering, batching, partitioning.

### Phase 4: Pipeline Composition (30 min)

**File: `pipeline.go`**

The `Pipeline` composes samber/ro operators into a single processing chain:

```go
// Pipeline processes an event stream through registered handlers with
// filtering, batching, and error recovery.
type Pipeline struct {
    registry  *HandlerRegistry
    opts      runnerOptions
}

// NewPipeline creates a pipeline for the given handler registry.
func NewPipeline(registry *HandlerRegistry, opts runnerOptions) *Pipeline

// ProcessAll builds and executes a pipeline for a slice of events.
// Used for replay from store.
func (p *Pipeline) ProcessAll(ctx context.Context, events []event.Event) error

// ProcessOne processes a single event through the pipeline.
// Used for live events from the bus.
func (p *Pipeline) ProcessOne(ctx context.Context, evt event.Event) error
```

**Pipeline construction with `ro.Pipe`:**

```go
func (p *Pipeline) buildObservable(events []event.Event) ro.Observable[event.Event] {
    ops := []ro.PipeOperator[event.Event]{
        // 1. Filter to registered event types
        stream.FilterByType(p.registry.EventTypes()...),
    }

    // 2. Optional: batch for bulk processing
    if p.opts.batchSize > 0 {
        // Switch to batched mode
    }

    // 3. Tap for checkpoint saving
    ops = append(ops, ro.Tap[event.Event](
        func(evt event.Event) { /* checkpoint after successful handle */ },
        nil,
        nil,
    ))

    return ro.Pipe(
        ro.FromSlice(events),
        ops...,
    )
}
```

**Key insight:** `ro.Pipe` is the composable pipeline builder. Each `stream.*` function returns a `ro.Operator[T, T]` (or `ro.Operator[T, U]` for transforms). `ro.Pipe` chains them.

For **live events**, we use `ro.PublishSubject[event.Event]`:

```go
subject := ro.NewPublishSubject[event.Event]()
// bus.SubscribeAll feeds into subject.Next(evt)
pipeline := ro.Pipe[event.Event, event.Event](
    subject,
    stream.FilterByType(registeredTypes...),
    ro.Tap[event.Event](handleAndCheckpoint),
)
pipeline.Subscribe(observer)
```

### Phase 5: Runner (45 min)

**File: `runner.go`**

```go
// RORunner processes events through projections using reactive streams.
// Supports replay from store, live subscription, batching, and partitioning.
type RORunner struct {
    store     event.Store
    bus       event.Bus
    checkpoint event.CheckpointStore
    registry  *HandlerRegistry
    opts      runnerOptions
    subject   *ro.PublishSubject[event.Event]  // live event feed
}

func NewRunner(
    store event.Store,
    bus event.Bus,
    checkpoint event.CheckpointStore,
    opts ...RunnerOption,
) *RORunner

func (r *RORunner) On(eventType string, handler event.Handler) error
func (r *RORunner) Run(ctx context.Context) error
func (r *RORunner) Close() error
```

**`Run()` lifecycle:**

```
1. Replay phase:
   ├─ Load checkpoint per projection
   ├─ Load events from store (LoadFromVersion or Load)
   ├─ Filter by registered event types
   ├─ Filter from checkpoint (skip already-processed)
   ├─ Feed through pipeline → handlers
   └─ Update checkpoint after each successful batch

2. Live phase:
   ├─ Subscribe to bus (SubscribeAll)
   ├─ Bus events → PublishSubject.Next()
   ├─ Subject → Filter → Handle → Checkpoint
   └─ Context cancellation → Complete subject → Cleanup
```

**File: `options.go`**

```go
type RunnerOption func(*runnerOptions)

type runnerOptions struct {
    batchSize    int           // 0 = no batching (default)
    batchWindow  time.Duration // 0 = no time windowing (default)
    retryCount   int           // 0 = no retry (default)
    retryDelay   time.Duration // initial retry delay
    concurrency  int           // per-partition concurrency (default 1)
    logger       interface{}   // slog.Logger or similar
}

func WithBatchSize(size int) RunnerOption
func WithBatchWindow(window time.Duration) RunnerOption
func WithRetry(count int, initialDelay time.Duration) RunnerOption
func WithConcurrency(n int) RunnerOption
```

### Phase 6: Tests (60 min)

**`runner_test.go`:**

- Runner creation with defaults
- Runner creation with options
- Replay from store (using `memory.MemoryStore`)
- Live subscription (using `memory.MemoryBus`)
- Replay + live handoff
- Context cancellation stops runner
- Close disposes subscriptions

**`handler_test.go`:**

- Register handlers
- Duplicate detection
- Wildcard handlers
- Lookup returns correct handlers

**`pipeline_test.go`:**

- Filter by type
- Batch events
- Process single event
- Process batch with errors
- Checkpoint update on success

**`integration_test.go`:**

- Full E2E: seed store → replay → subscribe live → publish → verify
- Multi-projection scenario
- Recovery after error (resume from checkpoint)

**`benchmark_test.go`:**

- BenchmarkProcessOne (single event through pipeline)
- BenchmarkProcessBatch (batch of 100 events)
- BenchmarkWithPartitioning (1000 events, 10 aggregates)

### Phase 7: Integration with Existing Modules (30 min)

**7.1 Update `go.work`**

```go
use (
    // ... existing ...
    ./projection
)
```

**7.2 Update `flake.nix`**

Add `projection` to the linted/tested modules list.

**7.3 Update `.golangci.yml`**

Add `samber/ro` to depguard allow list.

**7.4 Update `AGENTS.md`**

Add projection module to module table, dependency graph, coverage summary.

**7.5 Update `CHANGELOG.md`**

```markdown
### Added

- **Projection module** (`projection/`): Production-grade event projection
  runner backed by samber/ro reactive streams. Supports replay from store,
  live subscription, per-aggregate partitioning, batching, and retry.
```

---

## samber/ro Operator Usage Map

| CQRS Need                 | samber/ro Operator                         | Where Used                                     |
| ------------------------- | ------------------------------------------ | ---------------------------------------------- |
| Filter by event type      | `ro.Filter`                                | `internal/stream/filters.go`                   |
| Skip already-processed    | `ro.Filter` (custom)                       | `internal/stream/filters.go`                   |
| Per-aggregate ordering    | `ro.GroupBy`                               | `internal/stream/partition.go`                 |
| Batch writes              | `ro.BufferWithTime` / `ro.BufferWithCount` | `internal/stream/windows.go`                   |
| Error recovery            | `ro.Catch` / `ro.Retry`                    | `pipeline.go`                                  |
| Side effects (checkpoint) | `ro.Tap`                                   | `pipeline.go`                                  |
| Merge replay + live       | `ro.Merge`                                 | `runner.go`                                    |
| Limit replay              | `ro.TakeWhile`                             | `runner.go` (checkpoint boundary)              |
| Type transform            | `ro.Map`                                   | `internal/stream/windows.go` (event → []event) |
| Deduplication             | `ro.Distinct`                              | Optional: idempotency guard                    |

---

## Dependency Graph (After)

```
core (errors, ulid, go-branded-id, go-json-experiment/json)
 ├── memory
 ├── catalog
 ├── middleware
 ├── testhelpers
 ├── integration
 ├── storage (database/sql, pgx)
 └── projection (samber/ro)          ← NEW
```

Users who don't use projections never pull in samber/ro.

---

## Risk Assessment

| Risk                             | Likelihood | Impact                                  | Mitigation                                                   |
| -------------------------------- | ---------- | --------------------------------------- | ------------------------------------------------------------ |
| samber/ro v0.3.0 breaking change | Medium     | Low — all ro code in `internal/stream/` | Swappable behind internal interfaces                         |
| ro.GroupBy complexity            | Medium     | Medium                                  | Start without partitioning; add in Phase 3                   |
| PublishSubject memory leak       | Low        | Medium                                  | Proper disposal in `Close()` + context cancellation          |
| Performance overhead             | Low        | Low — ro is lightweight                 | Benchmark in Phase 6; compare with InMemoryRunner            |
| Replay/live gap (missed events)  | Medium     | High                                    | Replay-first, subscribe-second, merge overlap via checkpoint |

---

## Execution Order (Pareto)

```
Step 1: Module scaffold (go.mod, go.work)                    —  5 min
Step 2: errors.go + options.go                                — 10 min
Step 3: handler.go + handler_test.go                          — 25 min
Step 4: internal/stream/filters.go + tests                    — 20 min
Step 5: pipeline.go (minimal: Filter + Tap, no batching)      — 25 min
Step 6: runner.go (replay-only, no live yet) + tests          — 30 min
Step 7: Verify replay works end-to-end                        — 10 min
── checkpoint: working replay pipeline ──
Step 8: internal/stream/windows.go + tests                    — 20 min
Step 9: Add batching to pipeline                              — 15 min
Step 10: internal/stream/partition.go + tests                 — 25 min
Step 11: Add live subscription to runner (PublishSubject)     — 25 min
Step 12: Merge replay + live streams                          — 20 min
Step 13: Full integration test                                — 15 min
Step 14: Benchmarks                                           — 15 min
── checkpoint: full production runner ──
Step 15: flake.nix, .golangci.yml, AGENTS.md, CHANGELOG       — 15 min
Step 16: Final lint + test + review                           — 10 min
```

**Total estimate: ~5 hours of focused work**

---

## What This Does NOT Change

- `core/event/` stays dependency-free
- `InMemoryRunner` stays in `core/event/runner.go` — no changes
- `event.Projection` interface stays as-is
- `event.CheckpointStore` interface stays as-is
- `memory/` module unchanged
- `middleware/` module unchanged (retry middleware still exists for commands/queries)
- `storage/` module unchanged

This is purely additive: a new `projection/` module that depends on `core` and `samber/ro`.
