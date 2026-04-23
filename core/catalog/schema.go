package catalog

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
)

func SchemaFromType[T any]() *Schema {
	var zero T

	return schemaFromReflect(reflect.TypeOf(zero))
}

// SchemaFromReflect creates a schema from a reflect.Type at runtime.
// Useful for adapters that extract schemas from interface-typed values.
func SchemaFromReflect(t reflect.Type) *Schema {
	return schemaFromReflect(t)
}

func schemaFromReflect(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: "null"}
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return &Schema{
			Type:  "array",
			Items: propertyFromReflect(t.Elem()),
		}
	}

	if t.Kind() == reflect.Map {
		return &Schema{
			Type: "object",
			Properties: map[string]Property{
				"(key)":   *propertyFromReflect(t.Key()),
				"(value)": *propertyFromReflect(t.Elem()),
			},
		}
	}

	if t.Kind() != reflect.Struct {
		return &Schema{Type: goTypeToJSON(t.Kind())}
	}

	if t == reflect.TypeFor[time.Time]() {
		return &Schema{Type: "string", Properties: map[string]Property{}}
	}

	props := make(map[string]Property)

	var required []string

	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}

		if field.Anonymous {
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

		prop := *propertyFromReflect(field.Type)

		prop.Description = field.Tag.Get("doc")
		if prop.Description == "" {
			prop.Description = field.Tag.Get("description")
		}

		if format := field.Tag.Get("format"); format != "" {
			prop.Format = format
		}

		if v := field.Tag.Get("default"); v != "" {
			prop.Default = v
		}

		if v := field.Tag.Get("enum"); v != "" {
			prop.Enum = strings.Split(v, ",")
		}

		if _, ok := field.Tag.Lookup("nullable"); ok {
			prop.Nullable = true
		}

		if _, ok := field.Tag.Lookup("deprecated"); ok {
			prop.Deprecated = true
		}

		if v := field.Tag.Get("pattern"); v != "" {
			prop.Pattern = v
		}

		props[name] = prop
		if !omit {
			required = append(required, name)
		}
	}

	return &Schema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func collectionSchema(t reflect.Type) *Property {
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return &Property{
			Type:  "array",
			Items: propertyFromReflect(t.Elem()),
		}
	}

	return &Property{Type: "object"}
}

func propertyFromReflect(t reflect.Type) *Property {
	if t == nil {
		return &Property{Type: "null"}
	}

	if t.Kind() == reflect.Pointer {
		return propertyFromReflect(t.Elem())
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return collectionSchema(t)
	}

	if t.Kind() == reflect.Map {
		return &Property{Type: "object"}
	}

	if t.Kind() == reflect.Struct {
		if t == reflect.TypeFor[time.Time]() {
			return &Property{Type: "string", Format: "date-time"}
		}

		schema := schemaFromReflect(t)

		return &Property{
			Type:       schema.Type,
			Properties: schema.Properties,
			Required:   schema.Required,
		}
	}

	return &Property{Type: goTypeToJSON(t.Kind())}
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
	case reflect.Interface:
		return "object"
	default:
		return "string"
	}
}

func parseJSONTag(tag string) (name string, omit bool) {
	if tag == "" {
		return "", false
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	omit = len(parts) > 1 && parts[1] == "omitempty"

	return name, omit
}

func SchemaToJSON(schema *Schema) ([]byte, error) {
	if schema == nil {
		//nolint:err113 // nil check must return specific error
		return nil, errors.New("schema is nil")
	}

	//nolint:wrapcheck // MarshalIndent returns bytes, error from json.MarshalIndent
	return json.MarshalIndent(schema, "", "  ")
}
