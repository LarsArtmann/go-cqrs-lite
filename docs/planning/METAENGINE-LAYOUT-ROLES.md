# Metaengine Layout Roles — Design

> Status: **DECIDED** (2026-08-15). Implements ADR-0124 §"Runtime Backend Addition +
> Parallel Projections" and §"Aggregate Boundaries". Unblocks TODO_LIST
> "Layout roles" (7 items).

Related: [ADR-0124](../adr/0124-operator-driven-layout-planning.md) ·
[METAENGINE-LIVE-LATENCY-MODEL.md](METAENGINE-LIVE-LATENCY-MODEL.md) ·
[METAENGINE-LAYOUT-PLANNING-MODEL.md](METAENGINE-LAYOUT-PLANNING-MODEL.md)

---

## 1. Role Model

An engine's **role** decides two independent things:

| Role | Read routing | Sync strategy | Hosts |
| --- | --- | --- | --- |
| `Active` | eligible | fold pipeline (synchronous) | collections routed to it |
| `DualUse` | eligible | fold pipeline (synchronous) | collections routed to it |
| `Migration` | **not** eligible | async replication | **all** collections (mirror) |
| `Backup` | **not** eligible | async replication | **all** collections (mirror) |

Invariants:

- **I1 — Routing closure.** `planQuery` only ever sees engines whose role is
  `Active` or `DualUse` ("routable"). Shadow engines (`Migration`/`Backup`) can
  never serve a read, so a lagging mirror is never observed by a query.
- **I2 — Mirror completeness.** Shadow engines apply the folds of **every**
  query, regardless of which engine the query is routed to. A promoted mirror
  can therefore serve any query after cutover.
- **I3 — Failure isolation.** A shadow engine's failure (slow, erroring,
  unreachable, buffer overflow) never blocks or fails the primary write path.
  The shadow is marked *stale*, replication for it halts loudly, and the
  operator resolves it (see §4.4).
- **I4 — No cross-engine atomicity.** Within one engine, one event's fold batch
  commits atomically (`RunInTx`). Across engines there is no 2PC — the design
  accepts per-engine atomicity (strong per engine, not across engines).

`Active` vs `DualUse` share identical mechanics. `DualUse` exists as operator
*intent metadata*: "a second engine is deliberately serving a different query
shape in parallel." It exists so tooling (Doctor, audit trail) can distinguish
"two engines on purpose" from "accidental dual write paths."

### 1.1 API

```go
store.AddEngine(ctx, eng, metaengine.WithEngineRole(metaengine.RoleBackup))
store.EngineRole("backup-1")          // (ProjectionRole, bool)
store.PromoteEngine(ctx, "backup-1")  // Backup|Migration → Active
store.ReplicationStatus("backup-1")   // (ReplicationStatus, bool)
```

- `AddEngine` keeps its signature (`opts ...AddEngineOption` is additive);
  default role remains `Active`.
- Engines present at `Plan()` time default to `Active`.

## 2. Fold-Pipeline Sync (Active + DualUse)

Already the shipped behavior, made explicit and role-gated:

1. Event arrives (`Apply`/`ApplyRecord`) → recorded to the attached
   `EventLog` (if any).
2. `dispatchFolds` collects the folds matching the event type across all
   queries, groups them **by engine**, and applies each engine's group inside a
   single `RunInTx` when the engine is `Transactional` (batch atomicity,
   ADR-0124; covered by `batch_atomicity*_test.go`).
3. Shadow engines: the event is enqueued to each shadow's replicator (§3).

Strong consistency claim: *per engine*. All Active/DualUse projections on one
engine move atomically with the event. Projections on a second engine may
briefly lag the first (no 2PC). This is the documented contract.

## 3. Async Replication (Migration + Backup) — v1

### 3.1 Scope

