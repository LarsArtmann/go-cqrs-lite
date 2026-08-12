# Review: CircuitBreaker and DLQ Extraction Request

**Source feedback:** [`new/2026-08-09_file-renamer_extract-circuitbreaker-and-dlq.md`](../new/2026-08-09_file-renamer_extract-circuitbreaker-and-dlq.md)
**Date reviewed:** 2026-08-13
**Outcome:** No new modules. Both requests correctly resolved as "existing tooling is sufficient."

---

## CircuitBreaker → failsafe-go (docs, not a module) — ✅ COMPLETED

The feedback's prependix decision is confirmed. `middleware/circuit_breaker.go` delegates the state machine to failsafe-go — it does not implement one. A SKILL.md FAQ entry has been added at `.agents/skills/go-cqrs-lite/references/faq.md` directing standalone circuit-breaker consumers to import `failsafe-go/circuitbreaker` directly, with a cross-reference to `middleware/circuit_breaker.go` as the CQRS integration pattern.

**Rationale:** A thin facade would add naming cosmetics at the cost of a leaky abstraction. failsafe-go already exposes the exact API the consumer wants.

---

## DLQ → out of scope — ✅ CONFIRMED

The consumer's `pkg/deadletter/deadletter.go` is an application-level retry queue for failed file-rename operations, not a dead-letter queue. The projectionhost `DeadLetterStore` is tightly coupled to `event.Event` by design (ProjectionName, EventID, EventType, StreamID, ErrorCode, ErrorFamily). Genericizing it into `Entry[P any]` would either lose that richness or force a parallel typed layer.

**Rationale:** A generic failed-work queue is general-purpose application infrastructure, not CQRS/ES building material.

---

## Summary

| Request | Disposition | Rationale |
| --- | --- | --- |
| `circuitbreaker/v4` | Docs pointer, no module | failsafe-go IS the standalone breaker |
| `dlq/v4` | No module, out of scope | Consumer needs app-level retry queue, not event-specific DLQ |
