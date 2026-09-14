# Dedup seam decision — native partial-unique index, NOT go-idempotency (T10)

**Date:** 2026-09-14
**Task:** SUPERB plan T10 — "Dedup'd enqueue via go-idempotency contract seam".
**Verdict:** the seam IS the `task.New.DedupKey` contract (shipped, conformance-pinned);
a go-idempotency-backed dedup adapter is REJECTED. This note is the design
record the plan's T10.1 asked for, and the rejection rationale for T10.2/T10.3.

## The two dedup semantics are different animals

| Property            | queue dedup (DedupKey)                        | go-idempotency keys                          |
| ------------------- | --------------------------------------------- | -------------------------------------------- |
| Question answered   | "does this WORK ITEM exist yet?"              | "was this COMMAND already executed?"         |
| Lifetime            | FOREVER — terminal tasks still suppress       | TTL-bounded (sweep expires old keys)         |
| Suppression shape   | returns the STORED task unchanged             | returns the stored RESULT of the command     |
| Consistency domain  | the same tx as the task INSERT (one index)    | a separate store/table, separate round trip  |
| Conformance pin     | T4.1–T4.3 (cancelled/dead keys suppress)      | expiry by design                             |

The forever-suppression pin is the load-bearing difference: a TTL-expired
dedup key re-enqueues completed work — the exact convergence failure the
donor's harvester exists to prevent (repeated harvest runs converge, not
duplicate). Wiring queue dedup onto go-idempotency would import TTL
semantics into a forever contract; disabling the TTL to fix that makes
the key store a strictly worse copy of the partial unique index the
engines already have (no in-tx pairing, extra round trips, second schema
to operate).

## Where go-idempotency DOES compose (already true)

Consumers running the full cqrs stack get command-side dedup from
`middleware` + `idempotency/{kvstore,sqlstore}` at DISPATCH time — the
same layer tq's ecosystem uses it at. Queue dedup guards the entity;
idempotency guards the command; both exist, at their own layers, without
an adapter between them.

## Engine note

Both engines implement dedup natively and identically (SQLite partial
unique index `WHERE dedup_key != ''`; Postgres the same), pinned by the
shared suite: same-key returns the stored task unchanged with no
duplicate fact, terminal keys suppress forever, keyless tasks never
collide.
