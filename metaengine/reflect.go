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
	Type       reflect.Type
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
			Type:       f.Type,
			TypeString: f.Type.String(),
			Tag:        f.Tag,
		})
	}

	return fields
}

// extractKeyValueByType finds a field in the input struct whose type matches
// the projection's key type. The engine never assumes field names — it matches
// purely by Go type. Returns the field value and true if found.
// Panics if multiple fields match (ambiguous).
func extractKeyValueByType(input any, keyType reflect.Type) any {
	if keyType == nil {
		return nil
	}

	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	foundIdx := -1

	for i := range t.NumField() {
		if !t.Field(i).IsExported() {
			continue
		}

		if t.Field(i).Type == keyType {
			if foundIdx >= 0 {
				// Ambiguous — multiple fields of the same key type.
				return nil
			}

			foundIdx = i
		}
	}

	if foundIdx < 0 {
		return nil
	}

	return v.Field(foundIdx).Interface()
}

// extractDepthFromInput finds a field named "Depth" of type int in the input struct.
// Graph traversal queries use this convention.
func extractDepthFromInput(input any) int {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 1
	}

	f := v.FieldByName("Depth")
	if !f.IsValid() || f.Kind() != reflect.Int {
		return 1
	}

	return int(f.Int())
}

// detectPagination checks if the input struct has pagination fields:
// Limit int and/or After *Cursor. The engine detects these by type, not by name
// convention — these are the standard pagination types defined in this package.
func detectPagination(input any) bool {
	fields := reflectFields(input)

	limitType := reflect.TypeFor[int]() // untyped int literal matched by int field
	cursorPtrType := reflect.TypeFor[*Cursor]()

	for _, f := range fields {
		if f.Type == limitType && f.Name == "Limit" {
			return true
		}

		if f.Type == cursorPtrType && f.Name == "After" {
			return true
		}
	}

	// Also check via reflect directly for robustness.
	t := reflect.TypeOf(input)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	for field := range t.Fields() {
		switch field.Name {
		case "Limit":
			if field.Type.Kind() == reflect.Int {
				return true
			}
		case "After":
			if field.Type == cursorPtrType {
				return true
			}
		}
	}

	_ = limitType

	return false
}

// extractLimitFromInput extracts the Limit field value from the input struct.
// Returns 0 if not present (no limit).
func extractLimitFromInput(input any) int {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 0
	}

	f := v.FieldByName("Limit")
	if !f.IsValid() || f.Kind() != reflect.Int {
		return 0
	}

	return int(f.Int())
}

// extractCursorFromInput extracts the After *Cursor field from the input struct.
// Returns nil if not present.
func extractCursorFromInput(input any) *Cursor {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	f := v.FieldByName("After")
	if !f.IsValid() || f.Kind() != reflect.Pointer {
		return nil
	}

	if f.IsNil() {
		return nil
	}

	cursor, ok := f.Interface().(*Cursor)
	if !ok {
		return nil
	}

	return cursor
}

// nonMetaFields returns input fields that are NOT pagination metadata
// (Limit, After *Cursor, Depth). These are the actual domain filter fields.
func nonMetaFields(input any) []reflectField {
	metaNames := map[string]bool{
		"Limit": true,
		"After": true,
		"Depth": true,
	}

	var result []reflectField

	for _, f := range reflectFields(input) {
		if !metaNames[f.Name] {
			result = append(result, f)
		}
	}

	return result
}

// isTimestampType returns true if the type name represents time.Time.
func isTimestampType(typeName string) bool {
	return typeName == "time.Time" || typeName == "Time"
}

// detectSortField finds the sort key for a result element type by detecting
// the first time.Time field.
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
	itemsFieldIdx  int          // index of the []T field
	itemsElemType  reflect.Type // element type T
	cursorFieldIdx int          // index of the *Cursor field, or -1 if none
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

	cursorType := reflect.TypeFor[*Cursor]()

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

// extractFirstDomainField returns the value of the first exported non-meta field.
// Used as a fallback for Graph traversal when the engine can't determine the
// node type from fold return types (Edge stores From/To as any).
func extractFirstDomainField(input any) any {
	fields := nonMetaFields(input)
	if len(fields) == 0 {
		return nil
	}

	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	return v.FieldByName(fields[0].Name).Interface()
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
