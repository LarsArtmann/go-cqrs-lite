package metaengine

import (
	"cmp"
	"fmt"
	"reflect"
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

// filterPredicate is a runtime filter test against a result item.
type filterPredicate struct {
	expected any
	test     func(item any) bool
}

// passesFilterSpecs evaluates declarative FilterSpec predicates against a scan
// row. Unlike passesFilters (which uses closure-based filterPredicate), this
// works with declarative FilterSpec values (Column + Op + Value) used by
// TypedReader.Scan when falling back to ScanBackend.
func passesFilterSpecs(item any, specs []FilterSpec) bool {
	for _, spec := range specs {
		actual := itemFieldByName(item, spec.Column)

		if !evalFilterOp(spec.Op, actual, spec.Value) {
			return false
		}
	}

	return true
}

// evalFilterOp evaluates a single filter comparison against an actual value.
func evalFilterOp(op FilterOp, actual, expected any) bool {
	switch op {
	case FilterEq:
		return reflect.DeepEqual(actual, expected)
	case FilterNe:
		return !reflect.DeepEqual(actual, expected)
	case FilterLt:
		return compareValue(actual, expected) < 0
	case FilterLe:
		return compareValue(actual, expected) <= 0
	case FilterGt:
		return compareValue(actual, expected) > 0
	case FilterGe:
		return compareValue(actual, expected) >= 0
	default:
		return false
	}
}

// compareValue performs a type-aware tri-state comparison: -1 (a < b), 0 (equal), +1 (a > b).
// Handles same-type comparison, cross-type numeric comparison (e.g., int from item
// vs float64 from a deserialized cursor), and falls back to string comparison.
func compareValue(left, right any) int {
	if left == nil || right == nil {
		if left == right {
			return 0
		}

		if left == nil {
			return -1
		}

		return 1
	}

	if result, ok := tryNumericCompare(left, right); ok {
		return result
	}

	switch vLeft := left.(type) {
	case string:
		if vRight, ok := right.(string); ok {
			return cmp.Compare(vLeft, vRight)
		}
	case time.Time:
		if vRight, ok := right.(time.Time); ok {
			return vLeft.Compare(vRight)
		}
	}

	return strings.Compare(fmt.Sprintf("%v", left), fmt.Sprintf("%v", right))
}

// tryNumericCompare attempts to compare two values as float64 when their Go types
// differ (e.g., int from an item vs float64 from a deserialized cursor).
func tryNumericCompare(left, right any) (int, bool) {
	fLeft, okLeft := toFloat64(left)

	fRight, okRight := toFloat64(right)
	if !okLeft || !okRight {
		return 0, false
	}

	return cmp.Compare(fLeft, fRight), true
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
