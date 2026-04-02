package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

type CreateUser struct {
	Email string `json:"email" doc:"User email address"`
	Name  string `json:"name"`
	Age   int    `json:"age,omitempty"`
}

type OrderItem struct {
	ProductID string  `json:"productId" doc:"Product identifier"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price,omitempty"`
}

type CreateOrder struct {
	Items      []OrderItem `json:"items"`
	Total      float64     `json:"total"`
	Active     bool        `json:"active"`
	External   any         `json:"external,omitempty"`
}

func TestSchemaFromType_Struct(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateUser]()

	if schema.Type != "object" {
		t.Fatalf("expected object, got %s", schema.Type)
	}
	if _, ok := schema.Properties["email"]; !ok {
		t.Error("expected email property")
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected name property")
	}
	if len(schema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d: %v", len(schema.Required), schema.Required)
	}
}

func TestSchemaFromType_Slice(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateOrder]()
	if schema.Type != "object" {
		t.Fatalf("expected object, got %s", schema.Type)
	}

	items, ok := schema.Properties["items"]
	if !ok {
		t.Fatal("expected items property")
	}
	if items.Type != "array" {
		t.Errorf("expected array, got %s", items.Type)
	}
	if items.Items == nil {
		t.Fatal("expected items.Items to be set")
	}
	if items.Items.Type != "object" {
		t.Errorf("expected object items, got %s", items.Items.Type)
	}
}

func TestSchemaFromType_Description(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateUser]()
	email := schema.Properties["email"]
	if email.Description != "User email address" {
		t.Errorf("expected description 'User email address', got %q", email.Description)
	}
}

func TestSchemaFromType_FieldTypes(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateOrder]()

	tests := map[string]string{
		"items":  "array",
		"total":  "number",
		"active": "boolean",
	}
	for field, expected := range tests {
		prop, ok := schema.Properties[field]
		if !ok {
			t.Errorf("missing property %s", field)
			continue
		}
		if prop.Type != expected {
			t.Errorf("property %s: expected type %s, got %s", field, expected, prop.Type)
		}
	}
}

func TestSchemaToJSON(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateUser]()
	data, err := catalog.SchemaToJSON(schema)
	if err != nil {
		t.Fatalf("SchemaToJSON() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if parsed["type"] != "object" {
		t.Errorf("expected type object, got %v", parsed["type"])
	}
}

func TestSchemaToJSON_Nil(t *testing.T) {
	t.Parallel()

	_, err := catalog.SchemaToJSON(nil)
	if err == nil {
		t.Error("expected error for nil schema")
	}
}

func TestSchemaFromType_PrimitiveTypes(t *testing.T) {
	t.Parallel()

	type Primitives struct {
		Str    string `json:"str"`
		IntVal int    `json:"intVal"`
		BoolV  bool   `json:"boolV"`
		Flt    float64 `json:"flt"`
	}

	schema := catalog.SchemaFromType[Primitives]()
	expected := map[string]string{
		"str":    "string",
		"intVal": "integer",
		"boolV":  "boolean",
		"flt":    "number",
	}
	for name, exp := range expected {
		prop, ok := schema.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)
			continue
		}
		if prop.Type != exp {
			t.Errorf("property %s: expected %s, got %s", name, exp, prop.Type)
		}
	}
}

func TestSchemaFromType_EmbeddedStruct(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Inner Inner `json:"inner"`
		Name  string `json:"name"`
	}

	schema := catalog.SchemaFromType[Outer]()
	inner, ok := schema.Properties["inner"]
	if !ok {
		t.Fatal("expected inner property")
	}
	if inner.Type != "object" {
		t.Errorf("expected object, got %s", inner.Type)
	}
	if _, ok := inner.Properties["value"]; !ok {
		t.Error("expected inner.value property")
	}
}
