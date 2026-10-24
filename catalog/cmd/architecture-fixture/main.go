// Command architecture-fixture renders a small but complete fixture catalog
// as a self-contained architecture reference. It backs
// scripts/check-architecture-reference.sh, which asserts the artifact's
// properties (self-containment, anchor integrity, section counts) outside
// the go test process — the same fixture/validate split the ec-fixture
// command uses for the EventCatalog gate.
//
// With a second path argument it also renders a REDACTED partner slice of
// the same catalog (ADR-0160): one domain in, the hostile-named store
// excluded, out-of-scope references degraded to names — so the gate can
// assert the slice carries no trace of the excluded resource.
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// fixtureVersion is the service/domain version the fixture documents.
const fixtureVersion = "1.0.0"

// HostileStoreName is the excluded partner-slice store's name: a URL that
// reads like a secret, so a leak is unambiguous (the leak-fixture pattern
// from the package tests).
const HostileStoreName = "https://internal.example.com/secret/quarterly-revenue"

type createOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"` //nolint:tagliatelle // mirrors the external snake_case API this fixture documents
	Amount  int    `json:"amount"   doc:"Total in minor units"`
}

type orderCreated struct {
	OrderID string `json:"order_id" doc:"Order identifier"` //nolint:tagliatelle // mirrors the external snake_case API this fixture documents
	Amount  int    `json:"amount"   doc:"Total in minor units"`
}

type getOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"` //nolint:tagliatelle // mirrors the external snake_case API this fixture documents
}

func fixtureCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("Acme", fixtureVersion)

	builder.AddService("order-service", "Order Service", fixtureVersion, "Takes and tracks orders.",
		catalog.Command[createOrder]("create-order",
			catalog.WithSummary("Create an order."),
			catalog.MsgOperation("POST", "/orders"),
		),
		catalog.Event[orderCreated]("order-created", catalog.Sends,
			catalog.WithSummary("An order was created."),
		),
		catalog.Query[getOrder]("get-order",
			catalog.WithSummary("Fetch one order."),
		),
	)

	builder.ConfigureService("order-service",
		catalog.ServiceOwners("ordering"),
		catalog.ServiceWritesTo("order-db"),
		catalog.ServiceReadsFrom("secret-store"),
		catalog.ServiceEntities("order"),
	)

	builder.AddDomain(
		"ordering",
		"Ordering",
		fixtureVersion,
		"Everything about orders.",
		"order-service",
	)

	builder.AddChannel(catalog.Channel{
		ID:      "orders",
		Name:    "Orders",
		Version: fixtureVersion,
		Summary: "Order lifecycle events.",
	})
	builder.ConfigureChannel("orders", catalog.ChannelMessages("order-created"))

	builder.AddDataStore(catalog.DataStore{
		ID:            "order-db",
		Name:          "Order Database",
		Version:       fixtureVersion,
		Summary:       "System of record for orders.",
		Technology:    "postgres@16",
		Authoritative: true,
	})

	// The hostile-named store: in the whole reference it is a normal
	// section; the partner slice excludes it, and the gate asserts its name
	// appears nowhere in the slice.
	builder.AddDataStore(catalog.DataStore{
		ID:         "secret-store",
		Name:       catalog.Name(HostileStoreName),
		Version:    fixtureVersion,
		Summary:    "Quarterly revenue, unreleased.",
		Technology: "postgres@16",
	})

	builder.AddFlow(catalog.Flow{
		ID:      "place-order",
		Name:    "Place an order",
		Version: fixtureVersion,
		Steps: []catalog.FlowStep{
			{
				ID:      "create",
				Title:   "Create the order",
				Service: &catalog.Ref{ID: "order-service"},
				Message: &catalog.Ref{ID: "create-order"},
			},
		},
	})

	builder.AddEntity(catalog.Entity{
		ID:            "order",
		Name:          "Order",
		Version:       fixtureVersion,
		Summary:       "A customer order.",
		AggregateRoot: true,
	})

	builder.AddADR(catalog.ADR{
		ID:      "0001-events",
		Name:    "Model orders with events",
		Version: fixtureVersion,
		Status:  catalog.ADRAccepted,
		Date:    "2026-01-02",
		Summary: "Events are the order history.",
		AppliesTo: []catalog.ADRResourceRef{
			{Type: catalog.ADRAppliesToService, ID: "order-service"},
		},
	})

	builder.AddTeam(catalog.Team{ID: "ordering", Name: "Ordering Team", Summary: "Owns ordering."})
	builder.AddCustomDoc(catalog.CustomDoc{
		ID:      "getting-started",
		Title:   "Getting started",
		Summary: "How to read this reference.",
		Content: "Start with the [Order Service](#service-order-service).\n",
	})

	return builder.Build()
}

func main() {
	// argWhole is the minimum argv: the command plus the whole-reference
	// path; argSlice adds the partner-slice path.
	const argWhole, argSlice = 2, 3

	if len(os.Args) < argWhole || len(os.Args) > argSlice {
		failf("usage: architecture-fixture <whole.html> [slice.html]")
	}

	exp := architecture.NewExporter(
		architecture.WithTitle("Acme"),
		architecture.WithDescription("Acme's architecture."),
		architecture.WithCommand("catalog-export -architecture architecture.html"),
	)

	if err := exp.ExportFile(fixtureCatalog(), os.Args[1]); err != nil {
		failf("render: %v", err)
	}

	if len(os.Args) == argSlice {
		slice := &architecture.ExportSpec{
			Title:       "Acme — partner slice",
			Description: "What partners may see.",
			Include:     []architecture.Selector{{Kind: "domain", ID: "ordering"}},
			Exclude:     []architecture.Selector{{Kind: "container", ID: "secret-store"}},
			References:  architecture.ReferencesNameOnly,
		}

		sliced := architecture.NewExporter(append([]architecture.Option{
			architecture.WithCommand("catalog-export -spec partner.mdx"),
		}, slice.Options()...)...)

		if err := sliced.ExportFile(fixtureCatalog(), os.Args[2]); err != nil {
			failf("render slice: %v", err)
		}
	}
}

func failf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "architecture-fixture: "+format+"\n", args...)

	os.Exit(1)
}
