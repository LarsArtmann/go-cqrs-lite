package catalog_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
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

	reg.AddDomain(catalog.Domain{
		ID: "ordering", Name: "Ordering", Version: "1.0.0", Summary: "Order management domain",
		Services: []string{"order-service"},
	})

	cat := reg.Build()

	if cat.Title != "E-Commerce API" {
		t.Errorf("title = %q", cat.Title)
	}
	if len(cat.Services) != 1 {
		t.Fatalf("services = %d, want 1", len(cat.Services))
	}
	svc := cat.Services[0]
	if svc.ID != "order-service" {
		t.Errorf("service ID = %q", svc.ID)
	}
	if len(svc.Commands) != 1 || len(svc.Events) != 1 || len(svc.Queries) != 1 {
		t.Errorf(
			"commands=%d events=%d queries=%d",
			len(svc.Commands),
			len(svc.Events),
			len(svc.Queries),
		)
	}

	asyncExp := asyncapi.NewExporter("E-Commerce API", "1.0.0")
	doc := asyncExp.Export(cat)

	if doc.AsyncAPI != "3.0.0" {
		t.Errorf("asyncapi version = %q", doc.AsyncAPI)
	}
	if len(doc.Channels) != 3 {
		t.Errorf("channels = %d, want 3", len(doc.Channels))
	}
	if len(doc.Operations) != 3 {
		t.Errorf("operations = %d, want 3", len(doc.Operations))
	}
	if len(doc.Components.Messages) != 3 {
		t.Errorf("messages = %d, want 3", len(doc.Components.Messages))
	}

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
	data, err := os.ReadFile(svcIndex)
	if err != nil {
		t.Fatalf("read service index: %v", err)
	}
	content := string(data)
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
	data, err = os.ReadFile(cmdIndex)
	if err != nil {
		t.Fatalf("read command index: %v", err)
	}
	if !containsAll(string(data), "schemaPath: schemas/schema.json") {
		t.Errorf("command missing schemaPath: %s", string(data))
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
	data, err = os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schemaMap map[string]any
	if err := json.Unmarshal(data, &schemaMap); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if schemaMap["type"] != "object" {
		t.Errorf("schema type = %v", schemaMap["type"])
	}

	domainIndex := filepath.Join(ecDir, "domains", "ordering", "index.mdx")
	if _, err := os.Stat(domainIndex); os.IsNotExist(err) {
		t.Error("domain index not created")
	}

	cfgFile := filepath.Join(ecDir, "eventcatalog.config.js")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		t.Error("eventcatalog.config.js not created")
	}
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
