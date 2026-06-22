# v3 Migration Guide

> **Status:** Complete — all 11 breaking changes shipped. Ready to tag v3.0.0.
> **Last updated:** 2026-06-21

---

## Summary

v3 is the next major version of go-cqrs-lite. It removes ghost code, tightens
type safety, and consolidates the module structure. All 11 breaking changes
are complete and shipped.

| #   | Breaking Change                                       | ADR                                                     | Severity | Status                                                                        |
| --- | ----------------------------------------------------- | ------------------------------------------------------- | -------- | ----------------------------------------------------------------------------- |
| 1   | Delete ghost bus implementations                      | [0028](../adr/0028-watermill-as-delivery-layer.md)      | High     | **Done** — memory buses + event/reactive\*.go deleted                         |
| 2   | Move memory/ stores → stack/memory/                   | [0029](../adr/0029-storage-consolidation.md)            | Done     | Shipped in v2.8                                                               |
| 3   | Break command/query Metadata = event.Metadata alias   | [0031](../adr/0031-metadata-split.md)                   | Medium   | **Done** — each module owns its Metadata embedding event.Tracing              |
| 4   | Version → uint64                                      | —                                                       | Done     | Shipped in v2.8                                                               |
| 5   | Remove io.Closer from core interfaces                 | [0010](../adr/0010-remove-io-closer-from-interfaces.md) | Medium   | **Done** — callers type-assert to io.Closer                                   |
| 6   | Delete readmodel/ module (merged into kv/)            | [0032](../adr/0032-merge-readmodel-into-kv.md)          | Done     | Shipped in v2.8                                                               |
| 7   | query.Handler signature: any → generic                | —                                                       | Medium   | TypedHandler shipped                                                          |
| 8   | Rename Decider.Fold → Apply                           | —                                                       | Medium   | **Done** — field, errors, helpers renamed                                     |
| 9   | Make event.Event a concrete type                      | —                                                       | Low      | **Done** — `type Event = *ImmutableEvent`; zero call-site changes             |
| 10  | encoding/json/v2 migration                            | [0026](../adr/0026-experimental-features.md)            | Low      | Deferred — pending Go stdlib stabilization                                    |
| 11  | Move SSE → transport/http/, delete generic HTTP utils | [0025](../adr/0025-transport-adapter-strategy.md)       | Medium   | **Done** — SSE moved; healthcheck/metrics_http/pprof deleted (zero consumers) |

---

## Migration Steps (do these NOW on v2)

### Step 1: Replace memory.MemoryBus with watermill.EventBus

```go
// AFTER (v3)
import cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"

bus := cqrswatermill.NewEventBus()
```

The EventBus API is identical (Publish, Subscribe, SubscribeAll, Use, UsePublish, Close).
For multi-process, use `watermill.WithBackend(pub, sub, closer)` to inject Kafka/NATS.

Note: `memory.NewMemoryBus()` was already deleted in v2. Use `watermill.NewEventBus()` for
in-process delivery. `storage.PostgresBus` remains as the opt-in distributed bus via
`stack/postgres.WithDistributedBus(listener)` — it is NOT ghost code.

### Step 2: Replace readmodel.Store with kv.TypedStore

```go
// BEFORE (v2, deprecated module)
store := readmodel.NewStore[T, K](backend)
cache := readmodel.NewCachedStore[T, K](backend)

// AFTER (v3)
store := kv.NewTypedStore[T, K](backend)
cache := kv.NewCache[T, K](backend)
```

### Step 3: Use TypedHandler instead of any-returning Handler

```go
// BEFORE (v2 — Handler returns any, caller must type-assert)
d.Register("user.get", func(ctx context.Context, q query.Query) (any, error) {
    return result, nil
})

// AFTER (v3)
query.RegisterTyped[*GetUserQuery, *GetUserResult](d, "user.get",
    func(ctx context.Context, q *GetUserQuery) (*GetUserResult, error) {
        return result, nil
    })
```

### Step 4: Use event.Tracing / event.TombstoneMark / event.Causation

```go
// BEFORE (v2 — stringly-typed via Custom metadata)
evt, _ := event.NewEvent("user.created", id, "User", 1, payload,
    event.WithCustom(event.MetadataKey("correlation_id"), corrID))

// AFTER (v3 — typed options)
evt, _ := event.NewEvent("user.created", id, "User", 1, payload,
    event.WithCorrelationID(corrID))
// or via context enricher (sets CorrelationID on all events built from ctx):
ctx = cqrsotel.WithCorrelationID(ctx, corrID)
```

### Step 5: Decider.Fold → Apply

The aggregate event-applier field was renamed. "Fold" implied a generic
left-fold over a collection; it is the domain event applier (one event).

```go
// BEFORE
decider.Decider[State]{Initial: State{}, Fold: applyFunc}
// AFTER
decider.Decider[State]{Initial: State{}, Apply: applyFunc}
```

