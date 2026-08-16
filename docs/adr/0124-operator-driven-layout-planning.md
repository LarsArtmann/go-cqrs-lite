# ADR-0124: Operator-Driven Layout Planning

**Date:** 2026-08-11
**Status:** Accepted
**Related:** ADR-0116 (layered auto-projection), ADR-0111 (Record type), ADR-0062 (metaengine dependency boundary), ADR-0063 (metaengine pushdown), `METAENGINE-LAYOUT-PLANNING-MODEL.md`, `METAENGINE-LIVE-LATENCY-MODEL.md`

---

## Context

The original M9 task proposed auto-generating child collections from `[]Attachment`
fields via reflection. **This was wrong** for three reasons:

1. **Normalization is not universally correct.** Whether to embed a slice as a
   JSON blob or split it into a child collection + join depends on the backend
   (KV favors embed, SQL favors normalize, graph favors normalize) and the
   workload (append-heavy children favor normalize for write speed, read-whole-
   aggregate favors embed for read speed). Hardcoding "always split" throws away
   the metaengine's core cost-based value.

2. **It puts storage intent on the developer.** The project's north star is
   unambiguous: *"Developers declare ONLY Commands + Events + Queries and their
   relationships... developers REALLY NEVER need to think about the storage
   layer."* M9 forced the developer to model types carefully to influence layout
   — a direct violation.

3. **It ignores the cost model.** The metaengine exists to be cost-based and
   multi-engine. M9 bypassed the cost model entirely with a reflection-based
   rule: "slice detected → split." This is the antithesis of cost-based planning.

Full design rationale: [`docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md`](../planning/METAENGINE-LAYOUT-PLANNING-MODEL.md).

### The `[]Attachment` vs `[]AttachmentID` Distinction

The type distinction (`[]Attachment` vs `[]AttachmentID`) is **payload reality**,
not storage intent:

- `[]Attachment` (full struct values) → embed is *possible* (bytes exist)
- `[]AttachmentID` (IDs only) → embed is *impossible* (no bytes to embed)

The developer's type choice *constrains* the planner's option set (domain shape).
The operator's priority *selects within* that option set (storage intent). A
constraint that eliminates an option is not the same as an intent that expresses
a preference.

---

## Decision

Layout planning (embed vs. normalize vs. hybrid) is **100% the operator's call**
via a priority system that weights the cost model. The developer expresses zero
storage intent — ever.

### Layer 4: Physical Layout (extends ADR-0116)

ADR-0116 defines three layers of auto-projection:

| Layer | ADR-0116 | Concern |
| --- | --- | --- |
| Layer 1 | Auto-Generate | Fold inference from event/query struct shapes |
| Layer 2 | Explicit Folds | Consumer-written `On(...)` handlers |
| Layer 3 | Auto-Route | Cost-based engine selection per query |

**This ADR adds Layer 4:**

| Layer | This ADR | Concern |
| --- | --- | --- |
| **Layer 4** | **Physical Layout** | Embed vs. normalize within the chosen engine |

Layer 4 is orthogonal to Layers 1–3. It does not replace any layer. Fold
inference (Layer 1) generates *how events map to projection entries*. Layout
planning (Layer 4) decides *the physical storage shape of those entries* (one
embedded row vs. parent + child collection). Both run independently.

### Operator Priority System

Four priorities that weight the cost model's scoring function:

| Priority | Planner behavior |
| --- | --- |
| `WriteSpeed` | Penalize layouts that rewrite large rows on child mutation (favor normalized) |
| `ReadSpeed` | Penalize layouts requiring joins/secondary lookups (favor denormalized) |
| `StorageSpace` | Penalize data duplication (favor normalized) |
| `Balanced` | Even weighting — the default |

### Priority Hierarchy (most specific wins)

```
GLOBAL (whole deployment)  →  per-Engine  →  per-Query
```

An operator can say: *"The whole deployment optimizes for ReadSpeed, but the
Pebble engine prioritizes WriteSpeed, and this one analytics query prioritizes
StorageSpace."*

### Three Planner Modes

1. **Static (default):** Operator-configured priorities → cost model → chosen
   layouts at Plan() time. No runtime observation needed.

2. **Adaptive:** Runtime stats (the existing `Store.Replan` / `CheckRouting` /
   `StartAutoReplan` infra from the live-latency model) shift layouts based on
   observed reality.

