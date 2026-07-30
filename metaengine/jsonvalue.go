package metaengine

// jsonValue carries raw JSON bytes from a SQL engine, deferring decode until
// the typed result is reconstructed. ExecuteTyped recognizes this type and
// unmarshals directly into R, avoiding the intermediate map[string]any +
// reify round-trip (3 JSON ops → 1).
//
// This is an internal optimization: memory engines return typed Go values
// directly (no jsonValue), and the closure-based MapScan path returns decoded
// any values (filtering needs them). Only pushdown paths (PushdownMapScan,
// MapGet, MultiGet, LogTail) return jsonValue.
type jsonValue []byte
