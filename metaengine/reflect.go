package metaengine

import (
	"reflect"
)

// Pagination/metadata field names recognised on query input structs. Centralised
// as constants so the literal strings are not repeated (goconst) and the set of
// "meta" fields has a single source of truth.
const (
	limitField = "Limit"
	afterField = "After"
	depthField = "Depth"
)

func structValue(input any) (reflect.Value, bool) {
	v := reflect.ValueOf(input)
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, false
		}

		v = v.Elem()
	}

	return v, v.Kind() == reflect.Struct
}

func structType(input any) (reflect.Type, bool) {
	t := reflect.TypeOf(input)
	if t == nil {
		return nil, false
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t, t.Kind() == reflect.Struct
}

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
	t, ok := structType(v)
	if !ok {
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
// the projection's key type. The engine matches purely by Go type, not name.
func extractKeyValueByType(input any, keyType reflect.Type) any {
	if keyType == nil {
		return nil
	}

	v, ok := structValue(input)
	if !ok {
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
func extractDepthFromInput(input any) int {
	v, ok := structValue(input)
	if !ok {
		return 1
	}

	f := v.FieldByName("Depth")
	if !f.IsValid() || f.Kind() != reflect.Int {
		return 1
	}

	return int(f.Int())
}

// detectPagination checks if the input struct has pagination fields.
func detectPagination(input any) bool {
	t := reflect.TypeOf(input)
	if t == nil {
		return false
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	cursorPtrType := reflect.TypeFor[*Cursor]()

	for field := range t.Fields() {
		switch field.Name {
		case limitField:
			if field.Type.Kind() == reflect.Int {
				return true
			}
		case afterField:
			if field.Type == cursorPtrType {
				return true
			}
		}
	}

	return false
}

// extractLimitFromInput extracts the Limit field value from the input struct.
func extractLimitFromInput(input any) int {
	v, ok := structValue(input)
	if !ok {
		return 0
	}

	f := v.FieldByName(limitField)
	if !f.IsValid() || f.Kind() != reflect.Int {
		return 0
	}

	return int(f.Int())
}

// extractCursorFromInput extracts the After *Cursor field from the input struct.
func extractCursorFromInput(input any) *Cursor {
	v, ok := structValue(input)
	if !ok {
		return nil
	}

	f := v.FieldByName(afterField)
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

// nonMetaFields returns input fields that are NOT pagination metadata.
func nonMetaFields(input any) []reflectField {
	metaNames := map[string]bool{
		limitField: true,
		afterField: true,
		depthField: true,
	}

	var result []reflectField

	for _, f := range reflectFields(input) {
		if !metaNames[f.Name] {
			result = append(result, f)
		}
	}

	return result
}

// extractFirstDomainField returns the value of the first exported non-meta field.
func extractFirstDomainField(input any) any {
	fields := nonMetaFields(input)
	if len(fields) == 0 {
		return nil
	}

	v, ok := structValue(input)
	if !ok {
		return nil
	}

	return v.FieldByName(fields[0].Name).Interface()
}
