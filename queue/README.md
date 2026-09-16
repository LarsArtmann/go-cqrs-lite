# queue — Durable Work Queue

A durable, journal-first work queue for Go: task lifecycle with lease-based
claims, retries with backoff, a dead-letter queue, priorities with bounded
aging, DAG dependency gating, idempotent (dedup-keyed) enqueue, and an
append-only fact journal written in the SAME transaction as every state
change.

The semantics are production-proven, not designed on paper: they are
transcribed from go-taskqueue's internal queue stores (its `Store` contract
is the spec donor). Engines are held to identical semantics by ONE shared
conformance suite.

## Modules

| Module              | Role                                                                              |
| ------------------- | --------------------------------------------------------------------------------- |
| `queue/v4`          | The `Store[T]` contract, task/filter/fact types, error sentinels                   |
| `queue/sqlite/v4`   | Engine: single-writer SQLite (WAL) — zero-ops default                              |
| `queue/postgres/v4` | Engine: SKIP LOCKED claims over pgxpool — concurrent workers                       |
| `queue/conformance` | The mirrored suite every engine must pass (`-tags integration` for SQL engines)    |
| `claiming/v4`       | The extracted claim core (Spec, SKIP LOCKED / single-writer / two-statement claims) |

## Quickstart (SQLite)

```go
store, err := sqlite.Open[payload]("file:tasks.db")
if err != nil {
    return err
}
defer store.Close()

t, err := store.Enqueue(ctx, task.New[payload]{
    Project:  "docs",
    Type:     "render",
    Payload:  p,
    DedupKey: "docs:readme", // repeated producers converge, no duplicates
})
if err != nil {
    return err
}

claim, err := store.ClaimDue(ctx, "worker-1", 30*time.Second)
if errors.Is(err, queue.ErrNoTaskDue) {
    return nil // nothing claimable right now
}
if err != nil {
    return err
}
defer store.Heartbeat(ctx, claim.ID(), "worker-1", 30*time.Second)

// ... do the work ...
err = store.Complete(ctx, claim.ID(), "worker-1", result)
```

## Quickstart (Postgres)

```go
store, err := postgres.Open[payload](ctx, dsn, 0) // 0 = pgxpool default MaxConns
```

The schema is created on open. Claims use `SELECT ... FOR UPDATE SKIP LOCKED`
inside a transaction, so any number of workers can poll concurrently.

## The contract in one minute

- **Lifecycle:** `pending → running → completed | dead | cancelled`. Terminal
  states are terminal; `RescueDead` is the explicit, journaled way out.
- **Claims are leases.** `ClaimDue(owner, lease)` hands the task to one owner;
  `Heartbeat` extends the lease; expiry makes the task claimable again
  (at-least-once delivery — executors must tolerate redelivery).
- **Retries + DLQ.** `Fail` requeues with backoff until the attempt budget
  (default `task.DefaultMaxAttempts`) is spent, then the task goes `Dead`.
  `FailPermanent` skips the remaining budget. `DismissDead` closes DLQ items.
- **Journal-first.** Every state change appends its fact(s) in the same
  transaction. `Facts(after, limit)` is the audit stream; a state change
  without its fact did not happen.
- **Priorities + aging.** Stored priority is the fairness order; bounded aging
  breaks starvation at claim time. `UpdatePendingPriority` rewrites enqueue-time
  truth (pending only) and journals the provenance.
- **DAG gating.** `task.New.Deps` holds a task back until its dependencies
  complete; unblock happens transactionally with the completing write.
- **Cooperative cancel.** `CancelRunning` requests, the worker observes via
  `CancelRequested`, finalizes via `CancelOwned`. Lease expiry finalizes it.

## Partial-literal types (exhaustruct exemptions)

`task.New`, `task.Task`, `queue.Filter`, and `facts.Fact` are deliberately
exempt from the `exhaustruct` linter: their zero values are meaningful, so
partial literals are the intended call shape — `task.New[T]{Type: "render"}`
is a complete enqueue template (ID, status, attempts, and timestamps are
store-assigned; `Normalize` applies the default attempt budget), the zero
`Filter` lists everything, and `facts.Fact`'s optional fields collapse to
their zero values. Listing every field at every call site would be noise, not
safety.

## Testing an engine

Engines must pass the shared suite — that is the parity bar:

```go
func TestConformance(t *testing.T) {
    conformance.Run(t, conformance.Harness{
        NewStore: func(t *testing.T) queue.Store[payload] { /* open engine */ },
        Backdate: backdate, // white-box hook for the aging pins
    })
}
```