3. **Benchmark:** The planner tries multiple plans against real or simulated
   workloads and shows the operator measured results + scaling predictions.
   Delivered as both CLI (pre-deployment exploration) and runtime API (ongoing
   monitoring).

### Runtime Backend Addition + Parallel Projections

The planner maintains **parallel projections** across engines simultaneously,
with explicit roles:

| Role | Sync strategy | Purpose |
| --- | --- | --- |
| **Active** | Fold pipeline (strong) | Serving live queries |
| **DualUse** | Fold pipeline (strong) | Two engines serving different query shapes |
| **Migration** | Async replication | New engine being populated; cutover when caught up |
| **Backup** | Async replication | Redundant copy for disaster recovery |

New backends can be added at runtime. The planner generates a plan for the new
engine, backfills from the event log, and brings it online.

> **Implemented 2026-08-15** — see
> [`docs/planning/METAENGINE-LAYOUT-ROLES.md`](../planning/METAENGINE-LAYOUT-ROLES.md):
> `AddEngine(ctx, eng, WithEngineRole(...))` assigns roles; shadows
> (Migration/Backup) mirror ALL collections via async replication and are
> never routed until `Store.PromoteEngine(ctx, name)`. Invariants I1–I4 are
> proven by test.

### Re-layout Trigger (threshold-based)

When the operator changes a priority:

1. Planner computes the new plan and identifies affected projections.
2. **Small projections** (below threshold, default: 100K events or 1GB): rebuild
   automatically from the event log.
3. **Large projections** (above threshold): present the plan diff + cost estimate
   and wait for explicit operator confirmation.

### Pathological Layouts: Obey + WARN LOUDLY

If the operator configures a priority that produces pathological layouts (e.g.,
`StorageSpace` globally → 20-way joins on Pebble), the planner **obeys but warns
loudly.** Consistent with the project north star: graceful degradation, never
failure. Benchmark mode is the tool that prevents the operator from choosing blind.

### Aggregate Boundaries

- **Default:** Local child (each `[]T` is a child of its carrying event's
  collection). Matches DDD aggregate boundaries.
- **Opt-in:** Shared by Go type (operator config merges collections by type name).

---

## Alternatives Considered

### A. Developer-Driven Layout (original M9)

Developer annotates types or relies on reflection to express storage intent.

**Rejected:** Violates "zero storage thinking." Developer's job is to declare the
domain, not the deployment. Also hardcodes normalization as always-correct,
ignoring per-backend cost differences.

### B. Full ORM Relationship Inference

Planner infers cascade rules, on-delete behavior, lazy/eager loading from type
relationships.

**Rejected:** Scope creep. The metaengine is not an ORM. Local-child default
covers the 80% case. Cascade/on-delete inference is a separate, much larger
problem with its own design surface.

### C. Planner Refuses Pathological Layouts

Planner rejects operator priorities that produce bad layouts.

**Rejected:** Violates "graceful degradation, never failure." The operator is the
decision-maker. The planner's job is to inform (benchmark mode, warnings), not to
refuse. The operator may have good reasons for a seemingly pathological layout
(testing, migration, cost optimization on a specific workload).

---

## Consequences

- **Positive:** The developer's surface area stays minimal: Commands + Events +
  Queries + relationships. Zero storage thinking. Fully aligned with the north
  star.
- **Positive:** Layout decisions are cost-based and multi-engine-aware. The same
  deployment config produces different physical layouts on Pebble vs. PostgreSQL
  vs. Dgraph — automatically.
- **Positive:** Re-layout is always possible via event-log replay. Projections
  are rebuildable caches. Migration is never lossy.
- **Positive:** Benchmark mode gives operators measured data, not guesses. The
  cost model becomes visible and tunable, not a black box.
- **Positive:** Runtime backend addition enables zero-downtime engine migration,
  dual-use deployments, and backup replicas — all from the same event stream.
- **Negative:** The priority system adds deployment configuration surface area.
  Operators must understand the priority options and their cost implications.
  Benchmark mode mitigates this but doesn't eliminate it.
- **Negative:** Parallel projections with different sync strategies add
  operational complexity. The role system (Active/Migration/Backup/DualUse)
  must be understood and monitored.
- **Negative:** Re-layout of large projections is expensive (full event-log
  replay). The threshold-based confirmation mechanism prevents accidental
  massive rebuilds but doesn't make them cheap.

---

## Addendum: Calibration Correction (2026-08-11, post-implementation)

