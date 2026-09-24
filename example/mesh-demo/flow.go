// The demo's in-miniature event-sourcing loop: replay history through the
// fold, hand the state + next version to a DecideFunc, append. This is the
// same load→fold→decide→save core decider.Repository executes against a
// store — shown bare so the cross-context FLOW is the visible part.
package main

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func replay[S any](history []event.Event, initial S, fold func(S, event.Event) (S, error)) (S, error) {
	state := initial

	var err error

	for _, evt := range history {
		if state, err = fold(state, evt); err != nil {
			return initial, err
		}
	}

	return state, nil
}

// decideInto replays history, runs the DecideFunc at the next version, and
// returns history + the newly decided events.
func decideInto[S any](
	history []event.Event,
	initial S,
	fold func(S, event.Event) (S, error),
	decide decider.DecideFunc[S],
) ([]event.Event, error) {
	state, err := replay(history, initial, fold)
	if err != nil {
		return nil, err
	}

	newEvents, err := decide(state, event.Version(len(history)))
	if err != nil {
		return nil, err
	}

	return append(history, newEvents...), nil
}

// issueInvoiceForOrder is what billing's projection does when the
// replicated order.placed lands: fold it, then issue the invoice.
func issueInvoiceForOrder(
	orderEvents []event.Event,
) decider.DecideFunc[InvoiceState] {
	state, err := replay(orderEvents, initialInvoiceState(), foldInvoice)
	if err != nil {
		return failingDecide[InvoiceState](err)
	}

	return issueInvoice(IssueInvoiceCmd{OrderID: state.OrderID, AmountCents: state.AmountCents})
}

// completeOrderFromInvoice is orders' reaction to the replicated
// invoice.issued.
func completeOrderFromInvoice(
	orderID string,
	billingEvents []event.Event,
) decider.DecideFunc[OrderState] {
	invoice, err := replay(billingEvents, initialInvoiceState(), foldInvoice)
	if err != nil {
		return failingDecide[OrderState](err)
	}

	if invoice.IssuedRef == "" {
		return completeOrder(orderID)
	}

	return completeOrder(orderID)
}

func failingDecide[S any](err error) decider.DecideFunc[S] {
	return func(S, event.Version) ([]event.Event, error) { return nil, err }
}
