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

**On-disk calibration addendum (2026-08-11):** 60s benchmarks on real Pebble
and bbolt databases plus the memory engine (see `metaengine/layout_scoring.go`)
show the per-priority split on KV/LSM is the OPPOSITE of the naive "KV always
embeds" reading of the table above:

| Priority | Winner on KV/LSM | Measured ratios (normalize ÷ embed) |
| --- | --- | --- |
| ReadSpeed | **Embed** | KV read 1.8 vs 0.5; LSM read 1.45 vs 0.74 |
| WriteSpeed | **Normalize** | KV write 0.48 vs 1.0; LSM write 0.75 vs 1.10 |
| StorageSpace | **Normalize** | KV storage 0.63 vs 1.3; LSM storage 0.80 vs 1.15 |

Embedding's single-key read advantage survives measurement; its assumed write
and storage advantage does not. Normalized child inserts are O(1) with no
read-modify-write, and embedding duplicates the aggregate across projections.

**Row/Columnar calibration addendum (2026-08-15):** benchmarks on file-backed
SQLite, Postgres 16, MySQL (QEMU), and DuckDB (60s confirmation run) measured
the normalize÷embed ratios below (see `metaengine/layout_scoring.go`):

| Engine | read | write | storage | Per-priority winner |
| --- | --- | --- | --- | --- |
| SQLite (Row) | 1.95x | 0.66x | 0.33 | Normalize in all Row priority cells (write+storage dominate) |
| Postgres 16 (Row) | 1.00x | 0.38x | 0.33 | — |
| MySQL (Row) | 1.06x | 0.56x | 0.41 | — |
| DuckDB (Columnar) | 2.62x | 0.20x | 0.59 | Columnar: Embed for ReadSpeed, Normalize otherwise |

Notable correction vs the analytical estimate: a LEFT JOIN read is NOT cheaper
than a JSON-column read on server engines (≈1.0x) and is 2x+ worse on SQLite —
the old 0.8 guess was wrong. The old exact-tie Columnar × ReadSpeed cell is
now a measured 0.08-margin Embed win. All calibrated pairs are
geomean-centered (embed × normalize = 1.0 per dimension).

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

### Constraint vs Intent — Two Different Things

Initial thinking: the type distinction (`[]Attachment` vs `[]AttachmentID`)
could express developer storage intent (embed vs. normalize).

**Corrected.** There are two orthogonal axes that must not be conflated:

| Axis | Who controls it | What it means |
| --- | --- | --- |
| **Domain shape (constraint)** | Developer (involuntarily) | What layouts are *physically possible* given the payload bytes |
| **Storage intent (objective)** | Operator (voluntarily) | What layout to *prefer* given cost trade-offs |

`[]AttachmentID` is not storage intent — it is **payload reality.** The developer
chose to put IDs in the event (a domain modeling decision), and that choice
*constrains* what the planner can do. It does not express a preference for any
particular layout.

| Event field type | What's in the payload | Embed possible? | Split possible? | Shared collection possible? |
| --- | --- | --- | --- | --- |
| `[]Attachment` | Full struct values | **Yes** (has bytes) | **Yes** (can decompose) | Yes — if same type appears across events |
| `[]*Attachment` | Full struct values (ptr encoding) | **Yes** | **Yes** | Yes |
| `[]AttachmentID` | Only IDs (no payload) | **No** — nothing to embed | **Yes** (store IDs) | **Yes** — but join requires the Attachment collection to exist elsewhere |
| `[]*AttachmentID` | Only IDs | **No** | **Yes** | **Yes** — same as above |

The planner can't embed bytes it wasn't given. When the event carries
`[]AttachmentID`, normalization is the *only valid layout* — not because the
developer "wanted" normalization, but because embedding is physically
impossible. Conversely, when the event carries `[]Attachment` full struct
values, the planner CAN embed OR normalize — and the operator's priority
decides which.

**This is not a contradiction.** The developer's type choice constrains the
planner's option set (domain shape). The operator's priority selects within
that option set (storage intent). A constraint that eliminates an option is not
the same as an intent that expresses a preference.

