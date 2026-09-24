package main

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// foldTests exercise both contexts' decide/fold cores — the domain logic a
// consumer ships — with zero infrastructure.
func TestOrders_PlaceOrderEmitsPlacedContract(t *testing.T) {
	events, err := decideInto(nil, initialOrderState(), foldOrder,
		placeOrder(PlaceOrderCmd{OrderID: "o-1", CustomerID: "c-9", TotalCents: 4200}))
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	if len(events) != 1 || events[0].Type() != evtOrderPlaced {
		t.Fatalf("want one order.placed event, got %v", events)
	}

	state, err := replay(events, initialOrderState(), foldOrder)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if !state.Placed || state.TotalCents != 4200 {
		t.Errorf("unexpected state after fold: %+v", state)
	}
}

func TestOrders_RejectsEmptyCart(t *testing.T) {
	if _, err := decideInto(nil, initialOrderState(), foldOrder,
		placeOrder(PlaceOrderCmd{OrderID: "o-2", TotalCents: 0})); !errors.Is(err, ErrEmptyOrder) {
		t.Fatalf("want ErrEmptyOrder, got %v", err)
	}
}

func TestBilling_FoldsForeignOrderPlacedAndIssuesInvoice(t *testing.T) {
	orderEvents, err := decideInto(nil, initialOrderState(), foldOrder,
		placeOrder(PlaceOrderCmd{OrderID: "o-3", CustomerID: "c-1", TotalCents: 1500}))
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	billingEvents, err := decideInto(nil, initialInvoiceState(), foldInvoice,
		issueInvoiceForOrder(orderEvents))
	if err != nil {
		t.Fatalf("issue invoice: %v", err)
	}

	if len(billingEvents) != 1 || billingEvents[0].Type() != evtInvoiceIssued {
		t.Fatalf("want one invoice.issued event, got %v", billingEvents)
	}

	// billing's journal holds the consumed order.placed plus its own
	// invoice.issued — cross-context integration is a fold (ADR-0146).
	state, err := replay(append(orderEvents, billingEvents...), initialInvoiceState(), foldInvoice)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if state.OrderID != "o-3" || state.AmountCents != 1500 || state.IssuedRef == "" {
		t.Errorf("unexpected invoice state: %+v", state)
	}
}

func TestBilling_InvoiceIssueIsIdempotent(t *testing.T) {
	first, err := decideInto(nil, initialInvoiceState(), foldInvoice,
		issueInvoice(IssueInvoiceCmd{OrderID: "o-4", AmountCents: 777}))
	if err != nil {
		t.Fatalf("first issue: %v", err)
	}

	second, err := decideInto(first, initialInvoiceState(), foldInvoice,
		issueInvoice(IssueInvoiceCmd{OrderID: "o-4", AmountCents: 777}))
	if err != nil {
		t.Fatalf("second issue: %v", err)
	}

	if len(second) != len(first) {
		t.Errorf("re-issuing an issued invoice must decide no events: %v", second)
	}
}

func TestLifecycle_CrossDomainRoundTripCompletesOrder(t *testing.T) {
	const orderID = "o-5"

	orderEvents, err := decideInto(nil, initialOrderState(), foldOrder,
		placeOrder(PlaceOrderCmd{OrderID: orderID, CustomerID: "c-2", TotalCents: 9900}))
	if err != nil {
		t.Fatalf("place: %v", err)
	}

	billingEvents, err := decideInto(nil, initialInvoiceState(), foldInvoice,
		issueInvoiceForOrder(orderEvents))
	if err != nil {
		t.Fatalf("invoice: %v", err)
	}

	// orders consumes the invoice by folding it through its own fold.
	for _, evt := range billingEvents {
		if orderEvents, err = appendForeign(orderEvents, evt, initialOrderState(), foldOrder); err != nil {
			t.Fatalf("consume invoice: %v", err)
		}
	}

	orderEvents, err = decideInto(orderEvents, initialOrderState(), foldOrder,
		completeOrderFromInvoice(orderID, billingEvents))
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	state, err := replay(orderEvents, initialOrderState(), foldOrder)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if !state.Completed || state.InvoiceRef == "" {
		t.Errorf("order should be completed with an invoice reference: %+v", state)
	}
}

func appendForeign[S any](
	history []event.Event,
	evt event.Event,
	initial S,
	fold func(S, event.Event) (S, error),
) ([]event.Event, error) {
	if _, err := replay(append(history, evt), initial, fold); err != nil {
		return nil, err
	}

	return append(history, evt), nil
}
