# ADR-0042: Pure Replay for Dead-Letter Queue

## Status

Accepted — 2026-06-29 (implemented: `projectionhost.Host.ReplayDeadLetters`)

## Context

`projectionhost.Host.ReplayDeadLetters` re-feeds poison messages to their
projection after the handler bug that originally poisoned them has been fixed.

The first implementation (commit prior to `9fda1454`) was **mutating**: after
one entry replayed successfully, it called `dlq.Purge(projectionName)` — which
wipes **ALL** entries for that projection. If "orders" had 3 poisoned events
and one replayed OK, the other 2 still-failing entries were silently dropped.

**This was silent data loss in a dead-letter queue** — whose entire purpose is
to never lose data. The bug survived all tests because every test used a single
entry, making `Purge` and a hypothetical `Delete` indistinguishable.

When challenged ("Why add a Delete method?"), the first instinct was to grow
the interface with `Delete(name, eventID)`. The right answer was different:
make replay **pure** and let the caller decide cleanup.

## Decision

**`ReplayDeadLetters` is pure: it returns `ReplayResult{Replayed, StillFailing}`
and does NOT mutate the `DeadLetterStore`. Cleanup is the caller's explicit
responsibility via the existing `Purge`.**

```go
type ReplayResult struct {
    Replayed     []DeadLetterEntry
    StillFailing []ReplayFailure
}

func (h *Host) ReplayDeadLetters(ctx, projectionName) (ReplayResult, error)
```

### Why pure over mutating

| Approach | Mutation-as-side-effect (rejected) | Pure replay (chosen) |
| --- | --- | --- |
| Bug surface | `Purge(name)` after one success drops ALL entries | None — store untouched |
| Composability | Couples "did the handler work?" to "drop the entry" | Caller can log, alert, or partial-purge |
| Testability | Needs multi-entry tests to expose the bug | Single-entry tests are honest |
| CQS | Violates command-query separation (it's both) | Query only; cleanup is a separate command |

### The general lesson

**"Mutation-as-side-effect-of-query" is a code smell.** Replay answers a
question ("does this handler work now?"); purging is a command. Mixing them
couples two concerns and produces exactly this class of partial-success bug.
This applies beyond DLQs: any retry/replay/dry-run API should be pure, with
the caller choosing whether to commit the result.

## Consequences

- Callers must explicitly `Purge` after a successful replay. This is documented
  on the method and demonstrated in `example/projectionhost`.
- `DeadLetterEntry` carries the original `event.Event` so pure replay can
  re-feed it without an external lookup. The field may be nil for entries
  created by older code (documented).
- The DLQ interface stays at 3 methods (`Store`/`List`/`Purge`). No `Delete`
  was added — entry-scoped removal is a separate, future decision (see plan D-tier).

## Alternatives considered

1. **Add `Delete(name, eventID)` and call it per successful entry.** Rejected:
   growing the interface to fix a side-effect bug papers over the real issue
   (mutation coupled to query). Pure replay is simpler and safer.
2. **Return a callback the caller must invoke to commit.** Rejected: callbacks
   are easy to forget and harder to test than an explicit `Purge` call.
3. **Auto-purge only the successfully replayed entries.** Tempting, but still
   mutation-as-side-effect — and the next request is inevitably "let me inspect
   what was purged", which forces re-adding them anyway.

## References

- Original bug fix: commit `9fda1454`
- Plan entry: S5 in `docs/planning/2026-06-29_brutal-self-review-execution-plan.md`
- Related: ADR-0043 (DLQ unification options) — deferred, pending user decision
