package cattest

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

func NewCatalogCore(
	tb testing.TB,
	eventType string,
	meta event.CatalogMeta,
) (*event.CatalogCore, error) {
	tb.Helper()

	core, err := event.NewCatalogCore(
		event.Type(eventType),
		id.NewAggregateID(),
		"Order",
		1,
		nil,
		meta,
	)
	if err != nil {
		return nil, fmt.Errorf("new event catalog core: %w", err)
	}

	return core, nil
}

func BuildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order", Version: "1.0.0",
		Summary: "Create a new order",
		Schema: &catalog.Schema{Type: "object", Properties: map[string]catalog.Property{
			"orderId": {Type: "string"},
		}},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created", Version: "1.0.0",
		Summary: "Order was created", Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order", Version: "1.0.0",
		Summary: "Get order by ID",
	})

	return reg.Build()
}
