package adttest

import (
	"testing"
)

// paginationRowFields extracts the embedded map key and sort value from a
// returned row. Memory engines return the stored Go values (int sort fields);
// SQL and KV engines return decoded JSON (float64 sort fields).
func paginationRowFields(item any) (key string, sortVal float64, ok bool) {
	m, isMap := item.(map[string]any)
	if !isMap {
		return "", 0, false
	}

	k, isStr := m["key"].(string)
	if !isStr {
		return "", 0, false
	}

	switch sv := m["sort"].(type) {
	case int:
		return k, float64(sv), true
	case int64:
		return k, float64(sv), true
	case float64:
		return k, sv, true
	default:
		return "", 0, false
	}
}

// paginationSortFn builds the numeric sort comparator handed to MapScan. The
// comparator accepts both rows (maps with a numeric "sort" field) and bare
// cursor sort values (engines call sortFn(item, cursor.Sort) with the raw
// field value), and normalizes int vs float64 across engine families.
func paginationSortFn(t *testing.T) func(a, b any) int {
	return func(a, b any) int {
		av, aok := paginationSortableValue(a)
		bv, bok := paginationSortableValue(b)
		if !aok || !bok {
			t.Fatalf("pagination conformance: unsortable values %#v / %#v", a, b)
			return 0
		}

		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		default:
			return 0
		}
	}
}

func paginationSortableValue(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case map[string]any:
		if _, sv, ok := paginationRowFields(x); ok {
			return sv, true
		}
	}

	return 0, false
}
