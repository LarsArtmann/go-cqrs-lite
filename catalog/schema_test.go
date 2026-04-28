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

	if prop.Type != "string" {
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

	if prop.Type != "object" {
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

	if prop.Type != "array" {
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

	if prop.Type != "string" {
		t.Errorf("expected type string for *time.Time, got %q", prop.Type)
	}

	if prop.Format != "date-time" {
		t.Errorf("expected format date-time for *time.Time, got %q", prop.Format)
	}
}

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
