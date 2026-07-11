package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func assertPropertyCount(t *testing.T, s *schema.Schema, expected int) {
	t.Helper()

	if len(s.Properties) != expected {
		t.Errorf(
			"expected %d properties, got %d: %v",
			expected,
			len(s.Properties),
			s.Properties,
		)
	}
}

type CreateUser struct {
	Email string `doc:"User email address" json:"email"`
	Name  string `                         json:"name"`
	Age   int    `                         json:"age,omitempty"`
}

type OrderItem struct {
	ProductID string  `doc:"Product identifier" json:"productId"`
	Quantity  int     `                         json:"quantity"`
	Price     float64 `                         json:"price,omitempty"`
}

type CreateOrder struct {
	Items    []OrderItem `json:"items"`
	Total    float64     `json:"total"`
	Active   bool        `json:"active"`
	External any         `json:"external,omitempty"`
}

func TestFromType_Struct(t *testing.T) {
	t.Parallel()

	s := schema.FromType[CreateUser]()

	if s.Type != schema.TypeObject {
		t.Fatalf("expected object, got %s", s.Type)
	}

	if _, ok := s.Properties["email"]; !ok {
		t.Error("expected email property")
	}

	if _, ok := s.Properties["name"]; !ok {
		t.Error("expected name property")
	}

	if len(s.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d: %v", len(s.Required), s.Required)
	}
}

func TestFromType_Slice(t *testing.T) {
	t.Parallel()

	s := schema.FromType[CreateOrder]()
	if s.Type != schema.TypeObject {
		t.Fatalf("expected object, got %s", s.Type)
	}

	items, ok := s.Properties["items"]
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

func TestFromType_Description(t *testing.T) {
	t.Parallel()

	s := schema.FromType[CreateUser]()

	email := s.Properties["email"]
	if email.Description != "User email address" {
		t.Errorf("expected description 'User email address', got %q", email.Description)
	}
}

func TestFromType_FieldTypes(t *testing.T) {
	t.Parallel()

	s := schema.FromType[CreateOrder]()

	tests := map[string]schema.Type{
		"items":  schema.TypeArray,
		"total":  schema.TypeNumber,
		"active": "boolean",
	}
	for field, expected := range tests {
		prop, ok := s.Properties[field]
		if !ok {
			t.Errorf("missing property %s", field)

			continue
		}

		if prop.Type != expected {
			t.Errorf("property %s: expected type %s, got %s", field, expected, prop.Type)
		}
	}
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	s := schema.FromType[CreateUser]()

	data, err := schema.ToJSON(s)
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var parsed map[string]any

	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("expected type object, got %v", parsed["type"])
	}
}

func TestToJSON_Nil(t *testing.T) {
	t.Parallel()

	_, err := schema.ToJSON(nil)
	if err == nil {
		t.Error("expected error for nil schema")
	}
}

func TestFromType_PrimitiveTypes(t *testing.T) {
	t.Parallel()

	type Primitives struct {
		Str    string  `json:"str"`
		IntVal int     `json:"intVal"`
		BoolV  bool    `json:"boolV"`
		Flt    float64 `json:"flt"`
	}

	s := schema.FromType[Primitives]()

	expected := map[string]schema.Type{
		"str":    schema.TypeString,
		"intVal": schema.TypeInteger,
		"boolV":  schema.TypeBoolean,
		"flt":    schema.TypeNumber,
	}
	for name, exp := range expected {
		prop, ok := s.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)

			continue
		}

		if prop.Type != exp {
			t.Errorf("property %s: expected %s, got %s", name, exp, prop.Type)
		}
	}
}

func TestToAny_Nil(t *testing.T) {
	t.Parallel()

	result, err := schema.ToAny(nil)
	if err == nil {
		t.Fatal("expected error for nil schema")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %T", result)
	}
}

func TestToAny_EmptySchema(t *testing.T) {
	t.Parallel()

	result, err := schema.ToAny(&schema.Schema{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["type"] != "" {
		t.Errorf("expected empty type, got %v", m["type"])
	}
}

func TestToAny_WithProperties(t *testing.T) {
	t.Parallel()

	s := &schema.Schema{
		Type: schema.TypeObject,
		Properties: map[string]schema.Property{
			"name": {Type: schema.TypeString},
		},
	}

	result, err := schema.ToAny(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["type"] != "object" {
		t.Errorf("expected object, got %v", m["type"])
	}
}
