// The mesh-demo's CATALOG declarations — one registry per bounded context,
// exactly as each context would ship its own export. The bilateral-contract
// rule is visible in the code: orders declares order.placed (Sends) with an
// explicit consumer, billing declares the SAME event (Receives) with an
// explicit producer, and vice versa for invoice.issued. A hub union-merging
// both trees can take either copy of a shared event — both tell the whole
// relationship.
package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

const (
	ordersCatalogVersion  = "1.0.0"
	billingCatalogVersion = "1.0.0"

	ordersTeamID  = "orders-team"
	billingTeamID = "billing-team"

	evtOrderPlacedID   = catalog.MessageID("order.placed")
	evtInvoiceIssuedID = catalog.MessageID("invoice.issued")
)

// buildOrdersCatalog declares the orders context: its service, its two
// outbound events (order.placed, order.completed), the consumed
// invoice.issued, and the order-lifecycle DATA PRODUCT whose output port is
// the order.placed contract.
func buildOrdersCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("Orders", ordersCatalogVersion)

	reg.AddService(catalog.Service{
		ID: "orders-svc", Name: "Orders Service", Version: ordersCatalogVersion,
		Summary: "Owns the order lifecycle",
		Owners:  []string{ordersTeamID},
	})
	reg.AddEvent("orders-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: evtOrderPlacedID, Name: "Order Placed",
		Version: ordersCatalogVersion, Summary: "An order was placed and is awaiting invoicing",
		Direction: catalog.Sends,
		// Bilateral contract, producing side: name the consumer.
		Consumers: []catalog.ServiceID{"billing-svc"},
		Schema:    catalog.SchemaFromType[OrderPlacedPayload](),
		Owners:    []string{ordersTeamID},
	})
	reg.AddEvent("orders-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: catalog.MessageID("order.completed"),
		Name: "Order Completed", Version: ordersCatalogVersion,
		Summary:   "The order's invoice was settled; lifecycle closed",
		Direction: catalog.Sends,
		Schema:    catalog.SchemaFromType[OrderCompletedPayload](),
		Owners:    []string{ordersTeamID},
	})
	reg.AddEvent("orders-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: evtInvoiceIssuedID, Name: "Invoice Issued",
		Version: ordersCatalogVersion, Summary: "Billing issued an invoice (consumed)",
		Direction: catalog.Receives,
		// Bilateral contract, consuming side: the EXPLICIT external producer
		// keeps ValidateCoeffects happy — the event is imported, not dangled.
		Producers: []catalog.ServiceID{"billing-svc"},
		Schema:    catalog.SchemaFromType[InvoiceReceivedPayload](),
		Owners:    []string{ordersTeamID},
	})

	reg.AddDomain(catalog.Domain{
		ID: "ordering", Name: "Ordering", Version: ordersCatalogVersion,
		Summary: "The ordering bounded context", Services: []catalog.ServiceID{"orders-svc"},
		Owners: []string{ordersTeamID},
	})

	reg.AddDataProduct(catalog.DataProduct{
		ID: "order-lifecycle", Name: "Order Lifecycle", Version: ordersCatalogVersion,
		Summary: "Order placement + completion facts for downstream analytics",
		Owners:  []string{ordersTeamID},
		Inputs:  []catalog.Ref{{ID: evtInvoiceIssuedID, Version: ordersCatalogVersion}},
		Outputs: []catalog.DataProductOutput{
			{
				Ref: catalog.Ref{ID: evtOrderPlacedID, Version: ordersCatalogVersion},
				Contract: &catalog.DataContract{
					Path: "contracts/order-placed.yaml",
					Name: "order-placed",
				},
			},
		},
	})

	reg.AddTeam(catalog.Team{
		ID: ordersTeamID, Name: "Orders Team", Summary: "Owns ordering", Members: []string{"marta"},
	})

	return reg.Build()
}

// buildBillingCatalog declares the billing context: consumes order.placed,
// produces invoice.issued, serves the billing-ledger data product.
func buildBillingCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("Billing", billingCatalogVersion)

	reg.AddService(catalog.Service{
		ID: "billing-svc", Name: "Billing Service", Version: billingCatalogVersion,
		Summary: "Owns invoicing",
		Owners:  []string{billingTeamID},
	})
	reg.AddEvent("billing-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: evtOrderPlacedID, Name: "Order Placed",
		Version: billingCatalogVersion, Summary: "An order was placed (consumed from ordering)",
		Direction: catalog.Receives,
		// Bilateral contract, consuming side: name the external producer.
		Producers: []catalog.ServiceID{"orders-svc"},
		Schema:    catalog.SchemaFromType[OrderPlacedReceived](),
		Owners:    []string{billingTeamID},
	})
	reg.AddEvent("billing-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: evtInvoiceIssuedID, Name: "Invoice Issued",
		Version: billingCatalogVersion, Summary: "An invoice was issued for an order",
		Direction: catalog.Sends,
		// Bilateral contract, producing side.
		Consumers: []catalog.ServiceID{"orders-svc"},
		Schema:    catalog.SchemaFromType[InvoiceIssuedPayload](),
		Owners:    []string{billingTeamID},
	})

	reg.AddDomain(catalog.Domain{
		ID: "invoicing", Name: "Invoicing", Version: billingCatalogVersion,
		Summary: "The billing bounded context", Services: []catalog.ServiceID{"billing-svc"},
		Owners: []string{billingTeamID},
	})

	reg.AddDataProduct(catalog.DataProduct{
		ID: "billing-ledger", Name: "Billing Ledger", Version: billingCatalogVersion,
		Summary: "Issued-invoice facts for finance consumers",
		Owners:  []string{billingTeamID},
		Inputs:  []catalog.Ref{{ID: evtOrderPlacedID, Version: ordersCatalogVersion}},
		Outputs: []catalog.DataProductOutput{
			{
				Ref: catalog.Ref{ID: evtInvoiceIssuedID, Version: billingCatalogVersion},
				Contract: &catalog.DataContract{
					Path: "contracts/invoice-issued.yaml",
					Name: "invoice-issued",
				},
			},
		},
	})

	reg.AddTeam(catalog.Team{
		ID: billingTeamID, Name: "Billing Team", Summary: "Owns invoicing",
		Members: []string{"juno"},
	})

	return reg.Build()
}

// exportOptions carries the headless-export flags shared by both domains.
type exportOptions struct {
	plain         bool
	skipBootstrap bool
}

// exportDomain runs the headless export for one bounded context — the
// command shape the federation hub's sources.json invokes per source.
func exportDomain(domain, outDir string, opts exportOptions) error {
	var cat *catalog.Catalog

	switch domain {
	case "orders":
		cat = buildOrdersCatalog()
	case "billing":
		cat = buildBillingCatalog()
	default:
		return errUnknownDomain
	}

	options := []eventcatalog.Option{}
	if opts.plain {
		options = append(options, eventcatalog.WithPlainRefIDs())
	}

	if opts.skipBootstrap {
		options = append(options, eventcatalog.WithSkipBootstrapFiles())
	}

	return eventcatalog.NewExporter(outDir, options...).Export(cat)
}
