package openapi

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/cattest"
)

func TestExporter_Export(t *testing.T) {
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
				Schema:    cattest.StringSchema("id", "name"),
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

func TestToKebab(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CreateItem", "create-item"},
		{"GetItemByID", "get-item-by-id"},
		{"HTTPRequest", "http-request"},
		{"item_created", "item-created"},
		{"item created", "item-created"},
	}
	runToKebabTests(t, tests)
}

func TestToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create-item", "CreateItem"},
		{"item_created", "ItemCreated"},
		{"item created", "ItemCreated"},
		{"", ""},
	}

	for _, tt := range tests {
		got := toPascal(tt.input)
		if got != tt.expected {
			t.Errorf("toPascal(%q) = %q, want %q", tt.input, got, tt.expected)
		}
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

func TestToKebab_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"item2name", "item-2name"},
		{"HTTP", "http"},
		{"JSONParser", "json-parser"},
		{"already-lower", "already-lower"},
		{"lowercase", "lowercase"},
		{"ABC123", "abc-123"},
		{"v2User", "v-2user"},
	}
	runToKebabTests(t, tests)
}

func runToKebabTests(t *testing.T, tests []struct {
	input    string
	expected string
},
) {
	t.Helper()
	for _, tt := range tests {
		got := toKebab(tt.input)
		if got != tt.expected {
			t.Errorf("toKebab(%q) = %q, want %q", tt.input, got, tt.expected)
		}
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
