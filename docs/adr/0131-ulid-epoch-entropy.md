# ADR-0131: ULID Generation Uses Epoch-Sharded Entropy (Not a Global Monotonic Reader)

- **Status:** Accepted (implemented; this ADR documents the decision)
- **Date:** 2026-08-29
- **Context:** plan V3 T40; deep-review question "ULID sharded entropy — the
  global `ulid.MonotonicEntropy` reader serializes every ID generation
  process-wide behind one mutex" (status 2026-08-16_17-39).

## Context

The previous `id` implementation wrapped a shared `ulid.MonotonicEntropy` in
a global mutex. Correct, but every `id.New*` call in the process contended on
one lock — measurable on the event-append hot path, which mints an ID per
event.

## Decision

`id/entropy.go` replaces the shared reader with a lock-free epoch design:

- **Layout:** ULID = 48-bit millisecond timestamp ‖ 6 crypto-random epoch
  bytes ‖ 4-bit…4-byte atomic counter (per-millisecond suffix).
- Each millisecond gets a fresh 48-bit crypto-random prefix, published as an
  immutable epoch under a mutex that is taken **at most once per millisecond**.
- The fast path is one atomic pointer load + one atomic add; same-millisecond
  IDs are strictly increasing across ALL goroutines via a global atomic
  counter, so ULID sort order still equals issuance order.

## Tradeoffs accepted

- An observer of one ID can predict its same-millisecond successors (equally
  true of the monotonic reader it replaces).
- Cross-millisecond IDs carry 48 random bits instead of 80. ULIDs are
  identifiers, not secrets; 2^48 makes cross-millisecond guessing infeasible.
- Wall-clock steps backwards pin the current millisecond and keep the counter
  increasing: ULIDs never regress.

## Consequences

No further work queued: the "should we shard entropy?" question is closed by
this design. If a future consumer needs unpredictable same-millisecond
successors, that is a new requirement (a keyed/blind ID scheme), not a tweak
to this one.
