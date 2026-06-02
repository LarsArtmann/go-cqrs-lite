//lint:exhaustruct This package creates partial schemas via helper functions.

package schema

import (
	"cmp"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

var ErrNilSchema = errorfamily.NewRejection("catalog.nil_schema", "schema is nil")

const jsonKeyType = "type"

func FromType[T any]() *Schema {
	var zero T

	return fromReflect(reflect.TypeOf(zero))
}

func FromReflect(t reflect.Type) *Schema {
	return fromReflect(t)
}

func ToJSON(s *Schema) ([]byte, error) {
	if s == nil {
		return nil, ErrNilSchema
	}

	//nolint:wrapcheck // MarshalIndent returns bytes, error from json.MarshalIndent
	return json.MarshalIndent(s, "", "  ")
}

func ToAny(s *Schema) (any, error) {
	if s == nil {
		return nil, ErrNilSchema
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal schema to JSON: %w", err)
	}

	var result any

	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshal schema to any: %w", err)
	}

	return result, nil
}

var schemaCache sync.Map

func fromReflect(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: TypeNull}
	}

	if cached, ok := schemaCache.Load(t); ok {
		return cached.(*Schema)
	}

	s := buildSchema(t)
	schemaCache.Store(t, s)

	return s
}

func buildSchema(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{Type: TypeNull}
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if isCollectionKind(t.Kind()) {
		return &Schema{
			Type:  TypeArray,
			Items: propertyFromReflect(t.Elem()),
		}
	}

	if t.Kind() == reflect.Map {
		return &Schema{
			Type: TypeObject,
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
		return &Schema{Type: TypeString}
	}

	return structSchema(t)
}

func structSchema(t reflect.Type) *Schema {
	props := make(map[string]Property)

	var required []string

	for field := range t.Fields() {
		name, prop, omit, include := fieldToProperty(field)
		if !include {
			continue
		}

		props[name] = prop

		if !omit {
			required = append(required, name)
		}
	}

	return &Schema{
		Type:       TypeObject,
		Properties: props,
		Required:   required,
	}
}

func fieldToProperty(field reflect.StructField) (string, Property, bool, bool) {
	if !field.IsExported() || field.Anonymous {
		return "", Property{}, false, false
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		return "", Property{}, false, false
	}

	name, omit := parseJSONTag(jsonTag)
	name = cmp.Or(name, field.Name)

	prop := *propertyFromReflect(field.Type)

	prop.Description = tagValue(field, "doc", "description")

	if v := field.Tag.Get("format"); v != "" {
		prop.Format = v
	}

	prop.Default = field.Tag.Get("default")

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

	return name, prop, omit, true
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

		s := fromReflect(t)

		return &Property{
			Type:       s.Type,
			Properties: s.Properties,
			Required:   s.Required,
		}
	}

	return &Property{Type: goTypeToJSON(t.Kind())}
}

func goTypeToJSON(k reflect.Kind) Type {
	//nolint:exhaustive // default handles remaining kinds
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

func tagValue(field reflect.StructField, tags ...string) string {
	for _, tag := range tags {
		if v := field.Tag.Get(tag); v != "" {
			return v
		}
	}

	return ""
}
