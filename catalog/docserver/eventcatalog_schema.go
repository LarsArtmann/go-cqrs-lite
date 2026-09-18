package docserver

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// Schema pretty-rendering helpers for the event catalog message detail page:
// flatten properties into table rows and render schemas/examples as indented
// JSON.

// schemaProperties flattens the top-level schema properties into table rows.
func schemaProperties(s *schema.Schema) []eventCatalogProperty {
	required := map[string]bool{}
	for _, name := range s.Required {
		required[name] = true
	}

	rows := make([]eventCatalogProperty, 0, len(s.Properties))
	for _, name := range sortedKeys(s.Properties) {
		prop := s.Properties[name]
		rows = append(rows, eventCatalogProperty{
			Name:        name,
			Type:        propertyTypeLabel(prop),
			Description: prop.Description,
			Required:    required[name],
			Details:     propertyDetails(prop),
		})
	}

	return rows
}

func propertyTypeLabel(p schema.Property) string {
	if p.Type == schema.TypeArray && p.Items != nil {
		return "array of " + string(p.Items.Type)
	}

	return string(p.Type)
}

// propertyDetails concatenates enum values, format, default, and nullability
// into a compact, human-readable hint line.
func propertyDetails(p schema.Property) string {
	parts := make([]string, 0, 4)
	if len(p.Enum) > 0 {
		parts = append(parts, "enum: "+strings.Join(p.Enum, " | "))
	}

	if p.Format != "" {
		parts = append(parts, "format: "+p.Format)
	}

	if p.Default != "" {
		parts = append(parts, "default: "+p.Default)
	}

	if p.Nullable {
		parts = append(parts, "nullable")
	}

	return strings.Join(parts, " · ")
}

// prettySchema renders the schema as indented JSON for the detail page.
func prettySchema(s *schema.Schema) string {
	if s == nil {
		return ""
	}

	b, err := json.Marshal(s, jsontext.WithIndent("  "))
	if err != nil {
		return ""
	}

	return string(b)
}

// prettyExamples renders each stored example payload as indented JSON.
func prettyExamples(examples []jsontext.Value) []string {
	out := make([]string, 0, len(examples))
	for _, ex := range examples {
		if err := ex.Indent(jsontext.WithIndent("  ")); err != nil {
			continue
		}

		out = append(out, string(ex))
	}

	return out
}

func sortedKeys(m map[string]schema.Property) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
