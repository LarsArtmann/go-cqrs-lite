# ADR-0059: DLQ Unification Proposal

## Status

**Proposed** — 2026-07-23. Drafted as a forward path for the consumer burden
documented in [ADR-0043](0043-dlq-unification-options.md). Not yet committed;
awaiting a decision on whether the burden justifies a major-version breaking
change.

## Context

ADR-0043 (Option C) decided to keep `middleware.DeadLetterEntry` and
`projectionhost.DeadLetterEntry` as separate types. The rationale is sound:
they serve genuinely different lifecycles (dispatch-retry-exhaustion vs
projection-poison), have different fields, and unifying them would force
middleware to carry `event.Event` even for command/query dead-letters.

However, consumers who use **both** retry middleware and projectionhost face
a real ergonomic cost: two parallel DLQ APIs with divergent field shapes
(`Error error` vs `Error string`, `AggregateID id.AggregateID` vs `string`,
no shared Store interface). A dead-letter dashboard or alerting hook cannot
treat both uniformly without writing adapter code.

This ADR proposes a narrow unification that addresses the consumer burden
without forcing the structural merge that ADR-0043 rejected.

## Proposal: Option D (Bridge via Shared Replay + Typed Adapter)

Keep both types, but:

1. **Add `Event event.Event` to `middleware.DeadLetterEntry`** (optional —
   nil for non-event messages). This gives the dispatch DLQ the same replay
   capability as the projection DLQ, without forcing it for commands/queries.

2. **Add `Replay(handler)` to `middleware.MemoryDeadLetterStore`** — mirrors
   `projectionhost.Host.ReplayDeadLetters`. Both DLQ layers gain replay
   parity.

3. **Introduce a `dlq.Summary` adapter type** — a read-only struct that both
   entry types can be converted to, for dashboards and alerting:

```go
package dlq

type Summary struct {
    Source      string // "dispatch" or "projection"
    Kind        string // "command", "event", "query" (dispatch) or "event" (projection)
    MessageID   string
    EventType   string
    AggregateID string
    Error       string
    ErrorFamily string
    OccurredAt  time.Time
}

func FromMiddleware(e middleware.DeadLetterEntry) Summary { ... }
func FromProjection(e projectionhost.DeadLetterEntry) Summary { ... }
```

This does NOT merge the types. It gives consumers a single read model for
monitoring while preserving the structural separation that ADR-0043
justified.

## What This Does NOT Do

- Does NOT create a new `dlq/` module (Option A from ADR-0043). The Summary
  type can live in a small shared package or as a consumer-side adapter.
- Does NOT change the Store interface shape. `middleware` keeps its callback;
  `projectionhost` keeps its Store interface.
- Does NOT force `event.Event` on command/query dead-letters. The field is
  optional (nil for non-events).

## Open Questions

1. **Is the consumer burden large enough to justify this?** ADR-0043's
   rationale for separation is strong. This proposal is only warranted if
   real consumers report the two-API pain.
2. **Where does `dlq.Summary` live?** Options: new `dlq/` module (adds a
   module), `middleware/` (projectionhost depends on middleware — wrong
   direction), or consumer-side adapter pattern (no library change).
3. **Timeline?** If adopted, this is a v4 or v5 change — it adds fields to
   `middleware.DeadLetterEntry` (additive but changes the struct shape).

## References

- [ADR-0042](0042-pure-replay-dead-letters.md) — Pure replay design (projection side)
- [ADR-0043](0043-dlq-unification-options.md) — Decision to keep types separate (Option C)