> **Key distinction:** `[]AttachmentID` *constrains* the planner (can't embed
> absent data) but does NOT express storage *intent*. The developer expresses
> zero storage intent — ever. Layout selection within the constraint set is
> 100% the operator's call via priorities + the cost model.

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

**One sentence:** The planner selects the layout from the operator priority
plus the calibrated cost model — on KV/LSM, ReadSpeed selects Embed while
WriteSpeed and StorageSpace select Normalize — and lets the operator benchmark
real plans before committing, because the developer's job is to declare the
domain, and the operator's job is to tune the deployment.

---

## 13. Current Infer() Behavior with Slice Fields (Code Audit)

**Audited:** `metaengine/fold_inference.go`, `metaengine/auto_fold.go`

### What works today

1. **Scalar field matching** — `matchFields()` maps event fields to result
   fields by name + type assignability. `MessageCreated.Title` →
   `MessageView.Title` ✓

2. **Nested struct flattening** — If an event has `Address{City, Zip}` (a
   nested struct), `matchFields()` flattens it via `matchNestedFields()` and
   maps `City` and `Zip` individually to result fields. ✓

3. **Collection result types** — `collectionElementType()` handles `Items []T`
   in *result* types: extracts element type T and uses it for field matching.
   A query returning `Result{Items []UserView}` matches against `UserView`
   fields, not the wrapper. ✓

4. **Embedded slice mapping** — If event has `Attachments []Attachment` and
   result has `Attachments []Attachment` (same type), `matchFields()` maps the
   whole slice as a single field. This is embedding behavior. ✓

### The gap (what does NOT work)

**No slice decomposition in fold inference.** When an event has
`Attachments []Attachment`, Infer() cannot generate a fold that iterates the
slice and inserts each attachment as a separate projection entry. There is no
mechanism to:

- Generate a fold for a child collection (e.g., `attachments` keyed by
  attachment ID within message ID)
- Map individual slice elements' fields to result fields
- Generate a "fan-out" fold (one event → N projection entries)

### Why this is correct (not a bug)

Slice decomposition is a **layout planning** concern (Layer 4, ADR-0124), not
a fold inference concern (Layer 1, ADR-0116). Fold inference generates *how
events map to projection entries*. Layout planning decides *the physical shape
of those entries* (one embedded row vs. parent + child collection).

The pre-Phase-6b behavior — embed the whole slice as a single field — was a
reasonable starting point, but on-disk calibration corrected the assumed
default: embedding wins only under ReadSpeed priority. Under WriteSpeed or
StorageSpace priority the cost model selects normalization even on KV/LSM
engines (see the §2 calibration addendum). Normalization (decomposing into a
child collection) happens when the operator's priority + the calibrated cost
model justify it, not when the type shape triggers it.

### What needs to happen for normalization support

When layout planning decides to normalize a `[]T` field:

1. The fold must change from "store whole slice" to "iterate and insert each
   element into child collection"
2. This is a **fold transformation** applied at plan time, not a fold inference
   change
3. The existing `OnRecord` fold API already supports this — consumers can write
   explicit folds that iterate slices. The gap is only in *auto-inferred* folds.

**Action item:** When implementing layout planning (Phase 6b), add a fold
transformer that converts an embedded-slice fold into a normalized multi-
collection fold when the cost model selects normalization.

---

## 14. Worked Example: Message + Attachments

### Domain types (developer declares)

```go
type Attachment struct {
    ID       AttachmentID
    Filename string
    Size     int64
}

type MessageCreated struct {
    ID          MessageID
    Title       string
    Body        string
    Attachments []Attachment  // full struct values — embed is possible
}

type AttachmentAdded struct {
    MessageID  MessageID
    Attachment Attachment  // single attachment appended
}

type MessageDeleted struct {
    ID MessageID
}

// Query results
type MessageView struct {
    ID    MessageID
    Title string
}

type MessageDetail struct {
    ID          MessageID
    Title       string
    Body        string
    Attachments []Attachment  // full aggregate read
}
```

### Scenario 1: Static plan, ReadSpeed priority, Pebble engine

Operator config:
```yaml
priority:
  global: ReadSpeed
```

