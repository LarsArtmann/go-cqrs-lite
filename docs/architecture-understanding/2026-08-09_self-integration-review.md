# Self-Integration Review: Composition Roots & metaengine vs v1

> Point-in-time architecture snapshot — 2026-08-09.
> Scope: how well the library integrates with itself, and the current state of
> `metaengine/` versus the "v1" manual-backend path.

---

## TL;DR

The repo has accumulated **two unreconciled generations** of composition and
read-model design. The integration debt is real but **concentrated**, not spread
out. Three high-leverage consolidations would resolve almost all of it:

1. **Unify the composition root** — `system.System` is the stated future but
   only 2 of 9 engines are registered; it ships a toy bus instead of watermill;
   and `stack.Bundle` (8 backends, battle-tested) still coexists with no bridge.
2. **Unify the read model** — v1 tiers (`Materialize`, `RelationalProjection`,
   `GraphProjection`) and `metaengine` + `projectionadapter` both live as
   first-class, neither deprecated.
3. **Finish the Record consolidation** (ADR-0111 Phases 3–4) — duplicate
   metadata types (`event.Metadata`, `command.Metadata`, `metadata.Tracing`)
   still parallel `record.CommonMetadata`.

What already composes well: the `event.Store` (sink/source/journal) and
`projection.Projection` seams are clean and shared by **both** generations.
That is the spine to build on.

---

## Part 1: How go-cqrs-lite can better integrate with itself

### 1.1 Two composition roots doing the same job (biggest split-brain)

Both `stack.Bundle` and `system.System` sit in Tier 5 and solve the same
problem — wire a decider to storage, bus, and projections — with **fundamentally
different philosophies** and **no compatibility layer** between them.

| Dimension         | `stack.Bundle`                                                                | `system.System`                                                               |
| ----------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Composition model | Peer field bag; consumer wires via accessors                                  | Constructor-driven, auto-wired from 2 configs                                 |
| Config type       | Go code (options)                                                             | `DomainConfig` (closures) + `DeploymentConfig` (YAML data)                    |
| Storage layer     | `storage.SQLBackend` / `storage/pebble` / `kv.Store` (direct)                 | `metaengine.Engine` / `StreamLogBackend` (via adapters)                       |
| Event bus         | `watermill.EventBus` (external dep, persistent, retries)                      | `simpleBus` (internal, no persistence/retry/async)                            |
| Driver model      | Hardcoded preset functions                                                    | `database/sql`-style driver registry (`RegisterDriver`)                       |
| Projections       | `Materialize` (KV) + `RunProjections` (watermill channel)                     | `projectionadapter.Adapter` → `metaengine.Store` + `projectionhost.Host`      |
| Backends wired    | **8** presets (memory, sqlite, pebble, bbolt, duckdb, postgres, mysql, turso) | **2** built-in drivers (memory, sqlite); others must `RegisterDriver` at init |
| Config loading    | Go options only                                                               | YAML + env via koanf (`LoadConfig`)                                           |
| Safety            | None                                                                          | SCREAM checks, plan-drift detection, `CheckSafety`                            |
| Lifecycle         | `Bundle.Close()` (closer dedup)                                               | `System.Close()` + `GracefulClose(ctx)` + `Drain()` + drainers                |

**References:**

- `stack.Bundle` — `stack/bundle.go:34` (bag of peer interface fields)
- `system.System` — `system/system.go:111`
- Driver registry — `system/driver_registry.go:19` (`DriverFactory`), `:116` (only `memory` + `sqlite` registered)
- `system/` declares itself the replacement — `system/system.go:3`, `system/README.md:5`

**The core risk:** a consumer choosing `stack/sqlite.New()` gets a different
bus (watermill), different projection model (Materialize/KV), and different
storage abstraction (`storage.SQLBackend`) than `system.New()` with
`Driver: "sqlite"` (simpleBus, metaengine.Store, projectionadapter). There is
no bridge.

### 1.2 Two projection worlds

v1 tiers all implement the one good contract — `projection.Projection`
(`projection/projection.go:23`: `Name() / Handle(ctx, evt) / EventTypes()`):

| Tier           | Type                                              | Location                           |
| -------------- | ------------------------------------------------- | ---------------------------------- |
| KV/Materialize | `stack.Materialize[V,K]`                          | `stack/materialize.go:56`          |
| Relational/SQL | `storage.RelationalProjection` + `ProjectionSink` | `storage/relational/projection.go` |
| Graph          | `graph.GraphProjection` + `GraphSink`             | `graph/projection.go`              |

