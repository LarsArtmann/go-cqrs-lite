// Package main is the ORDERS bounded context of the mesh-demo example.
//
// A context owns its state, its commands, and its events — and publishes a
// contract for the events other contexts may consume. It also consumes
// another context's event (billing's invoice.issued) to complete its
// lifecycle: bilateral contracts flow BOTH ways.
package main

import (
	"errors"

	cbid "github.com/larsartmann/go-branded-id"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const (
	evtOrderPlaced    = event.Type("order.placed")
	evtOrderCompleted = event.Type("order.completed")
	evtInvoiceIssued  = event.Type("invoice.issued") // owned by billing

	cmdPlaceOrder = command.Type("order.place")

	ordersStreamType = "Order"
)

// OrderID is the caller-chosen order number — string-backed branded, the
// id.StreamID pattern (AGENTS.md 21(d)): ULID backing is for system-minted
// IDs only, so a caller-chosen key stays string-backed — the same number
// always derives the same stream, keeping retries idempotent.
type OrderID = cbid.ID[orderIDMarker, string]

// CustomerID brands the customer reference alongside OrderID.
type CustomerID = cbid.ID[customerIDMarker, string]

type (
	orderIDMarker    struct{}
	customerIDMarker struct{}
)

// NewOrderID brands a raw wire order number — contexts exchange plain
// strings in the bilateral payloads and brand at their boundaries.
func NewOrderID(raw string) OrderID { return cbid.NewID[orderIDMarker](raw) }

// NewCustomerID brands a raw wire customer reference.
func NewCustomerID(raw string) CustomerID { return cbid.NewID[customerIDMarker](raw) }

// orderStreamID derives a deterministic stream ID from the order number —
// same caller-chosen key ⇒ same stream (the id.StreamID pattern).
func orderStreamID(orderID OrderID) id.StreamID {
	return id.DeriveStreamID("order", orderID.Get())
}

// ErrEmptyOrder rejects a place-order command with no items.
var ErrEmptyOrder = errors.New("order: cart is empty")

// OrderPlacedPayload is the CONTRACT of the order.placed event — billing
// codes against this shape; breaking it is a version bump, not an edit.
// Fields stay plain strings: the payload is the wire form (JSON), and
// contexts brand at their own boundaries.
//branching-flow:ignore strong-id
type OrderPlacedPayload struct {
	OrderID    string `json:"orderId"`
	CustomerID string `json:"customerId"`
	TotalCents int64  `json:"totalCents"`
}

// OrderCompletedPayload is the contract of order.completed.
//branching-flow:ignore strong-id
type OrderCompletedPayload struct {
	OrderID    string `json:"orderId"`
	InvoiceRef string `json:"invoiceRef"`
}

// InvoiceReceivedPayload mirrors billing's invoice.issued contract from the
// CONSUMING side (orders' copy of the bilateral contract).
//branching-flow:ignore strong-id
type InvoiceReceivedPayload struct {
	InvoiceRef string `json:"invoiceRef"`
	OrderID    string `json:"orderId"`
}

// OrderState is the orders context's private state — billing never sees it.
type OrderState struct {
	Placed     bool
	TotalCents int64
	InvoiceRef string
	Completed  bool
}

func initialOrderState() OrderState { return OrderState{} }

// foldOrder applies the context's own events plus the consumed foreign
// event (invoice.issued) — cross-context integration is a fold, not a
// synchronous call (ADR-0146: replication, not federation).
func foldOrder(s OrderState, evt event.Event) (OrderState, error) {
	switch evt.Type() {
	case evtOrderPlaced:
		p, err := event.DecodePayloadAuto[OrderPlacedPayload](evt)
		if err != nil {
			return s, err
		}

		s.Placed, s.TotalCents = true, p.TotalCents
	case evtInvoiceIssued:
		p, err := event.DecodePayloadAuto[InvoiceReceivedPayload](evt)
		if err != nil {
			return s, err
		}

		s.InvoiceRef = p.InvoiceRef
	case evtOrderCompleted:
		s.Completed = true
	}

	return s, nil
}

// PlaceOrderCmd is the orders context's write-side API.
type PlaceOrderCmd struct {
	*command.BasicCommand

	OrderID    OrderID
	CustomerID CustomerID
	TotalCents int64
}

// placeOrder decides order.placed from the current state.
func placeOrder(cmd PlaceOrderCmd) decider.DecideFunc[OrderState] {
	return func(s OrderState, v event.Version) ([]event.Event, error) {
		if cmd.TotalCents <= 0 {
			return nil, ErrEmptyOrder
		}

		evt, err := event.New(evtOrderPlaced, orderStreamID(cmd.OrderID), ordersStreamType,
			v.Increment(), OrderPlacedPayload{
				OrderID:    cmd.OrderID.Get(),
				CustomerID: cmd.CustomerID.Get(),
				TotalCents: cmd.TotalCents,
			})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// completeOrder decides order.completed once the (replicated) invoice has
// arrived — the second half of the bilateral contract.
func completeOrder(orderID OrderID) decider.DecideFunc[OrderState] {
	return func(s OrderState, v event.Version) ([]event.Event, error) {
		if !s.Placed || s.InvoiceRef == "" || s.Completed {
			return nil, nil
		}

		evt, err := event.New(evtOrderCompleted, orderStreamID(orderID), ordersStreamType,
			v.Increment(), OrderCompletedPayload{OrderID: orderID.Get(), InvoiceRef: s.InvoiceRef})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}
