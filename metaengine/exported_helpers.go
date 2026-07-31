package metaengine

// Exported filter/sort helpers for non-SQL engines that implement
// RawScanReader. SQL engines (SQLite) push filters into WHERE clauses, but
// KV engines (Pebble) must evaluate FilterSpec/SortSpec in Go. These
// wrappers expose the existing internal helpers without changing call
// sites in the core package.

// PassesFilterSpecs evaluates declarative FilterSpec predicates against a
// scan row. Returns true when all specs match. Non-SQL engines use this in
// RawScanReader.ScanRawValues implementations to apply declarative filters
// in Go.
func PassesFilterSpecs(item any, specs []FilterSpec) bool {
	return passesFilterSpecs(item, specs)
}

// ItemFieldByName extracts a named field from a scan row by JSON key name.
// The item may be a map[string]any (decoded JSON) or a struct. Non-SQL
// engines use this to read filter/sort column values.
func ItemFieldByName(item any, name string) any {
	return itemFieldByName(item, name)
}

// CompareValues performs a type-aware tri-state comparison:
// -1 (left < right), 0 (equal), +1 (left > right). Non-SQL engines use
// this in RawScanReader.ScanRawValues for sorting.
func CompareValues(left, right any) int {
	return compareValue(left, right)
}

// EvalFilterOp evaluates a single filter comparison against an actual
// value. Non-SQL engines use this in RawScanReader.ScanRawValues to apply
// individual FilterSpec entries.
func EvalFilterOp(op FilterOp, actual, expected any) bool {
	return evalFilterOp(op, actual, expected)
}
