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
	meta event.CatalogMeta, //nolint:staticcheck
) (*event.CatalogCore, error) { //nolint:staticcheck
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
		return nil, fmt.Errorf("new event catalog core for %s: %w", eventType, err)
	}

	return core, nil
}

const testVersion = "1.0.0"

func BuildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", testVersion)
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: testVersion, Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order", Version: testVersion,
		Summary: "Create a new order",
		Schema: &catalog.Schema{Type: "object", Properties: map[string]catalog.Property{
			"orderId": {Type: "string"},
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

	return reg.Build()
}
