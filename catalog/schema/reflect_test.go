package schema_test

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2/schema"
)

func TestFromReflect_AllPrimitiveKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected schema.Type
	}{
		{"string", "", schema.TypeString},
		{"int", int(0), schema.TypeInteger},
		{"int8", int8(0), schema.TypeInteger},
		{"int16", int16(0), schema.TypeInteger},
		{"int32", int32(0), schema.TypeInteger},
		{"int64", int64(0), schema.TypeInteger},
		{"uint", uint(0), schema.TypeInteger},
		{"uint8", uint8(0), schema.TypeInteger},
		{"uint16", uint16(0), schema.TypeInteger},
		{"uint32", uint32(0), schema.TypeInteger},
		{"uint64", uint64(0), schema.TypeInteger},
		{"uintptr", uintptr(0), schema.TypeInteger},
		{"float32", float32(0), schema.TypeNumber},
		{"float64", float64(0), schema.TypeNumber},
		{"bool", true, schema.TypeBoolean},
		{"complex64", complex64(0), schema.TypeString},
		{"complex128", complex128(0), schema.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := schema.FromReflect(reflect.TypeOf(tt.input))
			if s.Type != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, s.Type)
			}
		})
	}
}

func TestFromReflect_Interface(t *testing.T) {
	t.Parallel()

	s := schema.FromReflect(reflect.TypeFor[any]())

	if s.Type != schema.TypeObject {
		t.Errorf("expected interface to map to object, got %q", s.Type)
	}
}

func TestFromReflect_Map(t *testing.T) {
	t.Parallel()

	s := schema.FromReflect(reflect.TypeFor[map[string]int]())

	if s.Type != schema.TypeObject {
		t.Errorf("expected map to map to object, got %q", s.Type)
	}
}

func TestFromReflect_NonSlice(t *testing.T) {
	t.Parallel()

	s := schema.FromReflect(reflect.TypeFor[string]())

	if s.Type != schema.TypeString {
		t.Errorf("expected string for direct string type, got %q", s.Type)
	}
}

func TestFromReflect_UnsignedIntegers(t *testing.T) {
	t.Parallel()

	type UnsignedTypes struct {
		U8  uint8  `json:"u8"`
		U16 uint16 `json:"u16"`
		U32 uint32 `json:"u32"`
		U64 uint64 `json:"u64"`
	}

	s := schema.FromType[UnsignedTypes]()

	unsignedFields := []string{"u8", "u16", "u32", "u64"}
	for _, name := range unsignedFields {
		prop, ok := s.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)

			continue
		}

		if prop.Type != schema.TypeInteger {
			t.Errorf("property %s: expected integer, got %s", name, prop.Type)
		}
	}
}

func TestFromReflect_ComplexTypes(t *testing.T) {
	t.Parallel()

	type ComplexTypes struct {
		C64  complex64  `json:"c64"`
		C128 complex128 `json:"c128"`
	}

	s := schema.FromType[ComplexTypes]()

	for _, name := range []string{"c64", "c128"} {
		prop, ok := s.Properties[name]
		if !ok {
			t.Errorf("missing property %s", name)

			continue
		}

		if prop.Type != schema.TypeString {
			t.Errorf("property %s: expected string (complex), got %s", name, prop.Type)
		}
	}
}

func TestFromReflect_InterfaceType(t *testing.T) {
	t.Parallel()

	type WithInterface struct {
		Val any `json:"val"`
	}

	s := schema.FromType[WithInterface]()

	prop, ok := s.Properties["val"]
	if !ok {
		t.Fatal("missing property val")
	}

	if prop.Type != schema.TypeObject {
		t.Errorf("expected object for interface, got %s", prop.Type)
	}
}

func TestFromReflect_ArrayType(t *testing.T) {
	t.Parallel()

	type WithArray struct {
		Items [3]int `json:"items"`
	}

	s := schema.FromType[WithArray]()

	prop, ok := s.Properties["items"]
	if !ok {
		t.Fatal("missing property items")
	}

	if prop.Type != schema.TypeArray {
		t.Errorf("expected array for fixed-size array, got %s", prop.Type)
	}
}
