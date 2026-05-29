package schema_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/schema"
)

func TestFromType_EnumTag(t *testing.T) {
	t.Parallel()

	type WithEnum struct {
		Status string `enum:"active,inactive,pending" json:"status"`
	}

	s := schema.FromType[WithEnum]()

	prop := s.Properties["status"]
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

func TestFromType_DefaultTag(t *testing.T) {
	t.Parallel()

	type WithDefault struct {
		Role string `default:"viewer" json:"role"`
	}

	s := schema.FromType[WithDefault]()

	prop := s.Properties["role"]
	if prop.Default != "viewer" {
		t.Errorf("default = %q, want %q", prop.Default, "viewer")
	}
}

func TestFromType_NullableTag(t *testing.T) {
	t.Parallel()

	type WithNullable struct {
		Email string `json:"email" nullable:"true"`
	}

	s := schema.FromType[WithNullable]()

	prop := s.Properties["email"]
	if !prop.Nullable {
		t.Error("expected nullable=true")
	}
}

func TestFromType_DeprecatedTag(t *testing.T) {
	t.Parallel()

	type WithDeprecated struct {
		OldField string `deprecated:"true" json:"oldField"`
	}

	s := schema.FromType[WithDeprecated]()

	prop := s.Properties["oldField"]
	if !prop.Deprecated {
		t.Error("expected deprecated=true")
	}
}

func TestFromType_PatternTag(t *testing.T) {
	t.Parallel()

	type WithPattern struct {
		Slug string `json:"slug" pattern:"^[a-z0-9-]+$"`
	}

	s := schema.FromType[WithPattern]()

	prop := s.Properties["slug"]
	if prop.Pattern != "^[a-z0-9-]+$" {
		t.Errorf("pattern = %q, want ^[a-z0-9-]+$", prop.Pattern)
	}
}

func TestFromType_AllTagsCombined(t *testing.T) {
	t.Parallel()

	type RichField struct {
		Status string `default:"active" doc:"Current status" enum:"active,inactive" json:"status"`
	}

	s := schema.FromType[RichField]()

	prop := s.Properties["status"]
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

func TestFromReflect_NilAndPrimitive(t *testing.T) {
	t.Parallel()

	t.Run("nil_type", func(t *testing.T) {
		t.Parallel()

		s := schema.FromReflect(nil)
		if s.Type != "null" {
			t.Errorf("expected null, got %s", s.Type)
		}
	})

	t.Run("primitive_string", func(t *testing.T) {
		t.Parallel()

		s := schema.FromReflect(reflect.TypeFor[string]())
		if s.Type != "string" {
			t.Errorf("expected string, got %s", s.Type)
		}
	})

	t.Run("uint_type", func(t *testing.T) {
		t.Parallel()

		type WithUint struct {
			Count uint `json:"count"`
		}

		s := schema.FromType[WithUint]()
		if s.Properties["count"].Type != "integer" {
			t.Errorf("expected integer for uint, got %s", s.Properties["count"].Type)
		}
	})
}

func TestFromType_ComplexAndChannelTypes(t *testing.T) {
	t.Parallel()

	type WeirdTypes struct {
		Complex complex128 `json:"c"`
		Ch      chan int   `json:"ch"`
	}

	s := schema.FromType[WeirdTypes]()

	if s.Properties["c"].Type != "string" {
		t.Errorf("expected string for complex128, got %s", s.Properties["c"].Type)
	}

	if s.Properties["ch"].Type != "string" {
		t.Errorf("expected string for chan, got %s", s.Properties["ch"].Type)
	}
}

func TestFromType_InterfaceField(t *testing.T) {
	t.Parallel()

	type WithInterface struct {
		Val any `json:"val"`
	}

	s := schema.FromType[WithInterface]()
	if s.Properties["val"].Type != "object" {
		t.Errorf("expected object for any/interface, got %s", s.Properties["val"].Type)
	}
}

func TestFromType_SliceOfPrimitives(t *testing.T) {
	t.Parallel()

	type WithSlice struct {
		Tags []string `json:"tags"`
	}

	s := schema.FromType[WithSlice]()

	prop := s.Properties["tags"]
	if prop.Type != "array" {
		t.Fatalf("expected array, got %s", prop.Type)
	}

	if prop.Items == nil || prop.Items.Type != "string" {
		t.Errorf("expected string items, got %v", prop.Items)
	}
}

func TestFromType_MapWithNonStringKey(t *testing.T) {
	t.Parallel()

	type WithMap struct {
		Counts map[int]string `json:"counts"`
	}

	s := schema.FromType[WithMap]()

	prop := s.Properties["counts"]
	if prop.Type != "object" {
		t.Errorf("expected object, got %s", prop.Type)
	}
}

func TestFromType_TopLevelSlice(t *testing.T) {
	t.Parallel()

	s := schema.FromType[[]string]()
	if s.Type != "array" {
		t.Errorf("expected array, got %s", s.Type)
	}

	if s.Items == nil || s.Items.Type != "string" {
		t.Errorf("expected string items, got %v", s.Items)
	}
}

func TestFromType_TopLevelMap(t *testing.T) {
	t.Parallel()

	s := schema.FromType[map[string]int]()
	if s.Type != "object" {
		t.Errorf("expected object, got %s", s.Type)
	}
}

func TestFromType_TopLevelTime(t *testing.T) {
	t.Parallel()

	s := schema.FromType[time.Time]()
	if s.Type != "string" {
		t.Errorf("expected string for time.Time, got %s", s.Type)
	}
}
