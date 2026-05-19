package openapi

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
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
				Schema: &catalog.Schema{
					Type: "object",
					Properties: map[string]catalog.Property{
						"name": {Type: "string", Description: "Item name"},
					},
					Required: []string{"name"},
				},
				Kind: catalog.CommandMessage,
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
				ID:      "item-created",
				Name:    "ItemCreated",
				Version: "1.0.0",
				Summary: "An item was created",
				Schema: &catalog.Schema{
					Type: "object",
					Properties: map[string]catalog.Property{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
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

	for _, tt := range tests {
		got := toKebab(tt.input)
		if got != tt.expected {
			t.Errorf("toKebab(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create-item", "CreateItem"},
		{"item_created", "ItemCreated"},
		{"item created", "ItemCreated"},
	}

	for _, tt := range tests {
		got := toPascal(tt.input)
		if got != tt.expected {
			t.Errorf("toPascal(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
