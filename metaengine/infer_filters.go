package metaengine

import (
	"reflect"
	"strings"
)

// filterPrefix maps a query-input field-name prefix to a FilterOp and the
// convention for stripping it. When a query input field doesn't exactly match
// a result field but starts with one of these prefixes followed by an
// uppercase letter, the operator is inferred and the remainder is matched
// against the result type.
//
// Example: input field "MinScore" → FilterGe on result field "Score".
var filterPrefixes = []struct {
	prefix string
	op     FilterOp
}{
	{"Min", FilterGe},
	{"Max", FilterLe},
	{"Since", FilterGe},
	{"Until", FilterLe},
	{"From", FilterGe},
	{"To", FilterLe},
	{"Start", FilterGe},
	{"End", FilterLe},
	{"Before", FilterLt},
	{"After", FilterGt},
}

// inferFilterOp checks whether fieldName starts with a known operator prefix.
// Returns the inferred operator, the stripped base name, and true when matched.
// The remainder must start with an uppercase letter to avoid false positives
// (e.g. "Minimum" should NOT match the "Min" prefix).
func inferFilterOp(fieldName string) (op FilterOp, baseName string, ok bool) {
	for _, p := range filterPrefixes {
		if !strings.HasPrefix(fieldName, p.prefix) {
			continue
		}

		remainder := fieldName[len(p.prefix):]
		if len(remainder) == 0 || remainder[0] < 'A' || remainder[0] > 'Z' {
			continue
		}

		return p.op, remainder, true
	}

	return "", "", false
}

// autoInferFilters inspects query input fields (excluding key fields and
// pagination meta) and generates FilterOnField options for any field whose
// name matches a result field. For collection result types (struct with an
// Items []T field), filters are matched against the element type T.
//
// Matching strategy (in priority order):
//  1. Exact name match → FilterEq
//  2. Prefix-based operator inference → FilterGe/FilterLe/etc.
//     (e.g. "MinScore" → FilterGe on "Score")
func autoInferFilters(
	queryType, resultType reflect.Type,
	keyFields []string,
	cfg QueryConfig,
) QueryConfig {
	resultFields := buildFieldIndex(collectionElementType(resultType))
	keySet := make(map[string]bool, len(keyFields))
	for _, k := range keyFields {
		keySet[k] = true
	}

	for i := range queryType.NumField() {
		f := queryType.Field(i)
		if !f.IsExported() {
			continue
		}

		if keySet[f.Name] {
			continue
		}

		if isMetaFieldName(f.Name) {
			continue
		}

		// Strategy 1: exact name match → FilterEq.
		if dst, ok := resultFields[f.Name]; ok && f.Type.AssignableTo(dst.Type) {
			cfg.filterAccessors = append(cfg.filterAccessors, filterAccessor{
				spec: &FilterSpec{Column: f.Name, Op: FilterEq},
			})

			continue
		}

		// Strategy 2: prefix-based operator inference.
		op, baseName, matched := inferFilterOp(f.Name)
		if !matched {
			continue
		}

		dst, ok := resultFields[baseName]
		if !ok || !f.Type.AssignableTo(dst.Type) {
			continue
		}

		cfg.filterAccessors = append(cfg.filterAccessors, filterAccessor{
			spec: &FilterSpec{
				Column:      baseName,
				Op:          op,
				InputColumn: f.Name,
			},
		})
	}

	return cfg
}

// isMetaFieldName returns true for pagination/metadata field names that
// should not become filter candidates.
func isMetaFieldName(name string) bool {
	return name == limitField || name == afterField || name == depthField
}
