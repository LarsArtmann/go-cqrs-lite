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

## Prior Art

- **PostgreSQL UPSERT (`INSERT ... ON CONFLICT DO UPDATE`):** The exact SQL
  idiom this ADR uses. Documented in PostgreSQL's "UPSERT" guide since 9.5.
  The metaengine's `MapUpdate` is the Go-level abstraction over this idiom.
- **SQLite (`INSERT ... ON CONFLICT DO UPDATE`):** Same syntax since 3.24.0.
  SQLite's single-writer serialization makes this pattern race-free by default.
- **Marten (C#/.NET):** `IDocumentSession` wraps all writes in a Postgres
  transaction. Patch operations use `jsonb` atomic updates — the C# equivalent
  of this ADR's `fn(prev) result` applied inside a `BEGIN...COMMIT`.
- **Ruby on Rails:** `find_or_create_by` has a well-documented race condition;
  the fix is `upsert` (Rails 6+). This ADR avoids the race by design.
- **EventStoreDB:** Uses `ExpectedVersion` for optimistic concurrency rather
  than pessimistic transactions. Different approach (optimistic vs pessimistic);
  the metaengine chose pessimistic for simplicity on single-writer SQLite.
