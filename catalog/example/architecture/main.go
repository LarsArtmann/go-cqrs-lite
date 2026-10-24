// Package main demonstrates exporting a catalog as the self-contained
// architecture reference HTML file (ADR-0159) and as a redacted slice of
// it (ADR-0160 selections).
//
// Run:
//
//	go run ./example/architecture
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
)

type placeOrder struct {
	OrderID string `json:"orderId" doc:"Order identifier"`
	Amount  int    `json:"amount"  doc:"Total in minor units"`
}

type orderPlaced struct {
	OrderID string `json:"orderId" doc:"Order identifier"`
}

type getOrder struct {
	OrderID string `json:"orderId" doc:"Order identifier"`
}

func buildCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("Ordering", "1.0.0")

	builder.AddService("order-service", "Order Service", "1.0.0", "Takes and tracks orders.",
		catalog.Command[placeOrder]("create-order",
			catalog.WithSummary("Create an order."),
			catalog.MsgOperation("POST", "/orders"),
		),
		catalog.Event[orderPlaced]("order-placed", catalog.Sends,
			catalog.WithSummary("An order was placed."),
		),
		catalog.Query[getOrder]("get-order",
			catalog.WithSummary("Fetch one order."),
		),
	)

	builder.ConfigureService("order-service",
		catalog.ServiceOwners("ordering"),
		catalog.ServiceWritesTo("order-db"),
	)

	builder.AddDataStore(catalog.DataStore{
		ID:            "order-db",
		Name:          "Order Database",
		Version:       "1.0.0",
		Summary:       "System of record for orders.",
		Technology:    "postgres@16",
		Authoritative: true,
	})

	builder.AddTeam(catalog.Team{ID: "ordering", Name: "Ordering Team", Summary: "Owns ordering."})
	builder.AddDomain("ordering", "Ordering", "1.0.0", "Everything about orders.", "order-service")

	return builder.Build()
}

func main() {
	cat := buildCatalog()

	whole := architecture.NewExporter(
		architecture.WithTitle("Ordering"),
		architecture.WithDescription("The whole ordering domain."),
		architecture.WithCommand("go run ./example/architecture"),
	)

	if err := whole.ExportFile(cat, "architecture.html"); err != nil {
		fmt.Fprintf(os.Stderr, "architecture export: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("wrote architecture.html (the whole catalog)")

	// The same catalog as a redacted slice: keep one domain, hide the
	// ownership attribution, let out-of-scope references degrade to names.
	slice := architecture.NewExporter(append([]architecture.Option{
		architecture.WithTitle("Ordering — partner slice"),
		architecture.WithCommand("go run ./example/architecture"),
	}, (&architecture.ExportSpec{
		Title:       "Ordering — partner slice",
		Description: "What partners may see.",
		Include:     []architecture.Selector{{Kind: "domain", ID: "ordering"}},
		Hide:        []architecture.HideSection{architecture.HideOwners},
		References:  architecture.ReferencesNameOnly,
	}).Options()...)...)

	if err := slice.ExportFile(cat, "architecture-partner.html"); err != nil {
		fmt.Fprintf(os.Stderr, "slice export: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("wrote architecture-partner.html (a redacted slice)")
}
