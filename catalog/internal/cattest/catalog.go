package cattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

const goldenFilePerm = 0o644

func Build(tb testing.TB, r *catalog.Registry) *catalog.Catalog {
	tb.Helper()

	return r.Build()
}

const testVersion = "1.0.0"

const testCreateOrderMsgID catalog.MessageID = "CreateOrder"

func BuildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", testVersion)
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: testVersion, Summary: "Manages orders",
		WritesTo: []catalog.DataStoreID{"orders-db"},
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      testCreateOrderMsgID,
		Name:    "Create Order",
		Version: testVersion,
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
		Summary: "All order-related events", Protocols: []catalog.Protocol{"kafka"},
	})
	reg.AddDataStore(catalog.DataStore{
		ID: "orders-db", Name: "Orders Database", Version: testVersion,
		ContainerType: "database", Technology: "postgres@16",
	})

	return reg.Build()
}

// BuildTestCatalogWithOps returns a catalog where messages carry explicit
// Operation and ResponseSpec data, exercising the REST export paths that
// BuildTestCatalog (which has no operations) does not cover.
func BuildTestCatalogWithOps() *catalog.Catalog {
	reg := catalog.NewRegistry("REST API", testVersion)
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: testVersion, Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      "CreateOrder",
		Name:    "Create Order",
		Version: testVersion,
		Summary: "Create a new order",
		Operation: &catalog.Operation{
			Method:      "POST",
			Path:        "/api/orders",
			StatusCodes: []string{"201", "400"},
		},
		Responses: []catalog.ResponseSpec{
			{
				StatusCode:  "201",
				Description: "Order created",
				Schema:      &catalog.Schema{Type: catalog.TypeObject},
			},
		},
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order",
		Version: testVersion, Summary: "Get order by ID",
		Operation: &catalog.Operation{Method: "GET", Path: "/api/orders/{id}"},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created",
		Version: testVersion, Summary: "Order was created", Direction: catalog.Sends,
		Operation: &catalog.Operation{Method: "POST", Path: "/api/orders/events"},
	})

	return reg.Build()
}

func AssertGolden(t *testing.T, goldenPath string, got []byte, update bool, mismatchMsg string) {
	t.Helper()

	if update {
		err := os.WriteFile(
			goldenPath,
			append(got, '\n'),
			goldenFilePerm,
		)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Error(mismatchMsg)
	}
}

func GoldenDir() string {
	return filepath.Join("..", "testdata", "golden")
}
