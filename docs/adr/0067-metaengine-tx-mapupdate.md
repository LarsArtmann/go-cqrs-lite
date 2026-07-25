# ADR-0067: Transaction-atomic MapUpdate for the SQLite engine

**Date:** 2026-07-25
**Status:** Accepted

## Context

`MapBackend.MapUpdate(collection, key, fn)` is a read-modify-write: it loads the
current value, applies `fn(prev)`, and stores the result. For the SQLite engine,
doing the read and the write as **separate statements** without a transaction
loses updates under concurrency: two concurrent `MapUpdate` calls can both read
the same `prev`, both compute a new value, and the second write clobbers the
first (lost update anomaly).

A regression test (`hardening_test.go`: 50 concurrent increments → expect +50)
would intermittently report fewer than 50, exposing the race.

## Decision

Wrap the SQLite `MapUpdate` read-modify-write in a single `BEGIN … COMMIT`
transaction. SQLite serializes writers (one writer at a time per database), so
the transaction makes the load+compute+store atomic: concurrent `MapUpdate`
calls are serialized, and no increment is lost.

```go
// sqlite_backends.go — MapUpdate runs inside RunInTx(ctx, db, func(tx) { … })
```

## Consequences

- **+** Correctness: concurrent `MapUpdate` calls never lose updates. The
  50-goroutine increment test is deterministic.
- **−** Write throughput on a single SQLite collection is serialized. This is
  acceptable: SQLite is chosen for persistence/portability, not write
  concurrency. For high-write-concurrency aggregations, the memory engine (or a
  dedicated counter backend) is the planner's pick.
- **−** A long-running `fn` holds the write lock. `fn` must be fast and
  side-effect-free (it is a pure fold by contract).

## Alternatives considered

- **Optimistic concurrency (CAS on a version column):** rejected — adds schema
  complexity and retry logic; SQLite's writer serialization already provides the
  guarantee more simply.
- **Application-level mutex:** rejected — only works single-process; the SQL
  engine exists precisely for multi-process correctness.