V1 is **in-process**: an in-memory bounded queue per shadow engine plus a
dedicated applier goroutine. Durable, cross-process, resumable replication
(position-tracked, WAL-backed) is future work (§3.5). V1 semantics are chosen
so the future subsystem can reuse the same contract (`ReplicationStatus`,
stale/halt semantics, promote-drain).

### 3.2 Write path

- After the primary dispatch succeeds, `applyWithRecord` enqueues
  `{eventType, record, payload}` to every shadow engine's replicator
  (non-blocking `select`).
- The replicator goroutine pops a job, snapshots the matching fold tasks under
  the store read lock, and applies them to the shadow engine with an
  **engine-override shim** (`shadowQuery` wraps `queryMeta`, overriding
  `QueryEngine()`), inside `RunInTx` when transactional.
- Watcher notifications are suppressed on the replication path (primaries
  already notified them; re-notifying would double-append replay sequences).

### 3.3 Failure semantics

- **Transient error** (fold/backend error): retried up to 3 times with short
  backoff. Transactional engines roll back per attempt, so retry is safe.
- **Permanent failure** (retries exhausted, panic recovered, non-transactional
  partial write): the shadow is marked **stale**, its replicator **halts**
  (keeps no backlog), and the error is recorded in `ReplicationStatus`.
  Halting — not skipping — is deliberate: a skipped event silently corrupts the
  mirror; a halted engine is loudly behind.
- **Buffer overflow** (queue full, default capacity 1024): the shadow goes
  stale and halts. The primary write path is never blocked.
- Unsupported backend capability on the shadow (e.g. a vector query on a KV
  engine) fails the same way: stale + halt. This mirrors the north star —
  degrade loudly, never crash.

### 3.4 Recovery

A stale shadow is repaired by: `RemoveEngine(name)` → fix/replace the engine →
`AddEngine(..., WithRole(...))` → `Backfill(ctx, WithBackfillForce())` (the
mirror is known-stale; force is safe *for the mirror* because
`RemoveEngine`+re-add implies an empty replacement engine).

`Backfill` replays into shadow engines synchronously through the same
engine-override path, subject to the same non-idempotent-fold guard
(`WithBackfillForce` overrides).

### 3.5 Future (explicitly out of scope for v1)

- Durable queue (WAL segment) + position-based resumption.
- Cross-process shipping (the shadow engine living in another node).
- Lag SLOs with auto-promote gating; resumable `JournalReadAll`-based catch-up
  once engine seqs are exposed (see TODO "Seq-carrying journal reads").

## 4. Role Transitions

### 4.1 `PromoteEngine(ctx, name)`

1. Role must be `Migration` or `Backup` (else error).
2. Stale/halted engines refuse promotion (they are behind; recover first).
3. Drain: wait until the queue is empty and no job is in flight (bounded by
   `ctx`).
4. Role becomes `Active`; replan with trigger `engine-promoted` (audited).
   The promoted engine is now routable; the planner may move queries to it.

### 4.2 Cutover runbook (Migration → Active)

```text
AddEngine(ctx, newEng, WithRole(RoleMigration))
Backfill(ctx, WithBackfillForce())     // populate the mirror
… run traffic; ReplicationStatus(lag) → 0 …
PromoteEngine(ctx, newEng.Profile().Name)
// optionally retire the old engine:
RemoveEngine(ctx, oldEng.Profile().Name)
```

### 4.3 Backup promote (disaster recovery)

Same as cutover minus the backfill step if the backup was kept current:
`PromoteEngine(ctx, backupName)` once `ReplicationStatus` shows it caught up.

### 4.4 No demote in v1

Active → Backup is not exposed; an Active engine may have queries routed to
it, and silently freezing it would strand reads. Demote requires
drain-then-unroute sequencing (future API: `DemoteEngine`).

## 5. Workload Trace Format

JSON Lines (one JSON object per `\n`-terminated line), versioned by `"v"`:

```json
{"v":1,"ts":"2026-08-15T12:00:00.123456789Z","op":"apply","name":"TaskCreated","dur_ms":0.42,"err":""}
{"v":1,"ts":"2026-08-15T12:00:00.2Z","op":"query","name":"tasks_by_id","dur_ms":0.18,"err":""}
```

