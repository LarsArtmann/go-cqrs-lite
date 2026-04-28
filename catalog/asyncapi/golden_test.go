package asyncapi_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
)

//nolint:gochecknoglobals // golden test pattern requires package-level flag
var update = flag.Bool("update", false, "update golden files")

func goldenDir() string {
	return filepath.Join("..", "testdata", "golden")
}

func buildTestCatalog() *catalog.Catalog {
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

func TestGolden_AsyncAPIJSON(t *testing.T) {
	cat := buildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "asyncapi.json")

	if *update {
		err := os.WriteFile(goldenPath, got, 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("AsyncAPI JSON mismatch (run with -update to refresh golden files)")
	}
}

func TestGolden_AsyncAPIYAML(t *testing.T) {
	cat := buildTestCatalog()
	exp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := exp.Export(cat)

	got, err := doc.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(), "asyncapi.yaml")

	if *update {
		err := os.WriteFile(goldenPath, got, 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("AsyncAPI YAML mismatch (run with -update to refresh golden files)")
	}
}
