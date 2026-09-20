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
//
// Column names may come from EITHER naming world: the Go field name
// (`FilterOnField[R]("ParentID", …)` plans a column named ParentID) or the
// JSON tag name (`parent_id`, what the stored document carries). A column
// therefore matches a field when its json tag matches (case-insensitive)
// OR the Go field name matches; map keys additionally fall back to the
// snake_case form of the column name. Without this, camelCase filterable
// fields over snake_case json tags extracted as NULL and every pushdown
// filter silently matched nothing (2026-09-18 Ledger CRM).
func ExtractFields(value any, columns []PlannedColumn) map[string]any {
	result := make(map[string]any, len(columns))

	if m, ok := value.(map[string]any); ok {
		for _, c := range columns {
			for k, v := range m {
				if strings.EqualFold(k, c.Name) || strings.EqualFold(k, snakeCase(c.Name)) {
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

				if strings.EqualFold(JSONFieldName(f), c.Name) || strings.EqualFold(f.Name, c.Name) {
					result[c.Name] = rv.Field(i).Interface()

					break
				}
			}
		}

		return result
	}

	return result
}

// snakeCase converts a Go identifier to its conventional snake_case json
// key form, keeping acronym runs intact: an underscore is inserted only at
// word boundaries — before an uppercase that follows a lowercase/digit, or
// before an uppercase run's last letter when a lowercase follows it
// (ParentID -> parent_id, DueAt -> due_at, URLKey -> url_key).
func snakeCase(name string) string {
	runes := []rune(name)
	var b strings.Builder
	b.Grow(len(name) + 4)

	for i, r := range runes {
		if r < 'A' || r > 'Z' {
			b.WriteRune(r)

			continue
		}

		lower := r - 'A' + 'a'
		prevLower := i > 0 && (runes[i-1] < 'A' || runes[i-1] > 'Z')
		nextLower := i+1 < len(runes) && (runes[i+1] < 'A' || runes[i+1] > 'Z')

		switch {
		case i == 0:
			b.WriteRune(lower)
		case prevLower, nextLower:
			b.WriteByte('_')
			b.WriteRune(lower)
		default: // inside an acronym run, not at its tail
			b.WriteRune(lower)
		}
	}

	return b.String()
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
