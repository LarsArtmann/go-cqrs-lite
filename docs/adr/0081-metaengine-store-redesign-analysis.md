# Metaengine Store Redesign Analysis: Eliminating Runtime Casts

> **Date:** 2026-07-31
> **Trigger:** "What if we redesign Store?" — evaluating whether an alternative Store architecture could eliminate the runtime casts documented in [ADR-0080](0080-metaengine-runtime-casts.md).
> **Status:** Analysis — no decision yet. This documents four alternative designs, their tradeoffs, and which casts each eliminates.

---

## The Problem

The current `Store` has 4 runtime cast sites, all caused by Go's inability to express heterogeneous generic containers (see [ADR-0080](0080-metaengine-runtime-casts.md)):

| # | Site | Signature | Root Cause |
|---|------|-----------|------------|
| 1 | `Plan` | `args ...any` | Heterogeneous `Query[I1,V1]`, `Query[I2,V2]` in one call |
| 2 | `Apply` | `payload any` | One event type fans out to folds expecting different payload types |
| 3 | `Execute` | returns `any` | Return type depends on which query was dispatched |
| 4 | `reify[V]` | `row any → V` | Engine stores values as `any`/`[]byte`; reader reifies |

The question: **can a Store redesign eliminate some or all of these casts?**

---

## Alternative A: Split Store into `TypedStore[I,V]`

### Design

One store per query. Each store is fully parameterized by its input and value types. No central `Plan` — each `TypedStore` plans independently.

```go
// One store per query — fully typed reads
counterStore := metaengine.PlanOne[taskCountsInput, map[string]int64](engines, taskCounts)
mapStore := metaengine.PlanOne[listTasksInput, TaskView](engines, taskViews)

// Read side — ZERO casts:
counts, _ := counterStore.Execute(ctx, taskCountsInput{})   // returns map[string]int64
item, _, _ := mapStore.Get(ctx, "task-1")                   // returns TaskView

// Write side — STILL needs any:
counterStore.Apply(ctx, "task.created", payload)            // payload is any
```

### Cast Analysis

| Cast | Eliminated? | Why |
|------|-------------|-----|
| 1 (Plan `...any`) | **Yes** | No heterogeneous args — each store takes one typed query |
| 2 (Apply `any`) | **No** | Each query still handles multiple event types with different payload types |
| 3 (Execute → `any`) | **Yes** | Return type is `V`, known at compile time |
| 4 (reify `any→V`) | **Yes** | Engine output is typed `V` from the start |

**Eliminates 3 of 4 casts. Keeps cast #2 (Apply).**

### Why Cast #2 Remains

Each query handles multiple event types (`TaskCreated`, `TaskCompleted`, ...) with different payload types. The store can't know which payload type at compile time — it's a map of `eventType → fold`, and each fold has a different input type. That's the existential problem, just moved inside.

### Tradeoffs

| Aspect | Current `Store` | `TypedStore[I,V]` |
|--------|----------------|-------------------|
| Consumer creates | 1 Store, 1 Plan call | N TypedStores, N PlanOne calls |
| Central plan diagnostics | Yes (cross-query write amplification) | Lost — no cross-query view |
| Multi-engine distribution | Planner sees all queries together | Each store plans independently |
| Read-side type safety | Via `TypedReader[V]` wrapper | Built-in |
| Write-side type safety | `any` payload | `any` payload (same problem) |
| Projection adapter | One adapter wraps one Store | One adapter per TypedStore, or a multi-store adapter |

### Verdict

Good for read-heavy workloads where the read-side casts are the hot path. But the current `TypedReader[V]` wrapper already gives consumers that type safety without splitting the Store. The write-side cast (the only truly unavoidable one) remains.

**Verdict: Marginal improvement. Not worth the API fragmentation.**

---

## Alternative B: Sum-Type Projection[E]

### Design

The consumer defines a sealed event interface. All projections share the same event type `E`. The host holds `[]Projection[E]` — homogeneous, no `any`.

