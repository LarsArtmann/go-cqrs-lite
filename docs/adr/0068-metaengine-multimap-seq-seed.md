# ADR-0068: Multimap seq-seed via sync.Once on first access

**Date:** 2026-07-25
**Status:** Accepted

## Context

The SQLite `MultimapBackend` models one key → many ordered values using a
`(collection, key, seq)` primary key, where `seq` orders the values per key. The
in-memory `seq` counter starts at zero for each new engine instance.

On **process restart**, a freshly constructed engine opens an existing database
file whose multimap rows already occupy `seq = 0, 1, 2, …`. A fresh in-memory
counter starting at 0 would emit `seq = 0, 1` for new values and collide with
existing rows on the primary key → `UNIQUE constraint failed`, losing data or
erroring. A regression test (`hardening_test.go`: reopen DB, add 2 more values,
expect 5 total) would fail.

## Decision

Seed the in-memory `seq` counter lazily on first use of a `(collection, key)`
via `SELECT MAX(seq) … ` guarded by `sync.Once` per key. The first `MultiAdd`
after (re)open reads the persisted high-water mark and continues from there, so
the sequence is monotonic across restarts without paying a `SELECT` on the hot
path after the first write.

```go
// multiSeqCounter: sync.Once guards a one-time SELECT MAX(seq) seed per key;
// subsequent increments are in-memory and allocation-free.
```

## Consequences

- **+** Restart-safety: reopening a database never reuses a `seq`, so multimap
  appends survive process restarts without PK collisions.
- **+** Hot path is unaffected: the `SELECT MAX(seq)` runs exactly once per
  `(collection, key)` per engine lifetime; every subsequent `MultiAdd` is an
  in-memory increment + a single `INSERT`.
- **−** The first write to a key after open incurs one extra read. This is
  negligible (once per key per process).
- **−** `sync.Once` makes the seeding goroutine-safe but not crash-safe across
  an ungraceful exit mid-seed. Because the seed reads the persisted `MAX(seq)`,
  a restart simply re-seeds from the committed high-water mark; no corruption.

## Alternatives considered

- **Eager seed for all keys at construction:** rejected — `O(rows)` scan at open
  for a multimap that may have millions of keys.
- **Database-side autoincrement seq:** rejected — would couple ordering to a
  global counter rather than per-`(collection,key)`, changing the semantics.

## Prior Art

- **PostgreSQL `SEQUENCE`:** Database-maintained counters (`CREATE SEQUENCE`).
  Per-key sequences would require a sequence per `(collection, key)` pair —
  impractical. This ADR's `sync.Once` approach is the application-level
  equivalent for composite-key sequences.
- **MongoDB `ObjectId`:** Embeds a timestamp for natural ordering; no explicit
  sequence needed. The multimap's `seq` serves a different purpose: stable
  within-collection ordering per key, not globally unique.
- **Redis `INCR`:** Atomically initializes to 0 and increments. If the
  metaengine's KV layer supported atomic `INCR` per `(collection, key)`, the
  `sync.Once` seed would be unnecessary. The current KV interface lacks an
  atomic-increment operation (it has `SetIfAbsent` but not `Increment`).
- **Django `bulk_create`:** Relies on the database's auto-increment for PK
  assignment after bulk insert. The seq-seed pattern is necessary here because
  the multimap's `(collection, key, seq)` composite PK doesn't map to a single
  auto-increment column.
