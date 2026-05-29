package catalog_test

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func TestSchemaFromReflect_AllPrimitiveKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected catalog.SchemaType
	}{
		{"string", "", catalog.TypeString},
		{"int", int(0), catalog.TypeInteger},
		{"int8", int8(0), catalog.TypeInteger},
		{"int16", int16(0), catalog.TypeInteger},
		{"int32", int32(0), catalog.TypeInteger},
		{"int64", int64(0), catalog.TypeInteger},
		{"uint", uint(0), catalog.TypeInteger},
		{"uint8", uint8(0), catalog.TypeInteger},
		{"uint16", uint16(0), catalog.TypeInteger},
		{"uint32", uint32(0), catalog.TypeInteger},
		{"uint64", uint64(0), catalog.TypeInteger},
		{"uintptr", uintptr(0), catalog.TypeInteger},
		{"float32", float32(0), catalog.TypeNumber},
		{"float64", float64(0), catalog.TypeNumber},
		{"bool", true, catalog.TypeBoolean},
		{"complex64", complex64(0), catalog.TypeString},
		{"complex128", complex128(0), catalog.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema := catalog.SchemaFromReflect(reflect.TypeOf(tt.input))
			if schema.Type != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, schema.Type)
			}
		})
	}
}

func TestSchemaFromReflect_Interface(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromReflect(reflect.TypeFor[any]())

	if schema.Type != catalog.TypeObject {
		t.Errorf("expected interface to map to object, got %q", schema.Type)
	}
}

func TestPropertyFromReflect_Map(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromReflect(reflect.TypeFor[map[string]int]())

	if schema.Type != catalog.TypeObject {
		t.Errorf("expected map to map to object, got %q", schema.Type)
	}
}

func TestCollectionSchema_NonSlice(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromReflect(reflect.TypeFor[string]())

	if schema.Type != catalog.TypeString {
		t.Errorf("expected string for direct string type, got %q", schema.Type)
	}
}

func TestSchemaFromReflect_UnsignedIntegers(t *testing.T) {
	t.Parallel()

	type UnsignedTypes struct {
		U8  uint8  `json:"u8"`
		U16 uint16 `json:"u16"`
		U32 uint32 `json:"u32"`
		U64 uint64 `json:"u64"`
	}

	schema := catalog.SchemaFromType[UnsignedTypes]()

	unsignedFields := []string{"u8", "u16", "u32", "u64"}
	for _, name := range unsignedFields {
		prop, ok := schema.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)

			continue
		}

		if prop.Type != catalog.TypeInteger {
			t.Errorf("property %s: expected integer, got %s", name, prop.Type)
		}
	}
}

func TestSchemaFromReflect_ComplexTypes(t *testing.T) {
	t.Parallel()

	type ComplexTypes struct {
		C64  complex64  `json:"c64"`
		C128 complex128 `json:"c128"`
	}

	schema := catalog.SchemaFromType[ComplexTypes]()

	for _, name := range []string{"c64", "c128"} {
		prop, ok := schema.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)

			continue
		}

		if prop.Type != catalog.TypeString {
			t.Errorf("property %s: expected string (complex), got %s", name, prop.Type)
		}
	}
}

func TestSchemaFromReflect_InterfaceType(t *testing.T) {
	t.Parallel()

	type WithInterface struct {
		Val any `json:"val"`
	}

	schema := catalog.SchemaFromType[WithInterface]()

	prop, ok := schema.Properties["val"]
	if !ok {
		t.Fatal("missing property val")
	}

	if prop.Type != catalog.TypeObject {
		t.Errorf("expected object for interface, got %s", prop.Type)
	}
}

func TestCollectionSchema_ArrayType(t *testing.T) {
	t.Parallel()

	type WithArray struct {
		Items [3]int `json:"items"`
	}

	schema := catalog.SchemaFromType[WithArray]()

	prop, ok := schema.Properties["items"]
	if !ok {
		t.Fatal("missing property items")
	}

	if prop.Type != catalog.TypeArray {
		t.Errorf("expected array for fixed-size array, got %s", prop.Type)
	}
}
