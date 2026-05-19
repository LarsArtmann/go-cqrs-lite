# Architecture Review — Session 74

**Date:** 2026-05-19 | **Scope:** Full system architecture | **Lens:** Library/SDK, not application

## Scalability

### Current State: GOOD with caveats

| Dimension              | Rating     | Detail                                                              |
| ---------------------- | ---------- | ------------------------------------------------------------------- |
| Horizontal scaling     | ✅ Good    | Stateless library — consumers add instances freely                  |
| Event store throughput | ⚠️ Limited | `projection.Runner.filterEvents` is O(n) per projection per startup |
| Module isolation       | ✅ Good    | 11 independent go.mod files, clean DAG                              |
| Catalog generation     | ✅ Good    | Immutable catalog after Build() — safe for concurrent reads         |

### Throughput Bottlenecks

1. **`GlobalLoader.LoadAll()` + `filterEvents`** — Projection replay loads ALL events into memory, then linearly scans for checkpoint position. At 1M events this becomes seconds of startup time. **Fix:** Add `LoadFromGlobalPosition(pos, limit)` to `GlobalLoader` interface, or accept a `CheckpointStore` in `GlobalLoader` for position-aware loading.

2. **`MemoryStore.LoadAll()`** — Copies all events from all aggregates into a sorted slice. Memory proportional to total event count. Acceptable for testing but should be documented as NOT suitable for large event stores.

3. **`storage/pebble_helpers.go` batch Commit with Sync** — Forces fsync on every write. Good for durability, bad for throughput. Should offer `WithAsyncWrites()` option.

## Modularity

### Current State: EXCELLENT

The 11-module structure is well-designed:

```
sync (stdlib only)
  ↓
core (oklog/ulid, go-branded-id, go-error-family)
  ↓
memory → core
testhelpers → core
middleware → core
catalog → core
storage → core (+ pebble, sqlite, turso)
  ↓
projection → core + memory (test)
integration → core + memory + middleware + projection + storage
example/user → core + memory + catalog + middleware
example/todo → core + memory + storage
```

### Module Boundary Issues

| Issue                                | Severity  | Detail                                                                                |
| ------------------------------------ | --------- | ------------------------------------------------------------------------------------- |
| core depends on memory + testhelpers | 🟡 MEDIUM | Production go.mod has test-only deps. Should be test-only requires or removed         |
| Published version mismatch           | 🔴 HIGH   | testhelpers v1.1.0 incompatible with current core. Modules building in isolation fail |
| catalog replace directive            | 🟢 LOW    | `catalog/go.mod` has `replace core => ../core` — should use go.work instead           |

### Coupling Analysis

**Low coupling (good):**

- `sync` has zero external deps — perfectly isolated
- `catalog` only depends on `core` types
- `middleware` only depends on `core` interfaces

**Moderate coupling (acceptable):**

- `storage` depends on `core` + pebble + sqlite + turso — heavy but justified for multi-engine support
- `projection` depends on `core` + `memory` (test only) — acceptable

**Tight coupling (needs attention):**

- `core/aggregate` and `core/decider` share ~200 lines of duplicated snapshot/outbox logic — should share helpers
- `core/event/runner` and `projection/runner` duplicate the handle-checkpoint pattern

## Service Orientation

### Current State: NOT APPLICABLE (by design)

This is a **library**, not a service. The "service-oriented" question translates to: "Can consumers compose these modules into services easily?"

### Composability Assessment

