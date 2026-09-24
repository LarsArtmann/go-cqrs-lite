// The mesh-demo's CATALOG declarations — one registry per bounded context,
// exactly as each context would ship its own export. The bilateral-contract
// rule is visible in the code: orders declares order.placed (Sends) with an
// explicit consumer, billing declares the SAME event (Receives) with an
// explicit producer, and vice versa for invoice.issued. A hub union-merging
// both trees can take either copy of a shared event — both tell the whole
// relationship.
package main

import (
	"os"
	"path/filepath"

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
	reg.AddUser(catalog.User{
		ID: "marta", Name: "Marta Vega", Role: "Lead Engineer", Email: "marta@orders.example.com",
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
	reg.AddUser(catalog.User{
		ID: "juno", Name: "Juno Ray", Role: "Staff Engineer", Email: "juno@billing.example.com",
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

	if err := eventcatalog.NewExporter(outDir, options...).Export(cat); err != nil {
		return err
	}

	return writeDomainContracts(domain, outDir)
}

// domainContract locates a bounded context's data-product output contract
// file inside the export tree.
type domainContract struct {
	dataProduct string // owning data product (directory name)
	file        string // contract file name under contracts/
	content     string
}

var domainContracts = map[string]domainContract{
	"orders": {
		dataProduct: "order-lifecycle",
		file:        "order-placed.yaml",
		content:     orderPlacedContract,
	},
	"billing": {
		dataProduct: "billing-ledger",
		file:        "invoice-issued.yaml",
		content:     invoiceIssuedContract,
	},
}

// writeDomainContracts materializes the data-product output contract FILES.
// The exporter copies nothing — a declared contract path is a promise the
// declaring repo keeps by shipping the file itself (same rule as
// catalog/cmd/ec-fixture). The hub's per-source governance lint
// (refs/file-exists) fails the build when a declared path is missing.
func writeDomainContracts(domain, outDir string) error {
	contract, ok := domainContracts[domain]
	if !ok {
		return errUnknownDomain
	}

	dir := filepath.Join(outDir, "data-products", contract.dataProduct, "contracts")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, contract.file), []byte(contract.content), 0o600)
}

// orderPlacedContract is the bilateral order.placed contract billing codes
// against (mirrors OrderPlacedPayload).
const orderPlacedContract = `# Data contract: order.placed (bilateral, orders -> billing)
id: order-placed
owner: orders-team
type: event
version: 1.0.0
description: Emitted when an order is placed and awaits invoicing
fields:
  - name: orderId
    type: string
    required: true
  - name: customerId
    type: string
    required: true
  - name: totalCents
    type: integer
    required: true
`

// invoiceIssuedContract is the bilateral invoice.issued contract orders
// folds (mirrors InvoiceIssuedPayload).
const invoiceIssuedContract = `# Data contract: invoice.issued (bilateral, billing -> orders)
id: invoice-issued
owner: billing-team
type: event
version: 1.0.0
description: Emitted when billing issues an invoice for an order
fields:
  - name: invoiceRef
    type: string
    required: true
  - name: orderId
    type: string
    required: true
  - name: amountCents
    type: integer
    required: true
`
