# Decision memo — ratify queue dep-validation semantics (M4 §f1)

**Date:** 2026-09-20
**Owner question** (open since 2026-09-14 §g3 and 2026-09-19 12-10 §g1):
ratify "every dependency must already exist at enqueue" as the queue
contract, or restore the donor's donor-faithful blindness?

## What shipped (v4.x, all three engines, conformance-pinned)

`task.New.Deps` entries are validated **at enqueue**: a dep that does not
exist is rejected with `queue.ErrDanglingDep` (Rejection family,
`queue.dangling_dep`) before any row or journal fact is written — SQLite
`json_each` anti-join, PG `jsonb_array_elements_text` anti-join, MySQL
per-dep EXISTS. Consequences, by construction:

- **DAG gating is total**: a task never waits on a typo'd or late-declared
  dep ID; that enqueue fails loudly instead of silently blocking forever.
- **Cycles are unrepresentable**: deps are fixed at enqueue and task IDs
  are store-minted, so the closing edge of any cycle would name a task
  that does not exist yet — rejected as dangling. No runtime cycle
  detector exists or is needed.
- **Cancelled/dead deps block forever** (rescue re-opens the gate), so
  stranded waiters stay visible instead of silently running.

Documented in `queue/doc.go`, `queue/README.md` ("DAG gating"), the Deps
conformance suite (4 pins), and the error-taxonomy gate (bidirectional).

## The deviation from the donor

go-taskqueue (the spec donor, five weeks of production dogfood) accepted
deps that did not exist yet at enqueue — its callers declared DAGs
incrementally, creating edges before targets. go-cqrs-lite requires
bottom-up creation order (deps first, dependents after). The TODO text
that scoped M4 listed "enqueue validation" as wanted, so it was executed;
the deviation itself was flagged for ratification rather than assumed.

## Options

| Option                            | Effect                                                                                                                                                                                                                                                                           |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A. Ratify as-is (recommended)** | Contract stands; donor-style incremental DAG declaration stays unsupported. Callers that wanted it can create placeholder tasks — but the simpler answer is "declare deps bottom-up", which the validation teaches at enqueue time with a precise error.                         |
| B. Restore donor blindness        | Remove the anti-join checks on 3 engines, delete the 4 conformance pins + error-taxonomy entries, lose the unrepresentable-cycle property (needs a separate cycle story or an explicit "cycles possible" caveat), renegotiate `ErrDanglingDep`'s public API before the tag wave. |

## Recommendation

**Option A.** The validation is load-bearing for two shipped guarantees
(total dep gating + cycle unrepresentability), is conformance-pinned on
all engines, and costs one rejection error at the exact moment the caller
can fix the problem cheaply. No consumer exists yet for donor-style
incremental declaration; if one appears, a `WithDepValidation(enabled)`
escape hatch can be added additively later without breaking anything.

**Decision needed:** owner reply "A" (or "B") — the queue family tag wave
(v4.0.0, one wave with `claiming`) freezes whichever semantics ship.