| Capability               | Score     | Detail                                                                          |
| ------------------------ | --------- | ------------------------------------------------------------------------------- |
| Custom event store       | ✅ Easy   | Implement `event.Store` interface (7 methods)                                   |
| Custom bus               | ✅ Easy   | Implement `event.Bus` (4 methods)                                               |
| Custom middleware        | ✅ Easy   | Middleware is just a function wrapper                                           |
| Custom serializer        | ✅ Easy   | Implement `event.Codec` (2 methods)                                             |
| Custom projection        | ✅ Easy   | Implement `event.Projection` (3 methods)                                        |
| Custom catalog exporter  | ⚠️ Medium | No shared exporter interface — must follow 4 existing patterns                  |
| Custom conflict resolver | ✅ Easy   | Implement `sync.ConflictResolver[T]` (1 method)                                 |
| Custom storage backend   | ⚠️ Medium | `Dialect` abstraction exists but schema DDL is free functions, not on interface |

### Missing Extension Points

1. **`catalog.Exporter` interface** — No common interface for the 4 exporters. Adding a new format (e.g., Markdown, HTML) requires copying the service/message iteration loop.

2. **Clock injection** — `time.Now()` hardcoded in `event.NewEvent()`, `sync.NewOperation()`, `outbox.Append()`. Should accept `func() time.Time` via option.

3. **Logger injection** — `projection.Runner` accepts `WithLogger`, but `OutboxPublisher`, `InMemoryRunner`, and `PebbleEventStore` use `slog.Default()` or their own logger parameter with no standard interface.

4. **Transaction abstraction** — `storage.TransactionalStore` is SQL-specific. No generic `event.Transaction` interface that could work with non-SQL stores (e.g., a MongoDB session).

## Composability Patterns

### What Composes Well

1. **Middleware stacking** — `dispatcher.Use(m1, m2, m3)` applies in reverse order. Clean, predictable.

2. **Decider pattern** — `Decider[State]` + `Repository[State]` + `event.Store` + `event.Publisher` compose with pure functions. Excellent testability.

3. **Catalog builder** — `CatalogBuilder.AddCommandFromType[T]()` + `.Build()` + any exporter. Type-safe and composable.

4. **Sync primitives** — `VectorClock` + `Operation[T]` + `ConflictResolver[T]` are independent, composable building blocks.

### What Doesn't Compose Well

1. **Aggregate + Decider choice** — Consumers must pick one pattern. No bridge between them. An aggregate using the OO pattern can't easily be tested with the decider's pure-function approach.

2. **Store + Bus wiring** — `aggregate.NewRepository(store, bus)` and `decider.NewRepository(store, bus)` both take concrete interfaces. But if you want outbox, you also need `event.Outbox`. If you want snapshots, you need `event.SnapshotStore`. If you want transactional outbox, you need `event.TransactionalStore`. The configuration grows to 5+ constructor params — consider a `RepositoryConfig` builder.

3. **Projection replay + live** — `projection.NewRunner(globalLoader, bus)` ties replay and live subscription together. No way to use just replay (for batch processing) or just live (for real-time only) without the other.

## Recommendations

### High Impact, Low Effort

| #   | Action                                         | Impact                   |
| --- | ---------------------------------------------- | ------------------------ |
| 1   | Fix testhelpers v1.1.0 version mismatch        | Unblocks isolated builds |
| 2   | Move test deps out of core's production go.mod | Cleaner dependency tree  |
| 3   | Add `catalog.Exporter` interface               | Enables custom exporters |
| 4   | Add clock injection option to NewEvent         | Deterministic testing    |

### High Impact, Medium Effort

| #   | Action                                          | Impact                             |
| --- | ----------------------------------------------- | ---------------------------------- |
| 5   | Unify aggregate/decider repository logic        | Fix bugs once, reduce 200 lines    |
| 6   | Add position-based loading to GlobalLoader      | Production-scale projection replay |
| 7   | Standardize logger injection across all modules | Consistent observability           |
| 8   | Move schema DDL onto Dialect interface          | Cleaner storage extensibility      |

### Lower Priority

| #   | Action                                                    | Impact                          |
| --- | --------------------------------------------------------- | ------------------------------- |
| 9   | Merge projection/ into core/event                         | Fewer packages to understand    |
| 10  | Unify error sentinels across aggregate/decider/projection | One errors.Is check per concept |