These coexist with `metaengine` + `projectionadapter`
(`metaengine/projectionadapter/adapter.go:63`).

- `system/` uses metaengine (`system/constructor.go:144` Plans, `:217` registers adapter).
- `stack/` uses v1 tiers with metaengine as an **opt-in**
  (`stack.WithMetaEngine`, `stack/metaengine_test.go:47`).
- Neither tier set is deprecated; `docs/projection-tiers.md` still documents
  the v1 path as first-class.

### 1.3 Type-level split-brain (Record consolidation unfinished)

`record.CommonMetadata` (ADR-0111) was meant to unify event/command/metadata.
Phases 1–2 done (Record defined; metaengine depends on it). **Phases 3–4 not
done** — `event.Metadata`, `command.Metadata`, `metadata.Tracing` still exist
in parallel with `record.CommonMetadata`.

### 1.4 Legacy re-exports

`storage/` still re-exports `eventstore/` + `readmodel/` types for backward
compat — legacy coupling that could be shed once consumers migrate.

### 1.5 High-leverage fixes (ordered)

1. **Unify the composition root.** Make `system/` the single entry point; port
   all 8 backends as registered drivers; adopt `watermill` as its bus (or
   expose bus selection) instead of reinventing `simpleBus`. Then deprecate
   `stack.Bundle`.
2. **Unify the read model.** Either complete metaengine's relational/graph
   capabilities so it subsumes the v1 tiers, then deprecate them — or publish
   an explicit decision matrix and stop presenting both as equally first-class.
3. **Finish the Record consolidation** (kill the duplicate metadata types).

---

## Part 2: State of metaengine vs v1 backends

### 2.1 The v1 path (today)

A **manual, per-tier pipeline**: pick a backend, hand-wire the decider,
hand-write a projection. No planner, no cost estimation, no ADT inference.

**Write-side seam:** `decider.Repository[State]` (`decider/decider.go:38`)
wraps `event.Store` + `event.Publisher` + a `Decider[State]`. Constructor:
`decider.NewRepository` (`decider.go:58`). Execution flow
(`Repository.Execute`, `decider.go:113`): load → fold → decide → save →
publish → (optional snapshot).

**Read-side: three manual tiers**, all implementing `projection.Projection`:
KV/Materialize, Relational/SQL (`storage.SQLViewStore` at
`storage/view/store.go:102`), Graph.

**KV store options for Materialize:** `kv.TypedStore` (blob-backed) or
`storage.SQLViewStore` (SQL-backed with queryable columns).

**Runner seams:** `stack.RunProjections` / `bundle.RunProjections`, or
`projectionhost.Host` (`projectionhost/host.go:32`).

**Consumer steps (v1):**

1. Pick backend (`storage.NewSQLiteViewStore` or `kv.TypedStore`).
2. `decider.NewRepository(store, pub, Decider{...})`.
3. Hand-write `stack.Materialize[V,K]{Store, KeyFromEvent, OnCreate, OnUpdate, OnTombstone}` — OR a `RelationalProjection` handler — OR a `GraphProjection` handler.
4. `bundle.RunProjections(...)` or register with `projectionhost.Host`.

**The pain:** every read model is hand-coded; tier choice is manual; no
inference of query shape; filter/sort/pagination each require a different tier.

### 2.2 What metaengine provides (today)

**Engine interface** — intentionally tiny (`metaengine/engine.go:547`):

```go
type Engine interface {
    Profile() EngineProfile
    Closer
}
```

Backends declare capability via **optional per-ADT backend interfaces** (ISP),
not one fat interface: `MapBackend`, `MapUpdater`, `ScanBackend`,
`PushdownScan`, `LayoutPlanner`, `LayoutPlanApplier`, `RawValueReader`,
`SetBackend`, `CounterBackend`, `GraphBackend`, `MultimapBackend`,
`LogBackend`, `StreamLogBackend`, `AtomicAppender`, `StreamTemporalReader`,
`SnapshotBackend`.

**ADTs** (`metaengine/types.go:6`): `ADTMap, ADTSet, ADTCounter, ADTGraph,
ADTLog, ADTStreamLog, ADTSortedMap, ADTMultimap` (+ `ADTVector, ADTSearch,
ADTSpatial` referenced in profiles).

**Plan function** (`metaengine/planner.go:75`):

```go
func Plan(engines []Engine, args ...any) (*Store, error)
```

