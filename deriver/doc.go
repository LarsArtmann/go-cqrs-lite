// Package deriver provides event→command derivation with functional composition.
//
// A [Deriver] transforms events into zero or more derived commands. Derivers are
// deterministic: the same event always produces the same commands. This makes
// them safe for at-least-once delivery — re-processing an event produces the
// same derived commands, which idempotency checks at the command handler level
// can deduplicate via [command.WithCommandID] with a deterministic key derived
// from the source event's causation chain.
//
// Derivers compose via [Deriver.Then] (fan-out) and [Deriver.Filter] (event-type
// matching). The resulting handler is wired into the event bus:
//
//	d := sendWelcomeEmail.Then(syncToCrm)
//	bus.SubscribeAll(d.Filter("user.created").AsHandler(cmdDispatcher))
//
// Design rationale: see ADR-0040. The functional/composable API was chosen over
// a declarative rule registry because go-cqrs-lite is a library, not a database
// engine — there is no query optimizer to benefit from declarative rules.
package deriver
