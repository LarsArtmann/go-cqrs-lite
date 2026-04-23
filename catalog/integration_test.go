package catalog_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

type CreateOrderPayload struct {
	OrderID string `json:"orderId"`
	Product string `json:"product"`
	Amount  int    `json:"amount"`
}

func TestIntegration_FullCatalogFlow(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	reg := catalog.NewRegistry("E-Commerce API", "1.0.0")

	reg.AddService(catalog.Service{
		ID: "order-service", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})

	schema := catalog.SchemaFromType[CreateOrderPayload]()

	reg.AddCommand("order-service", catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        "CreateOrder",
		Name:      "CreateOrder",
		Version:   "1.0.0",
		Summary:   "Create a new order",
		Direction: catalog.Receives,
		Schema:    schema,
	})
	reg.AddEvent("order-service", catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        "OrderCreated",
		Name:      "OrderCreated",
		Version:   "1.0.0",
		Summary:   "Order was created",
		Direction: catalog.Sends,
		Schema:    schema,
	})
	reg.AddQuery("order-service", catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      "GetOrder",
		Name:    "GetOrder",
		Version: "1.0.0",
		Summary: "Get order by ID",
	})

	cattest.AddDomain(
		t,
		reg,
		"ordering",
		"Ordering",
		"1.0.0",
		"Order management domain",
		[]string{"order-service"},
	)

	cat := reg.Build()

	if cat.Title != "E-Commerce API" {
		t.Errorf("title = %q", cat.Title)
	}

	cattest.AssertSliceLen(t, "cat.Services", cat.Services, 1)

	svc := cat.Services[0]
	if svc.ID != "order-service" {
		t.Errorf("service ID = %q", svc.ID)
	}

	cattest.AssertSliceLen(t, "svc.Commands", svc.Commands, 1)
	cattest.AssertSliceLen(t, "svc.Events", svc.Events, 1)
	cattest.AssertSliceLen(t, "svc.Queries", svc.Queries, 1)

	asyncExp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := asyncExp.Export(cat)

	if doc.AsyncAPI != "3.0.0" {
		t.Errorf("asyncapi version = %q", doc.AsyncAPI)
	}

	cattest.AssertMapLen(t, "doc.Channels", doc.Channels, 3)

	cattest.AssertMapLen(t, "doc.Operations", doc.Operations, 3)

	cattest.AssertMapLen(t, "doc.Components.Messages", doc.Components.Messages, 3)

	yamlBytes, err := doc.MarshalYAML()
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Error("yaml output is empty")
	}

	jsonBytes, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	var jsonDoc map[string]any
	if err := json.Unmarshal(jsonBytes, &jsonDoc); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if jsonDoc["asyncapi"] != "3.0.0" {
		t.Errorf("json asyncapi = %v", jsonDoc["asyncapi"])
	}

	ecDir := filepath.Join(tmpDir, "eventcatalog")

	ecExp := eventcatalog.NewExporter(ecDir)
	if err := ecExp.Export(cat); err != nil {
		t.Fatalf("eventcatalog export: %v", err)
	}

	svcIndex := filepath.Join(ecDir, "services", "order-service", "index.mdx")

	content := cattest.MustReadFile(t, svcIndex)
	if !containsAll(content, "sends:", "- OrderCreated/1.0.0") {
		t.Errorf("service frontmatter missing sends: %s", content)
	}

	cmdIndex := filepath.Join(
		ecDir,
		"services",
		"order-service",
		"commands",
		"CreateOrder",
		"index.mdx",
	)

	cmdContent := cattest.MustReadFile(t, cmdIndex)
	if !containsAll(cmdContent, "schemaPath: schemas/schema.json") {
		t.Errorf("command missing schemaPath: %s", cmdContent)
	}

	schemaFile := filepath.Join(
		ecDir,
		"services",
		"order-service",
		"commands",
		"CreateOrder",
		"schemas",
		"schema.json",
	)

	schemaContent := cattest.MustReadFile(t, schemaFile)

	var schemaMap map[string]any
	if err := json.Unmarshal([]byte(schemaContent), &schemaMap); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("schema type = %v", schemaMap["type"])
	}

	domainIndex := filepath.Join(ecDir, "domains", "ordering", "index.mdx")
	cattest.AssertFileExists(t, domainIndex)

	cfgFile := filepath.Join(ecDir, "eventcatalog.config.js")
	cattest.AssertFileExists(t, cfgFile)
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !containsStr(s, sub) {
			return false
		}
	}

	return true
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s[:len(sub)] == sub || containsStr(s[1:], sub))
}