Cost-based ranking per query via `EngineProfile` (NsPerOp / ReadCosts /
Persistence / Replication), plus a **rule pipeline** producing diagnostics
(write-amplification, durability, layout, replication rules).

**Store** (`metaengine/store.go:15`): `Plan()`, `Collections()`,
`ReplicationMode()`, `Persistence()`, `EventTypes()`, `Close()`. Apply path:
`Apply`, `ApplyRecord`, `ApplyBatch`, `ApplyIdempotent`, `ApplyEncoded` —
routes events to every matching query's projection. Execute path: `Execute` /
`ExecuteTyped[Q,R]`.

**Consumer declaration API** (`metaengine/query.go:160`):

```go
func Query[Q any, R any](name string, args ...any) QueryDecl[Q, R]
```

with `On[P]` / `OnTyped` / `OnRecord` folds (`metaengine/fold.go:38`,
`record_fold.go:35`), declarative `FilterOnField` / `SortOnField` (SQL
pushdown), `WithColumnarLayout`.

**Engines that EXIST (separate modules):**

| Engine        | Path                                    | Status                                          |
| ------------- | --------------------------------------- | ----------------------------------------------- |
| memory        | `metaengine/memory_engine.go`           | Full — all ADTs                                 |
| sqlite        | `metaengine/sqliteengine/`              | Full — pushdown + layout                        |
| pebble        | `metaengine/pebbleengine/`              | KV/log + layout planner                         |
| duckdb        | `metaengine/duckdbengine/`              | CGo, columnar, layout planner (ADR-0092)        |
| pg (postgres) | `metaengine/pgengine/`                  | SQL/KV + pushdown                               |
| badger        | `metaengine/badgerengine/`              | KV/log (ADR-0118)                               |
| dgraph        | `metaengine/dgraphengine/`              | graph (ADR-0119)                                |
| iroh          | `metaengine/irohengine/`                | distributed eval (ADR-0096) — loopback + quic   |
| graphadapter  | `metaengine/graphadapter/adapter.go:20` | wraps `graph.MemoryDriver` as Engine (ADR-0113) |

All share the `enginetest/` contract harness and the `adttest/` matrix.

### 2.3 projectionadapter (the ES bridge)

Bridges `metaengine.Store` → `projection.Projection`
(`metaengine/projectionadapter/adapter.go:63`):

- `New(name, store, decoder, opts...)` (`adapter.go:77`) — auto-derives event
  types via `store.EventTypes()`.
- `Handle(ctx, evt)` (`adapter.go:111`) → decode →
  `store.ApplyRecord(ctx, event.AsRecord(evt), decoded)` (`adapter.go:141`).
- `EventDecoder func(evt event.Event) (any, error)` (`adapter.go:36`) — has
  full event context (StreamID, metadata, version), required for Map-ADT
  queries keyed by entity ID. Takes precedence over `PayloadDecoder`.
- `TypeDecoder` + `Register[E]` + `NewWithDecoder`
  (`metaengine/projectionadapter/typed_decoder.go`) — replaces 70+ line
  switch/case decoders; one-liner constructor.

**Purpose of separate module:** preserve metaengine's dependency boundary —
this is the **only** place that imports `event/` and `projection/`. Core
`metaengine/` imports only `record/`.

### 2.4 End-to-end integration status

**metaengine is wired end-to-end through `system/`** (and is the default path
there):

`system.New(ctx, domain, deployment)` (`system/constructor.go:22`):

1. Creates engines from `DeploymentConfig.Engines` via driver registry (`:42`).
2. Wires the **event store** from a `StreamLogBackend` engine via
   `NewEventAdapter` (`:68`, `:89`).
3. Wires snapshot/command/query stores from the same backend (`:92-113`).
4. Calls `metaengine.Plan(projEngines, domain.Projections...)` (`:144`) →
   `sys.projStore`.
5. Creates `projectionhost.Host` from the event journal (`:210`).
6. **Auto-registers a `projectionadapter.Adapter`** on the host, feeding events
   into `projStore` (`:217-240`), with decoder priority: `TypeDecoder >
EventDecoder > PayloadDecoder > generic JSON`.

`system.System.MetaEngine()` (`system.go:163`) returns the projection-layer
store for typed reads.

**But metaengine is NOT the only/default path everywhere:**

- `stack.Bundle` still supports the v1 tiers and offers metaengine as a
  separate opt-in (`stack.WithMetaEngine`).
