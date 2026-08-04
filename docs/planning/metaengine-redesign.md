# Metaengine Redesign: From "Stack Hack" to Deployer-Driven Architecture

> **STATUS: DESIGN — decisions recorded, implementation not started.**
>
> This document captures the full context, research evidence, decisions, and
> target architecture for the next-generation composition model in go-cqrs-lite.
> It is the single source of truth for the redesign. Every claim is cited against
> source code or external research.
>
> **Companion artifacts:**
>
> - [`docs/architecture-understanding/2026-08-04_metaengine-goal-gap.html`](../architecture-understanding/2026-08-04_metaengine-goal-gap.html) — the point-in-time assessment that triggered this redesign.
> - [`docs/planning/meta-engine-design.md`](meta-engine-design.md) — the original vision (aspirational; some features shipped, some did not).
> - [`docs/planning/meta-engine-project-definition.md`](meta-engine-project-definition.md) — the research-contribution framing.
>
> **Date:** 2026-08-04
> **Participants:** Lars Artmann (goal owner), Crush (research + architecture)
> **Related ADRs:** none yet — ADRs will be cut from this document when implementation begins.

---

## Table of Contents

1. [The Goal (Verbatim)](#1-the-goal-verbatim)
2. [Why the Current Stack Is a "Hack"](#2-why-the-current-stack-is-a-hack)
3. [Research Evidence](#3-research-evidence)
4. [Decisions (Recorded)](#4-decisions-recorded)
5. [The Key Insight: Multi-Instance Metaengine](#5-the-key-insight-multi-instance-metaengine)
6. [Target Architecture](#6-target-architecture)
7. [Operator Configuration Surface](#7-operator-configuration-surface)
8. [Introspection API (for cqrs-htmx)](#8-introspection-api-for-cqrs-htmx)
9. [Scream Store (Safety Model)](#9-scream-store-safety-model)
10. [Open Questions](#10-open-questions)
11. [Glossary](#11-glossary)

---

## 1. The Goal (Verbatim)

Lars's goal, stated across two conversations:

> My Goal is for consumers of this lib should NOT decide on the
> implementation of infrastructure e.g. DB, MessageBuses, ....
> They should have a simple API that allows the person deploying the App,
> to decide where they want to keep there data and what they want to store.
> We should have recommendations though, e.g. some meta data or projections
> aka materialized views may be better places in a SQL/KV/columnar/Graph
> DB - but we should also be able to run fully with just a SQLite + Memory
> setup, maybe even multiple SQLite DBs e.g. 1 for Command + Event
> Sourcing, 1 for Query (logs), and 1 for materialized views.
>
> Basically knowing ONLY the Commands + Events + Queries and there relations we should be able to build superb Projections (aka. Materialized Views)!

Expanded in the second conversation:

> The old stack was more like a 'hack'/'in between solution' to get closer
> to where I want to get too. It may need a full redesign.
>
> We also should provide easy ways of giving the END Operator the right
> tools to configure the App's deployment via the metaengine.
>
> The App Code itself should basically never care about where and how
> it's deployed.
>
> We also need a way to add new backends/DBs and remove old ones - for
> BOTH the source of truth stores (i.e. Command/Event/Queries/... log)
> AND the Projections!
>
> We also need to provide proper APIs/SDKs for (e.g.
> /home/lars/projects/cqrs-htmx/) to provide a nice pluggable Web
> Interface, so Admins and Operations can see what ACTUALLY happens
> underneath the layers (i.e. in the metaengine)
>
> We also should consider some kind of store that will SCREAM BIG TIME
> (and prevents failures) if an Operator makes a metaengine change that
> is unsafe.

### Goal decomposition (six requirements)

| #   | Requirement                                                        | Source         |
| --- | ------------------------------------------------------------------ | -------------- |
| G1  | Consumers do NOT decide infrastructure (DB, bus, codec)            | Goal statement |
| G2  | The deployer/operator decides where data lives and what is stored  | Goal statement |
| G3  | Recommendations for storage placement (SQL/KV/columnar/Graph)      | Goal statement |
| G4  | Run fully with SQLite + Memory (minimal viable deployment)         | Goal statement |
| G5  | Multiple SQLite DBs (events, queries, projections as separate DBs) | Goal statement |
| G6  | From Commands + Events + Queries alone → superb projections        | Goal statement |
| G7  | Add/remove backends for BOTH source-of-truth and projections       | Expansion      |
| G8  | App code is deployment-agnostic                                    | Expansion      |
| G9  | Introspection API for a pluggable admin/ops web interface          | Expansion      |
| G10 | Scream store: prevent unsafe operator changes                      | Expansion      |

---

## 2. Why the Current Stack Is a "Hack"

The `stack.Bundle` (`stack/bundle.go:34-124`) is the current composition root.
It works, but it has structural debt that blocks the goal.

### 2.1 The Bundle is a capability bag, not an abstraction

The Bundle is **explicitly documented as "a bag of peer capability fields"**
(`bundle.go:21`). Every field is optional (nil-allowed). It holds 18 exported
interface fields (EventSink, EventSource, Journal, SeekableJournal,
CommandSink, SnapshotStore, CheckpointStore, ReadModels, etc.) plus
unexported infrastructure fields.

This is fine for wiring, but it means:

- **No backend identity** — the Bundle doesn't know "I am a SQLite deployment"
  in a behavioral sense, only via the descriptive `Capabilities` struct.
- **No lifecycle ownership** — the Bundle closes things, but doesn't own
  construction. Each preset's `New()` does the construction.
- **No topology** — the Bundle describes storage capabilities, not the wired
  CQRS system (dispatchers, hosts, projections, bus type).

### 2.2 Structural leaks (the "holes")

| Leak                                                | Evidence                                                                                                                                                                                 | Impact                                                                                                      |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Type-erased `db any`**                            | `bundle.go:86` — stores `*sql.DB` as `any` to avoid importing `database/sql`. `Database() any` (`bundle.go:236`) hands it back; SQL view constructors re-assert to `*sql.DB`.            | Defeats the type system. Admin UI can't introspect the DB safely.                                           |
| **Bolted-on metaengine**                            | `bundle.go:73` — `metaEngine *metaengine.Store` is the only capability stored as a **concrete pointer**, not an interface. Every other capability is an interface from a leaf module.    | Metaengine is not a first-class peer. `validate()` doesn't check it. Consumer must construct it themselves. |
| **Inconsistent `WithStack` passthrough**            | Only `sqlite` (`preset.go:117`) and `mysql` (`preset.go:44`) expose `WithStack`. Postgres, Turso, DuckDB, Pebble do NOT.                                                                 | A consumer using `postgres.New()` cannot inject `WithMetaEngine` via the preset API.                        |
| **`CatchUpSubscriber()` hard-depends on watermill** | `accessors.go:193` — type-asserts `b.Subscriber.(*cqrswatermill.EventBus)`.                                                                                                              | The Bundle claims backend-agnostic but this accessor is watermill-specific.                                 |
| **mysql skips `WithCapabilities`**                  | `mysql/preset.go:63-89` — the only SQL preset that omits the Capabilities struct.                                                                                                        | Bundle reports `Backend:""` for MySQL deployments.                                                          |
| **Wrapper-Bundle inconsistency**                    | `pebble.Bundle` (`pebble/preset.go:63`) and `turso.Bundle` (`turso/preset.go:76`) wrap `*stack.Bundle` to add backend-specific methods. The other 5 presets return bare `*stack.Bundle`. | Three-way naming overload; admin UI can't rely on a single type.                                            |

### 2.3 No backend registry or plugin system

There is **no registry, no `Provider` interface, no plugin system** anywhere in
the storage layer. `doc.go:27-33` explicitly defends this choice: _"Go does not
support partial interface implementation, so a Provider interface would force
every preset to implement every method."_

Each preset is a **hardcoded `New()` function in its own Go module**
(`stack/sqlite/go.mod`, `stack/pebble/go.mod`, etc.). Selecting a backend =
selecting an import path = recompiling.

**An operator who isn't also the developer changing imports cannot swap
backends.** This directly violates G2 (deployer decides).

### 2.4 The metaengine is not integrated at the goal level

The assessment (`docs/architecture-understanding/2026-08-04_metaengine-goal-gap.html`)
found:

- **Lifecycle-deep, semantically-shallow.** `WithMetaEngine` registers a
  pre-built store for Close + benchkit. No auto-wiring of engines from the
  bundle's backend.
- **Consumer opens the DB.** `example/taskmanager/setup.go:83-102` calls
  `sql.Open`, sets WAL pragmas, constructs `NewSQLiteEngine`, and hardcodes the
  engine list. This is the opposite of "deployer decides."
- **77-line hand-written event decoder.**
  `example/taskmanager/metaengine.go:75-151` (`taskEventDecoder`) — a parallel
  switch that re-decodes every event payload and wraps it in
  `eventWithID[P]{ID: evt.StreamID(), Payload: p}`. The fold handlers already
  encode event→view logic; the decoder duplicates that knowledge.
- **Only 1 of 3 examples uses it.** `getting-started` uses
  `stack.NewMaterialize`; `readme-quickstart` uses no projection at all.

### 2.5 The gap vs goal, clause by clause

| Goal clause                                | Status (current) | Evidence                                                                              |
| ------------------------------------------ | ---------------- | ------------------------------------------------------------------------------------- |
| G1: Consumers do NOT decide infrastructure | ❌ Not met       | Consumer opens DB + builds engines                                                    |
| G2: Deployer decides where data lives      | ❌ Not met       | No operator config; engine list is consumer code                                      |
| G3: Recommendations                        | ✅ Met           | ExplainPlan/Doctor + degradation/scale/write-amp diagnostics                          |
| G4: Run with SQLite + Memory               | ✅ Met           | taskmanager runs on Memory + SQLite                                                   |
| G5: Multiple SQLite DBs                    | ⚠️ Partial       | Stack supports multi-DB; metaengine re-opens its own DB                               |
| G6: Commands+Events+Queries → projections  | ⚠️ Partial       | Optimizer is superb; declaration is engine-level not domain-level                     |
| G7: Add/remove backends for both layers    | ❌ Not met       | No registry; compile-time imports only                                                |
| G8: App code is deployment-agnostic        | ❌ Not met       | App code touches DSNs, codecs, engine lists                                           |
| G9: Introspection API for admin UI         | ⚠️ Partial       | Rich methods exist (ExplainPlan, Doctor, Collections) but ad-hoc, no unified topology |
| G10: Scream store                          | ❌ Not met       | No Diff/Pin/Fingerprint; all diagnostics advisory; SwapEngine does zero validation    |

---

## 3. Research Evidence

### 3.1 Current stack architecture

Deep analysis of `stack/bundle.go`, `stack/options.go`, and all 7 presets
(stack/memory, sqlite, pebble, postgres, mysql, duckdb, turso):

- **27 core Option functions** across the stack package.
- **No shared "preset contract"** — just convention. SQL presets route through
  `sqlopt.InitStack`/`FinalizeBundle` (a de facto mini-framework); memory/pebble
  hand-assemble options.
- **`Capabilities`** (`capabilities.go:11-42`) is descriptive metadata (a struct
  of booleans/strings), not a behavioral contract. Two sources of truth: the
  struct vs. the actual PRAGMAs applied.

### 3.2 Backend extension model

For the source-of-truth stores, a new backend must implement the ISP-segregated
interfaces:

- `event.EventSink`/`EventSource`/`Store`/`Journal`/`SeekableJournal`
  (`event/store.go`)
- `command.CommandSink`/`CommandSource`/`Store`/`CommandJournal`/`SeekableCommandJournal`
  (`command/store.go`)
- `query.QuerySink`/`QuerySource`/`QueryStore`/`QueryJournal`/`SeekableQueryJournal`
  (`query/store.go`)
- `snapshot.SnapshotStore` (`snapshot/store.go`)
- `event.CheckpointStore` (`event/checkpoint.go`)

For projections:

- `projection.Projection` (`projection/projection.go:23`) — `Name()`, `Handle()`,
  `EventTypes()`
- `kv.Store` / `kv.ViewStore[V,K]` (`kv/kv.go`, `kv/view_store.go`)
- `storage.RelationalProjection` (concrete struct, not interface)
- `graph.GraphDriver` / `GraphProjection`
- `metaengine.Engine` + 16 optional ADT backend interfaces (`metaengine/engine.go`)

**There is NO registry/plugin pattern for storage backends.** Every backend is
a hardcoded constructor. Backend selection is compile-time (import path) only.

### 3.3 cqrs-htmx (the admin UI consumer)

`/home/lars/projects/cqrs-htmx/` is a library/SDK (not an app) — the HTTP/HTMX
binding layer for go-cqrs-lite. 19-module Go workspace.

**`dashboardui/`** is the existing CQRS/ES observability dashboard
(`cqrs-htmx/dashboardui/`):

| Panel                | Config interface             | Routes                                   |
| -------------------- | ---------------------------- | ---------------------------------------- |
| Overview             | `Journal`/`SeekableJournal`  | `GET /`                                  |
| Event Stream Browser | `Journal`/`SeekableJournal`  | `/events`, `/events/{id}`                |
| Aggregate Browser    | `StreamReader`/`EventSource` | `/aggregates`                            |
| Projection Dashboard | `*projectionhost.Host`       | `/projections`, reset POST               |
| Dead-Letter Queue    | `DeadLetterStore`            | `/dead-letters`, replay/delete/purge     |
| Command Audit        | `command.CommandJournal`     | `/commands`                              |
| Query Audit          | `query.QueryJournal`         | `/queries`                               |
| Time-Travel          | `EventSource`                | `/time-travel/{type}/{id}`               |
| Snapshot Inspector   | `snapshot.SnapshotStore`     | `/snapshots`                             |
| SSE Live Updates     | `event.Bus`                  | `/-/events/stream`                       |
| Probes               | always                       | `/-/healthz`, `/-/readyz`, `/-/versionz` |

**Gaps an admin interface needs but doesn't exist today:**

1. **No metaengine panel** — module not consumed by cqrs-htmx at all.
2. **No storage health** — no disk usage, table sizes, WAL size, pool stats.
3. **No metrics/throughput** — no events/sec, write-amp, error-rate trend.
4. **No topology view** — no "what's wired" picture.
5. **No multi-store aggregation** — one Dashboard binds one set of interfaces.

### 3.4 Existing introspection surfaces (raw material)

The metaengine already has rich observability, but it returns ad-hoc types:

| Method                                                  | File                              | Returns                              |
| ------------------------------------------------------- | --------------------------------- | ------------------------------------ |
| `Store.Plan()`                                          | `metaengine/store.go:29`          | Full PlanResult                      |
| `Store.ExplainPlan()`                                   | `metaengine/explain.go:147`       | Human-readable plan string           |
| `Store.Doctor(ctx)`                                     | `metaengine/explain.go:199`       | Health + stats + poisoned report     |
| `Store.Inspect()` / `InspectJSON()`                     | `metaengine/sse.go:375,399`       | Collection metadata                  |
| `Store.Stats(ctx)`                                      | `metaengine/stats.go:19`          | Per-collection row counts            |
| `Store.HealthCheck(ctx)`                                | `metaengine/stats.go:63`          | Pings every engine                   |
| `Store.Collections()`                                   | `metaengine/store.go:53`          | Sorted collection metadata           |
| `Store.ReplicationMode(queryName)`                      | `metaengine/store.go:104`         | Per-query topology                   |
| `Store.IsPoisoned(collection)`                          | `metaengine/store.go:116`         | Poison error check                   |
| `Store.EventTypes()`                                    | `metaengine/store.go:125`         | Event types listened to              |
| `Store.Export(ctx, w)` / `Import(ctx, r)`               | `metaengine/export_import.go`     | JSON dump/restore                    |
| `Store.Verify(ctx, engines)`                            | `metaengine/consistency.go:53`    | Cross-engine consistency             |
| `PlanResult.Report()`                                   | `metaengine/plan_types.go:98`     | Human plan w/ diagnostics            |
| `PlanResult.DotGraph()`                                 | `metaengine/observability.go:110` | DOT graph of query→ADT→engine        |
| `Bundle.Debug()` / `DebugStructured()`                  | `stack/debug.go:55,11`            | Per-capability ✓/✗                   |
| `Bundle.HealthCheck(ctx)`                               | `stack/health.go:28`              | Pings DB + all HealthChecker closers |
| `Bundle.Durability()` / `Capabilities()` / `DiskSize()` | various                           | Introspection metadata               |

**Gap:** No unified `Topology`/`DeploymentManifest` type. No single "give me a
JSON snapshot of the whole system" endpoint. The Bundle describes storage
capabilities only, not the wired CQRS topology (dispatchers, hosts, projections,
bus).

### 3.5 Scream store raw materials (what exists vs. what's missing)

**Exists (building blocks):**

- `SerializablePlan` with full JSON round-trip (`metaengine/serializable.go`)
- `PlanResult.Version` counter + `ComputedAt` timestamp (`plan_types.go:81-87`)
- Per-event `SchemaVersion` + upcaster chain (`schema/`, `event/types.go`)
- Poison detection (runtime scream on corruption) (`store_collaborators.go`)
- `Verify()` replay consistency check (`consistency.go`)
- Layout conflict detection for additive changes (`planned_sqlite.go`)
- 7 plan rules emitting WARN/DEGRADED/INFO diagnostics (`rules.go`)
- api-stability golden-file diff pattern (directly portable)
  (`cmd/api-stability/main.go`)

**Missing (what the scream store needs):**

- Plan diff engine (compare two SerializablePlans)
- Plan fingerprint (hash/canonical form)
- Deployment manifest (pinned golden plan persisted across deploys)
- `SCREAM` severity level (a diagnostic that blocks deployment)
- Cross-deploy comparison (Plan doesn't see the prior plan)
- Durability/persistence metadata on EngineProfile
- ADT/key-type pinning
- SQL migration version tracking (no migration runner today)

### 3.6 LiveStore architecture (the proven reference model)

[LiveStore](https://github.com/livestorejs/livestore) is a TypeScript event-
sourcing library that splits the event log from materialized state using
**two separate SQLite databases**. Research via agentic_fetch confirmed:

- **`dbEventlog`** — the append-only event log (source of truth).
  Tables: `EVENTLOG_META_TABLE`, `SYNC_STATUS_TABLE`.
- **`dbState`** — the materialized state (read model). Fully rebuildable from
  the event log. Application tables + session changesets.
- **Bootstrap order:** the event log boots first. If state is missing or schema
  changed, state is rebuilt by replaying all events through materializers.
- **Two coordinated but not cross-DB-atomic transactions.** Writes to both DBs
  happen in lockstep inside one uninterruptible Effect. Not atomic across DBs
  under crash; divergence heals via rebuild (state is derived).
- **The materializer** is a pure function: `event → SQL mutations → state`.
  Runs identically on every client.

LiveStore's core insight: **separating the write model from read models means
evolving a read model never requires migrating source data.** This is the same
insight behind go-cqrs-lite's event sourcing, but LiveStore makes the physical
split explicit (two DB handles).

### 3.7 samber/do v2 (the DI composition model)

`samber/do` v2 is the dominant DI library across Lars's workspace (72 modules,
~443 Go files). The redesign will use it as the composition root.

Relevant capabilities:

- **Scopes:** `root.Scope("sourcetruth")`, `root.Scope("projections")` — child
  injectors with lifecycle isolation. Each scope can be shut down independently.
- **Lazy singletons:** `do.Provide` for most services.
- **Eager foundation:** `do.ProvideValue` for config, logger, DB connection.
- **Named services:** `do.ProvideNamed` when multiple implementations of the
  same interface exist.
- **Lifecycle interfaces:** `HealthcheckerWithContext`, `ShutdownerWithContextAndError`.
  `injector.Shutdown()` runs shutdowns in reverse invocation order.
- **Anti-patterns to avoid:** DO-1 (`Must*` on runtime paths), DO-2 (`do.New()`
  without `Shutdown()`), DO-4 (package-level injector).

---

## 4. Decisions (Recorded)

Five architectural forks were presented to Lars via the question tool. Decisions
recorded below.

### 4.1 Backend selection: Hybrid (registry + config), leaning runtime

**Decision:** Somewhere between runtime config and hybrid. Drivers are
registered at compile time (which are available), but operator config picks
which to use and how — the `database/sql` model. Plus: option for HTTP admin
runtime config changes.

**Rationale:** The goal (G2) requires the operator to decide, not the developer.
Pure compile-time (current model) violates this. Pure runtime (full plugin
loading) is unnecessary and hurts performance (Go plugin overhead). The hybrid
model gives compile-time safety (only vetted drivers are linked) + runtime
flexibility (operator picks which to activate).

**Performance note:** Lars flagged that performance matters. The hybrid model
has zero runtime overhead — driver registration is a map insert at init time;
lookup is a map read at startup. After construction, all calls go directly to
the concrete implementation.

**Resolved:** Config is Go struct (canonical) + YAML + env, merged via
[koanf](https://github.com/knadh/koanf/). Koanf handles multi-source merging
(file + env + flags → struct) with multiple parser backends. The Go struct
remains the single source of truth; koanf is the input mechanism. See [§7.2](#72-config-format).

HTTP admin runtime-change mechanism is hot-reload for additive changes (add
cache tier, add read replica) and graceful restart for structural changes
(swap event log engine, change durability tier). A transactional config-swap
mechanism validates the new config against the scream store before applying.
Design details deferred to implementation.

### 4.2 Redesign scope: Parallel (new alongside old), gradual migration

**Decision:** Build a new `System` type alongside the existing `Bundle`.
Migrate gradually. Deprecate `Bundle` later.

**Rationale:** A clean-slate v5 (breaking) would force all consumers to migrate
simultaneously. Incremental evolution carries the structural debt forever. The
parallel approach lets early adopters use the new `System` while existing
consumers keep using `Bundle`. Both coexist until migration is complete.

### 4.3 Backend abstraction: N-instance metaengine with operator-configured grouping

**Decision:** The metaengine manages BOTH the source-of-truth logs AND
projections, via **N metaengine.Store instances** (not just two). The operator
decides how to group collections into instances and which engine(s) each
instance uses. The Log ADT already exists in the metaengine; it will be extended
to support the full event.Store interface.

This was the key insight (see [§5](#5-the-key-insight-multi-instance-metaengine)),
further refined by Lars's observation that each collection (events, commands,
queries) may need its own instance with its own engine — or may share an
instance with other collections. The operator decides.

**Rationale:**

- **One abstraction** (metaengine.Engine + ADT backends) for all storage —
  source-of-truth and projections alike.
- **N instances, not 2** — the operator groups collections into instances
  freely (e.g., events+commands on SQLite, queries on DuckDB, projections split
  by domain). Each instance has its own engine pool, durability tier, and
  lifecycle.
- **Hard invariants per layer** — source-of-truth instances use
  persistent-only engine pools (the planner literally can't route the event
  log to Memory because Memory isn't in the pool). Projection instances use
  mixed pools (the planner routes freely for cost optimization).
- **LiveStore-proven model** — dbEventlog + dbState split, generalized to any
  number of instances on any engine combination.
- **Cost-based optimization applies per instance** — each instance has its own
  plan; the planner optimizes within the instance's engine pool.
- **Unified introspection** — one topology shows all N instances.

### 4.4 Admin web interface: Introspection API only

**Decision:** go-cqrs-lite ships a rich introspection API (Topology, Plan,
Health, Stats interfaces + JSON snapshot types). The UI code lives outside
go-cqrs-lite (cqrs-htmx/dashboardui renders everything).

**Rationale:** Separation of concerns. go-cqrs-lite is a library, not an app.
The UI is a consumer concern. cqrs-htmx/dashboardui already has a capable
dashboard framework; it just needs data to render.

### 4.5 Scream store: Tiered enforcement

**Decision:** Three tiers of safety enforcement:

1. **Loud warn + override flag** — for changes that are risky but may be
   intentional (e.g., durability downgrade, adding a degraded ADT). The system
   logs loudly, the dashboard shows red, and the operator can acknowledge with
   an explicit flag.
2. **Advisory (dashboard red only)** — for informational diagnostics that don't
   block startup but should be visible (e.g., replication lag, write
   amplification).
3. (Implied) **Hard block** — for changes that would cause data loss or
   corruption (e.g., removing a persistent engine that holds the only copy of
   data). Refuse to start unless the operator explicitly force-migrates.

**Rationale:** Lars selected "Loud warn + override" AND "Advisory" — a tiered
model. The system should scream loudly but not prevent informed operator
decisions, except for genuinely destructive changes.

---

## 5. The Key Insight: Multi-Instance Metaengine

### 5.1 The mistake that was corrected

During the Q&A, the question arose: should source-of-truth stores and projection
engines share ONE backend abstraction? Three options were presented:

- **A: Unified Backend** — one Backend type with role-mixin interfaces.
- **B: Parallel Registries** — two registries composed at the top.
- **C: Metaengine absorbs everything** — the event log becomes "just another
  engine."

Initial analysis rejected C with five objections:

1. Bootstrap circular dependency (metaengine needs event log to plan).
2. Write-path overhead (fold/apply pipeline tax on every event.Save).
3. Lost optimistic concurrency (no ADT has expectedVersion).
4. Dependency-direction violation (decider would import metaengine).
5. Conflated schema-evolution models.

**Lars's insight broke all five objections:** "Who says we can only have 1
instance of the MetaEngine Service?"

Multiple metaengine instances dissolves the circular dependency. The log
instances boot first. The projection instances boot second and read the
events instance's journal. It's a DAG, not a cycle.

**Lars's second refinement:** there should not be just TWO instances (one log,
one projection). Each collection (events, commands, queries) may get its own
instance — or share one — at the operator's discretion. Same for projections.
This is critical because source-of-truth collections have **hard invariants**
(must be persistent, must be durable) that the cost-planner must not override.
Separate instances with constrained engine pools enforce these invariants
structurally rather than hoping the planner respects them.

The model is **N instances across two semantic layers** (source-of-truth +
projections), with the operator deciding the grouping:

### 5.2 Verification: the Log ADT already exists

The metaengine **already has** a first-class Log ADT:

- `ADTLog` is defined in `metaengine/types.go:11`.
- `LogBackend` interface exists at `metaengine/engine.go:309`:
  `LogAppend(ctx, collection, value)` + `LogTail(ctx, collection, limit)`.
- **All 5 engines implement it** (Memory, SQLite, Pebble, DuckDB, Postgres —
  all declare `ADTLog` in their profiles with complexity ratings).
- The `Append` sentinel type exists for fold declaration (`types.go:59`).
- The fold classifier already maps `func(e) Append` → `ADTLog`
  (`fold_classify.go:52`).
- Cross-engine parity is tested (`adttest/harness.go:328`, `restart_test.go`,
  `concurrent_gaps_test.go:91`).

The metaengine can already BE a log. What's missing:

1. **Richer LogBackend operations** — current is LogAppend + LogTail. event.Store
   needs stream-keyed Save(ctx, ref, events, expectedVersion), Load(ctx, ref),
   LoadFromVersion, LoadToTimestamp, ReadAll (journal), ReadFrom (seekable).
   These are extensions to the existing ADT, not a redesign.
2. **A `metaengine-eventstore` adapter** — wraps a Log ADT collection as
   `event.Store` (like `projectionadapter` wraps a Store as
   `projection.Projection`).
3. **N-instance composition** — multiple `*metaengine.Store` instances wired
   via journals, managed by samber/do scopes. The operator decides how many
   instances and which collections go in each.

### 5.3 Why each objection dissolves

| Original objection                 | Why it dissolves with N instances                                                                                                                                                                           |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Bootstrap circular dependency**  | Log instances boot first. Projection instances boot second and read the events journal. DAG, not cycle.                                                                                                     |
| **Write-path overhead**            | The Log fold is `func(e) Append{Value: e}` — literally "append." One map lookup + one write. Nanoseconds. Same as `event.Save` today.                                                                       |
| **Lost optimistic concurrency**    | The engine already has `Transactional`/`RunInTx` + `MapUpdater` (atomic read-modify-write). ExpectedVersion = a RunInTx that reads current version, checks, appends. Engine capability, not ADT limitation. |
| **Dependency-direction violation** | The decider imports `event.Store` (Tier 1 interface) — unchanged. The metaengine Log instance _implements_ that interface. Dependency inversion, not violation.                                             |
| **Conflated schema evolution**     | Separate instances = separate evolution models. Event upcasters on log instances. Layout migration on projection instances.                                                                                 |

### 5.4 Why N instances, not 2 (the invariant constraint argument)

The metaengine cost-planner optimizes for **cheapest** assignment. If events,
commands, and queries share one Store instance with a mixed engine pool
(Memory + SQLite), the planner could route the **event log to Memory** (O(1)
is cheaper than SQLite O(logN)) — and you lose everything on restart.

Source-of-truth collections have **hard invariants** the planner must not
override:

- **Persistence required** — the event log MUST survive restart.
- **Durability required** — commits MUST be fsync'd (or the operator accepts
  data loss explicitly).
- **Optimistic concurrency** — `Save(ctx, ref, events, expectedVersion)` MUST
  be atomic (check-and-append under a lock/transaction).

Projection collections have **preferences**, not invariants:

- Persistence is nice-to-have (projections are rebuildable from the event log).
- Durability is a performance tradeoff (Relaxed is fine for projections that
  rebuild fast).
- No optimistic concurrency needed (projections are derived, not authoritative).

**The structural solution:** separate instances with constrained engine pools.

```go
// Source-of-truth instances: only persistent engines in the AUTHORITATIVE pool.
// The planner CANNOT route the event log to Memory — it's not in the list.
// (Memory can still serve as a read-through CACHE tier — see §5.5.)
eventsStore, _   := metaengine.Plan([]Engine{primaryEng}, eventsQuery)
commandsStore, _ := metaengine.Plan([]Engine{primaryEng}, commandsQuery)
queriesStore, _  := metaengine.Plan([]Engine{analyticsEng}, queriesQuery)

// Projection instances: mixed pool — planner routes freely for cost.
projStore, _     := metaengine.Plan([]Engine{memEng, primaryEng, pebbleEng}, projQueries...)
```

The constraint is enforced structurally (the authoritative engine pool), not
via advisory rules. The planner literally cannot make the wrong choice because
the wrong engine isn't available. Volatile engines can still participate as
**cache tiers** (§5.5) — read-only, never authoritative.

### 5.5 The Cache Tier: ZFS-ARC for Immutable Events

Lars's insight: events are immutable — so we can cache them in RAM without any
invalidation concern. This is the same property ZFS exploits with its ARC
(Adaptive Replacement Cache): copy-on-write blocks are never modified in place,
so the cache is always consistent.

**Why this is architecturally significant (not "just caching"):**

Cache invalidation is the hardest problem in caching. In mutable systems, every
write can stale cache entries, requiring complex invalidation protocols. Event
sourcing eliminates this entirely: once written, an event is valid forever. The
cache never goes stale — the only concern is eviction (capacity management).

**The model:** each instance can have an optional **read-through cache tier**
sitting in front of its authoritative engine:

```
Read  → [Cache (Memory)] → miss → [Authoritative (SQLite)]
              ↑                           │
              └─── populate on miss ──────┘

Write → [Authoritative (SQLite)]     (cache is not written to directly;
       ↑                              events are immutable so there is
       └── no invalidation needed      no stale-entry problem)
```

Writes always go to the authoritative engine (persistent). Reads check the
cache first; on a miss, they read through to the authoritative engine and
populate the cache. The cache is volatile — a dropped entry is just a cache
miss, never data loss.

**When this helps:**

| Scenario                     | Why                                                                     |
| ---------------------------- | ----------------------------------------------------------------------- |
| Hot aggregate loads          | Decider loads the same stream repeatedly. First load warms cache.       |
| Parallel projection rebuilds | Multiple projections replay the same log. First warms; others read RAM. |
| CatchUpSubscriber replay     | Live handoff replays recent events. Cache holds the recent tail.        |
| Time-travel queries          | Historical state reconstruction reads events up to a version.           |

**When it does NOT help:** write-once-read-rarely streams (most event logs in
practice), or very large volumes where total events >> RAM (low hit rate).

**Design decision: instance-level concern, NOT planner concern.**

The metaengine planner assigns engines to queries (a one-time cost decision).
Caching is orthogonal — it avoids the query entirely (runtime behavior).
Conflating them violates separation of concerns. The cache is a transparent
read-through wrapper around the instance's `event.Store` adapter. The planner
never knows about it.

**Complementary to `decider.StateCache`** — they cache at different levels:

| Cache             | What it caches | Cache hit effect                                |
| ----------------- | -------------- | ----------------------------------------------- |
| `StateCache`      | Folded state   | Skips event loading entirely (loads delta only) |
| Event cache (NEW) | Raw events     | Speeds up event loading on StateCache miss      |

**Config (named engines — see [§7.1](#71-driver-registry-the-databasesql-model)):**

```yaml
engines:
  primary: { driver: sqlite, dsn: "file:events.db" }
  hot-cache: { driver: memory }

instances:
  - role: events
    engine: primary # authoritative (persistent)
    cache: # optional read-through cache tier
      engine: hot-cache # references the named Memory engine
      # No invalidation policy — events are immutable.
      # Only eviction policy: LRU (default), or ARC (future).
```

**Scream store interaction:** removing a cache tier is always safe (read-through
falls back to authoritative). Adding a cache tier is ADVISORY at most. Changing
the authoritative engine from persistent to volatile is SCREAM (§9).

---

## 6. Target Architecture

### 6.1 High-level diagram (N-instance model)

The operator decides how many instances and how to group collections. Three
example topologies, from minimal to maximum isolation:

```
═══════════════════════════════════════════════════════════════
TOPOLOGY 1: Minimal (one instance per layer — SQLite + Memory only)
═══════════════════════════════════════════════════════════════

samber/do Root Scope
│
├── Source of Truth Layer
│   └── Instance: logs-store (engine: SQLite)
│       ├── "events"     (Log ADT) → event.Store
│       ├── "commands"   (Log ADT) → command.Store
│       ├── "queries"    (Log ADT) → query.Store
│       ├── "snapshots"  (Map ADT) → snapshot.SnapshotStore
│       └── "checkpoints" (Map ADT) → event.CheckpointStore
│
├── Projection Layer
│   └── Instance: proj-store (engines: Memory + SQLite, planner routes)
│       ├── "task_views"  (Map + FilterOnField)
│       └── "task_counts" (Counter)
│
└── System Scope
    ├── decider.Repository    ← writes to logs-store["events"]
    ├── projectionhost.Host   ← reads logs-store["events"] journal
    │                           feeds proj-store
    └── Introspection API     ← queries both layers

═══════════════════════════════════════════════════════════════
TOPOLOGY 2: Split (LiveStore-style — separate DBs per concern)
═══════════════════════════════════════════════════════════════

samber/do Root Scope
│
├── Source of Truth Layer
│   ├── Instance: events-store   (engine: SQLite, events.db)
│   │   └── "events" + "snapshots" + "checkpoints"
│   ├── Instance: commands-store (engine: SQLite, events.db, shared conn)
│   │   └── "commands"
│   └── Instance: queries-store  (engine: DuckDB, analytics.db)
│       └── "queries"  (analytics-grade audit log)
│
├── Projection Layer
│   ├── Instance: task-projections (engines: SQLite + Memory)
│   │   └── "task_views" + "task_counts"
│   └── Instance: graph-projections (engine: Pebble)
│       └── "user_graph"  (LSM-optimized adjacency)
│
└── System Scope (same wiring, multiple instances)

═══════════════════════════════════════════════════════════════
TOPOLOGY 3: Maximum isolation (one instance per collection)
═══════════════════════════════════════════════════════════════

samber/do Root Scope
│
├── Source of Truth Layer
│   ├── Instance: events     (engine: SQLite, events.db,     durability: strict)
│   ├── Instance: commands   (engine: SQLite, commands.db,   durability: normal)
│   ├── Instance: queries    (engine: DuckDB, queries.db)
│   ├── Instance: snapshots  (engine: SQLite, snapshots.db)
│   └── Instance: checkpoints (engine: SQLite, checkpoints.db)
│
├── Projection Layer
│   ├── Instance: task_views   (engine: SQLite, views.db)
│   ├── Instance: task_counts  (engine: Memory)
│   └── Instance: user_graph   (engine: Pebble, /data/graph)
│
└── System Scope (same wiring, fine-grained instances)
```

**Key invariant:** source-of-truth instances always use persistent-only engine
pools. The planner cannot route the event log to a volatile engine. Projection
instances use mixed pools — the planner routes freely for cost optimization.

### 6.2 The consumer experience (target state)

The consumer declares ONLY domain types, query patterns, and folds. The
operator provides engines. The system wires everything.

```go
// ── CONSUMER CODE (deployment-agnostic) ──

// 1. Domain types (pure Go structs)
type TaskCreated struct { ID TaskID; Title string; At time.Time }
type TaskCompleted struct { ID TaskID }

// 2. Query declarations (fold return types infer ADTs)
taskViews := metaengine.Query[ListTasks, TaskView]("task_views",
    metaengine.OnEvent("task.created", TaskCreated{},
        func(s id.StreamID, e TaskCreated) (string, TaskView) {
            return s.String(), TaskView{ID: e.ID, Title: e.Title, Status: "pending"}
        }),
    metaengine.OnEvent("task.completed", TaskCompleted{},
        func(_ id.StreamID, _ TaskCompleted, prev TaskView) TaskView {
            prev.Status = "completed"; return prev
        }),
    metaengine.FilterOnField[TaskView]("status", metaengine.FilterEq),
)

taskCounts := metaengine.Query[StatsInput, map[string]int64]("task_counts",
    metaengine.OnEvent("task.created", TaskCreated{},
        func(_ TaskCreated) metaengine.Delta { return metaengine.Delta{"pending": 1} }),
)

// 3. System construction (one call — operator config decides engines)
sys, err := system.New(ctx, system.Config{
    Queries: []metaengine.QueryDecl[any, any]{taskViews, taskCounts},
    // NO engine list, NO DSN, NO *sql.DB — that's the operator's job
})
```

```go
// ── OPERATOR CODE (deployment-specific) ──
//
// Engines are DECLARED by name (semantic roles, not engine types).
// Instances REFERENCE engines by name. Swap DuckDB→ClickHouse by changing
// the declaration — the topology stays the same.

// Minimal: one instance per layer
sys, err := system.New(ctx, system.Config{
    Queries: queries, // consumer's query declarations
    Engines: map[string]system.EngineConfig{
        "primary":   {Driver: "sqlite", DSN: "file:app.db"},
        "hot-cache": {Driver: "memory"},
    },
    Instances: []system.InstanceConfig{
        {
            Role:        system.RoleSourceOfTruth,
            Collections: []string{"events", "commands", "queries", "snapshots", "checkpoints"},
            Engine:      "primary",
            Durability:  system.DurabilityStrict,
        },
        {
            Role:        system.RoleProjections,
            Collections: []string{"task_views", "task_counts"},
            Engines:     []string{"primary", "hot-cache"}, // planner routes
            Durability:  system.DurabilityNormal,
        },
    },
})

// Split: events+commands on SQLite, queries on DuckDB, graph on Pebble
sys, err = system.New(ctx, system.Config{
    Queries: queries,
    Engines: map[string]system.EngineConfig{
        "primary":    {Driver: "sqlite",  DSN: "file:events.db"},
        "analytics":  {Driver: "duckdb",  DSN: "file:analytics.db"},
        "views":      {Driver: "sqlite",  DSN: "file:views.db"},
        "hot-cache":  {Driver: "memory"},
        "graph-lsm":  {Driver: "pebble",  DSN: "/data/graph"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleEvents,      Engine: "primary",   Durability: system.DurabilityStrict},
        {Role: system.RoleCommands,    Engine: "primary"},                     // shared connection
        {Role: system.RoleQueries,     Engine: "analytics"},                   // OLAP-grade audit
        {Role: system.RoleProjections, Collections: []string{"task_views", "task_counts"},
            Engines: []string{"views", "hot-cache"}},
        {Role: system.RoleProjections, Collections: []string{"user_graph"},
            Engine: "graph-lsm"},
    },
})
```

### 6.3 The key difference from the current stack

| Aspect                 | Current (stack.Bundle)         | Target (system.New)                       |
| ---------------------- | ------------------------------ | ----------------------------------------- |
| Who picks engines      | Consumer (hardcoded in Go)     | Operator (config string)                  |
| Who opens DB           | Consumer (`sql.Open`)          | System (via driver registry)              |
| Event decoder          | Consumer writes 77-line switch | System auto-decodes via `OnEvent`         |
| Metaengine integration | Bolted-on (`WithMetaEngine`)   | First-class (N instances)                 |
| Event log storage      | Separate from metaengine       | metaengine Log instance(s)                |
| Instance count         | Fixed (1 Bundle)               | N — operator decides grouping             |
| Multi-DB support       | Partial (metaengine re-opens)  | Native (one DB per instance, or shared)   |
| Source-of-truth safety | None                           | Persistent-only engine pools (structural) |
| Admin topology         | None                           | Unified (all instances)                   |
| Backend swap           | Recompile                      | Config change (+ restart or hot-reload)   |

### 6.4 LiveStore parallel

LiveStore splits `dbEventlog` + `dbState` into two SQLite databases. This design
generalizes that split to **any engine combination** and adds cost-based
optimization to both layers:

|                        | LiveStore                      | This design                                            |
| ---------------------- | ------------------------------ | ------------------------------------------------------ |
| Event log DB           | `dbEventlog` (SQLite, fixed)   | N log instances (any engine, operator picks)           |
| State DB               | `dbState` (SQLite, fixed)      | N projection instances (any engine(s), planner routes) |
| Instance count         | Fixed (2)                      | N per layer (1..many, operator decides)                |
| Materializer           | Pure function `event → SQL`    | Pure function `event → fold → ADT write`               |
| Bootstrap              | Event log first, state derived | Same (log instances boot first)                        |
| Rebuild                | Replay all events              | Same                                                   |
| Cost optimization      | None                           | Cost-based planner per instance                        |
| Source-of-truth safety | None                           | Persistent-only engine pools                           |

---

## 7. Operator Configuration Surface

### 7.1 Driver registry (the database/sql model)

```go
// Drivers register at init time (compile-time safety):
import _ "github.com/larsartmann/go-cqrs-lite/drivers/sqlite"
import _ "github.com/larsartmann/go-cqrs-lite/drivers/pebble"
import _ "github.com/larsartmann/go-cqrs-lite/drivers/duckdb"
// The binary now has sqlite + pebble + duckdb available.

// Engines are declared BY NAME (operator assigns semantic names).
// Instances reference engines BY NAME — swap engine type without touching
// the instance topology.
config := system.Config{
    Engines: map[string]system.EngineConfig{
        "primary":   {Driver: "sqlite", DSN: "file:events.db"},
        "analytics": {Driver: "duckdb", DSN: "file:analytics.db"},
        "views":     {Driver: "sqlite", DSN: "file:views.db"},
        "hot-cache": {Driver: "memory"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleEvents,     Engine: "primary"},   // persistent
        {Role: system.RoleCommands,   Engine: "primary"},   // shared connection
        {Role: system.RoleQueries,    Engine: "analytics"}, // OLAP engine
        {Role: system.RoleProjections, Engines: []string{"views", "hot-cache"}}, // planner routes
    },
}
```

This is directly analogous to `database/sql`:

- `import _ "modernc.org/sqlite"` registers the driver.
- `sql.Open("sqlite", dsn)` looks it up by name.

The two-level indirection (driver type → named engine → instance reference)
means the operator can swap `analytics` from DuckDB to ClickHouse by changing
ONE declaration — no instance topology changes needed.

### 7.2 Config format

TBD — see [§10](#10-open-questions). Options: Go struct (preferred for library
SDK), YAML (for file-based deployment), env vars (for 12-factor). All three can
coexist (Go struct is canonical; YAML/env are parsed into it).

### 7.3 HTTP admin runtime config

Lars requested "option for HTTP admin runtime config changes." This implies:

- A config endpoint served by the introspection API.
- Changes are validated by the scream store before applying.
- Some changes are hot-reloadable (e.g., adding a read replica); others require
  graceful restart (e.g., changing the event log engine).

Design TBD — see [§10](#10-open-questions).

### 7.4 Multi-DB SQLite (goal G5)

The operator can split databases by specifying separate DSNs per instance. The
N-instance model makes this natural — each instance can use a different engine
or the same engine with a different DSN:

```go
// Three-way SQLite split (the goal's exact example)
config := system.Config{
    Engines: map[string]system.EngineConfig{
        "events-db":  {Driver: "sqlite", DSN: "file:events.db"},
        "queries-db": {Driver: "sqlite", DSN: "file:queries.db"},
        "views-db":   {Driver: "sqlite", DSN: "file:views.db"},
        "hot-cache":  {Driver: "memory"},
    },
    Instances: []system.InstanceConfig{
        // DB 1: Command + Event Sourcing
        {Role: system.RoleEvents,   Engine: "events-db",  Durability: system.DurabilityStrict},
        {Role: system.RoleCommands, Engine: "events-db"},  // shared connection
        // DB 2: Query logs
        {Role: system.RoleQueries,  Engine: "queries-db"},
        // DB 3: Materialized views
        {Role: system.RoleProjections, Engine: "views-db"},
    },
}

// Or: SQLite for events + DuckDB for query analytics (different engines!)
config = system.Config{
    Engines: map[string]system.EngineConfig{
        "events-db": {Driver: "sqlite", DSN: "file:events.db"},
        "analytics": {Driver: "duckdb", DSN: "file:queries.db"},
        "views-db":  {Driver: "sqlite", DSN: "file:views.db"},
        "hot-cache": {Driver: "memory"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleEvents,     Engine: "events-db"},
        {Role: system.RoleCommands,   Engine: "events-db"},
        {Role: system.RoleQueries,    Engine: "analytics"},    // columnar audit
        {Role: system.RoleProjections, Engines: []string{"views-db", "hot-cache"}},
    },
}
```

---

## 8. Introspection API (for cqrs-htmx)

### 8.1 Design principle

go-cqrs-lite ships **structured types** + **JSON snapshot methods**. cqrs-htmx
renders them. No HTML, no handlers, no embedded UI in go-cqrs-lite.

### 8.2 The unified Topology type (NEW — does not exist today)

```go
// system.Topology describes the entire wired deployment as a graph.
// This is the single "what's actually running" snapshot for admin UIs.
// Supports N instances across both layers.
type Topology struct {
    Instances     []InstanceTopology  // all metaengine.Store instances
    Bus           string              // "watermill-gochannel", "nats", ""
    Dispatchers   []DispatcherInfo
    ProjectionHost *ProjectionHostInfo // nil if not configured
    ScreamStore    *ScreamReport       // nil if not configured
}

type InstanceTopology struct {
    Name          string              // operator-assigned instance name
    Role          InstanceRole        // RoleEvents, RoleCommands, RoleQueries, RoleProjections
    EngineName    string              // named engine reference (e.g., "primary", "analytics")
    Capabilities  Capabilities
    Collections   []CollectionInfo   // from metaengine.Store.Collections()
    Plan          *SerializablePlan  // from metaengine.Store.Plan()
    Durability    DurabilityTier
    DiskUsage     int64
    HealthStatus  HealthStatus
    Cache         *CacheTierInfo     // nil if no cache tier configured
}

type CacheTierInfo struct {
    EngineName string  // named cache engine (e.g., "hot-cache")
    HitRate   float64 // cache hits / total reads
    Size      int     // current entry count
    MaxSize   int     // capacity limit
}

type InstanceRole string
const (
    RoleEvents      InstanceRole = "events"
    RoleCommands    InstanceRole = "commands"
    RoleQueries     InstanceRole = "queries"
    RoleSnapshots   InstanceRole = "snapshots"
    RoleProjections InstanceRole = "projections"
)
```

### 8.3 The introspection methods (wiring existing surfaces)

The new `system.System` type exposes:

```go
// System.Snapshot returns a JSON-serializable topology snapshot.
func (s *System) Snapshot(ctx context.Context) (*Topology, error)

// System.Health returns aggregated health across all scopes.
func (s *System) Health(ctx context.Context) HealthStatus

// System.Plan returns the combined plan (both scopes).
func (s *System) Plan() *CombinedPlan

// System.Explain returns a human-readable explanation.
func (s *System) Explain(ctx context.Context) string

// System.Verify runs consistency checks across all scopes.
func (s *System) Verify(ctx context.Context) error
```

These compose the existing surfaces:

- `metaengine.Store.Collections()` / `Plan()` / `Doctor()` / `Stats()` /
  `HealthCheck()` / `ExplainPlan()` — per scope.
- `projectionhost.Host.Status()` / `LagPerProjection()` — projections.
- `stack.Bundle.DebugStructured()` — capabilities.
- The scream store report (see [§9](#9-scream-store-safety-model)).

### 8.4 What cqrs-htmx/dashboardui gets

With this API, dashboardui can add panels for:

- **Metaengine plan** — which engines serve which queries, cost estimates,
  diagnostics (DEGRADED/WARN/INFO).
- **Storage health** — disk usage, row counts, WAL size, connection pool stats.
- **Topology graph** — a visual "what's wired" picture (engines, stores,
  projections, bus, dispatchers).
- **Scream store status** — red/yellow/green per safety check.
- **Metrics** — events/sec, write-amp, error-rate, projection lag.

---

## 9. Scream Store (Safety Model)

### 9.1 Concept

A mechanism that **SCREAMS BIG TIME** and prevents failures when an operator
makes an unsafe metaengine or deployment change. Think: `cmd/api-stability` for
runtime plans instead of Go export surfaces.

### 9.2 Architecture

```
┌──────────────────────────────────────────────────┐
│             Scream Store                          │
│                                                   │
│  ┌─────────────┐    ┌─────────────┐              │
│  │  Pinned     │    │  Current    │              │
│  │  Manifest   │───▶│  Plan       │              │
│  │  (golden)   │    │  (computed) │              │
│  └─────────────┘    └─────────────┘              │
│         │                  │                      │
│         │     ┌────────────┘                      │
│         │     │                                   │
│         ▼     ▼                                   │
│  ┌─────────────────┐                              │
│  │  Plan Diff      │                              │
│  │  Engine         │                              │
│  └─────────┬───────┘                              │
│            │                                      │
│            ▼                                      │
│  ┌──────────────────────────────────────┐         │
│  │  Safety Rules (tiered)                │         │
│  │                                       │         │
│  │  SCREAM  (hard block)                 │         │
│  │  ├── Persistent engine removed        │         │
│  │  ├── Projection key type changed      │         │
│  │  └── ADT changed for existing col     │         │
│  │                                       │         │
│  │  WARN+OVERRIDE (loud, acknowledge)    │         │
│  │  ├── Durability downgraded            │         │
│  │  ├── Replicated → non-replicated      │         │
│  │  ├── Degraded ADT assigned            │         │
│  │  └── Write-amp budget exceeded        │         │
│  │                                       │         │
│  │  ADVISORY (dashboard red)             │         │
│  │  ├── Replication lag > threshold      │         │
│  │  ├── Estimated latency > budget       │         │
│  │  └── Cost drift detected              │         │
│  └──────────────────────────────────────┘         │
└──────────────────────────────────────────────────┘
```

### 9.3 The pinned manifest

The system persists a `SerializablePlan` snapshot (the "golden file") at
deployment time. On startup, the current plan is compared against the pinned
manifest. This is directly analogous to `cmd/api-stability`:

| api-stability (existing)             | Scream store (new)                       |
| ------------------------------------ | ---------------------------------------- |
| Collects exported Go symbols via AST | Collects plan via `SerializablePlan`     |
| Compares against golden file         | Compares against pinned manifest         |
| SCREAMS (exit 1) on removed exports  | SCREAMS (refuse start) on unsafe changes |
| `docs/api_surface.txt`               | `plan.pin.json`                          |

### 9.4 Unsafe changes the scream store catches

| Change                                   | Tier          | Failure mode                          |
| ---------------------------------------- | ------------- | ------------------------------------- |
| Removing a persistent engine (data loss) | SCREAM        | Collections silently empty on restart |
| Changing a projection's key type         | SCREAM        | Existing rows unfindable              |
| Changing ADT for an existing collection  | SCREAM        | Stored shape incompatible             |
| Removing a projection queries depend on  | SCREAM        | Runtime ErrNoQueryForInputType        |
| Durability downgrade (SQLite→Memory)     | WARN+OVERRIDE | Data loss on crash                    |
| Replicated → non-replicated swap         | WARN+OVERRIDE | Consistency loss                      |
| Degraded ADT assigned                    | WARN+OVERRIDE | Performance degradation               |
| Write-amp budget exceeded                | WARN+OVERRIDE | Write throughput drop                 |
| Replication lag > threshold              | ADVISORY      | Stale reads                           |
| Estimated latency > budget               | ADVISORY      | Slow queries                          |
| Cost drift (estimated vs actual)         | ADVISORY      | Cost model inaccuracy                 |

### 9.5 Implementation building blocks (all exist)

- `SerializablePlan` (`metaengine/serializable.go`) — JSON round-trip ready
- `PlanResult.Version` + `ComputedAt` (`metaengine/plan_types.go:81-87`)
- Poison detection (`metaengine/store_collaborators.go`) — runtime scream on
  corruption
- `Verify()` replay consistency check (`metaengine/consistency.go:53`)
- 7 plan rules with WARN/DEGRADED/INFO (`metaengine/rules.go`)
- `cmd/api-stability/main.go:216-244` — the diff function (~28 lines, directly
  portable)

**Missing pieces to build:**

- `PlanDiff(prev, current *SerializablePlan) (*DiffResult, error)`
- `PlanFingerprint(plan *SerializablePlan) string` (canonical hash)
- `Manifest` type (pinned plan + metadata)
- `SCREAM` severity level on `Diagnostics`
- Safety rules (the table above, codified)

---

## 10. Open Questions

These need resolution before implementation. Ordered by blocking impact.

### 10.1 Log ADT extension scope

The current `LogBackend` has `LogAppend` + `LogTail`. `event.Store` needs:
stream-keyed `Save(ctx, ref, events, expectedVersion)`, `Load(ctx, ref)`,
`LoadFromVersion`, `LoadToTimestamp`, `ReadAll` (journal), `ReadFrom` (seekable).

**Question:** Do we extend `LogBackend` with stream-keyed operations, or do we
create a new `EventLogBackend` interface that engines implement additionally?

### 10.1b Instance grouping defaults

**Question:** What are the default instance groupings if the operator doesn't
specify any? Options: (a) one instance per collection type (events, commands,
queries, snapshots, checkpoints — 5 instances), (b) one instance per layer (2
instances), (c) one instance for everything (1 instance, like current Bundle).
Probably (b) is the right default — simple but separated.

### 10.2 Config format

**Question:** Go struct (canonical) + YAML parser + env var override? Or just
Go struct (operators write `config.go`)? Or all three?

### 10.3 HTTP admin runtime changes

**Question:** Which changes are hot-reloadable (no restart) vs require graceful
restart? Is there a transactional config-swap mechanism?

### 10.4 Driver registration API

**Question:** `init()`-based (like database/sql) or explicit `system.Register()`
call? `init()` is more Go-idiomatic but harder to test.

### 10.5 samber/do scope boundaries

**Question:** Do source-of-truth and projections get separate `do.Scope` instances
with independent shutdown, or are they providers within the root scope?

### 10.6 Migration path from Bundle

**Question:** Do presets (sqlite, pebble, etc.) get dual implementations (Bundle +
System), or do they migrate one at a time?

### 10.7 Bus and dispatcher integration

The event bus, command dispatcher, and query dispatcher are currently wired
outside the Bundle by the consumer. The new System should own them.

**Question:** Does the System auto-construct a watermill GoChannel bus by
default, with operator config to swap for NATS/Redis? Or does the consumer still
pass the bus?

### 10.8 Codec defaults

Currently different layers have different default codecs (events: CBOR, read
models: CBOR, stack: configurable).

**Question:** Does the System unify codec selection? Does the operator configure
it, or is it always CBOR?

### 10.9 Cache tier policy

**Question:** What eviction policy for the event cache? LRU (simple, default)?
ARC (ZFS-style adaptive recency+frequency, optimal but complex)? Clock
(approximation of LRU, simpler)? Should the cache be per-stream or global?

### 10.10 Named engine sharing semantics

**Question:** When two instances reference the same named engine (e.g., events +
commands both → "primary"), do they share a connection pool / *sql.DB, or does
each instance get its own connection? Shared is efficient; isolated is safer.

---

## 11. Glossary

| Term                     | Definition                                                                                                                                                                                                                                                    |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ADT**                  | Abstract Data Type. The metaengine infers the ADT from fold return types: `func(e)(K,V)`→Map, `func(e)Delta`→Counter, `func(e)Append`→Log, etc.                                                                                                               |
| **Backend**              | A storage engine implementation (SQLite, Pebble, Postgres, DuckDB, Memory). Each implements `metaengine.Engine` + whichever ADT backends it supports.                                                                                                         |
| **Bundle**               | The current composition root (`stack.Bundle`). A bag of optional capability fields. To be replaced by `system.System`.                                                                                                                                        |
| **Cache Tier**           | An optional read-through volatile cache (typically Memory) sitting in front of an instance's authoritative engine. Exploits event immutability: no invalidation needed, only eviction. NOT a planner concern — a transparent adapter wrapper.                 |
| **Named Engine**         | A declared engine configuration (driver + DSN + options) identified by a semantic name (e.g., "primary", "analytics", "hot-cache"). Instances reference engines by name, enabling swaps without topology changes.                                             |
| **Deployer / Operator**  | The person configuring the deployment. Picks engines, DSNs, durability. Does NOT write domain code.                                                                                                                                                           |
| **Consumer / Developer** | The person writing the application. Declares events, commands, queries, folds. Does NOT pick infrastructure.                                                                                                                                                  |
| **Fold**                 | A pure function that maps an event to a projection update. The return type determines the ADT.                                                                                                                                                                |
| **Instance**             | A single `*metaengine.Store` created by `metaengine.Plan(engines, queries)`. The redesign uses N instances (one or more per layer), each with its own engine pool. Source-of-truth instances use persistent-only pools; projection instances use mixed pools. |
| **Layer**                | A semantic grouping of instances: Source of Truth (logs, snapshots, checkpoints) or Projections (derived views). Each layer has 1..N instances.                                                                                                               |
| **Journal**              | The cross-stream event reader (`event.Journal.ReadAll`). Used by projectionhost to replay events.                                                                                                                                                             |
| **Log ADT**              | The append-only ordered log ADT (`metaengine.ADTLog`). Already implemented by all 5 engines. The redesign uses it for event/command/query storage.                                                                                                            |
| **Plan**                 | The output of `metaengine.Plan()`. Assigns engines to queries based on cost. Contains diagnostics, layouts, rule traces.                                                                                                                                      |
| **Projection**           | A derived view built from events. Has a fold function and query patterns.                                                                                                                                                                                     |
| **Scope**                | A samber/do child injector with lifecycle isolation. The redesign uses separate scopes per instance (or per layer for simpler setups). Each scope can be shut down independently.                                                                             |
| **Scream Store**         | The safety mechanism that detects and blocks unsafe operator changes by diffing the current plan against a pinned manifest.                                                                                                                                   |
| **SerializablePlan**     | A JSON-serializable snapshot of PlanResult, stripping runtime closures and reflect.Type values.                                                                                                                                                               |
| **Source of Truth**      | The event log (and command/query audit logs). The authoritative, immutable record.                                                                                                                                                                            |
| **System**               | The new composition root (replacing Bundle). Owns construction, lifecycle, topology, and introspection.                                                                                                                                                       |
| **Topology**             | A structured snapshot of the entire wired deployment (engines, stores, projections, bus, dispatchers).                                                                                                                                                        |

---

_End of document. Implementation begins when open questions are resolved._
