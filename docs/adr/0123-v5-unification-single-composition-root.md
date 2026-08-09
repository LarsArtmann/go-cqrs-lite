# ADR-0123: v5 Unification — Single Composition Root, Universal Engines, Auto-Projection

**Date:** 2026-08-09
**Status:** Proposed
**Supersedes:** stack.Bundle (Tier 5), simpleBus, BusDriverFactory
**Builds on:** ADR-0111 (Record type), ADR-0112 (ES-native metaengine), ADR-0113 (delete GraphBackend), ADR-0116 (layered auto-projection)

---

## Context

go-cqrs-lite has accumulated two unreconciled generations of composition and
read-model design. The result is a split-brain where consumers face two valid,
overlapping stacks with no single blessed path:

| Dimension          | v1 (`stack.Bundle`)                                        | v2 (`system.System`)                      |
| ------------------ | ---------------------------------------------------------- | ----------------------------------------- |
| Backends wired     | 8 presets                                                  | 2 drivers                                 |
| Event bus          | watermill (persistent, retry)                              | simpleBus (synchronous, no persistence)   |
| Bus extension      | `watermill.WithBackend` (NATS/Redis/Kafka)                 | `BusDriverFactory` (redundant, 1 driver)  |
| Read models        | `Materialize` / `RelationalProjection` / `GraphProjection` | `metaengine` + `projectionadapter`        |
| Projection running | `RunProjections` (blocking loop)                           | `projectionhost.Host` (managed lifecycle) |

Both solve the same problem. Neither is deprecated. There is no bridge.

The metaengine vision (ADRs 0111-0117) is designed but not complete. The v1
tiers remain first-class. A consumer choosing `stack/sqlite.New()` gets a
different bus, storage abstraction, and projection model than
`system.New(Driver:"sqlite")`.

See `docs/architecture-understanding/2026-08-09_self-integration-review.md` for
the full analysis that motivated this decision.

---

## Decision

**Cut a v5 major release that unifies the library around a single composition
root, universal engines, and auto-projection.** No dual paths. No escape
hatches. The developer declares domain types only; the system handles
everything else.

### 1. `system.System` is the single composition root

`stack.Bundle` is **deleted** in v5. `system.New()` becomes the only entry
point. All capabilities (event store, bus, projections, deciders, queries,
health, lifecycle) are auto-wired from `DomainConfig` (consumer closures) +
`DeploymentConfig` (operator YAML).

### 2. `watermill.EventBus` replaces `simpleBus`

`simpleBus` and `BusDriverFactory` are **deleted**. `system/` adopts
`watermill.EventBus` as its bus. Watermill already abstracts multiple backends
via `WithBackend(pub, sub, closer)` — upstream Watermill ships NATS, Redis,
Kafka, SQL, SNS/SQS, GCP Pub/Sub. There is no need for a second abstraction.

`BusConfig` in `DeploymentConfig` maps directly to watermill backend
selection:

```yaml
buses:
  default:
    driver: gochannel # default (in-process, ordered)
    # driver: nats        # multi-process (blank-import watermill-nats)
    # driver: redis       # multi-process (blank-import watermill-redis)
    url: nats://localhost:4222
```

### 3. Driver registry moves to `metaengine/`; engines self-register

`RegisterDriver` + `DriverFactory` + `EngineConfig` move from `system/` to
`metaengine/` (which all 9 engines already depend on). Each engine module
self-registers in its own `init()`:

```go
// metaengine/pebbleengine/register.go
func init() {
    metaengine.RegisterDriver("pebble", func(_ context.Context, cfg metaengine.EngineConfig) (metaengine.Engine, error) {
        return NewPebbleEngine(cfg.DSN)
    })
}
```

Consumers blank-import the engines they want:

```go
import (
    _ "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v5"
    _ "github.com/larsartmann/go-cqrs-lite/metaengine/postgres/v5"
)
```

`system/` never imports engine modules directly. No dependency inversion.

### 4. All 8 backends ported before v5

memory, sqlite, pebble, bbolt, duckdb, postgres, mysql, turso — all
self-register as metaengine drivers. Feature parity with stack presets is a
v5 release gate.

### 5. Record consolidation completed (ADR-0111 Phases 3-4)

`event.Metadata`, `command.Metadata`, `metadata.Tracing` are consolidated into
`record.CommonMetadata`. The Record type becomes the single structural base
for events and commands. Duplicate metadata types are deleted.

### 6. Record-typed folds become the default

`OnRecord` / `OnRecordTyped` become the standard fold constructors, not
opt-in. Fold handlers receive `record.Record` (with Type, StreamID, Version,
Metadata) as their first parameter. The old payload-only `On` constructor is
deprecated and removed.

### 7. Auto-projection is the only consumer-facing read-model API

The developer declares **only** domain types — Events, Commands, Query inputs,
Query results. The auto-projection system (ADR-0116 Layer 1) infers everything
else:

