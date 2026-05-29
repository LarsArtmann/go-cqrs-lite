package catalog_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

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
