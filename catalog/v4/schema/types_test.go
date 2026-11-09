package schema_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func TestFromType_EmbeddedStruct(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Value string `json:"value"`
	}

	type Outer struct {
		Inner Inner  `json:"inner"`
		Name  string `json:"name"`
	}

	s := schema.FromType[Outer]()

	inner, ok := s.Properties["inner"]
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

func TestFromType_PointerField(t *testing.T) {
	t.Parallel()

	type WithPtr struct {
		Name *string `json:"name"`
	}

	s := schema.FromType[WithPtr]()

	prop, ok := s.Properties["name"]
	if !ok {
		t.Fatal("expected name property")
	}

	if prop.Type != schema.TypeString {
		t.Errorf("expected string, got %s", prop.Type)
	}
}

func TestFromType_MapField(t *testing.T) {
	t.Parallel()

	type WithMap struct {
		Meta map[string]string `json:"meta"`
	}

	s := schema.FromType[WithMap]()

	prop, ok := s.Properties["meta"]
	if !ok {
		t.Fatal("expected meta property")
	}

	if prop.Type != schema.TypeObject {
		t.Errorf("expected object, got %s", prop.Type)
	}
}

func TestFromType_FormatTag(t *testing.T) {
	t.Parallel()

	type WithFormat struct {
		Email     string `format:"email"     json:"email"`
		CreatedAt string `format:"date-time" json:"createdAt"`
	}

	s := schema.FromType[WithFormat]()
	for _, tc := range []struct {
		prop   string
		format string
	}{
		{"email", "email"},
		{"createdAt", "date-time"},
	} {
		if s.Properties[tc.prop].Format != tc.format {
			t.Errorf("expected format %q, got %q", tc.format, s.Properties[tc.prop].Format)
		}
	}
}

func TestFromType_DescriptionTag(t *testing.T) {
	t.Parallel()

	type WithDesc struct {
		Name string `description:"The name" json:"name"`
	}

	s := schema.FromType[WithDesc]()
	if s.Properties["name"].Description != "The name" {
		t.Errorf("expected description 'The name', got %q", s.Properties["name"].Description)
	}
}

func TestFromType_SkipsUnexportedAndIgnored(t *testing.T) {
	t.Parallel()

	type Mixed struct {
		Name    string `json:"name"`
		_       string // intentionally private to test field skipping
		Ignored string `json:"-"`
	}

	s := schema.FromType[Mixed]()
	if _, ok := s.Properties["private"]; ok {
		t.Error("unexported field should be skipped")
	}

	if _, ok := s.Properties["Ignored"]; ok {
		t.Error("json:'-' field should be skipped")
	}

	if len(s.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(s.Properties))
	}
}

func TestFromType_ArrayField(t *testing.T) {
	t.Parallel()

	type WithArray struct {
		IDs [3]string `json:"ids"`
	}

	s := schema.FromType[WithArray]()

	prop, ok := s.Properties["ids"]
	if !ok {
		t.Fatal("expected ids property")
	}

	if prop.Type != schema.TypeArray {
		t.Errorf("expected array, got %s", prop.Type)
	}
}

func TestFromType_EmptyTag(t *testing.T) {
	t.Parallel()

	type NoJSON struct {
		Name string
	}

	s := schema.FromType[NoJSON]()
	if _, ok := s.Properties["Name"]; !ok {
		t.Error("expected Name property (no json tag uses field name)")
	}
}

// encoding/json promotes exported anonymous struct (and pointer-to-struct)
// fields into the parent JSON object, so the schema must flatten them too —
// otherwise generated clients reject payloads the wire format accepts.
func TestFromType_PromotesAnonymousEmbeddedFields(t *testing.T) {
	t.Parallel()

	type Embed struct {
		ID string `json:"id"`
	}

	type WithEmbed struct {
		Embed

		Name  string `json:"name"`
		Email string `json:"email"`
	}

	s := schema.FromType[WithEmbed]()

	if _, ok := s.Properties["Embed"]; ok {
		t.Error("anonymous embedded struct must not appear as its own property")
	}

	if _, ok := s.Properties["id"]; !ok {
		t.Error("promoted embedded field 'id' missing from schema")
	}

	assertPropertyCount(t, s, 3)
}

func TestFromType_PromotesAnonymousPointerEmbeddedFields(t *testing.T) {
	t.Parallel()

	type Core struct {
		ID string `json:"id"`
	}

	type WithPtrEmbed struct {
		*Core

		Name string `json:"name"`
	}

	s := schema.FromType[WithPtrEmbed]()

	if _, ok := s.Properties["Core"]; ok {
		t.Error("anonymous embedded pointer must not appear as its own property")
	}

	if _, ok := s.Properties["id"]; !ok {
		t.Error("promoted embedded field 'id' missing from schema")
	}

	assertPropertyCount(t, s, 2)
}

func TestFromType_TimeTime(t *testing.T) {
	t.Parallel()

	type WithTime struct {
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
	}

	s := schema.FromType[WithTime]()

	createdAt, ok := s.Properties["createdAt"]
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

func TestFromType_PointerTimeTime(t *testing.T) {
	t.Parallel()

	type WithPtrTime struct {
		UpdatedAt *time.Time `json:"updatedAt"`
	}

	s := schema.FromType[WithPtrTime]()

	prop, ok := s.Properties["updatedAt"]
	if !ok {
		t.Fatal("expected updatedAt property")
	}

	if prop.Type != schema.TypeString {
		t.Errorf("expected type string for *time.Time, got %q", prop.Type)
	}

	if prop.Format != "date-time" {
		t.Errorf("expected format date-time for *time.Time, got %q", prop.Format)
	}
}
