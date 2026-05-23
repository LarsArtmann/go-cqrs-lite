package schemautil

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func TestSchemaToAny_Nil(t *testing.T) {
	t.Parallel()

	result := SchemaToAny(nil)

	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}

	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}
}

func TestSchemaToAny_WithProperties(t *testing.T) {
	t.Parallel()

	schema := &catalog.Schema{
		Type: catalog.TypeObject,
		Properties: map[string]catalog.Property{
			"name": {Type: catalog.TypeString},
		},
		Required: []string{"name"},
	}

	result := SchemaToAny(schema)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any")
	}

	if m["type"] != "object" {
		t.Errorf("type = %v, want object", m["type"])
	}

	if _, exists := m["properties"]; !exists {
		t.Error("missing properties")
	}

	required, _ := m["required"].([]any)
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("required = %v", m["required"])
	}
}

func TestObjectSchema(t *testing.T) {
	t.Parallel()

	schema := ObjectSchema()
	if schema["type"] != "object" {
		t.Errorf("type = %q, want object", schema["type"])
	}
}

func TestJSONToYAML(t *testing.T) {
	t.Parallel()

	input, _ := json.Marshal(map[string]any{"key": "value"}) //nolint:errchkjson
	output, err := JSONToYAML(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestJSONToYAML_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := JSONToYAML([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONToYAML_Empty(t *testing.T) {
	t.Parallel()

	output, err := JSONToYAML([]byte("null"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestSchemaToAny_EmptySchema(t *testing.T) {
	t.Parallel()

	result := SchemaToAny(&catalog.Schema{})
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if m["type"] != "" {
		t.Errorf("type = %v, want empty", m["type"])
	}
}
