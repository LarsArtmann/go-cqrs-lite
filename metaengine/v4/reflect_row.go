package metaengine

import (
	"strings"
)

// itemFieldByName extracts a named field from a scan row, which may be a
// map[string]any (SQL engines) or a struct (memory engine). Used by the
// closure-fallback path to evaluate declarative FilterOnField predicates.
func itemFieldByName(item any, name string) any {
	if m, ok := item.(map[string]any); ok {
		for k, v := range m {
			if strings.EqualFold(k, name) {
				return v
			}
		}

		return nil
	}

	return extractValueByName(item, name)
}

// isRowValue reports whether v is a scan row shape — a map[string]any (SQL
// engines) or a struct (memory engine) — rather than a bare scalar. Cursor
// comparisons use it to distinguish "row that lacks the sort field" (keep the
// nil extraction) from "bare scalar cursor operand" (compare the scalar
// directly).
func isRowValue(v any) bool {
	if _, ok := v.(map[string]any); ok {
		return true
	}

	_, ok := structValue(v)

	return ok
}
