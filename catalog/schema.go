package catalog

//lint:exhaustruct This package creates partial schemas via helper functions.

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
)

const (
	jsonTypeString  = "string"
	jsonTypeObject  = "object"
	jsonTypeInteger = "integer"
	jsonTypeNumber  = "number"
	jsonTypeBoolean = "boolean"
	jsonTypeArray   = "array"
	jsonTypeNull    = "null"
)

// SchemaFromType generates a JSON Schema from the struct type T using reflection.
// It reads json, doc/description, and format struct tags.
func SchemaFromType[T any]() *Schema {
	var zero T

	return schemaFromReflect(reflect.TypeOf(zero))
}

// SchemaFromReflect generates a JSON Schema from a reflect.Type.
func SchemaFromReflect(t reflect.Type) *Schema {
	return schemaFromReflect(t)
}

func collectionSchema(t reflect.Type) *Property {
	if isCollectionKind(t.Kind()) {
		return &Property{
			Type:  "array",
			Items: propertyFromReflect(t.Elem()),
		}
	}

	return &Property{Type: jsonTypeObject}
}

func isCollectionKind(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

func propertyFromReflect(t reflect.Type) *Property {
	if t == nil {
		return &Property{Type: jsonTypeNull}
	}

	if t.Kind() == reflect.Pointer {
		return propertyFromReflect(t.Elem())
	}

	if isCollectionKind(t.Kind()) {
		return collectionSchema(t)
	}

	if t.Kind() == reflect.Map {
		return &Property{Type: jsonTypeObject}
	}

	if t.Kind() == reflect.Struct {
		if t == reflect.TypeFor[time.Time]() {
			return &Property{Type: jsonTypeString, Format: "date-time"}
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
		return jsonTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return jsonTypeInteger
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return jsonTypeInteger
	case reflect.Float32, reflect.Float64:
		return jsonTypeNumber
	case reflect.Bool:
		return jsonTypeBoolean
	case reflect.Interface:
		return jsonTypeObject
	case reflect.Complex64, reflect.Complex128:
		return jsonTypeString
	case reflect.Array, reflect.Slice:
		return jsonTypeArray
	case reflect.Map, reflect.Struct:
		return jsonTypeObject
	case reflect.Pointer:
		return jsonTypeObject
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return jsonTypeString
	case reflect.Invalid:
		return jsonTypeNull
	default:
		return jsonTypeString
	}
}

func parseJSONTag(tag string) (string, bool) {
	if tag == "" {
		return "", false
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	omit := len(parts) > 1 && parts[1] == "omitempty"

	return name, omit
}

// ErrNilSchema is returned when a nil schema is passed to SchemaToJSON.
var ErrNilSchema = errors.New("schema is nil")

// SchemaToJSON serializes a Schema to indented JSON.
func SchemaToJSON(schema *Schema) ([]byte, error) {
	if schema == nil {
		return nil, ErrNilSchema
	}

	//nolint:wrapcheck // MarshalIndent returns bytes, error from json.MarshalIndent
	return json.MarshalIndent(schema, "", "  ")
}
