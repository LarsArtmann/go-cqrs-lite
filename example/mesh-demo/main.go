// mesh-demo: two bounded contexts (ordering + billing) wired together ONLY
// through bilateral event contracts, each shipping its own EventCatalog
// export for the federation hub.
//
// Usage:
//
//	mesh-demo demo                                     # run the cross-domain lifecycle in-process
//	mesh-demo export -domain orders -out work/out/orders [-plain] [-skip-bootstrap]
//	mesh-demo export -domain billing -out work/out/billing [-plain] [-skip-bootstrap]
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
)

var errUnknownDomain = errors.New("mesh-demo: unknown domain (want orders|billing)")

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error

	switch os.Args[1] {
	case "demo":
		err = runDemo()
	case "export":
		err = runExportCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		log.Fatalf("mesh-demo: %v", err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: mesh-demo demo | export -domain orders|billing -out DIR [-plain] [-skip-bootstrap]")
}

func runExportCmd(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)

	domain := fs.String("domain", "", "bounded context to export (orders|billing)")
	outDir := fs.String("out", "", "output directory for the EventCatalog tree")
	plain := fs.Bool("plain", false, "emit plain frontmatter-ID refs (governance/lint exports)")
	skipBootstrap := fs.Bool("skip-bootstrap", false, "omit eventcatalog.config.js + package.json (hub CI)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *domain == "" || *outDir == "" {
		return errors.New("mesh-demo export: -domain and -out are required")
	}

	return exportDomain(*domain, *outDir, exportOptions{plain: *plain, skipBootstrap: *skipBootstrap})
}

// runDemo walks the full cross-domain lifecycle with pure deciders — no
// infrastructure, just the decide/fold core the contexts share:
// place order → order.placed → billing folds it, issues invoice →
// invoice.issued → orders folds it, completes the order.
func runDemo() error {
	const orderID = "order-42"

	events, err := decideInto(nil, initialOrderState(), foldOrder,
		placeOrder(PlaceOrderCmd{OrderID: orderID, CustomerID: "cust-7", TotalCents: 9900}))
	if err != nil {
		return err
	}

	orderEvents := events

	// billing consumes order.placed out of orders' journal (replication —
	// the ADR-0146 mechanism) and issues the invoice.
	billingEvents, err := decideInto(nil, initialInvoiceState(), foldInvoice,
		issueInvoiceForOrder(orderEvents))
	if err != nil {
		return err
	}

	// orders consumes invoice.issued back (folded into its own history) and
	// completes.
	orderEvents = append(orderEvents, billingEvents...)
	orderEvents, err = decideInto(orderEvents, initialOrderState(), foldOrder,
		completeOrderFromInvoice(orderID, billingEvents))
	if err != nil {
		return err
	}

	finalOrder := initialOrderState()
	for _, evt := range orderEvents {
		if finalOrder, err = foldOrder(finalOrder, evt); err != nil {
			return err
		}
	}

	// billing's journal holds the consumed order.placed plus its own
	// invoice.issued — cross-context integration is a fold (ADR-0146).
	billingHistory := append(orderEvents[:len(orderEvents):len(orderEvents)], billingEvents...)
	finalInvoice := initialInvoiceState()
	for _, evt := range billingHistory {
		if finalInvoice, err = foldInvoice(finalInvoice, evt); err != nil {
			return err
		}
	}

	fmt.Printf("order:  placed=%v total=%d invoice=%q completed=%v\n",
		finalOrder.Placed, finalOrder.TotalCents, finalOrder.InvoiceRef, finalOrder.Completed)
	fmt.Printf("billing: order=%s amount=%d invoice=%q\n",
		finalInvoice.OrderID, finalInvoice.AmountCents, finalInvoice.IssuedRef)

	return nil
}
