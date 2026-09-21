# Advanced queue & delivery patterns (go-cqrs-lite view)

Companion to [SKILL.md](../SKILL.md) (contract) and
[backends.md](backends.md) (broker matrix). Everything here is verified
against this repo's source, not upstream docs.

## 1. Delayed delivery — `NotBefore`

A task becomes claimable only after its `NotBefore` instant
(`queue/store.go:45`): `ClaimDue` selects `NotBefore <= now`, so delay is a
SCHEDULED-START concept, not a sleep. Use it for retry backoff (Fail sets
`NotBefore = now + backoff`, `queue/store.go:60`), rate limiting a downstream,
or time-windowed work. There is no broker-level delayed message involved —
the delay lives in the queue's own table, so it survives restarts and needs
no broker support (works on every backend).

## 2. Requeue vs Fail — evidence-carrying requeue

`Requeue` returns a CLAIMED task to Pending **without** counting an attempt,
and it demands evidence: the stored fact is `facts.Requeued` carrying
`facts.RequeueEvidence` (`queue/store.go:86-91`). Fail is the opposite path —
it counts the attempt and applies backoff via `NotBefore`. Rule of thumb:

- infrastructure hiccup, task itself fine → `Requeue` with the reason
- task actually failed → `Fail` (attempts++, DLQ after max)

Cancel of work you no longer want is `CancelOwned` (finalize your own claim);
an expired lease can only be finalized by the re-claimer.

## 3. Metrics — `ClaimMetrics` and the OTel bridge

`scheduling/sqlstore.ClaimMetrics` (`claim_metrics.go:24`) is the opt-in,
zero-dependency observability surface: counters for Claimed / Renewed /
RenewRejected wired via `WithClaimMetrics[P]`, snapshot via `Metrics()`
(returns `ClaimMetricsSnapshot`, includes the `StartedAt` anchor used for
cross-restart claim rates). Two consumption paths:

- built-in `Metrics()` JSON snapshot on a `/status` endpoint,
- OTel counters exposed on `/metrics` via the OTel→Prometheus bridge —
  worked example: `example/scheduler-otel-status`.

Queue claim transitions are also FACTS (`facts.Released`, `facts.Requeued`),
so the journal itself is an audit-grade metrics source.

## 4. Fan-in with the Router

Watermill's Router has no special fan-in primitive — one `HandlerFunc` per
subscribed topic, handlers run in parallel (SKILL.md §1). The repo patterns:

- fan-in by AGGREGATION: each handler publishes its output onto one
  aggregated topic; a final handler consumes that topic. Ordering between
  inputs is then broker-partition ordering, not global.
- fan-in by CHECKPOINT: for projection reads, prefer `CatchUpSubscriber`
  (the journal is the outbox) — see internals.md; the Router is for
  cross-process delivery, not for rebuilding state.

## 5. Troubleshooting

| Symptom | Cause | Fix |
| ------- | ----- | ---- |
| `ErrLeaseNotHeld` on Complete/Fail/Renew | your lease lapsed and the task was re-claimed (crash, long handler, GC pause) | the claim token lost; re-do the work idempotently — never resurrect a lapsed claim (`queue/token.go`) |
| Same message processed twice | at-least-once delivery (redelivery after Nack/crash, consumer-group rebalance) | handler idempotency or `middleware.{Command,Event,Query}Idempotency` / projectionhost dedup — watermill's own `Deduplicator` is process-local only |
| Task stuck in Running forever | holder died without finalize | ClaimDue reclaims after the lease deadline (crash reclaim); shorten lease or add heartbeats (`Heartbeat` extends while the token is still owned) |
| Poison message loops | handler always errors | attempts exhaust → DLQ; inspect via the commandlifecycle projections if commands, or the queue's facts journal |
| Handlers slow down over time | Ack/Nack cancels the message context — post-ack work on a stashed ctx fails or blocks | do post-ack work on a background context with explicit dedup (SKILL.md §1) |
| Ordered consumers stall | one slow partition/head-of-line blocks the ordered stream | isolate the slow topic, or relax to at-least-once + idempotent handlers (backends.md ordering column) |
