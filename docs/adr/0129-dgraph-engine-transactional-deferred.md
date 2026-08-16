# ADR-0129: Dgraph Engine Does Not Implement Transactional (RunInTx) Yet

**Date:** 2026-08-16
**Status:** Accepted (deferred capability)

---

## Context

`metaengine.Transactional` (`RunInTx(ctx, fn)`) lets callers run
version-check-then-append sequences atomically. The `system` EventAdapter uses
it (via `AtomicAppender`-style optimistic concurrency) when the engine exposes
it; engines without it fall back to a best-effort append.

The Dgraph engine (`metaengine/dgraphengine`) does NOT implement
`Transactional`, and the TODO list asked for either an implementation or an
ADR explaining why not.

## Decision

Do not implement `Transactional` in `dgraphengine` for now. Keep the honest
non-implementation so the capability audit, Doctor, and the planner all see
the truth (no interface, no declaration).

## Why not now

1. **Per-op transactions are the current unit of work.** Every write path
   (`MapSet`, `MapDelete`, counter upserts, `DeleteJson`-with-null-predicates)
   already opens its own single-operation `dgo.Txn` — many via the upsert
   pattern that Dgraph requires for read-modify-write. Correct `RunInTx`
   support means ALL of these paths must detect and join an ambient
   transaction carried in `ctx`, otherwise the "transaction" silently commits
   per-op — the exact silent-divergence hazard documented for iroh's
   `Replicated` wrapper (see `metaengine/irohengine/engine_passthrough.go`).
2. **Dgraph txn semantics need mapping work.** Dgraph aborts conflicting
   transactions with `ErrAborted`; exposing that as the metaengine conflict
   family, retry semantics, and discard-on-panic behavior each need deliberate
   design plus a conformance run via `enginetest.RunTransactionalTest`.
3. **No consumer demand yet.** The planner routes event-sourcing workloads by
   `Profile()`; dgraph's role today is graph/scan/RAG-shaped queries where
   single-op atomicity suffices.

## Consequences

- `dgraphengine` stays out of `Transactional`/`AtomicAppender` code paths;
  optimistic-concurrency workloads are routed to engines that implement them.
- The capability conformance table stays green (nothing declared, nothing
  implemented).
- Revisit when: a consumer needs ES optimistic concurrency on Dgraph, or the
  planner begins requiring `Transactional` for ADTStreamLog routing.

## Implementation sketch (when revisited)

- Wrap `dgo.Txn` in a context value: `RunInTx` opens a read-write txn, stores
  it in `ctx`, runs `fn`, commits on nil error, discards otherwise.
- Every write path: `txnFrom(ctx)` helper returns the ambient txn or opens a
  fresh one.
- Map `y.ErrAborted` to the Conflict error family.
- Gate with `enginetest.RunTransactionalTest` in the module's test suite.
