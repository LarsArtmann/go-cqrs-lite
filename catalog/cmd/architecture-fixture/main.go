// Command architecture-fixture renders a small but complete fixture catalog
// as a self-contained architecture reference. It backs
// scripts/check-architecture-reference.sh, which asserts the artifact's
// properties (self-containment, anchor integrity, section counts) outside
// the go test process — the same fixture/validate split the ec-fixture
// command uses for the EventCatalog gate.
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

// fixtureVersion is the service/domain version the fixture documents.
const fixtureVersion = "1.0.0"

type createOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"` //nolint:tagliatelle // mirrors the external snake_case API this fixture documents
	Amount  int    `json:"amount"   doc:"Total in minor units"`
}

type orderCreated struct {
	OrderID string `json:"order_id" doc:"Order identifier"` //nolint:tagliatelle // mirrors the external snake_case API this fixture documents
	Amount  int    `json:"amount"   doc:"Total in minor units"`
}

type getOrder struct {
	OrderID string `json:"order_id" doc:"Order identifier"`
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
		catalog.ServiceEntities("order"),
	)

	builder.AddDomain("ordering", "Ordering", fixtureVersion, "Everything about orders.", "order-service")

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
	if len(os.Args) != 2 {
		fail("usage: architecture-fixture <output.html>")
	}

	exp := architecture.NewExporter(
		architecture.WithTitle("Acme"),
		architecture.WithDescription("Acme's architecture."),
		architecture.WithCommand("catalog-export -architecture architecture.html"),
	)

	if err := exp.ExportFile(fixtureCatalog(), os.Args[1]); err != nil {
		fail("render: %v", err)
	}
}

func fail(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "architecture-fixture: "+format+"\n", args...)
	os.Exit(1)
}