- `op` ∈ `apply` | `query`.
- `name` — event type (apply) or query name (query).
- `dur_ms` — wall duration in milliseconds.
- `err` — empty on success, else the error string.
- Unknown `op` values are skipped by the player (forward compatibility).
- Payloads are **not** serialized (arbitrary Go values are not JSON-round-trip
  safe). Replay synthesizes payloads via a caller-supplied factory; the trace
  carries the *shape and mix* of the workload, which is what calibration needs.

Components (metaengine package):

- `TraceRecorder` — attaches to a Store (chaining any existing hooks), writes
  JSONL to an `io.Writer`. New `Hooks.OnApply` gives one tap per applied event.
- `ReadTrace(io.Reader) ([]TraceOp, error)` — parse.
- `TraceStats(ops)` — op counts by name (calibration input).
- `ReplayTrace(ctx, ops, sink)` — sequential replay.
- `StoreTraceSink(store, payloadFor)` — sink that applies events (`payloadFor`
  synthesizes a payload per event type and occurrence index).

## 6. Aggregate Boundaries (shared collections)

Default (ADR-0124): **local child** — each `[]T` field belongs to its carrying
query's collection (DDD aggregate boundary). Opt-in:

```go
metaengine.Plan(engines, q1, q2, metaengine.WithSharedCollection("Attachment"))
```

`WithSharedCollection(typeNames ...string)` declares child Go types whose data
is shared across aggregates. Planning effects (v1, scoring-level — physical
child-collection materialization does not exist yet):

1. Any query whose **result type** contains a field of a shared type (directly,
   as `*T`, or as `[]T`) is forced to `LayoutNormalize` — embedding duplicates
   a shared child into every embedding collection.
2. Diagnostics: `INFO` when a shared type affects one query; `WARN` listing the
   affected collections when it spans ≥2 (the duplication the operator opted
   out of would have been real).

## 7. Fold Locking

The global `foldMu` (added to fix a `SetCurrentRecord` + invoke data race)
serialized **all** fold execution store-wide. Replaced by **per-query** locks:

- A fold instance is owned by exactly one query (`QueryDecl`), and the shared
  mutable state (`RecordAwareFold.SetCurrentRecord`) is per fold instance.
  Locking per query name therefore serializes exactly the folds that share
  state, and lets folds of different queries apply in parallel.
- Concurrent fold application to one engine is safe: Memory engine is
  internally locked; sqlite engine pins `MaxOpenConns(1)` (WAL +
  busy_timeout); other engines serialize writes in their backends.
- The replication applier shares fold instances with the primary path, so
  per-query locks are also what make replication race-free.
- Lock ordering: query locks are always acquired *inside* the store read lock
  (dispatch) or outside it (replicator snapshot-then-apply); the store write
  lock never wraps a query lock. No deadlock cycle exists.

Soak/race coverage: dedicated concurrent apply test (mixed queries × goroutines
× events) run under `-race -count=3`, plus the existing batch-atomicity and
concurrency suites.

## 8. Multi-Collection Batch Atomicity — status

**Shipped** (previously delivered): `dispatchFolds` groups all folds an event
triggers — across collections — per engine into one `RunInTx`. Covered by
`batch_atomicity_test.go` (multi-query, single event) and
`batch_atomicity_rollback_test.go` (rollback on mid-batch failure, commit on
success, documented non-atomic fallback for non-transactional engines). The
"replaces `RelationalProjection`'s per-event tx" clause is consumer migration
guidance (v5 Phase 8, tracked separately) — nothing further to build here.

## 9. Delivery order

1. Per-query fold locking (prerequisite: replication shares fold instances).
2. Role-aware engine management (roles, routable filter, promote).
3. Replicator v1.
4. Workload trace.
5. `WithSharedCollection` rule.
