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
		return filterValuesEqual(actual, expected)
	case FilterNe:
		return !filterValuesEqual(actual, expected)
	case FilterLt:
		return compareValue(actual, expected) < 0
	case FilterLe:
		return compareValue(actual, expected) <= 0
	case FilterGt:
		return compareValue(actual, expected) > 0
	case FilterGe:
		return compareValue(actual, expected) >= 0
	case FilterIn:
		values, ok := expected.([]any)
		if !ok {
			return false
		}

		for _, v := range values {
			if filterValuesEqual(actual, v) {
				return true
			}
		}

		return false
	default:
		return false
	}
}

// filterValuesEqual compares a stored field value against a filter value by
// VALUE, not Go type identity. Engines round-trip rows through JSON where
// named types (e.g. `type Status string`) vanish, so a filter written
// against a typed enum field must match both the typed form (memory engine)
// and the decoded primitive form (JSON-backed engines). DeepEqual alone
// makes the same query engine-dependent.
func filterValuesEqual(actual, expected any) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}

	av, ev := reflect.ValueOf(actual), reflect.ValueOf(expected)
	if !av.IsValid() || !ev.IsValid() {
		return false
	}

	if av.Kind() == reflect.String && ev.Kind() == reflect.String {
		return av.String() == ev.String()
	}

	if isNumericKind(av.Kind()) && isNumericKind(ev.Kind()) {
		return compareValue(actual, expected) == 0
	}

	return false
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// matchFilter evaluates whether itemVal satisfies the given FilterOp against
// expected. Used by the closure-fallback filter path for declarative filters
// (FilterOnField) on engines that don't support pushdown.
func matchFilter(itemVal any, op FilterOp, expected any) bool {
	switch op {
	case FilterEq:
		return filterValuesEqual(itemVal, expected)
	case FilterNe:
		return !filterValuesEqual(itemVal, expected)
	case FilterLt, FilterLe, FilterGt, FilterGe:
		if itemVal == nil {
			return false
		}

		cmp := compareValue(itemVal, expected)

		return switchCompare(cmp, op)
	default:
		return false
	}
}

func switchCompare(cmp int, op FilterOp) bool {
	switch op {
	case FilterLt:
		return cmp < 0
	case FilterLe:
		return cmp <= 0
	case FilterGt:
		return cmp > 0
	case FilterGe:
		return cmp >= 0
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