```go
// Consumer defines a sealed event interface
type TaskEvent interface{ isTaskEvent() }

type TaskCreatedEvt struct{ ID string; Title string } // implements TaskEvent
type TaskCompletedEvt struct{ ID string }              // implements TaskEvent

// Projections are homogeneous — all share the same E:
type Projection[E any] interface {
    Apply(ctx context.Context, evt E) error
}

// Host holds []Projection[TaskEvent] — homogeneous, no any!
host := NewHost[TaskEvent](counterProj, mapProj)
host.Apply(ctx, "task.created", evt)  // fully typed
```

### Cast Analysis

| Cast | Eliminated? | Why |
|------|-------------|-----|
| 1 (Plan `...any`) | **Yes** | All projections share type `E` — homogeneous slice |
| 2 (Apply `any`) | **Yes** | Apply takes `E`, not `any` |
| 3 (Execute → `any`) | **Yes** | Each projection has its own typed result |
| 4 (reify `any→V`) | **Yes** | Values are typed from the start |

**Eliminates ALL 4 casts.**

### The Cost: Worse Fold DX

Every fold handler must type-switch internally:

```go
// Current DX — clean, one handler per event type:
metaengine.On(TaskCreated{}, func(e TaskCreated) (string, TaskView) { ... }),
metaengine.On(TaskCompleted{}, func(e TaskCompleted, prev TaskView) TaskView { ... }),

// Sum-type DX — switch statement in every fold:
func(e TaskEvent) (string, TaskView) {
    switch evt := e.(type) {
    case TaskCreatedEvt:
        return evt.ID, TaskView{...}
    case TaskCompletedEvt:
        // update
    }
}
```

That's worse DX and more boilerplate. The current `On` pattern dispatches automatically — each fold receives its exact payload type without a switch.

### Additional Costs

- **Consumer must define a sealed interface** for every aggregate's events
- **No more `metaengine.On` auto-dispatch** — each fold handles all event types
- **Event type strings must be managed manually** — currently inferred from `reflect.Type.Name()`
- **Multi-aggregate stores** (events from different aggregates in one projection) require a shared event interface spanning all aggregates — a leaky abstraction

### Verdict

Fully type-safe, but at a significant DX cost. The boilerplate of type-switches in every fold handler is worse than the hidden runtime casts in the current design.

**Verdict: Eliminates casts but degrades DX. Not worth it.**

---

## Alternative C: Per-Event-Type Registration (Codegen Path)

### Design

Generate a fully typed Store per query set. Each event type gets its own `Apply` method with the correct payload type. No `any` anywhere.

```go
//go:generate cqrs-gen

// Generated code — one method per event type, fully typed:
type TaskViewsStore struct {
    engine Engine
    onTaskCreated   func(TaskCreated) (string, TaskView)
    onTaskCompleted func(TaskCompleted, TaskView) TaskView
}

func (s *TaskViewsStore) ApplyTaskCreated(ctx context.Context, e TaskCreated) error { ... }
func (s *TaskViewsStore) ApplyTaskCompleted(ctx context.Context, e TaskCompleted) error { ... }
func (s *TaskViewsStore) Get(ctx context.Context, id string) (TaskView, bool, error) { ... }
```

### Cast Analysis

**Eliminates ALL 4 casts.** Every method is fully typed. The generated code is type-checked by the compiler.

### The Cost: Build Complexity

- **Requires a code generation step** (`go generate` / `cqrs-gen`)
- **Generated code per query set** — one file per projection
- **Must regenerate when queries change** — build-time burden
- **Tooling investment** — the generator must parse query declarations and emit typed code
- **Debugging generated code** — stack traces point at generated files

### What the Generated Code Eliminates

The generator would produce:
1. A typed `Apply` method per event type (no `any` payload)
2. A typed `Get`/`Scan` returning `V` (no `any` result)
3. A typed constructor that validates query shapes at compile time (no `queryMeta` interface)

### Verdict

