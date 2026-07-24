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
// the projection's key type. The engine matches purely by Go type, not name.
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

	return false
}

// extractLimitFromInput extracts the Limit field value from the input struct.
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

// nonMetaFields returns input fields that are NOT pagination metadata.
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

// extractFirstDomainField returns the value of the first exported non-meta field.
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
