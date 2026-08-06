package metaengine

// JSONValue carries raw JSON bytes from a SQL engine, deferring decode until
// the typed result is reconstructed. ExecuteTyped and TypedReader recognize
// this type and unmarshal directly into the target type, avoiding the
// intermediate map[string]any + reify round-trip (3 JSON ops → 1).
//
// This is an internal optimization: memory engines return typed Go values
// directly (no JSONValue), and the closure-based MapScan path returns decoded
// any values (filtering needs them). Only pushdown paths (PushdownMapScan,
// GetRawValue, ScanRawValues) return JSONValue.
type JSONValue []byte

// jsonValue is a backward-compatible alias.
type jsonValue = JSONValue
