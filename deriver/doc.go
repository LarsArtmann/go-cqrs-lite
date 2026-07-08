// Package deriver provides event→command derivation with functional composition.
//
// A [Deriver] transforms events into zero or more derived commands. Derivers are
// deterministic: the same event always produces the same commands. This makes
// them safe for at-least-once delivery — re-processing an event produces the
// same derived commands, which idempotency checks at the command handler level
// can deduplicate.
//
// For at-least-once safety, chain [Deriver.Idempotent] before [Deriver.AsHandler].
// Idempotent re-stamps each command with a deterministic [id.CommandID] derived
// from the source event's ID, so an idempotency store keyed on the command ID
// (see middleware.CommandIdempotency with a nil keyExtractor) deduplicates
// redeliveries automatically:
//
//	d := sendWelcomeEmail.Then(syncToCrm)
//	bus.SubscribeAll(d.Filter("user.created").Idempotent().AsHandler(cmdDispatcher))
//
// Derivers compose via [Deriver.Then] (fan-out), [Deriver.Filter] (event-type
// matching), and [Deriver.Idempotent] (deterministic IDs). The resulting handler
// is wired into the event bus.
//
// Design rationale: see ADR-0040. The functional/composable API was chosen over
// a declarative rule registry because go-cqrs-lite is a library, not a database
// engine — there is no query optimizer to benefit from declarative rules.
package deriver
