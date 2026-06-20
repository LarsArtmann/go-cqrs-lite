# v3 Migration Guide

> **Status:** Planning — v3 has not been released. This document tracks all
> planned breaking changes so consumers can prepare.
> **Last updated:** 2026-06-20

---

## Summary

v3 is the next major version of go-cqrs-lite. It removes ghost code, tightens
type safety, and consolidates the module structure. Every breaking change has
been prepared additively in v2 — the v2 types you should migrate to already exist.

| #   | Breaking Change                                     | ADR                                                     | Severity     | Files Affected                                                                |
| --- | --------------------------------------------------- | ------------------------------------------------------- | ------------ | ----------------------------------------------------------------------------- |
| 1   | Delete ghost bus implementations                    | [0028](../adr/0028-watermill-as-delivery-layer.md)      | High         | memory/bus.go (deleted), memory/command_bus.go (deleted), event/reactive\*.go |
| 2   | Move memory/ stores → storage/memory/               | [0029](../adr/0029-storage-consolidation.md)            | **Done**     | Already shipped in v2.8                                                       |
| 3   | Break command/query Metadata = event.Metadata alias | [0031](../adr/0031-metadata-split.md)                   | Medium       | storage/sql/marshal.go + cascade                                              |
| 4   | Version → uint64                                    | —                                                       | **Done**     | Already shipped in v2.8                                                       |
| 5   | Remove io.Closer from core interfaces               | [0010](../adr/0010-remove-io-closer-from-interfaces.md) | **Rejected** | Removes type safety (verschlimmbessern). ADR-0010 stays Proposed              |
| 6   | Delete readmodel/ module (merged into kv/)          | [0032](../adr/0032-merge-readmodel-into-kv.md)          | **Done**     | Already shipped in v2.8                                                       |
| 7   | query.Handler signature: any → generic              | —                                                       | Medium       | query/dispatcher.go                                                           |
| 8   | encoding/json/v2 migration                          | [0026](../adr/0026-experimental-features.md)            | Low          | All golden tests                                                              |

---

## Migration Steps (do these NOW on v2)

### Step 1: Replace memory.MemoryBus with watermill.EventBus

```go
// AFTER (v2 now, v3 only option)
import cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v2"

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

// AFTER (v2 now, v3 only option)
store := kv.NewTypedStore[T, K](backend)
cache := kv.NewCache[T, K](backend)
```

### Step 3: Use TypedHandler instead of any-returning Handler

```go
// BEFORE (v2, deprecated)
query.RegisterTyped(d, "user.get", func(ctx context.Context, q *query.Query) (any, error) {
    return result, nil
})

// AFTER (v2 now, v3 only option)
query.RegisterTyped[*GetUserQuery, *GetUserResult](d, "user.get",
    func(ctx context.Context, q *GetUserQuery) (*GetUserResult, error) {
        return result, nil
    })
```

### Step 4: Use event.Tracing / event.TombstoneMark / event.Causation

```go
// BEFORE (v2 — stringly-typed via Custom metadata)
evt, _ := event.NewEvent("user.created", id, "User", 1, payload,
    event.WithCustom("correlation_id", corrID))

// AFTER (v2 now, v3 preferred)
evt, _ := event.NewEvent("user.created", id, "User", 1, payload,
    event.WithCorrelationID(corrID))
// or via context:
ctx = event.WithCorrelationID(ctx, corrID)
```

---

## What Gets Deleted in v3

### Ghost bus code (ADR-0028)

| File                      | LOC | Status                                                                                                                        |
| ------------------------- | --- | ----------------------------------------------------------------------------------------------------------------------------- |
| `memory/bus.go`           | 250 | **Already deleted** in v2.8                                                                                                   |
| `memory/command_bus.go`   | 150 | **Already deleted** in v2.8                                                                                                   |
| `storage/pg_bus.go`       | 265 | **NOT ghost** — live code (ADR-0027, PostgresBus). Replaced by `watermill.EventBus` with Postgres backend only at v3 boundary |
| `event/reactive.go`       | 188 | **NOT ghost** — consumed by `projection/runner_live.go`. Removed when projection/ dissolves                                   |
| `event/reactive_dedup.go` | 70  | **NOT ghost** — consumed by `projection/runner_live.go`. Removed when projection/ dissolves                                   |

### Module moves (ADR-0029)

| From              | To                | Reason                                               |
| ----------------- | ----------------- | ---------------------------------------------------- |
| `storage/memory/` | `storage/memory/` | Stores belong under storage/; bus code deleted first |
| `readmodel/`      | (deleted)         | Merged into `kv/` (ADR-0032)                         |

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
