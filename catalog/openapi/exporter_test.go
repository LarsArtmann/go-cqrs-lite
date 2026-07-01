package openapi

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/internal/cattest"
)

func TestExporter_Export(t *testing.T) {
	eventSchema, err := cattest.StringSchema("id", "name")
	if err != nil {
		t.Fatal(err)
	}

	reg := catalog.NewRegistry("Test API", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "test-svc",
		Name:    "Test Service",
		Version: "1.0.0",
		Commands: []catalog.Message{
			{
				ID:      "create-item",
				Name:    "CreateItem",
				Version: "1.0.0",
				Summary: "Creates a new item",
				Schema:  cattest.CreateItemSchema(),
				Kind:    catalog.CommandMessage,
			},
		},
		Queries: []catalog.Message{
			{
				ID:      "get-item",
				Name:    "GetItem",
				Version: "1.0.0",
				Summary: "Gets an item by ID",
				Schema: &catalog.Schema{
					Type: "object",
					Properties: map[string]catalog.Property{
						"id": {Type: "string", Description: "Item ID"},
					},
				},
				Kind: catalog.QueryMessage,
			},
		},
		Events: []catalog.Message{
			{
				ID:        "item-created",
				Name:      "ItemCreated",
				Version:   "1.0.0",
				Summary:   "An item was created",
				Schema:    eventSchema,
				Kind:      catalog.EventMessage,
				Direction: catalog.Sends,
			},
		},
	})

	cat := reg.Build()
	exp := NewExporter("Test API", "1.0.0", WithDescription("A test API"))
	doc := exp.Export(cat)

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("expected OpenAPI version 3.0.3, got %s", doc.OpenAPI)
	}

	if doc.Info.Title != "Test API" {
		t.Errorf("expected title 'Test API', got %s", doc.Info.Title)
	}

	if doc.Info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", doc.Info.Version)
	}

	if doc.Info.Description != "A test API" {
		t.Errorf("expected description 'A test API', got %s", doc.Info.Description)
	}

	if len(doc.Paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(doc.Paths))
	}

	if len(doc.Components.Schemas) != 3 {
		t.Errorf("expected 3 schemas, got %d", len(doc.Components.Schemas))
	}
}

func TestExporter_WithBasePath(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "svc",
		Name:    "Svc",
		Version: "1.0.0",
		Commands: []catalog.Message{
			{ID: "do-thing", Name: "DoThing", Version: "1.0.0", Kind: catalog.CommandMessage},
		},
	})

	cat := reg.Build()
	exp := NewExporter("Svc", "1.0.0", WithBasePath("/api/v2"))
	doc := exp.Export(cat)

	found := false
	for path := range doc.Paths {
		if path == "/api/v2/svc/do-thing" {
			found = true
		}
	}

	if !found {
		t.Errorf("expected path /api/v2/do-thing in %v", doc.Paths)
	}
}

func TestExporter_NilSchema(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Svc", "1.0.0")
	reg.AddService(catalog.Service{
		ID:      "svc",
		Name:    "Svc",
		Version: "1.0.0",
		Commands: []catalog.Message{
			{
				ID: "no-schema", Name: "NoSchema", Version: "1.0.0",
				Kind: catalog.CommandMessage, Schema: nil,
			},
		},
	})

	cat := reg.Build()
	doc := NewExporter("Svc", "1.0.0").Export(cat)

	if len(doc.Paths) != 1 {
		t.Fatalf("paths = %d, want 1", len(doc.Paths))
	}

	for _, pi := range doc.Paths {
		if pi.Post == nil {
			t.Fatal("expected POST operation")
		}

		body := pi.Post.RequestBody
		if body == nil || body.Content == nil {
			t.Fatal("expected request body with content")
		}

		jsonContent, ok := body.Content["application/json"]
		if !ok {
			t.Fatal("expected application/json content")
		}

		if jsonContent.Schema == nil {
			t.Fatal("expected schema in content")
		}
	}
}

