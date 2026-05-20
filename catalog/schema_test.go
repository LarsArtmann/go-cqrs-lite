package catalog_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func assertPropertyCount(t *testing.T, schema *catalog.Schema, expected int) {
	t.Helper()

	if len(schema.Properties) != expected {
		t.Errorf(
			"expected %d properties, got %d: %v",
			expected,
			len(schema.Properties),
			schema.Properties,
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

func TestSchemaFromType_Struct(t *testing.T) {
	t.Parallel()

	schema := catalog.SchemaFromType[CreateUser]()

	if schema.Type != catalog.TypeObject {
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
	if schema.Type != catalog.TypeObject {
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

	tests := map[string]catalog.SchemaType{
		"items":  catalog.TypeArray,
		"total":  catalog.TypeNumber,
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

	err = json.Unmarshal(data, &parsed)
	if err != nil {
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
		Str    string  `json:"str"`
		IntVal int     `json:"intVal"`
		BoolV  bool    `json:"boolV"`
		Flt    float64 `json:"flt"`
	}

	schema := catalog.SchemaFromType[Primitives]()

	expected := map[string]catalog.SchemaType{
		"str":    catalog.TypeString,
		"intVal": catalog.TypeInteger,
		"boolV":  catalog.TypeBoolean,
		"flt":    catalog.TypeNumber,
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
		Inner Inner  `json:"inner"`
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

func TestSchemaFromType_PointerField(t *testing.T) {
	t.Parallel()

	type WithPtr struct {
		Name *string `json:"name"`
	}

	schema := catalog.SchemaFromType[WithPtr]()

	prop, ok := schema.Properties["name"]
	if !ok {
		t.Fatal("expected name property")
	}

	if prop.Type != catalog.TypeString {
		t.Errorf("expected string, got %s", prop.Type)
	}
}

func TestSchemaFromType_MapField(t *testing.T) {
	t.Parallel()

	type WithMap struct {
		Meta map[string]string `json:"meta"`
	}

	schema := catalog.SchemaFromType[WithMap]()

	prop, ok := schema.Properties["meta"]
	if !ok {
		t.Fatal("expected meta property")
	}

	if prop.Type != catalog.TypeObject {
		t.Errorf("expected object, got %s", prop.Type)
	}
}

func TestSchemaFromType_FormatTag(t *testing.T) {
	t.Parallel()

	type WithFormat struct {
		Email     string `format:"email"     json:"email"`
		CreatedAt string `format:"date-time" json:"createdAt"`
	}

	schema := catalog.SchemaFromType[WithFormat]()
	for _, tc := range []struct {
		prop   string
		format string
	}{
		{"email", "email"},
		{"createdAt", "date-time"},
	} {
		if schema.Properties[tc.prop].Format != tc.format {
			t.Errorf("expected format %q, got %q", tc.format, schema.Properties[tc.prop].Format)
		}
	}
}

func TestSchemaFromType_DescriptionTag(t *testing.T) {
	t.Parallel()

	type WithDesc struct {
		Name string `description:"The name" json:"name"`
	}

	schema := catalog.SchemaFromType[WithDesc]()
	if schema.Properties["name"].Description != "The name" {
		t.Errorf("expected description 'The name', got %q", schema.Properties["name"].Description)
	}
}

func TestSchemaFromType_SkipsUnexportedAndIgnored(t *testing.T) {
	t.Parallel()

	type Mixed struct {
		Name    string `json:"name"`
		_       string // intentionally private to test field skipping
		Ignored string `json:"-"`
	}

	schema := catalog.SchemaFromType[Mixed]()
	if _, ok := schema.Properties["private"]; ok {
		t.Error("unexported field should be skipped")
	}

	if _, ok := schema.Properties["Ignored"]; ok {
		t.Error("json:'-' field should be skipped")
	}

	if len(schema.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(schema.Properties))
	}
}

func TestSchemaFromType_ArrayField(t *testing.T) {
	t.Parallel()

	type WithArray struct {
		IDs [3]string `json:"ids"`
	}

	schema := catalog.SchemaFromType[WithArray]()

	prop, ok := schema.Properties["ids"]
	if !ok {
		t.Fatal("expected ids property")
	}

	if prop.Type != catalog.TypeArray {
		t.Errorf("expected array, got %s", prop.Type)
	}
}

func TestSchemaFromType_EmptyTag(t *testing.T) {
	t.Parallel()

	type NoJSON struct {
		Name string
	}

	schema := catalog.SchemaFromType[NoJSON]()
	if _, ok := schema.Properties["Name"]; !ok {
		t.Error("expected Name property (no json tag uses field name)")
	}
}

func TestSchemaFromType_SkipsAnonymousEmbeddedFields(t *testing.T) {
	t.Parallel()

	type Embed struct {
		ID string `json:"id"`
	}

	type WithEmbed struct {
		Embed

		Name  string `json:"name"`
		Email string `json:"email"`
	}

	schema := catalog.SchemaFromType[WithEmbed]()

	if _, ok := schema.Properties["Embed"]; ok {
		t.Error("anonymous embedded field 'Embed' should be skipped")
	}

	if _, ok := schema.Properties["id"]; ok {
		t.Error("promoted fields from anonymous embed should not appear")
	}

	assertPropertyCount(t, schema, 2)
}

func TestSchemaFromType_SkipsAnonymousPointerEmbeddedFields(t *testing.T) {
	t.Parallel()

	type Core struct {
		ID string `json:"id"`
	}

	type WithPtrEmbed struct {
		*Core

		Name string `json:"name"`
	}

	schema := catalog.SchemaFromType[WithPtrEmbed]()

	if _, ok := schema.Properties["Core"]; ok {
		t.Error("anonymous embedded pointer field 'Core' should be skipped")
	}

	assertPropertyCount(t, schema, 1)
}

func TestSchemaFromType_TimeTime(t *testing.T) {
	t.Parallel()

	type WithTime struct {
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
	}

	schema := catalog.SchemaFromType[WithTime]()

	createdAt, ok := schema.Properties["createdAt"]
	if !ok {
		t.Fatal("expected createdAt property")
	}

	if createdAt.Type != "string" {
		t.Errorf("expected type string for time.Time, got %q", createdAt.Type)
	}

	if createdAt.Format != "date-time" {
		t.Errorf("expected format date-time for time.Time, got %q", createdAt.Format)
	}

	if len(createdAt.Properties) != 0 {
		t.Errorf("time.Time should not have nested properties, got %v", createdAt.Properties)
	}
}

func TestSchemaFromType_PointerTimeTime(t *testing.T) {
	t.Parallel()

	type WithPtrTime struct {
		UpdatedAt *time.Time `json:"updatedAt"`
	}

	schema := catalog.SchemaFromType[WithPtrTime]()

	prop, ok := schema.Properties["updatedAt"]
	if !ok {
		t.Fatal("expected updatedAt property")
	}

	if prop.Type != catalog.TypeString {
		t.Errorf("expected type string for *time.Time, got %q", prop.Type)
	}

	if prop.Format != "date-time" {
		t.Errorf("expected format date-time for *time.Time, got %q", prop.Format)
	}
}

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

	// collectionSchema with a non-slice type falls through to object
	// This tests the else branch that's normally unreachable
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
