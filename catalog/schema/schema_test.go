package schema_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/schema"
)

func TestToAny_Nil(t *testing.T) {
	t.Parallel()

	result := schema.ToAny(nil)
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["type"] != "object" {
		t.Errorf("expected object, got %s", m["type"])
	}
}

func TestToAny_EmptySchema(t *testing.T) {
	t.Parallel()

	result := schema.ToAny(&schema.Schema{})
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

	result := schema.ToAny(s)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["type"] != "object" {
		t.Errorf("expected object, got %v", m["type"])
	}
}

func TestFromType_Basic(t *testing.T) {
	t.Parallel()

	type Simple struct {
		Name string `json:"name"`
	}

	s := schema.FromType[Simple]()
	if s.Type != schema.TypeObject {
		t.Errorf("expected object, got %s", s.Type)
	}

	if _, ok := s.Properties["name"]; !ok {
		t.Error("expected name property")
	}
}

func TestToJSON_Nil(t *testing.T) {
	t.Parallel()

	_, err := schema.ToJSON(nil)
	if err == nil {
		t.Error("expected error for nil schema")
	}
}
