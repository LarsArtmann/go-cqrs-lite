# Durable Work Queue Module — proposal (2026-09-13)

**Status:** PROPOSED (owner-triggered: "go-cqrs-lite should have a solution
for this" — go-taskqueue storage-adoption challenge, 2026-09-13 session)
**Trigger consumer:** go-taskqueue (`internal/queue` hand-rolled SQLite +
Postgres stores); second consumer: PapDashboard ("worker pools over durable
queues" in production today).

## Problem

The ecosystem has no durable WORK QUEUE primitive. Every consumer
hand-rolls the same store: lease-based claims with expiry reclaim, task
lifecycle, retries with backoff, dead-lettering, priorities, and a journal
that explains what happened. go-taskqueue ships 1.8k + 1.5k LOC of two
mirrored backends + a conformance suite to pin them. That is the exact
library-shaped problem this repo exists to solve — and the pieces are
already here, unassembled.

## What already exists here (verified first-hand 2026-09-13)

| Capability | Where | Notes |
| --- | --- | --- |
| Atomic claim w/ lease + expiry reclaim | `scheduling/sqlstore/claiming.go` (`ClaimingTimerStore`) | PG `FOR UPDATE SKIP LOCKED` CTE→UPDATE→RETURNING; SQLite single-writer UPDATE..RETURNING; MySQL/MariaDB 10.6+ SKIP LOCKED; `RenewLease`; `ClaimMetrics`; idempotent lease-column migration. THE hard part, already 3-dialect |
| Per-key atomic RMW | `metaengine` `MapUpdater` (pgengine: `SELECT … FOR UPDATE` in-tx) | claim-adjacent; no multi-key conditional claim |
| Filtered/sorted/keyset listing | `metaengine` planned tables (`FilterSpec`/`SortSpec`/`PushdownMapScan`, json_extract pushdown) | read side for dashboards; single collection, no cross-collection anti-joins |
| Journal + cursors | `event.Store`/`SeekableJournal`, `storage`, `watermill.CatchUpSubscriber`, `CheckpointStore` | ≈ go-taskqueue's facts + watermarks |
| Deadline scheduling | `scheduling` (`Timer[P]`, fire-once) | claims exist HERE but rows are DELETED on fire — timer semantics, not task semantics |

## The gap (what nobody assembles today)

Task lifecycle (pending→running→completed/dead), attempts + backoff +
DLQ **at the store level** (scheduling retries at dispatch level only),
priorities (+aging) in the claim order, DAG dependency gating,
owner-bearing claims (heartbeat, cooperative cancel), dedup'd enqueue,
and journal/facts appended **in the same transaction** as the state
change (go-taskqueue ADR-0001's load-bearing invariant).

## Design sketch — new sibling module `queue/`

- **`ClaimableTaskStore[T]`** — the contract: `Enqueue` (dedup key),
  `Claim(owner, lease)` (due + priority-ordered + dep-gated),
  `Heartbeat`, `Complete`, `Fail(backoff)` (→DLQ after budget),
  `Requeue`, `Cancel`. Payload-generic like `Timer[P]`.
- **Spec source of truth: go-taskqueue's `internal/queue.Store`** —
  five weeks of production dogfood (agent-pool on live repos), mirrored
  conformance suites, both backends. Upstream the PROVEN semantics onto
  this repo's claim core; do not invent new ones.
- **Engines:** generalize `scheduling/sqlstore`'s 3-dialect claim core
  (extract to a shared internal; timers keep delegating). Optional
  metaengine adapter later for dashboard-grade listing.
- **Journal option:** facts/events appended in the same tx as the claim
  transition (capability flag); consumers wanting pure CQRS wire
  `watermill` on top.
- **Conformance:** one suite, every dialect — the go-taskqueue ADR-0007
  mirrored-suite pattern, upstreamed.
- **Non-goals:** executors/worker pools stay consumer-side (go-taskqueue
  owns that UX); no scheduler process — store only.

## Phasing

1. **P0 (S):** extract the claim core from `scheduling/sqlstore` into a
   shared internal; timers + queue both consume.
2. **P1 (M):** task lifecycle + attempts/backoff/DLQ + dedup'd enqueue;
   conformance suite; SQLite + PG.
3. **P2 (S):** priorities + aging in claim order; MySQL dialect.
4. **P3 (S):** deps table + NOT EXISTS gating (composite-key shape).
5. **P4 (S):** same-tx journal option + watermark/cursor API.
6. **P5 (owner):** go-taskqueue adoption decision — its own ADR there
   (re-point facades or keep internal stores until parity is proven);
   never forced.

## Consumers

- **go-taskqueue** — first consumer candidate (reference semantics
  donor; its AGENTS.md verdict note gates re-adoption on exactly this
  module existing).
- **PapDashboard** — worker pools over durable queues, production today.
- **`example/taskmanager`** — upgrades from demo to real consumer.
