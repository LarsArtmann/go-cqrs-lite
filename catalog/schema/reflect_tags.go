//lint:exhaustruct This package creates partial schemas via helper functions.

package schema

import (
	"cmp"
	"reflect"
	"strings"
)

const (
	paramQuery  = "query"
	paramPath   = "path"
	paramHeader = "header"
	paramCookie = "cookie"
)

// fieldToProperty converts a struct field into its JSON-Schema property name,
// the Property value, whether the field is omittable, whether it should be
// included in the parent schema, and an optional HTTP Parameter (populated
// when the field carries a query/path/header/cookie tag).
//
// The (name, include) return pair lets callers skip fields tagged `json:"-"`
// without a parameter location, while still emitting a Parameter for fields
// that combine `json:"-"` with a parameter tag (a common idiom for
// request-only fields that should not appear in the body schema).
func fieldToProperty(field reflect.StructField) (string, Property, bool, bool, Parameter) {
	if !field.IsExported() || field.Anonymous {
		return "", Property{}, false, false, Parameter{}
	}

	paramLoc, paramName := readParamLocation(field)

	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		if paramLoc != "" {
			pName := cmp.Or(paramName, field.Name)
			prop := *propertyFromReflect(field.Type)

			return "", Property{}, false, false, Parameter{
				Name:        pName,
				In:          paramLoc,
				Description: tagValue(field, "doc", "description"),
				Required:    paramLoc == paramPath,
				Schema:      &prop,
			}
		}

		return "", Property{}, false, false, Parameter{}
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

	if paramLoc != "" {
		pName := cmp.Or(paramName, name)

		return name, prop, omit, true, Parameter{
			Name:        pName,
			In:          paramLoc,
			Description: prop.Description,
			Required:    paramLoc == paramPath || !omit,
			Schema:      &prop,
		}
	}

	return name, prop, omit, true, Parameter{}
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

func readParamLocation(field reflect.StructField) (string, string) {
	for _, loc := range [...]struct{ tag, value string }{
		{paramQuery, paramQuery},
		{paramPath, paramPath},
		{paramHeader, paramHeader},
		{paramCookie, paramCookie},
	} {
		if v := field.Tag.Get(loc.tag); v != "" {
			return loc.value, strings.Split(v, ",")[0]
		}
	}

	return "", ""
}