Errors renamed too: `ErrNilFold` → `ErrNilApply`, `ErrFoldFailed` → `ErrApplyFailed`.

### Step 6: io.Closer removed from core interfaces

Core interfaces no longer embed `io.Closer`. If you implemented a custom
store/bus purely to satisfy `Close()`, you can delete the no-op method.
If you relied on the interface guaranteeing Close, type-assert instead:

```go
// BEFORE — Close guaranteed by interface
store.Close()
// AFTER — type-assert; concrete impls still have Close()
if c, ok := store.(io.Closer); ok { c.Close() }
```

All shipped concrete stores/buses retain their `Close()` method unchanged.

### Step 7: command/query Metadata is no longer event.Metadata

`command.Metadata` and `query.Metadata` are now their own structs
(embedding `event.Tracing` + a `Custom` map), not aliases of
`event.Metadata`. They no longer carry event-only fields (Tombstone,
Causation). The tracing options (`WithCorrelationID`, etc.) are unchanged.

If you called `event.EnsureCustom` on a command/query Metadata, use the
module-local helper instead: `command.EnsureCustom(&m)` / `query.EnsureCustom(&m)`.

### Step 8: event.Event is a concrete type

`event.Event` is now `type Event = *ImmutableEvent` (a concrete pointer
type alias), not an interface. This is transparent to almost all code —
method calls, `[]event.Event`, `func(event.Event)` all work unchanged.

If you wrote a custom type implementing the old `event.Event` interface
(rare — `*ImmutableEvent` was the only production implementation), it no
longer satisfies `event.Event`. Construct events via `event.NewEvent` /
`event.New` instead.

### Step 9: SSE moved to transport/http/

```go
// BEFORE (v2)
import "github.com/larsartmann/go-cqrs-lite/middleware/v2"
broker, _ := middleware.NewSSEBroker(bus)
handler := middleware.SSEHandler(broker)

// AFTER (v3)
import "github.com/larsartmann/go-cqrs-lite/transport/http/v3"
broker, _ := http.NewSSEBroker(bus)
handler := http.SSEHandler(broker)
```

The following middleware exports were **deleted** (generic utilities with
zero CQRS dependencies and zero consumers):

- `HealthCheckHandler`, `HealthChecker`, `HealthStatus`, `HealthCheckResponse`
- `MetricsHandler`, `MetricsMiddleware`, `MetricsCollector`, `NewMetricsCollector`
- `ProfilingHandler`, `RegisterProfiling`

For health checks, write a 10-line `http.HandlerFunc`. For metrics, use
the `prometheus/` module (OTel→Prometheus bridge). For pprof, use
`import _ "net/http/pprof"` directly.

---

### Ghost bus code (ADR-0028)

| File                      | LOC | Status                                                                                                                        |
| ------------------------- | --- | ----------------------------------------------------------------------------------------------------------------------------- |
| `memory/bus.go`           | 250 | **Already deleted** in v2.8                                                                                                   |
| `memory/command_bus.go`   | 150 | **Already deleted** in v2.8                                                                                                   |
| `storage/pg_bus.go`       | 265 | **NOT ghost** — live code (ADR-0027, PostgresBus). Replaced by `watermill.EventBus` with Postgres backend only at v3 boundary |
| `event/reactive.go`       | 239 | **Already deleted** — removed with projection/ dissolution (ADR-0030)                                                         |
| `event/reactive_dedup.go` | 104 | **Already deleted** — removed with projection/ dissolution (ADR-0030)                                                         |

### Module moves (ADR-0029)

| From              | To              | Reason                                                         |
| ----------------- | --------------- | -------------------------------------------------------------- |
| `storage/memory/` | `stack/memory/` | Stores moved to stack/ presets package; bus code deleted first |
| `readmodel/`      | (deleted)       | Merged into `kv/` (ADR-0032)                                   |

### Type changes

| Type               | v2                      | v3                          | Impact                                         |
| ------------------ | ----------------------- | --------------------------- | ---------------------------------------------- |
| `event.Version`    | `int`                   | `uint64`                    | 164 files — negative versions were never valid |
| `command.Metadata` | `= event.Metadata`      | own struct                  | SQL marshal cascades                           |
| `query.Handler`    | `func(...)(any, error)` | generic `TypedHandler[Q,R]` | Already additive in v2                         |

---

## Blocked / Deferred

### encoding/json/v2

Requires `GOEXPERIMENT=jsonv2` build tag (experimental). The `codec/jsonv2_experiment.go`
file exists behind the tag. Migration deferred until Go stdlib stabilizes json/v2.

### catalog.Message / catalog.Service splits (v4)

17-field `catalog.Message` and 16-field `catalog.Service` structs need splitting
into Message + MessageMeta. Deferred to v4 — lower priority, internal types only.
