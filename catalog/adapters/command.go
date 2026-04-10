package adapters

import (
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/command"
)

// AddCommand registers a command instance with the catalog builder.
// The command must implement command.Catalogable (embed command.CatalogCore).
// The schema is auto-extracted from the command's payload fields via reflection.
// Use this when your commands embed command.CatalogCore.
func (b *CatalogBuilder) AddCommand(serviceID string, cmd command.Catalogable) {
	meta := cmd.CatalogInfo()
	schema := extractCommandSchema(cmd)

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        string(cmd.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Receives,
	}

	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

// AddCommandWithSchema registers a command with an explicit schema override.
// Use this when you want to provide a custom schema instead of auto-extracting.
func (b *CatalogBuilder) AddCommandWithSchema(
	serviceID string,
	cmd command.Catalogable,
	schema *catalog.Schema,
) {
	meta := cmd.CatalogInfo()

	msg := catalog.Message{
		Kind:      catalog.CommandMessage,
		ID:        string(cmd.Type()),
		Name:      meta.Name,
		Version:   meta.Version,
		Summary:   meta.Summary,
		Schema:    schema,
		Direction: catalog.Receives,
	}

	b.addMessageToService(serviceID, catalog.CommandMessage, msg)
}

func extractCommandSchema(cmd command.Catalogable) *catalog.Schema {
	t := reflect.TypeOf(cmd).Elem()
	return schemaFromReflect(t)
}

func schemaFromReflect(t reflect.Type) *catalog.Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	props := make(map[string]catalog.Property)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name, omit := parseJSONTag(jsonTag)
		if name == "" {
			name = field.Name
		}

		prop := catalog.Property{Type: goTypeToJSON(field.Type.Kind())}
		prop.Description = field.Tag.Get("doc")
		if prop.Description == "" {
			prop.Description = field.Tag.Get("description")
		}

		if format := field.Tag.Get("format"); format != "" {
			prop.Format = format
		}

		props[name] = prop
		if !omit {
			required = append(required, name)
		}
	}

	return &catalog.Schema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func parseJSONTag(tag string) (name string, omit bool) {
	if tag == "" {
		return "", false
	}
	parts := splitTag(tag)
	return parts[0], len(parts) > 1 && parts[1] == "omitempty"
}

func splitTag(tag string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			parts = append(parts, tag[start:i])
			start = i + 1
		}
	}
	parts = append(parts, tag[start:])
	return parts
}

func goTypeToJSON(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return "object"
	}
}
