package metaengine

// Exported filter/sort helpers for non-SQL engines that implement
// RawScanReader. SQL engines (SQLite) push filters into WHERE clauses, but
// KV engines (Pebble) must evaluate FilterSpec/SortSpec in Go. These
// wrappers expose the existing internal helpers without changing call
// sites in the core package.
//
// Usage pattern (from pebbleengine.ScanRawValues):
//
//	for iter.First(); iter.Valid(); iter.Next() {
//	    raw := copy(iter.Value())
//	    decoded := decodeJSON(raw)
//	    if !metaengine.PassesFilterSpecs(decoded, filters) { continue }
//	    pairs = append(pairs, kvPair{value: decoded, raw: raw})
//	}
//	// Sort: compare by ItemFieldByName + CompareValues
//	// Cursor: compare by ItemFieldByName + CompareValues

// PassesFilterSpecs evaluates all declarative FilterSpec predicates against
// a decoded scan row (typically map[string]any from JSON). Returns true when
// every spec matches (AND semantics). An empty specs slice always passes.
//
// Non-SQL engines call this in ScanRawValues to apply WHERE-equivalent
// filtering in Go. SQL engines do not need this — they push FilterSpec into
// SQL WHERE clauses via json_extract.
func PassesFilterSpecs(item any, specs []FilterSpec) bool {
	return passesFilterSpecs(item, specs)
}

// ItemFieldByName extracts a named field from a decoded scan row by JSON key
// name. The item may be a map[string]any (the common case from JSON decode)
// or a struct (via reflection). Returns nil if the field does not exist.
//
// Non-SQL engines use this to read filter and sort column values from
// decoded JSON rows.
func ItemFieldByName(item any, name string) any {
	return itemFieldByName(item, name)
}

// CompareValues performs a type-aware tri-state comparison returning -1
// (left < right), 0 (equal), or +1 (left > right). Handles cross-type
// numeric comparison (e.g., int from a struct field vs float64 from a
// deserialized cursor) by promoting both to float64. Non-numeric values
// fall back to string comparison via fmt.Sprintf.
//
// Non-SQL engines use this in ScanRawValues for sorting and keyset cursor
// pagination. The comparison is consistent with evalFilterOp's FilterLt /
// FilterLe / FilterGt / FilterGe semantics.
func CompareValues(left, right any) int {
	return compareValue(left, right)
}

// EvalFilterOp evaluates a single FilterOp comparison (FilterEq, FilterNe,
// FilterLt, FilterLe, FilterGt, FilterGe, FilterIn) against an actual value
// from a scan row and an expected value from the FilterSpec. Returns true
// when the comparison holds.
//
// Non-SQL engines can use this for fine-grained filter evaluation, though
// PassesFilterSpecs is the higher-level entry point that calls this
// internally.
func EvalFilterOp(op FilterOp, actual, expected any) bool {
	return evalFilterOp(op, actual, expected)
}
