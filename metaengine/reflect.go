package metaengine

import (
	"reflect"
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

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []reflectField

	for f := range t.Fields() {
		f := f
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

// isTimestampType returns true if the type name represents time.Time.
func isTimestampType(typeName string) bool {
	return typeName == "time.Time" || typeName == "Time"
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
// Detects by field shape (Items []T, Next *Cursor, HasMore bool), not by name,
// because Go reflect returns "Page[pkg.Type]" as the Name for generic instantiations.
func unwrapPageType(resultType reflect.Type) (reflect.Type, bool) {
	if resultType == nil || resultType.Kind() != reflect.Struct {
		return nil, false
	}

	if resultType.NumField() < 1 {
		return nil, false
	}

	itemsField := resultType.Field(0)
	if itemsField.Name != "Items" || itemsField.Type.Kind() != reflect.Slice {
		return nil, false
	}
	// Verify it's our Page by checking remaining fields match the shape.
	if resultType.NumField() < 3 {
		return nil, false
	}

	if resultType.Field(1).Name != "Next" || resultType.Field(2).Name != "HasMore" {
		return nil, false
	}

	return itemsField.Type.Elem(), true
}

// getFieldValue extracts a field value from a struct by name using reflection.
func getFieldValue(v any, fieldName string) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		return nil
	}

	return f.Interface()
}

