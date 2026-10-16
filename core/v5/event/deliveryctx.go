package event

import "context"

// The synchronous-bus-delivery marker: bus implementations that deliver to
// handlers on a single goroutine under a publish-held lock (the watermill
// EventBus/CommandBus under BlockPublishUntilSubscriberAck) mark the
// context they hand to handlers, so a nested synchronous Publish on that
// same context can fail fast with a named error instead of deadlocking
// (ADR-0154; live stack evidence in
// docs/evidence/2026-10-09_deriver-bus-deadlock.md of the go-cqrs-lite
// repo). The marker is context-carried — goroutine-ID-free by design.
//
// Three functions, three roles:
//   - bus delivery paths call [MarkInDelivery] before invoking handlers;
//   - bus Publish paths consult [ContextInDelivery] and reject reentrant
//     publishes with their named reentrancy error;
//   - sanctioned async escapes (goroutines that may publish back — e.g.
//     deriver.WithAsyncDispatch) call [WithoutDeliveryMark] so the nested
//     publish runs clean on its own goroutine.

// deliveryMarkKey is the unexported marker key; a false value explicitly
// clears an inherited mark (WithoutDeliveryMark), so absence and cleared
// are distinguishable only through ContextInDelivery's boolean result.
type deliveryMarkKey struct{}

// MarkInDelivery returns ctx with the in-delivery marker set. Bus
// implementations call this on the context they pass into handler
// dispatch; the marker then flows through synchronous handler chains
// (command dispatch, derived work) to any nested Publish.
func MarkInDelivery(ctx context.Context) context.Context {
	return context.WithValue(ctx, deliveryMarkKey{}, true)
}

// ContextInDelivery reports whether ctx is marked as being inside a
// synchronous bus delivery. Bus Publish paths reject with their named
// reentrancy error when this is true: a nested synchronous publish on the
// delivering goroutine deadlocks buses that hold a per-topic lock across
// publish while waiting for the subscriber to ack.
func ContextInDelivery(ctx context.Context) bool {
	marked, _ := ctx.Value(deliveryMarkKey{}).(bool)

	return marked
}

// WithoutDeliveryMark returns ctx with the in-delivery marker cleared.
// Call this when escaping the delivery goroutine asynchronously and the
// escape may legitimately publish back to the bus (deriver.WithAsyncDispatch
// does): values still flow from the original context, but the nested
// publish is no longer reentrant — it runs on its own goroutine, outside
// the delivering publish's lock-wait cycle.
func WithoutDeliveryMark(ctx context.Context) context.Context {
	return context.WithValue(ctx, deliveryMarkKey{}, false)
}
