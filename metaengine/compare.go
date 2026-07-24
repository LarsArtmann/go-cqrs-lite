package metaengine

import (
	"cmp"
	"fmt"
	"strings"
	"time"
)

// passesFilters checks if a value passes all filter predicates.
// Each predicate is a runtime closure from FilterOn — no field name strings.
func passesFilters(value any, filters []filterPredicate) bool {
	if len(filters) == 0 {
		return true
	}

	for _, f := range filters {
		if !f.test(value) {
			return false
		}
	}

	return true
}

// compareValue performs a type-aware tri-state comparison: -1 (a < b), 0 (equal), +1 (a > b).
// Handles same-type comparison, cross-type numeric comparison (e.g., int from item
// vs float64 from a deserialized cursor), and falls back to string comparison.
func compareValue(a, b any) int {
	if a == nil || b == nil {
		if a == b {
			return 0
		}

		if a == nil {
			return -1
		}

		return 1
	}

	if result, ok := tryNumericCompare(a, b); ok {
		return result
	}

	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return cmp.Compare(va, vb)
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			return va.Compare(vb)
		}
	}

	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// tryNumericCompare attempts to compare two values as float64 when their Go types
// differ (e.g., int from an item vs float64 from a deserialized cursor).
func tryNumericCompare(a, b any) (int, bool) {
	fa, okA := toFloat64(a)

	fb, okB := toFloat64(b)
	if !okA || !okB {
		return 0, false
	}

	return cmp.Compare(fa, fb), true
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
