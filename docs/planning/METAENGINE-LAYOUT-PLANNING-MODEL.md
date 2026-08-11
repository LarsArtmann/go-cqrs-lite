# Metaengine Layout Planning Model

**Date:** 2026-08-11
**Status:** Design discussion (pre-ADR)
**Related:** ADR-0116 (layered auto-projection), ADR-0111 (Record type), `METAENGINE-LIVE-LATENCY-MODEL.md`, TODO M9 (struct-composition-driven multi-collection)

> This document captures a design session that reframed how the metaengine should
> decide projection layout (embed vs. normalize vs. hybrid). It replaces the
> original M9 framing ("auto-generate a second collection from `[]Attachment`")
> with an operator-driven, cost-aware model.

---

## 1. The Problem with M9

The original M9 task (TODO_LIST.md Phase 6):

> *"When an event has a `[]Attachment` field and a query requests `MessageView`
> (which has `Attachments`), auto-generate a second collection for attachments.
> Planner sees the relationship and generates a join-aware read path."*

**This assumes normalization is always correct.** It isn't. The right layout
depends on the backend AND the workload. Hardcoding "always split into a child
collection" throws away the metaengine's core value: being cost-based and
multi-engine.

---

## 2. Normalization vs. Denormalization: The Per-Backend Truth

| Backend | Denormalize (embed slice) | Normalize (child collection + join) | Winner |
| --- | --- | --- | --- |
| **Pebble / bbolt / memory** (KV/log) | **Native.** One key lookup → whole aggregate. O(1). | Two lookups + in-memory merge. Pessimistic. | **Denormalize** |
| **SQLite / PG / MySQL / Turso** (SQL) | JSON/TEXT column. Loses queryability into children. | **Native.** Child table + FK + index-backed JOIN. | **Normalize** |
| **Dgraph** (graph) | Predicate with list value. OK. | Edges to attachment nodes. **Native.** | **Normalize** (graph-shaped) |
| **DuckDB** (columnar) | Nested/repeated column. **Fast** for analytics. | Long/narrow child table. Also fine. | **Either** (workload-dependent) |

The choice isn't universal. It's per-engine, per-query.

---

## 3. The Event-Sourcing Wrinkle (Write Side)

Event sourcing changes the calculus. You don't "update attachment Y" — you emit
`AttachmentAdded{messageID, attachment}`. The fold that applies it:

- **Denormalized:** load parent, append to slice, write back entire row →
  **write amplification** on large aggregates.
- **Normalized:** insert one row into child collection → **O(1) write**.

For append-heavy child patterns (the ES sweet spot), normalization has a real
write win. But for "read whole aggregate" (the dominant read pattern in most
UIs), denormalization wins. Genuinely per-workload.

---

## 4. The `[]Attachment` vs `[]AttachmentID` Insight

Initial thinking: the type distinction (`[]Attachment` vs `[]AttachmentID`)
could express developer storage intent (embed vs. normalize).

**Corrected:** `[]AttachmentID` is just a serializable pointer to
`[]Attachment`. The type distinction tells the planner **nothing about desired
layout** — it tells the planner **what data is literally in the event payload.**

| Event field type | What's in the payload | Embed possible? | Split possible? | Shared collection possible? |
| --- | --- | --- | --- | --- |
| `[]Attachment` | Full struct values | **Yes** (has bytes) | **Yes** (can decompose) | Yes — if same type appears across events |
| `[]*Attachment` | Full struct values (ptr encoding) | **Yes** | **Yes** | Yes |
| `[]AttachmentID` | Only IDs (no payload) | **No** — nothing to embed | **Yes** (store IDs) | **Yes** — but join requires the Attachment collection to exist elsewhere |
| `[]*AttachmentID` | Only IDs | **No** | **Yes** | **Yes** — same as above |

The planner can't embed bytes it wasn't given. `[]AttachmentID` *forces*
normalization because the event literally doesn't contain the attachment data.
That's not a storage decision — it's a consequence of what the developer chose
to put in the event. **The planner reads reality, not intent.**

