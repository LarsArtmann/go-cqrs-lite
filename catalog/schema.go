package catalog

//lint:exhaustruct This package creates partial schemas via helper functions.

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
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
			Type:  TypeArray,
			Items: propertyFromReflect(t.Elem()),
		}
	}

	return &Property{Type: TypeObject}
}

func isCollectionKind(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

func propertyFromReflect(t reflect.Type) *Property {
	if t == nil {
		return &Property{Type: TypeNull}
	}

	if t.Kind() == reflect.Pointer {
		return propertyFromReflect(t.Elem())
	}

	if isCollectionKind(t.Kind()) {
		return collectionSchema(t)
	}

	if t.Kind() == reflect.Map {
		return &Property{Type: TypeObject}
	}

	if t.Kind() == reflect.Struct {
		if t == reflect.TypeFor[time.Time]() {
			return &Property{Type: TypeString, Format: "date-time"}
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

func goTypeToJSON(k reflect.Kind) SchemaType {
	switch k {
	case reflect.String:
		return TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInteger
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return TypeInteger
	case reflect.Float32, reflect.Float64:
		return TypeNumber
	case reflect.Bool:
		return TypeBoolean
	case reflect.Interface:
		return TypeObject
	case reflect.Complex64, reflect.Complex128:
		return TypeString
	case reflect.Array, reflect.Slice:
		return TypeArray
	case reflect.Map, reflect.Struct:
		return TypeObject
	case reflect.Pointer:
		return TypeObject
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return TypeString
	case reflect.Invalid:
		return TypeNull
	default:
		return TypeString
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
var ErrNilSchema = errorfamily.NewRejection("catalog.nil_schema", "schema is nil")

// SchemaToJSON serializes a Schema to indented JSON.
func SchemaToJSON(schema *Schema) ([]byte, error) {
	if schema == nil {
		return nil, ErrNilSchema
	}

	//nolint:wrapcheck // MarshalIndent returns bytes, error from json.MarshalIndent
	return json.MarshalIndent(schema, "", "  ")
}