```
Developer declares:                    System infers:
  Event types (structs)          →     Fold operations (insert/update/delete)
  Query input (filter/sort)      →     ADT classification (Map/Set/Counter/...)
  Query result (struct shape)    →     Collection layout (columns, indexes)
```

The planner inspects event and query struct shapes at `Plan()` time and
synthesizes folds automatically. `AutoInsert`/`AutoCRUD`/`AutoCRUDByConvention`
are the current building blocks; the v5 work is making them the **default path**
through planner-time inference, not an opt-in helper.

Explicit `On`/`OnRecord` folds remain available for the 5% of cases where
auto-projection's inference is wrong — but this is an **override**, not an
escape hatch to a different system. There is no second API.

### 8. All v1 read-model tiers are deleted

| Module                         | Fate                                                                                               |
| ------------------------------ | -------------------------------------------------------------------------------------------------- |
| `stack.Materialize`            | **Deleted.** Auto-projection replaces it.                                                          |
| `storage.RelationalProjection` | **Deleted.** Multi-table concepts absorbed as engine internals (multi-collection batch atomicity). |
| `storage.SQLViewStore`         | **Deleted.** sqliteengine/pgengine with layout planning replaces it.                               |
| `graph.GraphProjection`        | **Deleted.** Auto-projection + graphadapter replaces it.                                           |
| `stack.RunProjections`         | **Deleted.** `projectionhost.Host` is the only runner.                                             |

RelationalProjection's `ProjectionSink` operations (Upsert, Ensure, Update,
Increment, DeleteWhere, UpsertCols, UpsertExpr) survive as **internal engine
implementation details** — how SQL engines execute ADT operations under the
hood. The public-facing API dies.

### 9. Every engine implements every ADT

Engines are **universal**: each must implement all ADT backends (Map, Set,
Counter, Log, StreamLog, Multimap, SortedMap, Graph, Vector, Search, Spatial).
If an engine cannot implement an ADT natively, it provides a degraded fallback
(e.g., graph traversal via recursive CTE on SQLite, brute-force vector search
on Memory).

The planner's **capability-degradation rule** emits honest diagnostics:

```
WARN: query "friend_graph" routed to SQLite — graph traversal via recursive CTE,
      O(depth × degree). Consider a GraphDB engine for high-degree traversals.
      Estimated cost: 12ms/query at N=1000 (vs 0.3ms on Dgraph).
```

The operator's backend choice determines **performance characteristics**, not
**capability**. Everything works everywhere. The planner tells you when it's
suboptimal.

### 10. Multi-collection batch atomicity

When one event triggers folds for multiple collections (messages +
attachments + member_roles), all writes commit atomically in one engine
transaction. The batch boundary is the **event**, not the collection.

This replaces RelationalProjection's per-event transaction guarantee — same
atomicity, different abstraction level.

### 11. GraphBackend deleted (ADR-0113 enforced)

`metaengine.GraphBackend` is deleted. Graph operations route through
`graphadapter` (which wraps `graph.MemoryDriver` as a `metaengine.Engine`).
No engine implements `GraphBackend` directly; graph-capable engines
implement the graph traversal as part of their universal ADT coverage.

---

## Consequences

### Positive

- **One composition root.** Consumers learn one API. No dual-path confusion.
- **One read-model system.** Auto-projection handles everything. No tier
  selection, no manual projection writing.
- **Universal engines.** Any backend works for any workload. The planner
  warns, never blocks.
- **Operator-driven infrastructure.** Backend choice is a deployment concern,
  not a code concern.
- **Type safety.** Record-typed folds, generic query declarations, no `any`
  in the consumer-facing API.
- **Smaller surface area.** Fewer modules, fewer concepts, fewer decisions.

### Negative

- **v5 is a breaking release.** All consumers must migrate from stack presets
  and v1 projection tiers. A migration guide is required.
- **Auto-projection is hard.** Inferring folds from struct shapes requires
  sophisticated reflection and convention-matching. Edge cases will need
  explicit overrides.
- **Universal ADT coverage is work.** Each engine must implement every ADT,
  even if degraded. The pebble/duckdb/bbolt engines need graph traversal
  fallbacks; dgraph needs StreamLog support.
- **Large migration surface.** 8 stack presets + 3 read-model tiers +
  stack.Bundle + simpleBus all get deleted in one release.

### Migration Path

1. v4.x: ship auto-projection alongside v1 tiers. Let consumers try it.
2. v4.x+1: mark v1 tiers as deprecated. system/ reaches feature parity.
3. v5.0: delete v1 tiers, stack.Bundle, simpleBus. Clean break.

---

## Implementation Order

See [TODO_LIST.md](../../TODO_LIST.md) → v5 Unification for the ordered task
list. The dependency chain is:

```
Record consolidation (foundation)
  → GraphBackend delete + watermill swap (quick wins)
    → Registry move to metaengine/ + self-registration
      → All 8 backends self-register
        → Record-typed default folds
          → Auto-projection (planner-time fold inference)
            → Multi-collection batch atomicity
              → Universal ADT coverage + degradation rule
                → Delete v1 tiers + stack.Bundle
                  → Cut v5
```
