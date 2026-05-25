package cattest

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func Build(tb testing.TB, r *catalog.Registry) *catalog.Catalog {
	tb.Helper()

	return r.Build()
}

func MustExport(
	tb testing.TB,
	exp interface {
		Export(cat *catalog.Catalog) error
	},
	cat *catalog.Catalog,
) {
	tb.Helper()

	err := exp.Export(cat)
	if err != nil {
		tb.Fatalf("export catalog: %v", err)
	}
}

const testVersion = "1.0.0"

func BuildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", testVersion)
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: testVersion, Summary: "Manages orders",
		WritesTo: []catalog.DataStoreID{"orders-db"},
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order", Version: testVersion,
		Summary: "Create a new order",
		Schema: &catalog.Schema{Type: catalog.TypeObject, Properties: map[string]catalog.Property{
			"orderId": {Type: catalog.TypeString},
		}},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created", Version: testVersion,
		Summary: "Order was created", Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order", Version: testVersion,
		Summary: "Get order by ID",
	})
	reg.AddChannel(catalog.Channel{
		ID: "order-events", Name: "Order Events", Version: testVersion,
		Summary: "All order-related events", Protocols: []string{"kafka"},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders Database", Version: testVersion,
		ContainerType: "database", Technology: "postgres@16",
	})

	return reg.Build()
}