- `storage/relational/`, `storage/view/`, `graph/` (v1 tiers) all remain
  first-class and are **not deprecated**.

### 2.5 Comparison matrix

|               | **v1 (today)**                                                   | **metaengine (today)**                                                   | **v2 vision (ADRs 0111–0117)**                       |
| ------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------ | ---------------------------------------------------- |
| Model         | Manual, per-tier                                                 | Declarative + cost-planned                                               | ES-native, auto-generating                           |
| Consumer does | Pick backend, wire `decider.Repository`, hand-write a projection | Declare `Query[Q,R]` + `On[P]` fold; `Plan()` cost-routes across engines | Declare domain types only; planner synthesizes folds |
| Read tiers    | KV/Materialize, SQL `RelationalProjection`, `GraphProjection`    | 8 ADTs (+ Vector/Search/Spatial)                                         | Same ADTs, planner-inferred                          |
| Backends      | 8 stack presets                                                  | 9 engines                                                                | same                                                 |
| End-to-end?   | Yes, via `stack.Bundle`                                          | Yes, via `system/`                                                       | n/a                                                  |

---

## Part 3: The v2 vision (ADRs 0111–0117) — designed and scaffolded, not complete

### ADR-0111: Record Type Extraction — IMPLEMENTED (Phases 1–2), UNFINISHED (3–4)

`record.Record` and `record.CommonMetadata` exist (`record/record.go:64`, `:25`)
and are imported by metaengine. Shared base for Commands+Events. Phases 3–4
(remove duplicate metadata types, remove Tombstone) **not yet done**.

### ADR-0112: ES-Native Metaengine — PARTIALLY DONE

**Vision:** metaengine _is_ the ES projection planner; fold handlers receive
`Record` not `any`; planner reasons about event relationships; auto-projection;
materialize-vs-replay first-class.

**Implemented:**

- `record.Record` is a metaengine dependency.
- `RecordAwareFold` interface (`record_fold.go:18`) + `OnRecord`/`OnRecordTyped`
  constructors (`:35`/`:40`).
- `ApplyRecord` (`store.go:267`) sets the Record on record-aware folds.
- `StreamLogBackend` is the ES primitive (`engine.go:423`); `system/` uses it
  as the event store.

**Not done:**

- Folds still receive decoded `any` payload by default; `OnRecord` is opt-in,
  not enforced.
- Materialize-vs-replay is a diagnostic, not an executor branch.

### ADR-0116: Layered Auto-Projection — PARTIALLY DONE

**Vision:** 3 layers — (1) auto-generate 80% of folds from type inspection,
(2) explicit folds for 20%, (3) auto-route 100%.

**Implemented:**

- Layer 3 (auto-route): **done** — `Plan()` cost-based routing works fully.
- Layer 2 (explicit `On`/`OnRecord` folds): **done**.
- Layer 1 (auto-generate): **partial** — `AutoInsert[E,R]`, `AutoUpdate[E,R]`,
  `AutoDelete[E]` (`auto_fold.go:80`/`:117`/`:94`) via reflection field-matching
  (`matchFields`, `auto_fold.go:36`); `AutoCRUD[C,U,D,R]` (`:135`);
  `AutoCRUDByConvention[R]` (`auto_naming.go:150`); record auto-stamping.

**Not done:** true "zero-config from event struct shapes" where the planner
itself inspects event+query type pairs at `Plan()` time and synthesizes folds
without the consumer calling `AutoInsert`/`AutoCRUD`.

### ADR-0117: Command Lifecycle as Events — VISION, NOT IMPLEMENTED

Commands are immutable intents; lifecycle (received/failed/retried/
dead-lettered) lives in separate `CommandLifecycle/*` event streams; DLQ/retry
are projections. **No `CommandLifecycle` stream or lifecycle event types exist
in code today** — design only. `system/` does wire `CommandAdapter` from
`StreamLogBackend` so commands ARE stored as records, but the lifecycle-event
projection machinery is absent.

### ADR-0114: Tombstone as Domain Event — NOT IMPLEMENTED

Tombstones should be domain events, not mutable metadata. Not done.

### ADR-0113: Delete GraphBackend — NOT DONE

Should delete `metaengine.GraphBackend` in favor of `graphadapter`. But
`GraphBackend` is still defined at `engine.go:394` and the memory engine still
asserts it (`engine.go:560`).

---

## Part 4: What's missing for metaengine to be the default read-model path