func TestSchemaToAny_Nil(t *testing.T) {
	result := schemaToAny(nil)
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}

	if m["type"] != "object" {
		t.Errorf("type = %q, want object", m["type"])
	}
}

func TestExtractIDParameter_NilSchema(t *testing.T) {
	t.Parallel()

	path, params := extractIDParameter("/api/svc/get-item", nil)
	if path != "/api/svc/get-item" {
		t.Errorf("path = %q, want unchanged", path)
	}
	if len(params) != 0 {
		t.Errorf("params = %d, want 0", len(params))
	}
}

func TestExtractIDParameter_NilProperties(t *testing.T) {
	t.Parallel()

	path, params := extractIDParameter(
		"/api/svc/get-item",
		&catalog.Schema{Type: catalog.TypeObject},
	)
	if path != "/api/svc/get-item" {
		t.Errorf("path = %q, want unchanged", path)
	}
	if len(params) != 0 {
		t.Errorf("params = %d, want 0", len(params))
	}
}

func TestExtractIDParameter_NoIDField(t *testing.T) {
	t.Parallel()

	path, params := extractIDParameter("/api/svc/list-items", &catalog.Schema{
		Type: catalog.TypeObject,
		Properties: map[string]catalog.Property{
			"name": {Type: catalog.TypeString},
		},
	})
	if path != "/api/svc/list-items" {
		t.Errorf("path = %q, want unchanged", path)
	}
	if len(params) != 0 {
		t.Errorf("params = %d, want 0", len(params))
	}
}

func TestDocument_MarshalYAML(t *testing.T) {
	t.Parallel()

	doc := &Document{
		OpenAPI:    "3.0.3",
		Info:       catalog.DocumentInfo{Title: "Test", Version: "1.0.0"},
		Paths:      map[string]*PathItem{},
		Components: Components{Schemas: map[string]any{}},
	}

	data, err := doc.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty YAML")
	}
}

func TestExporter_EmptyCatalog(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Empty", "1.0.0")
	cat := reg.Build()
	doc := NewExporter("Empty", "1.0.0").Export(cat)

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("openapi = %q, want 3.0.3", doc.OpenAPI)
	}

	if len(doc.Paths) != 0 {
		t.Errorf("paths = %d, want 0", len(doc.Paths))
	}
}

func TestExporter_EntitySchemas(t *testing.T) {
	t.Parallel()

	reg := catalog.NewRegistry("Test API", "1.0.0")
	reg.AddEntity(catalog.Entity{
		ID:   "user-entity",
		Name: "User",
		Schema: &catalog.Schema{
			Type: catalog.TypeObject,
			Properties: map[string]catalog.Property{
				"id":    {Type: catalog.TypeString, Description: "User ID"},
				"email": {Type: catalog.TypeString, Description: "User email"},
			},
		},
	})
	reg.AddEntity(catalog.Entity{
		ID:   "order-entity",
		Name: "Order",
		Schema: &catalog.Schema{
			Type: catalog.TypeObject,
			Properties: map[string]catalog.Property{
				"id":     {Type: catalog.TypeString},
				"status": {Type: catalog.TypeString},
			},
		},
	})
	reg.AddEntity(catalog.Entity{
		ID:   "schemaless",
		Name: "Schemaless",
	})

	cat := reg.Build()
	doc := NewExporter("Test API", "1.0.0").Export(cat)

	if _, ok := doc.Components.Schemas["entity.user-entity"]; !ok {
		t.Errorf("expected entity.user-entity in schemas, got keys: %v", schemaKeys(doc))
	}

	if _, ok := doc.Components.Schemas["entity.order-entity"]; !ok {
		t.Errorf("expected entity.order-entity in schemas, got keys: %v", schemaKeys(doc))
	}

	if _, ok := doc.Components.Schemas["entity.schemaless"]; ok {
		t.Error("schemaless entity should not appear in components")
	}
}

func schemaKeys(doc *Document) []string {
	keys := make([]string, 0, len(doc.Components.Schemas))
	for k := range doc.Components.Schemas {
		keys = append(keys, k)
	}
	return keys
}