The original cost model assumed KV/LSM engines natively favor embedding across
all priorities. 60-second on-disk calibration benchmarks
(`BenchmarkDiskLayoutCalibration_*` in `metaengine/bench` on real Pebble and
bbolt databases, plus `BenchmarkLayoutCalibration_*` on the memory engine)
corrected this to a per-priority split. The measured ratios are encoded in
`metaengine/layout_scoring.go`:

| Storage layout | Embed (read/write/storage) | Normalize (read/write/storage) | ReadSpeed winner | WriteSpeed winner | StorageSpace winner |
| --- | --- | --- | --- | --- | --- |
| **KV** (memory engine) | 0.5 / 1.0 / 1.3 | 1.8 / 0.48 / 0.63 | Embed | **Normalize** | **Normalize** |
| **LSM** (Pebble + bbolt, geomean) | 0.74 / 1.10 / 1.15 | 1.45 / 0.75 / 0.80 | Embed | **Normalize** | **Normalize** |

Conclusions:

- Embedding's single-key read advantage survived measurement (normalize reads
  cost 1.8-2.4x embed reads due to multi-key lookup + application-level merge).
- Embedding's assumed write advantage did not: a normalized child insert is a
  single O(1) write with no parent read-modify-write, measuring 0.48x (KV) and
  0.75x (LSM) of an embed write.
- Embedding duplicates the aggregate across projections; normalize stores one
  copy of each fact (0.63x / 0.80x storage).
- Row (SQLite/PG/MySQL) and Columnar (DuckDB) multipliers were analytical
  estimates until the 2026-08-15 calibration below.

The planner behavior is unchanged — it already selects by weighted score — but
the design doc's "defaults to embedding" phrasing was corrected to reflect that
the default depends on priority, not engine family.

## Addendum: Row/Columnar Calibration + Replan Convergence + DemoteEngine (2026-08-15)

**Row and Columnar multipliers are now measurement-derived**
(`BenchmarkRowLayoutCalibration_*` on file-backed SQLite, Postgres 16, and
MySQL; `BenchmarkColumnarLayoutCalibration_*` on file-backed DuckDB with a
literal 60s confirmation run within 2%). The `layout_scoring.go` cells:

| Storage layout | Embed (read/write/storage) | Normalize (read/write/storage) | ReadSpeed winner | WriteSpeed winner | StorageSpace winner |
| --- | --- | --- | --- | --- | --- |
| **Row** (SQLite/PG/MySQL geomean 1.27x/0.52x/0.35x) | 0.89 / 1.39 / 1.68 | 1.13 / 0.72 / 0.59 | **Normalize** | **Normalize** | **Normalize** |
| **Columnar** (DuckDB 2.62x/0.20x/0.59x) | 0.62 / 2.23 / 1.30 | 1.62 / 0.45 / 0.77 | **Embed** | **Normalize** | **Normalize** |

The analytical guess that a LEFT JOIN read beats a JSON-column read was
wrong (measured ≈1.0x on server engines, 1.95x on SQLite). The
float-fragility of the old exact-tie Columnar × ReadSpeed cell (2.65 vs
2.65) is resolved as a measured 0.08-margin Embed win; the 16-cell
regression matrix passes on real constants.

**`ReplanLayout` converged (§5).** The separate scoring/priority-resolution
copy in `relayout.go` is deleted. `ReplanLayout(ctx, pc)` applies `pc` as the
store priority config (non-nil is now equivalent to `SetPriority` + `Replan`,
audited `priority-change`), funnels through the single `replanWithTrigger`
path, and returns old-plan vs new-plan layout diffs computed from the two
plan snapshots (planQuery records `Layout` in both).

**`DemoteEngine` closes the role-transition square (§7).** Active → shadow
(Backup/Migration) is the inverse of `PromoteEngine`, implemented as an
atomic drain-then-unroute: role flip, replicator registration, EventLog
snapshot, and query re-assignment happen under one write-lock section inside
the re-plan (`replanWithTransition` hook, trigger `engine-demoted`).
Targeted catch-up replays the demoted engine's never-served collections onto
its mirror and the re-routed queries' history onto their new engines
(non-idempotent folds demand `WithDemoteForce`). `applyWithRecord` records,
dispatches, and replicates under one read-lock section so the EventLog
snapshot splits history at exactly the routing flip — exactly-once delivery
per engine, race-tested. `PromoteEngine` is hardened through the same atomic
path. Full design: `METAENGINE-LAYOUT-ROLES.md` §4.4.
