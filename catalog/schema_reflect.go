//lint:exhaustruct This package creates partial schemas via helper functions.
package catalog

import (
	"reflect"
	"strings"
	"time"
)

func schemaFromReflect(t reflect.Type) *Schema {
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
	if name == "" {
		name = field.Name
	}

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

func tagValue(field reflect.StructField, tags ...string) string {
	for _, tag := range tags {
		if v := field.Tag.Get(tag); v != "" {
			return v
		}
	}

	return ""
}
