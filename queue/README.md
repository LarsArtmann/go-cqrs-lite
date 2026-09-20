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

| Module              | Role                                                                                |
| ------------------- | ----------------------------------------------------------------------------------- |
| `queue/v4`          | The `Store[T]` contract, task/filter/fact types, error sentinels                    |
| `queue/sqlite/v4`   | Engine: single-writer SQLite (WAL) — zero-ops default                               |
| `queue/postgres/v4` | Engine: SKIP LOCKED claims over pgxpool — concurrent workers                        |
| `queue/mysql/v4`    | Engine: two-statement SKIP LOCKED claims on MySQL 8+/MariaDB 10.6+                  |
| `queue/conformance` | The mirrored suite every engine must pass (`-tags integration` for SQL engines)     |
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
defer store.Heartbeat(ctx, claim.ID(), claim.Token, 30*time.Second)

// ... do the work ...
err = store.Complete(ctx, claim.ID(), claim.Token, result)
```

Every claim mints an unguessable token (ADR-0134): finalize calls
(`Complete`, `Fail`, `Requeue`, `Heartbeat`, `CancelOwned`) present the
token, and a worker whose lease lapsed and was re-claimed gets
`queue.ErrLeaseNotHeld` — the finalize path is the theft detector.

## Quickstart (Postgres)

```go
store, err := postgres.Open[payload](ctx, dsn, 0) // 0 = pgxpool default MaxConns
```

The schema is created on open. Claims use `SELECT ... FOR UPDATE SKIP LOCKED`
inside a transaction, so any number of workers can poll concurrently.

## Quickstart (MySQL / MariaDB)

```go
store, err := mysql.Open[payload]("user:pass@tcp(127.0.0.1:3306)/tasks?parseTime=true")
```

Same shape as Postgres: schema on open (InnoDB — the dedup emulation needs
a nullable unique index), two-statement `SKIP LOCKED` claims, `BIGINT`
millisecond timestamps, automatic InnoDB deadlock retry. Pool knobs:
`mysql.WithMaxOpenConns[payload](16)` (default 8) and
`mysql.WithMaxIdleConns[payload](0..n)` (default 2).

The engine's own suite resolves its server via `MYSQL_TEST_DSN`, a local
Docker MariaDB 11.4 (`testutil/mysqltestcontainer`), or skips when neither
is available (`-race -count=2` green on MariaDB 11.4).

Deadlock scope: only `ClaimDue` retries deadlocks internally (bounded,
backoff+jitter) — claims are the one constant-contention path. Enqueue and
the finalize calls surface a rare deadlock to the caller instead, whose
retry is safe: `Enqueue` is idempotent under its `DedupKey` (keyless
enqueues are deliberately non-idempotent) and every finalize is
token-fenced (a retry either lands or reports `ErrLeaseNotHeld` after a
theft).

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
  Every dep must already exist at enqueue (`queue.ErrDanglingDep`
  otherwise) — which also makes dependency cycles unrepresentable: the
  closing edge of any cycle would name a task that does not exist yet.
  Cancelled and dead deps block forever (rescue the dep to re-open the
  gate) so stranded waiters stay visible instead of silently running.
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

### MySQL reality notes

Dialect traps found and handled in the MySQL engine (worth knowing when
porting further engines or debugging live):

- **No multi-statement DDL by default**: the schema is applied one
  statement at a time (`schemaStmts` slice) so the DSN never needs
  `multiStatements=true`.
- **Strict mode inserts**: strict-mode servers reject implicit defaults —
  the nullable `last_error LONGTEXT NOT NULL` column gets an explicit
  `''` in the INSERT.
- **InnoDB deadlocks under concurrent claims are normal** for the
  two-statement `SKIP LOCKED` pattern; `ClaimDue` retries them internally
  (3 retries, 25ms→200ms full-jitter backoff). See the deadlock-scope note
  in the MySQL quickstart above for why enqueue/finalize do not retry.
- **Testing**: set `MYSQL_TEST_DSN` (server DSN, e.g.
  `root@tcp(127.0.0.1:3306)/?parseTime=true`) — every subtest creates a
  throwaway database on that server. Without the env var, the suite boots
  a local Docker MariaDB 11.4 via `testutil/mysqltestcontainer`, and skips
  when neither is available or under `-short`.
