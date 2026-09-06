// Command ec-fixture exports a fixed demo catalog with the EventCatalog
// exporter. It backs `nix run .#check-eventcatalog`: the render-validation
// flow (generate → npm install → eventcatalog build) needs a stable fixture
// that exercises every resource kind EventCatalog renders.
//
// Usage: ec-fixture <output-dir>
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ec-fixture <output-dir>")
		os.Exit(2)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "ec-fixture: %v\n", err)
		os.Exit(1)
	}
}

func run(outputDir string) error {
	reg := catalog.NewRegistry("Demo", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0",
		Summary: "Manages orders", WritesTo: []catalog.DataStoreID{"orders-db"},
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order",
		Version: "1.0.0", Summary: "Create a new order",
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"orderId": {Type: catalog.TypeString},
		}},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created",
		Version: "1.0.0", Summary: "Order was created", Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order",
		Version: "1.0.0", Summary: "Get order by ID",
	})
	reg.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: "1.0.0",
		Summary: "All order-related events", Protocols: []catalog.Protocol{"kafka"},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders Database", Version: "1.0.0",
		ContainerType: "database", Technology: "postgres@16",
	})

	return eventcatalog.NewExporter(outputDir).Export(reg.Build())
}
