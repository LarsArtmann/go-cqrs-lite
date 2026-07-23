package metaengine

import (
	"fmt"
	"reflect"
	"strings"
)

// reflectField is a simplified view of a struct field for type inference.
type reflectField struct {
	Name       string
	TypeString string
	Tag        reflect.StructTag
}

// reflectFields extracts the exported fields of a struct instance or pointer.
func reflectFields(v any) []reflectField {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var fields []reflectField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fields = append(fields, reflectField{
			Name:       f.Name,
			TypeString: f.Type.String(),
			Tag:        f.Tag,
		})
	}
	return fields
}

// domainFields returns ALL exported fields of a query input struct.
// Since pagination is now encoded in Page[T] (not in the query input),
// every field in the query input IS a domain filter criterion.
func domainFields(v any) []reflectField {
	return reflectFields(v)
}

// matchFilterFields finds fields shared by name + type between query input
// and result element type. The query input carries ONLY domain intent now —
// every matched field is a filter criterion.
func matchFilterFields(queryInput any, resultElementType any) []FieldPath {
	qFields := domainFields(queryInput)
	rFields := reflectFields(resultElementType)
	rByName := make(map[string]reflectField, len(rFields))
	for _, f := range rFields {
		rByName[f.Name] = f
	}
	var matched []FieldPath
	structName := reflect.TypeOf(resultElementType).Name()
	for _, qf := range qFields {
		if rf, ok := rByName[qf.Name]; ok && rf.TypeString == qf.TypeString {
			matched = append(matched, FieldPath{
				Struct: structName,
				Field:  qf.Name,
				GoType: qf.TypeString,
			})
		}
	}
	return matched
}

// detectSortField finds the sort key for a result element type.
// Priority: struct tag `metaengine:"sort"` > first time.Time field > none.
func detectSortField(elementType any) string {
	fields := reflectFields(elementType)
	for _, f := range fields {
		if tag := f.Tag.Get("metaengine"); tag == "sort" {
			return f.Name
		}
	}
	for _, f := range fields {
		if isTimestampType(f.TypeString) {
			return f.Name
		}
	}
	return ""
}

// unwrapPageType extracts the element type T from a Page[T].
// Returns nil if the type is not a Page.
func unwrapPageType(resultType reflect.Type) (reflect.Type, bool) {
	if resultType.Kind() != reflect.Struct {
		return nil, false
	}
	if resultType.Name() != "Page" || resultType.PkgPath() == "" {
		// Check if it embeds Page[T]
		for i := 0; i < resultType.NumField(); i++ {
			f := resultType.Field(i)
			if f.Type.Name() == "Page" && f.Type.PkgPath() != "" && strings.HasSuffix(f.Type.PkgPath(), "/metaengine/v4") {
				if f.Type.Kind() == reflect.Struct && f.Type.NumField() > 0 {
					gf := f.Type.Field(0)
					if gf.Name == "Items" {
						if gf.Type.Kind() == reflect.Slice {
							return gf.Type.Elem(), true
						}
					}
				}
			}
		}
		return nil, false
	}
	if resultType.NumField() > 0 && resultType.Field(0).Name == "Items" {
		itemsField := resultType.Field(0)
		if itemsField.Type.Kind() == reflect.Slice {
			return itemsField.Type.Elem(), true
		}
	}
	return nil, false
}

// describeType returns a human-readable description of a type for diagnostics.
func describeType(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	if t.Name() != "" && t.Name() != t.Kind().String() {
		return fmt.Sprintf("%s (%s)", t.Name(), t.Kind())
	}
	return t.String()
}
