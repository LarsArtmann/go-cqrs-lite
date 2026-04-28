package catalog_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func TestSchemaFromType_EnumTag(t *testing.T) {
	t.Parallel()

	type WithEnum struct {
		Status string `enum:"active,inactive,pending" json:"status"`
	}

	schema := catalog.SchemaFromType[WithEnum]()

	prop := schema.Properties["status"]
	if len(prop.Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(prop.Enum))
	}

	expected := []string{"active", "inactive", "pending"}
	for i, v := range expected {
		if prop.Enum[i] != v {
			t.Errorf("enum[%d] = %q, want %q", i, prop.Enum[i], v)
		}
	}
}

func TestSchemaFromType_DefaultTag(t *testing.T) {
	t.Parallel()

	type WithDefault struct {
		Role string `default:"viewer" json:"role"`
	}

	schema := catalog.SchemaFromType[WithDefault]()

	prop := schema.Properties["role"]
	if prop.Default != "viewer" {
		t.Errorf("default = %q, want %q", prop.Default, "viewer")
	}
}

func TestSchemaFromType_NullableTag(t *testing.T) {
	t.Parallel()

	type WithNullable struct {
		Email string `json:"email" nullable:"true"`
	}

	schema := catalog.SchemaFromType[WithNullable]()

	prop := schema.Properties["email"]
	if !prop.Nullable {
		t.Error("expected nullable=true")
	}
}

func TestSchemaFromType_DeprecatedTag(t *testing.T) {
	t.Parallel()

	type WithDeprecated struct {
		OldField string `deprecated:"true" json:"oldField"`
	}

	schema := catalog.SchemaFromType[WithDeprecated]()

	prop := schema.Properties["oldField"]
	if !prop.Deprecated {
		t.Error("expected deprecated=true")
	}
}

func TestSchemaFromType_PatternTag(t *testing.T) {
	t.Parallel()

	type WithPattern struct {
		Slug string `json:"slug" pattern:"^[a-z0-9-]+$"`
	}

	schema := catalog.SchemaFromType[WithPattern]()

	prop := schema.Properties["slug"]
	if prop.Pattern != "^[a-z0-9-]+$" {
		t.Errorf("pattern = %q, want ^[a-z0-9-]+$", prop.Pattern)
	}
}

func TestSchemaFromType_AllTagsCombined(t *testing.T) {
	t.Parallel()

	type RichField struct {
		Status string `default:"active" doc:"Current status" enum:"active,inactive" json:"status"`
	}

	schema := catalog.SchemaFromType[RichField]()

	prop := schema.Properties["status"]
	if prop.Description != "Current status" {
		t.Errorf("description = %q", prop.Description)
	}

	if len(prop.Enum) != 2 {
		t.Errorf("enum = %v", prop.Enum)
	}

	if prop.Default != "active" {
		t.Errorf("default = %q", prop.Default)
	}
}

func TestSchemaFromReflect(t *testing.T) {
	t.Parallel()

	t.Run("nil_type", func(t *testing.T) {
		t.Parallel()

		schema := catalog.SchemaFromReflect(nil)
		if schema.Type != "null" {
			t.Errorf("expected null, got %s", schema.Type)
		}
	})

	t.Run("primitive_string", func(t *testing.T) {
		t.Parallel()

		schema := catalog.SchemaFromReflect(reflect.TypeFor[string]())
		if schema.Type != "string" {
			t.Errorf("expected string, got %s", schema.Type)
		}
	})

	t.Run("uint_type", func(t *testing.T) {
		t.Parallel()

		type WithUint struct {
			Count uint `json:"count"`
		}

		schema := catalog.SchemaFromType[WithUint]()
		if schema.Properties["count"].Type != "integer" {
			t.Errorf("expected integer for uint, got %s", schema.Properties["count"].Type)
		}
	})
}

func TestSchemaFromType_ComplexAndChannelTypes(t *testing.T) {
	t.Parallel()

	type WeirdTypes struct {
		Complex complex128 `json:"c"`
		Ch      chan int   `json:"ch"`
	}

	schema := catalog.SchemaFromType[WeirdTypes]()

	if schema.Properties["c"].Type != "string" {
		t.Errorf("expected string for complex128, got %s", schema.Properties["c"].Type)
	}

	if schema.Properties["ch"].Type != "string" {
		t.Errorf("expected string for chan, got %s", schema.Properties["ch"].Type)
	}
}

func TestSchemaFromType_InterfaceField(t *testing.T) {
	t.Parallel()

	type WithInterface struct {
		Val any `json:"val"`
	}

	schema := catalog.SchemaFromType[WithInterface]()
	if schema.Properties["val"].Type != "object" {
		t.Errorf("expected object for any/interface, got %s", schema.Properties["val"].Type)
	}
}

func TestSchemaFromType_SliceOfPrimitives(t *testing.T) {
	t.Parallel()

	type WithSlice struct {
		Tags []string `json:"tags"`
	}

	schema := catalog.SchemaFromType[WithSlice]()

	prop := schema.Properties["tags"]
	if prop.Type != "array" {
		t.Fatalf("expected array, got %s", prop.Type)
	}

	if prop.Items == nil || prop.Items.Type != "string" {
		t.Errorf("expected string items, got %v", prop.Items)
	}
}

func TestSchemaFromType_MapWithNonStringKey(t *testing.T) {
	t.Parallel()

	type WithMap struct {
		Counts map[int]string `json:"counts"`
	}

	schema := catalog.SchemaFromType[WithMap]()

	prop := schema.Properties["counts"]
	if prop.Type != "object" {
		t.Errorf("expected object, got %s", prop.Type)
	}
}

func TestSchemaFromType_TopLevelSlice(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[[]string]()
	if schema.Type != "array" {
		t.Errorf("expected array, got %s", schema.Type)
	}

	if schema.Items == nil || schema.Items.Type != "string" {
		t.Errorf("expected string items, got %v", schema.Items)
	}
}

func TestSchemaFromType_TopLevelMap(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[map[string]int]()
	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}
}

func TestSchemaFromType_TopLevelTime(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[time.Time]()
	if schema.Type != "string" {
		t.Errorf("expected string for time.Time, got %s", schema.Type)
	}
}
