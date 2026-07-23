package metaengine

import (
	"reflect"
)

// qualifiedTypeName returns a package-qualified type name to prevent collisions
// between types with the same name from different packages.
func qualifiedTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t == nil {
		return ""
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if pkg := t.PkgPath(); pkg != "" {
		return pkg + "." + t.Name()
	}

	return t.Name()
}

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

// matchFilterFields finds fields shared by name + type between query input
// and result element type. Each matched field is a filter criterion.
func matchFilterFields(queryInput any, resultElementType any) []FieldPath {
	qFields := reflectFields(queryInput)
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
// Uses the first time.Time field as the default sort key.
func detectSortField(elementType any) string {
	fields := reflectFields(elementType)
	for _, f := range fields {
		if isTimestampType(f.TypeString) {
			return f.Name
		}
	}

	return ""
}

// colResultInfo describes a collection result type.
type colResultInfo struct {
	itemsFieldIdx  int           // index of the []T field
	itemsElemType  reflect.Type  // element type T
	cursorFieldIdx int           // index of the *Cursor field, or -1 if none
}

// collectionResultInfo inspects a result type R for collection characteristics.
// A collection result is a struct with at least one slice field ([]T).
// If a *Cursor field exists, it is used for pagination continuation.
func collectionResultInfo(t reflect.Type) (*colResultInfo, bool) {
	if t == nil || t.Kind() != reflect.Struct {
		return nil, false
	}

	info := &colResultInfo{cursorFieldIdx: -1}
	foundSlice := false

	cursorType := reflect.TypeOf((*Cursor)(nil))

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		if !foundSlice && f.Type.Kind() == reflect.Slice {
			info.itemsFieldIdx = i
			info.itemsElemType = f.Type.Elem()
			foundSlice = true
		}

		if info.cursorFieldIdx < 0 && f.Type == cursorType {
			info.cursorFieldIdx = i
		}
	}

	if !foundSlice {
		return nil, false
	}

	return info, true
}

func isCollectionResult[R any]() bool {
	var zero R
	_, ok := collectionResultInfo(reflect.TypeOf(zero))

	return ok
}

// reconstructCollection builds a typed collection result from []any returned
// by the engine. It fills in the slice field and optionally the cursor field.
func reconstructCollection[R any](raw any, limit int) R {
	var zero R

	t := reflect.TypeOf(zero)

	info, ok := collectionResultInfo(t)
	if !ok {
		return zero
	}

	items, ok := raw.([]any)
	if !ok {
		return zero
	}

	hasMore := limit > 0 && len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	slice := reflect.MakeSlice(reflect.SliceOf(info.itemsElemType), 0, len(items))

	var lastItem any

	for _, item := range items {
		if item == nil {
			continue
		}

		val := reflect.ValueOf(item)
		if val.Type().ConvertibleTo(info.itemsElemType) {
			slice = reflect.Append(slice, val.Convert(info.itemsElemType))
			lastItem = item
		}
	}

	result := reflect.New(t).Elem()
	result.Field(info.itemsFieldIdx).Set(slice)

	if hasMore && info.cursorFieldIdx >= 0 && lastItem != nil {
		cursor := &Cursor{Value: lastItem}
		result.Field(info.cursorFieldIdx).Set(reflect.ValueOf(cursor))
	}

	return result.Interface().(R)
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