The strongest type safety of all alternatives. Every cast is eliminated at the source. But requires building and maintaining a code generator.

**Verdict: Best types, worst build complexity. Worth it for a future `cmd/cqrs-gen` feature, but not as a replacement for the current runtime API.**

---

## Alternative D: Hybrid — Typed Reader Layer + Erased Write Layer

### Design

Keep the current `Store` for writes (the erased layer), but make `TypedReader[V]` the primary read API (the typed layer). This is essentially what we already have, but formally documented as a two-layer architecture.

```go
// Write layer (erased — accepts any, routes to folds)
store.Apply(ctx, "task.created", payload)  // payload is any

// Read layer (typed — no casts)
reader := metaengine.NewReader[TaskView](store, "task_views")
item, _, _ := reader.Get(ctx, "task-1")     // returns TaskView, not any
items, _ := reader.Scan(ctx, ...)           // returns []TaskView
```

### Cast Analysis

| Cast | Eliminated? | Why |
|------|-------------|-----|
| 1 (Plan `...any`) | **No** | Store still accepts heterogeneous queries |
| 2 (Apply `any`) | **No** | Write layer stays erased |
| 3 (Execute → `any`) | **Yes** (for consumers) | `TypedReader[V]` and `ExecuteTyped[I,V]` hide the cast |
| 4 (reify `any→V`) | **Yes** (for consumers) | `TypedReader[V]` handles reification internally |

**Consumer-visible casts: 0. Internal casts: 2 (at the Store/Engine boundary).**

### Why This Is Already the Right Answer

The current design already implements this. Consumer code never touches `any`:

- `metaengine.NewReader[V]` → typed reads
- `metaengine.ExecuteTyped[I,V]` → typed execute
- `metaengine.On(E{}, func(e E) ...)` → typed fold registration
- `projectionadapter.WithEventDecoder` → typed event decoding

The casts are internal to Store/Engine — they're implementation details that consumers never see.

### Verdict

This is the current design, formalized. The consumer-facing API is already fully typed. The internal casts are at the exact boundary where Go's type system gives up, and they're hidden behind type-safe wrappers.

**Verdict: This is what we have. It's the right answer.**

---

## Comparison Matrix

| Design | Casts Eliminated | Consumer DX | Build Complexity | Verdict |
|--------|-----------------|-------------|------------------|---------|
| **Current (Store + `any`)** | 0 (baseline) | Best — one Store, one Plan, typed readers | None | **Status quo** |
| **A: TypedStore[I,V]** | 3 of 4 (read side) | N stores, fragmented API | None | Marginal — reader wrapper already covers this |
| **B: Sum-Type Projection[E]** | All 4 | Worse — switch statements in every fold | None | DX degradation not worth it |
| **C: Codegen** | All 4 | Best types, generated boilerplate | High — requires generator | Future option via `cmd/cqrs-gen` |
| **D: Hybrid (current, formalized)** | Consumer: all. Internal: 2. | Best — typed readers, erased writes | None | **This is already the design** |

---

## The Uncomfortable Truth

The current design already hides the casts behind `TypedReader[V]` and `ExecuteTyped[I,V]`. **Consumer code never touches `any`.** The casts are internal implementation details at the exact boundary where Go's type system gives up.

**The only cast that truly matters is #2 (Apply).** That's the event fan-out — one decoded event routed to folds with different signatures. No redesign eliminates this without either:
- A sum type (worse DX — Alternative B)
- Codegen (build complexity — Alternative C)

**If codegen ever lands (`cmd/cqrs-gen`):** Alternative C becomes the premium path — fully typed everything, generated from query declarations. Until then, the current design (Alternative D) is the correct tradeoff.

---

## References

- [ADR-0080: Why metaengine uses runtime casts](0080-metaengine-runtime-casts.md)
- [Go #77273: Generic type parameters on methods](https://github.com/golang/go/issues/77273) (Go 1.27)
- [Go #80448: Associated types in interfaces](https://github.com/golang/go/issues/80448) (Open, Jul 2026)
