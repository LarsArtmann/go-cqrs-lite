# PapDashboard → go-cqrs-lite/queue adoption evaluation (T20 spike)

**Date:** 2026-09-20
**Scope:** TODO_LIST "Durable Work Queue module" T20 — evaluate migrating
PapDashboard's worker pools onto `queue/{sqlite,postgres}`; record the
verdict (adopt / blockers).
**Method:** first-hand source read of `/home/lars/projects/PapDashboard`
(every load-bearing claim below verified against the cited file:line on
2026-09-20), mapped onto `queue/v4` contract + `queue/sqlite` engine
(conformance-green incl. `-race -count=2` 2026-09-19/20).

## T20.1 — What PapDashboard actually runs today

The TODO's premise ("worker pools over durable queues in production")
overstates the code reality. PapDashboard is a **single-node SQLite app**
(`mattn/go-sqlite3` v1.14.52, `SetMaxOpenConns(1)`,
`internal/di/container.go:236-244`) with one Docker Compose service
(`docker-compose.yml`, `PAP_DB_PATH` volume) and **no durable worker
queue**. What exists instead:

| Component                                            | Where                                                                    | Mechanism                                                                                                                                                               |
| ---------------------------------------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Notification delivery (the closest thing to a queue) | `internal/notify/{service,retry,channel,deadletter}.go` (~850 LOC core)  | In-memory bus drain → synchronous channel send → retry (3 attempts, 500ms×2 expo, 30s cap, ±20% jitter, 429/5xx/transport only) → terminal failure parked to SQLite DLQ |
| DLQ claim                                            | `internal/notify/deadletter.go:164-166`, `service.go:228-248`            | Atomic claim-by-**delete** `DELETE … RETURNING` (exactly one winner; no lease, no redelivery-on-crash)                                                                  |
| DLQ operator API                                     | `internal/api/deadletters.go` (~120 LOC)                                 | list / requeue / discard endpoints + Prometheus counters                                                                                                                |
| Idempotency                                          | `internal/api/idempotency.go` (182 LOC)                                  | `idempotency_responses` replay-by-key, conditional upsert, TTL + sweep                                                                                                  |
| Expiration worker                                    | `internal/worker/{expiration,service}.go` (211 LOC)                      | Single ticker goroutine scanning expirable rows — not a queue consumer                                                                                                  |
| Event fanout                                         | `internal/events/bus.go` (~312 LOC)                                      | In-memory `BufferedEventBus` (channels, drop policies) — **lossy on restart**                                                                                           |
| Event journal                                        | `internal/events/store_sqlite.go`, `migrations/001_initial.up.sql:22-35` | Append-only `events` with `UNIQUE(aggregate_id, version)` OCC                                                                                                           |

Queue-adjacent code: **~1,600–1,700 LOC** (≈2,200 with the five delivery
channels). Already a go-cqrs-lite consumer (`go.mod:13-19`: catalog,
decider, event, id, metadata, query).

## T20.2 — Mapping onto the queue contract

| PapDashboard today                                       | `queue/sqlite` equivalent                                                                                                        | Fit                                                                     |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Sync notify send + hand-rolled retry/backoff + DLQ park  | `Enqueue` per delivery attempt-schedule → worker `ClaimDue` → `Fail` (built-in backoff → `Dead`) → `DismissDead`/`FailPermanent` | **Direct** — the store subsumes `retry.go` + `deadletter.go` semantics  |
| DLQ claim-by-delete (crash loses the in-flight delivery) | Lease-based `ClaimDue` + `Heartbeat` + expiry reclaim (at-least-once)                                                            | **Upgrade** — crash-safe redelivery, the one real operational gap today |
| `Idempotency-Key` replay                                 | `task.New.DedupKey` enqueue dedup                                                                                                | Direct                                                                  |
| Cooperative shutdown (context + drain)                   | `CancelRunning`/`CancelRequested`/`CancelOwned` + lease expiry                                                                   | Direct                                                                  |
| Audit of state changes (Prometheus counters only)        | Same-tx fact journal (`facts.Enqueued/Completed/…`, `Facts(after, limit)`)                                                       | Upgrade                                                                 |
| Expiration ticker                                        | `NotBefore`-scheduled tasks claimed by the same worker loop                                                                      | Possible, not justified (see verdict)                                   |
| In-memory event bus fanout                               | Not a queue problem — live-push fanout with per-subscriber drop policies ≠ durable task queue                                    | Out of scope                                                            |

Semantics that carry over unchanged: single-writer SQLite
(`queue/sqlite` is single-writer WAL — matches their `MaxOpenConns(1)`
posture), JSON payloads (event snapshot JSON is already produced for the
DLQ), zero new infrastructure (same single file, same volume).

## T20.3 — Gap list (blockers / considerations)

1. **Packaging gate (hard, temporary):** the queue family
   (`queue`, `queue/sqlite`, `queue/postgres`, `queue/mysql`, `claiming`)
   is **not tagged yet** — the ONE-wave v4.0.0 tag wave is still pending
   in TODO_LIST. PapDashboard consumes tagged modules only; adoption
   starts after that wave.
2. **Dual SQLite drivers (consideration):** PapDashboard runs
   `mattn/go-sqlite3` (CGO); `queue/sqlite` runs `modernc.org/sqlite`
   (pure Go). Two drivers in one binary works (separate connection
   domains) but doubles the dependency. Migrating the app's own DB to
   modernc is the cleaner end state (its `_journal_mode` DSN pragmas need
   a modernc-equivalent rewrite) — optional, app-side, not a queue
   blocker.
3. **At-least-once redelivery (behavior change):** lease expiry makes
   redelivery possible where claim-by-delete was exactly-once-ish.
   Notification channels are idempotency-tolerant (HTTP senders), and the
   current code already retries, so this is acceptable — but the notify
   channel layer should pass a send-idempotency key where an API supports
   it.
4. **No PostgreSQL anywhere in PapDashboard:** `queue/postgres` is
   irrelevant to this consumer; `queue/sqlite` is the only relevant
   engine.

## T20.4 — Verdict: ADOPT (targeted), gated on the tag wave

- **Adopt `queue/sqlite` for the notify pipeline** (~850 LOC of
  hand-rolled retry/DLQ/claim collapses into the store; gains
  crash-safe redelivery, priorities if ever needed, and the journaled
  facts the Prometheus counters only approximate). Effort: M (worker loop
  - Enqueue-on-receive rework in `internal/notify/service.go`; delete
    `retry.go` + `deadletter.go` core; keep the operator API — requeue maps
    to `RescueDead`/`Requeue`, discard to `DismissDead`).
- **Keep** the in-memory bus (live-push fanout is a different problem) and
  the expiration ticker (a durable queue adds failure modes with no gain
  for a single-node timer).
- **Re-evaluate after** multi-node deployment ever lands — at that point
  `queue/postgres` + a real worker-pool topology becomes the right
  upgrade, which is the scenario the 2026-09-13 proposal anticipated.

Blockers: only #1 (tag wave), which is this repo's own pending item —
no PapDashboard-side blocker.
