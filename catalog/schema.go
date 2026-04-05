package catalog

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

func SchemaFromType[T any]() *Schema {
	var zero T

	return schemaFromReflect(reflect.TypeOf(zero))
}

func schemaFromReflect(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: "null"}
	}

	if t.Kind() == reflect.Ptr {
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

	props := make(map[string]Property)

	var required []string

	for field := range t.Fields() {
		field := field
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

		prop := *propertyFromReflect(field.Type)

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

	return &Schema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func propertyFromReflect(t reflect.Type) *Property {
	if t == nil {
		return &Property{Type: "null"}
	}

	if t.Kind() == reflect.Ptr {
		return propertyFromReflect(t.Elem())
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return &Property{
			Type:  "array",
			Items: propertyFromReflect(t.Elem()),
		}
	}

	if t.Kind() == reflect.Map {
		return &Property{Type: "object"}
	}

	if t.Kind() == reflect.Struct {
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
		return nil, errors.New("schema is nil")
	}

	return json.MarshalIndent(schema, "", "  ")
}
