package metaengine

import (
	"reflect"
	"strings"
)

// ExtractFields pulls field values from a Go value (struct or map) for the
// planned columns. Missing fields produce nil (stored as NULL).
//
// Structs use a reflect fast path (no JSON marshal/unmarshal on writes).
// Maps and other types fall back to JSON round-trip.
func ExtractFields(value any, columns []PlannedColumn) map[string]any {
	result := make(map[string]any, len(columns))

	if m, ok := value.(map[string]any); ok {
		for _, c := range columns {
			for k, v := range m {
				if strings.EqualFold(k, c.Name) {
					result[c.Name] = v

					break
				}
			}
		}

		return result
	}

	rv := reflect.ValueOf(value)

	if rv.IsValid() && rv.Kind() == reflect.Struct {
		rt := rv.Type()

		for _, c := range columns {
			for i := range rt.NumField() {
				f := rt.Field(i)
				if !f.IsExported() {
					continue
				}

				fieldName := JSONFieldName(f)

				if strings.EqualFold(fieldName, c.Name) {
					result[c.Name] = rv.Field(i).Interface()

					break
				}
			}
		}

		return result
	}

	return result
}

// JSONFieldName returns the JSON field name for a struct field, respecting
// json tags. Falls back to the Go field name when no tag is present.
func JSONFieldName(f reflect.StructField) string {
	if tag := f.Tag.Get("json"); tag != "" {
		if name, _, _ := strings.Cut(tag, ","); name != "" {
			return name
		}
	}

	return f.Name
}

// PlansColumnCompatible checks whether two layout plans have matching
// column names (order-independent). Used to detect layout conflicts.
func PlansColumnCompatible(a, b LayoutPlan) bool {
	ac := a.ColumnNames()

	bc := b.ColumnNames()
	if len(ac) != len(bc) {
		return false
	}

	bset := make(map[string]bool, len(bc))
	for _, c := range bc {
		bset[c] = true
	}

	for _, c := range ac {
		if !bset[c] {
			return false
		}
	}

	return true
}