Planner reasoning:
- Pebble is a KV engine → embedding is native (O(1) lookup)
- ReadSpeed penalizes joins → embedding wins (no join needed)
- **Selected layout:** `MessageDetail` stores `Attachments` as a single CBOR-
  encoded value in the row. `AttachmentAdded` triggers a read-modify-write
  (load parent, append, write back).

Cost: write amplification on `AttachmentAdded`, but O(1) reads.

### Scenario 2: Operator switches to StorageSpace priority

Operator config change:
```yaml
priority:
  global: StorageSpace
```

Planner re-plans:
- StorageSpace penalizes data duplication
- Attachments are duplicated across `MessageDetail` and the event log
- Normalization eliminates duplication: `attachments` becomes a child collection
  keyed by `(MessageID, AttachmentID)`
- **Selected layout:** `MessageDetail` stores `AttachmentID` list. Separate
  `attachments` collection holds `{AttachmentID, Filename, Size}`.
- `AttachmentAdded` → insert into child collection (O(1) write, no read-modify-
  write). `MessageDetail` read requires a join (two point lookups on Pebble).

### Scenario 3: Re-layout trigger

Changing from Scenario 1 to Scenario 2 requires rebuilding `MessageDetail`
projections:

1. Planner computes new plan: `MessageDetail.Attachments` goes from embedded to
   normalized.
2. Estimates rebuild cost: 50K events, ~200MB projected data.
3. Threshold check: 50K < 100K threshold → **auto-rebuild**.
4. Rebuild: replay event log from position 0, apply new folds.
5. `attachments` child collection created from scratch.
6. `MessageDetail` re-materialized without embedded attachments.

### Scenario 4: Pathological layout — obey + warn

Operator config:
```yaml
priority:
  global: StorageSpace
  engines:
    pebble: StorageSpace  # KV engine, normalization is expensive
```

Planner:
- Normalization on Pebble requires in-memory joins (no native JOIN)
- WARN LOUDLY: "StorageSpace on Pebble: 12 queries require multi-lookup joins
  (estimated 3x read latency increase). Consider ReadSpeed for this engine."
- **Obeys the configuration.** The operator may have good reasons (e.g., Pebble
  is a backup engine, reads are rare here).
- Benchmark mode available: `cqrs-bench layout --engine pebble --priority
  storage-space` shows measured latency impact before the operator commits.

---

## 15. WARN LOUDLY Specification

### Where warnings appear

| Surface | What it shows |
| --- | --- |
| `Doctor()` output | `--- Layout Warnings ---` section: priority conflicts, pathological layouts, rebuild backlog |
| `EXPLAIN <query>` | Layout annotation: "embedded (ReadSpeed priority)", "normalized via child collection (StorageSpace priority)", "WARNING: 3-way join on KV engine" |
| Structured logs | `slog.Warn` with fields: `layout.warn`, `priority.conflict`, `engine.name`, `query.name`, `cost.estimate` |
| `GetEngineStats()` | `LayoutWarnings []LayoutWarning` field: machine-readable warning list |

### Warning types

| Warning | Trigger | Severity |
| --- | --- | --- |
| `PRIORITY_MISMATCH` | Engine can't efficiently serve the selected layout (e.g., normalized on KV) | WARN |
| `REBUILD_BACKLOG` | Large projections pending rebuild after priority change | WARN |
| `JOIN_AMPLIFICATION` | Query requires N-way join where N > 2 on a non-SQL engine | WARN |
| `WRITE_AMPLIFICATION` | Embedded layout with high child-mutation rate (write amplification) | INFO |
| `COST_MODEL_STALE` | Cost estimates based on compile-time priors, no live calibration | INFO |

### Priority conflict resolution

When priorities conflict (GLOBAL says one thing, Engine says another):

1. **Most specific wins** (per-Query > per-Engine > GLOBAL). No ambiguity.
2. If the resolved priority produces a suboptimal layout for the engine, emit
   `PRIORITY_MISMATCH` warning with the reasoning.
3. The operator sees both the resolved priority and the warning in `Doctor()`.
4. **The planner never refuses.** Obey + warn. The operator is the decision-maker.
