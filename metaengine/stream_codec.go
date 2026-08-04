package metaengine

import "encoding/json/v2"

// EncodeStreamValue encodes a value for storage in a StreamLogBackend.
// Strings are stored as-is (zero-copy); all other types are JSON-encoded.
// This is the canonical encoding used by all SQL-backed stream log engines
// (SQLite, DuckDB, Postgres) to ensure cross-engine value compatibility.
func EncodeStreamValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	data, _ := json.Marshal(v)

	return string(data)
}

// DecodeStreamValue decodes a value from its stored representation.
// Falls back to returning the raw string if JSON decoding fails.
// This mirrors the encoding logic in [EncodeStreamValue].
func DecodeStreamValue(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}

	return v
}
