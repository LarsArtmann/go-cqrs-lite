package architecture

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// attributeTable renders a schema's top-level properties as a table.
func attributeTable(schema *catalog.Schema) markup {
	if schema == nil || len(schema.Properties) == 0 {
		return ""
	}

	rows := make([][]tableCell, 0, len(schema.Properties))

	for _, name := range sortedFieldNames(schema.Properties) {
		prop := schema.Properties[name]
		rows = append(rows, []tableCell{
			monoCell(name),
			monoCell(typeLabel(prop)),
			plainCell(requiredLabel(schema.Required, name)),
			plainCell(prop.Description),
		})
	}

	return writeTable([]string{headerField, headerType, headerRequired, headerDescription}, rows)
}

// sortedKeysOf returns a map's keys in stable order.
func sortedKeysOf[V any](values map[string]V) []string {
	return slices.Sorted(maps.Keys(values))
}

// sortedFieldNames returns the property names in stable order.
func sortedFieldNames(props map[string]catalog.Property) []string {
	return sortedKeysOf(props)
}

// typeLabel renders a property's declared type, array-ness and format.
func typeLabel(prop catalog.Property) string {
	label := string(prop.Type)

	if prop.Items != nil {
		label += "[]"
	}

	if prop.Format != "" {
		label += " (" + prop.Format + ")"
	}

	if prop.Nullable {
		label += ", nullable"
	}

	return label
}

// requiredLabel reports whether a field is required.
func requiredLabel(required []string, name string) string {
	if slices.Contains(required, name) {
		return "yes"
	}

	return ""
}

// typeScriptInterface renders a schema as a TypeScript interface.
func typeScriptInterface(name string, schema *catalog.Schema) string {
	if schema == nil || len(schema.Properties) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("/** " + name + " */\nexport interface " + exportName(name) + " {\n")

	for _, field := range sortedFieldNames(schema.Properties) {
		prop := schema.Properties[field]

		optional := ""
		if requiredLabel(schema.Required, field) == "" {
			optional = "?"
		}

		b.WriteString("  " + tsField(field) + optional + ": " + tsType(prop) + ";\n")
	}

	b.WriteString("}\n")

	return b.String()
}

// exportName turns a display name into a TypeScript-friendly identifier.
func exportName(name string) string {
	var b strings.Builder

	upperNext := true

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			if upperNext && r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}

			upperNext = false

			b.WriteRune(r)
		default:
			upperNext = true
		}
	}

	if b.Len() == 0 {
		return "Payload"
	}

	return b.String()
}

// tsField quotes field names that are not valid identifiers.
func tsField(field string) string {
	for _, r := range field {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '$':
		default:
			return "'" + strings.ReplaceAll(field, "'", `\'`) + "'"
		}
	}

	return field
}

// tsType maps a schema property to a TypeScript type.
func tsType(prop catalog.Property) string {
	if prop.Nullable {
		return tsType(
			catalog.Property{Type: prop.Type, Items: prop.Items, Properties: prop.Properties},
		) + " | null"
	}

	switch string(prop.Type) {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if prop.Items != nil {
			return tsType(*prop.Items) + "[]"
		}

		return "unknown[]"
	case "object":
		return tsObject(prop)
	default:
		return "unknown"
	}
}

// tsObject renders a nested object literal for a property.
func tsObject(prop catalog.Property) string {
	if len(prop.Properties) == 0 {
		return "Record<string, unknown>"
	}

	parts := make([]string, 0, len(prop.Properties))

	for _, field := range sortedFieldNames(prop.Properties) {
		nested := prop.Properties[field]

		optional := ""
		if requiredLabel(prop.Required, field) == "" {
			optional = "?"
		}

		parts = append(parts, tsField(field)+optional+": "+tsType(nested))
	}

	return "{ " + strings.Join(parts, "; ") + " }"
}

// prettyJSON re-indents a JSON document, falling back to the raw text.
func prettyJSON(raw []byte) string {
	var buf bytes.Buffer

	if err := func() error {
		out, err := jsontext.AppendFormat(nil, raw, jsontext.WithIndent("  "))
		(&buf).Write(out)

		return err
	}(); err != nil {
		return string(raw)
	}

	return buf.String()
}

// schemaJSON renders a schema as the raw JSON Schema document.
func schemaJSON(schema *catalog.Schema) string {
	if schema == nil {
		return ""
	}

	raw, err := catalog.SchemaToJSON(schema)
	if err != nil {
		return ""
	}

	return prettyJSON(raw)
}

// examplePayload renders the most specific example available for a message,
// synthesising one from the schema when the catalog carries none.
func examplePayload(schema *catalog.Schema, examples []jsontext.Value) string {
	if len(examples) > 0 {
		return prettyJSON(examples[0])
	}

	if schema == nil {
		return ""
	}

	if len(schema.Examples) > 0 {
		return prettyJSON(schema.Examples[0])
	}

	synth := synthesize(schema)
	if synth == "" {
		return ""
	}

	return synth
}

// synthesize builds a placeholder payload from a schema's structure.
func synthesize(schema *catalog.Schema) string {
	if len(schema.Properties) == 0 {
		return ""
	}

	value := make(map[string]any, len(schema.Properties))

	for _, field := range sortedFieldNames(schema.Properties) {
		value[field] = sampleValue(schema.Properties[field])
	}

	raw, err := json.Marshal(value, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		return ""
	}

	return string(raw)
}

// sampleValue returns a zero-ish value for a property, used only to show the
// shape of a payload the catalog does not exemplify.
func sampleValue(prop catalog.Property) any {
	switch string(prop.Type) {
	case "string":
		return defaultIfEmpty(prop.Default, "string")
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []any{}
	case "object":
		nested := make(map[string]any, len(prop.Properties))

		for _, field := range sortedFieldNames(prop.Properties) {
			nested[field] = sampleValue(prop.Properties[field])
		}

		return nested
	default:
		return nil
	}
}
