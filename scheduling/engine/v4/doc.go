// Package engine provides the ADR-0142 scheduling facade: a
// scheduling.TimerStore[P] backed by ANY metaengine engine that implements
// metaengine.DueClaimer — timers become claims on the engine, so the ONE
// claim stack serves timers, queues, and dedup (no satellite SQL).
//
// Mapping:
//
//	Schedule   → ClaimInsert (idempotent by TimerID, NotBefore = FireAt)
//	Due(now)   → ClaimDue claim-loop (due-ordered, lease-fenced: two Due
//	             callers never receive the same unfired timer)
//	MarkFired  → ClaimDeleteIfDue with the claiming Due's epoch — a stale
//	             finalizer cannot delete a re-scheduled generation (the
//	             documented MarkFired race, fixed structurally)
//	Cancel     → ClaimDelete
//
// Payloads are the JSON-encoded scheduling.Timer[P]; the collection name
// defaults to "timers" (one collection per payload type P is the norm).
//
// Delivery semantics are at-least-once inside the lease window, identical to
// scheduling/sqlstore's ClaimingTimerStore: a dispatcher that crashes after
// Due keeps its timers leased until the lease lapses, then another poller
// may claim them again. Dispatch handlers should be idempotent.
package engine
