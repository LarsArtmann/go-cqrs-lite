// The BILLING bounded context: consumes orders' order.placed event,
// issues invoices, publishes invoice.issued back. It is a fully separate
// context — different state, different team, different catalog declaration
// — wired to orders ONLY through the event contracts.
package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const (
	cmdIssueInvoice = command.Type("billing.issue_invoice")

	billingStreamType = "Invoice"
)

// InvoiceIssuedPayload is the CONTRACT of invoice.issued — orders codes
// against this shape.
type InvoiceIssuedPayload struct {
	InvoiceRef  string `json:"invoiceRef"`
	OrderID     string `json:"orderId"`
	AmountCents int64  `json:"amountCents"`
}

// OrderPlacedReceived mirrors orders' order.placed contract from the
// CONSUMING side.
type OrderPlacedReceived struct {
	OrderID    string `json:"orderId"`
	CustomerID string `json:"customerId"`
	TotalCents int64  `json:"totalCents"`
}

// InvoiceState is billing's private state.
type InvoiceState struct {
	OrderID     string
	AmountCents int64
	IssuedRef   string
}

func initialInvoiceState() InvoiceState { return InvoiceState{} }

// foldInvoice applies the consumed order.placed plus billing's own
// invoice.issued.
func foldInvoice(s InvoiceState, evt event.Event) (InvoiceState, error) {
	switch evt.Type() {
	case evtOrderPlaced:
		p, err := event.DecodePayloadAuto[OrderPlacedReceived](evt)
		if err != nil {
			return s, err
		}

		s.OrderID, s.AmountCents = p.OrderID, p.TotalCents
	case evtInvoiceIssued:
		p, err := event.DecodePayloadAuto[InvoiceIssuedPayload](evt)
		if err != nil {
			return s, err
		}

		s.IssuedRef = p.InvoiceRef
	}

	return s, nil
}

// IssueInvoiceCmd is billing's write-side API (here driven by the demo
// loop; in production the replicated order.placed event drives it).
type IssueInvoiceCmd struct {
	*command.BasicCommand

	OrderID     string
	AmountCents int64
}

// issueInvoice decides invoice.issued from the current state — idempotent:
// an already-issued invoice never re-issues.
func issueInvoice(cmd IssueInvoiceCmd) decider.DecideFunc[InvoiceState] {
	return func(s InvoiceState, v event.Version) ([]event.Event, error) {
		if s.IssuedRef != "" {
			return nil, nil
		}

		ref := "inv-" + cmd.OrderID

		evt, err := event.New(evtInvoiceIssued, id.DeriveStreamID("invoice", cmd.OrderID),
			billingStreamType, v.Increment(), InvoiceIssuedPayload{
				InvoiceRef:  ref,
				OrderID:     cmd.OrderID,
				AmountCents: cmd.AmountCents,
			})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}