1. **Planner-time fold inference (ADR-0116 Layer 1).** Consumers still must
   write or `AutoInsert`/`AutoCRUD` every fold. The "declare only domain types,
   planner generates folds" vision is unmet.
2. **Fold handlers still receive `any`, not `Record`.** ADR-0112's
   "fold handlers receive Record" is opt-in via `OnRecord`; the default `On`
   path is payload-only. The ES-native retype is incomplete.
3. **Duplicate metadata types not removed** (ADR-0111 Phases 3–4):
   `event.Metadata`, `command.Metadata`, `metadata.Tracing` still exist in
   parallel with `record.CommonMetadata`. No tombstone removal (ADR-0114) yet.
4. **Command lifecycle as events (ADR-0117) is entirely unbuilt** — no
   `CommandLifecycle` stream, no DLQ/retry projections.
5. **Materialize-vs-replay is advisory only** — the planner emits INFO/WARN
   diagnostics (`WithWorkloadStats`, `planner.go:49`) but the executor always
   materializes; there's no replay-on-read executor path.
6. **GraphBackend duplication** — ADR-0113 says delete it in favor of
   `graphadapter`, but it's still defined (`engine.go:394`) and asserted.
7. **v1 tiers are not deprecated** and remain the documented path in
   `docs/projection-tiers.md` — there is no migration making metaengine the
   single recommended read-model builder.

---

## Bottom line

metaengine is **real and integrated via `system/`** (event store + projections

- host), works today with explicit folds + `projectionadapter`, and supports 9
  engines with cost-based routing. It is **not** experimental or parallel-only.

But it is a "declare queries + write folds" layer **today**, not the "declare
types, get everything" vision of ADRs 0111–0117. The v2 vision is
**designed and partially scaffolded** (Record type, OnRecord, AutoCRUD,
StreamLogBackend, projectionadapter) but not complete.

Until v1 tiers are either **subsumed** (port relational/graph as metaengine
capabilities) or **formally deprecated**, a consumer faces two valid,
overlapping read-model stacks with no single blessed path — which is exactly
the self-integration gap described in Part 1.

The cleanest path forward: finish the composition-root unification (`system/`
with all 8 backends + watermill), then complete enough of the v2 vision
(planner-time fold inference, Record-typed default folds, materialize-vs-replay
executor) that the v1 tiers can be deprecated with a clear migration guide.

---

## Key file references

| Concern                                           | Location                                        |
| ------------------------------------------------- | ----------------------------------------------- |
| `stack.Bundle` (peer field bag)                   | `stack/bundle.go:34`                            |
| `stack.Repository[State]` accessor                | `stack/accessors.go:31`                         |
| `stack.Materialize[V,K]` (v1 KV projection)       | `stack/materialize.go:56`                       |
| `stack.RunProjections`                            | `stack/run_projections.go:35`                   |
| `system.System`                                   | `system/system.go:111`                          |
| `system.New` (composition)                        | `system/constructor.go:22`                      |
| `system.DomainConfig` / `DeploymentConfig`        | `system/config_types.go:15` / `:94`             |
| `system` driver registry (only memory+sqlite)     | `system/driver_registry.go:116`                 |
| `system` EventAdapter (metaengine→event.Store)    | `system/adapter_event.go:27`                    |
| `system.RegisterDecider`                          | `system/register.go:41`                         |
| `decider.Repository[State]`                       | `decider/decider.go:38`                         |
| `event.Store` (the shared seam)                   | `event/store.go:93`                             |
| `projection.Projection` (the shared contract)     | `projection/projection.go:23`                   |
| `projectionhost.Host`                             | `projectionhost/host.go:32`                     |
| `metaengine.Engine` interface                     | `metaengine/engine.go:547`                      |
| `metaengine.Plan`                                 | `metaengine/planner.go:75`                      |
| `metaengine.Store`                                | `metaengine/store.go:15`                        |
| `metaengine.Query[Q,R]`                           | `metaengine/query.go:160`                       |
| `metaengine` fold DSL (`On`/`OnRecord`)           | `metaengine/fold.go:38` / `record_fold.go:35`   |
| `metaengine` auto-folds (`AutoInsert`/`AutoCRUD`) | `metaengine/auto_fold.go:80`                    |
| `projectionadapter.Adapter`                       | `metaengine/projectionadapter/adapter.go:63`    |
| `projectionadapter.TypeDecoder`                   | `metaengine/projectionadapter/typed_decoder.go` |
| `record.Record` / `CommonMetadata`                | `record/record.go:64` / `:25`                   |