**Conclusion: The developer expresses zero storage intent. Ever.** Layout is
100% the operator's call via priorities + the planner's cost model.

---

## 5. The Revised Vision: Developer Is Silent, Operator Controls

Two cleanly separated concerns:

### Developer expresses: *what the data IS*

- Event types, field types, query shapes.
- This defines the **constraint set** — what layouts are even valid (you can't
  embed data that isn't in the payload; you can always normalize data that is).
- The developer NEVER makes a storage decision.

### Operator expresses: *what to optimize FOR*

A priority objective that weights the cost model:

| Priority | Planner behavior |
| --- | --- |
| `WriteSpeed` | Penalize layouts that rewrite large rows on child mutation (favor normalized) |
| `ReadSpeed` | Penalize layouts requiring joins/secondary lookups (favor denormalized) |
| `StorageSpace` | Penalize data duplication (favor normalized) |
| `Balanced` | Even weighting — the sane default |

### Priority Hierarchy (most specific wins)

```
GLOBAL (whole deployment)  →  per-Engine  →  per-Query
```

An operator can say: *"The whole deployment optimizes for ReadSpeed, but the
Pebble engine specifically prioritizes WriteSpeed (KV rewrites are cheap
there), and this one analytics query optimizes for StorageSpace."*

---

## 6. Three Planner Modes

The planner operates in three modes:

### 6.1 Static Plan (default)

Operator-configured priorities → cost model → chosen layouts upfront at Plan()
time. No runtime observation needed. This is the baseline.

### 6.2 Adaptive Override

Runtime stats (the existing `Store.Replan` / `Store.CheckRouting` /
`StartAutoReplan` infra from the live-latency model) shift layouts based on
observed reality. The operator configures the thresholds; the planner adapts.

### 6.3 Benchmark Mode (new — critical)

The planner tries multiple plans against real or simulated workloads and shows
the operator **measured results + scaling predictions** so they choose
intelligently, not blindly.

This is the answer to "how does the operator know what priority to set?" — they
don't guess, they **measure**.

Capabilities:
- Try all (or a Pareto-frontier subset of) valid plans for a workload.
- Show real latency, throughput, storage size per plan.
- Predict scaling (what happens at 10x, 100x data volume).
- Accept a **real workload trace** from the operator, or **synthesize** one
  from declared queries.

**Delivery: both** — CLI for pre-deployment "what if" exploration, runtime API
for ongoing monitoring + adaptive re-tuning.

**Workload source: both** — synthesize from declared queries by default (zero
operator effort, covers the 80% case), accept real operator-provided traces for
calibration when precision matters.

---

## 7. Runtime Backend Addition + Dual-Use / Migration / Backup

The planner isn't picking one layout per query — it can maintain **parallel
projections** across engines simultaneously, with explicit roles:

| Role | Purpose |
| --- | --- |
| **Active read** | Serving live queries (the primary projection) |
| **Migration target** | New engine being populated; switch over when caught up |
| **Backup replica** | Redundant copy for disaster recovery |
| **Dual-use** | Two engines serving different query shapes simultaneously |

**New backends can be added at runtime.** The planner generates a plan for the
new engine, backfills from the event log, and brings it online.

### Retroactive Re-layout

Changing a priority triggers a **rebuild from the event log**. The ES
foundation pays off here: the event stream is the source of truth, projections
are rebuildable caches, so re-layout is always possible. Migration is never
lossy.

**Sync strategy: role-based** — active read + dual-use roles sync via the fold
pipeline (event → all projections in one transaction, strong consistency).
Backup + migration roles sync via async replication (eventual consistency,
failure-isolated). This matches the operational reality: roles that serve live
traffic need atomicity; roles that are safety nets don't.

---

## 8. Aggregate Boundaries

If `Message` has `[]Attachment` and `Order` has `[]Attachment`, are those the
same Attachment collection or two different ones? Without the developer telling
the planner, how are boundaries discovered?

| Approach | How it works | Trade-off |
| --- | --- | --- |
| **Always local child** (recommended default) | Each `[]T` is a child of whatever event carries it. `Message.attachments` and `Order.attachments` are separate. | Simple. No sharing. Matches DDD aggregate boundaries. |
| **Shared by Go type** (operator opt-in) | Same `Attachment` type anywhere = one global collection, referenced by FK from multiple parents. | True normalization. Assumes same type = same lifecycle (not always true: `Address` on `User` and `Order`). |
| **Operator decides** | Planner proposes, operator confirms boundaries in deployment config. | Maximum flexibility. More operator burden. |

**Recommendation:** Local child by default, shared-by-type when operator opts
in. Matches the "developer says nothing, operator tunes" model.

---

## 9. Normalize Anything (Not Just Slices)

Normalization is not limited to repeated fields (`[]T`). Even a single nested
struct (`Address{City, Zip}`) can be normalized to its own collection if the
operator priority justifies it (e.g., `StorageSpace` on an engine where the
address repeats across many aggregates).

The operator's control has no structural floor. The cost model decides.

---

## 10. Pathological Layouts: Obey + WARN LOUDLY

If the operator configures a priority that produces pathological layouts (e.g.,
`StorageSpace` globally → 20-way joins on Pebble), the planner **obeys but
warns loudly.** This is consistent with the project's north star:

> *"Graceful degradation, never failure. Unsupported/unsuited query shapes emit
> advisory diagnostics (WARN: slow), not errors."*

The operator is the decision-maker. The planner's job is to inform, not to
refuse. Benchmark mode (§6.3) is the tool that prevents the operator from
choosing blind.

---

## 11. Resolved Decisions

All four open questions resolved 2026-08-11:

| Decision | Choice | Rationale |
| --- | --- | --- |
| **Benchmark delivery** | Both (CLI + runtime) | CLI for pre-deployment "what if" exploration. Runtime API for ongoing monitoring + adaptive re-tuning. |
| **Benchmark workload** | Both (synthesize + real trace) | Synthesize from declared queries by default (zero operator effort). Accept real traces for calibration when precision matters. |
| **Dual-use sync** | Role-based | Fold pipeline (strong) for active read + dual-use. Async replication (eventual) for backup + migration. Matches operational reality. |
| **Re-layout trigger** | Threshold-based | Small projections (<N events) rebuild automatically. Large ones require explicit operator confirmation. Balances safety and automation. |

### Re-layout Trigger (detail)

When the operator changes a priority, the planner:

1. Computes the new plan and identifies which projections must change.
2. For each affected projection, estimates rebuild cost (event count, data size).
3. **Small projections** (below threshold): rebuild automatically from the event
   log. No operator intervention.
4. **Large projections** (above threshold): present the plan diff + cost estimate
   and wait for explicit operator confirmation before rebuilding.

The threshold is operator-configurable (default: e.g. 100K events or 1GB
projected data). This prevents a global priority change from silently launching
massive parallel rebuilds that could overwhelm the system.

---

## 12. Summary: What Replaces M9

The original M9 ("auto-generate child collection from `[]Attachment` via
reflection") is **wrong** because it:

- Hardcodes normalization as always-correct (it isn't — backend-dependent)
- Puts storage intent on the developer (violates "zero storage thinking")
- Ignores the cost model (the metaengine's core value)

**What replaces it:**

| Dimension | M9 (original) | This model |
| --- | --- | --- |
| Who decides layout | Developer (via types) + reflection | **Operator** (via priorities) |
| When decided | Planner-time (reflection) | Static + adaptive + benchmark |
| Normalization trigger | Slice-of-struct field detected | Cost model + operator priority |
| Backend awareness | None (always splits) | **Per-engine cost** (the whole point) |
| Developer burden | Must model types carefully | **Zero** — developer is silent |
| Layout migration | Not considered | Rebuild from event log (retroactive) |

**One sentence:** The planner defaults to embedding, adapts based on operator
priorities and measured costs, and lets the operator benchmark real plans
before committing — because the developer's job is to declare the domain, and
the operator's job is to tune the deployment.
