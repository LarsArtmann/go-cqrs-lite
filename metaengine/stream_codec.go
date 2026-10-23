package metaengine

import (
	"encoding/json/v2"
	"fmt"
)

// EncodeStreamValue encodes a value for storage in a StreamLogBackend.
// Strings are stored as-is (zero-copy); all other types are JSON-encoded.
// This is the canonical encoding used by all SQL-backed stream log engines
// (SQLite, DuckDB, Postgres) to ensure cross-engine value compatibility.
//
// A marshal failure yields an empty string, which [DecodeStreamValue] reads
// back as "". Write paths that persist caller data must use
// [EncodeStreamValueErr] instead: silently storing "" loses the value.
//
// Deprecated: use [EncodeStreamValueErr]. This wrapper stays for compatibility
// with external consumers and is removed in v5.
func EncodeStreamValue(v any) string {
	encoded, _ := EncodeStreamValueErr(v)

	return encoded
}

// EncodeStreamValueErr is [EncodeStreamValue] with the marshal error surfaced,
// so a write path can refuse to persist a placeholder when the value is not
// JSON-representable (a channel, a function, a cyclic structure, ...).
func EncodeStreamValueErr(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("metaengine.EncodeStreamValue: marshal %T: %w", v, err)
	}

	return string(data), nil
}

// DecodeStreamValue decodes a value from its stored representation.
// Falls back to returning the raw string if JSON decoding fails.
// This mirrors the encoding logic in [EncodeStreamValue].
//
// The fallback is intentional: strings are stored unquoted, so a failed JSON
// decode cannot be distinguished from a legitimate plain-string row here.
// Callers that already know the row was JSON-encoded and must reject corruption
// use [DecodeStreamValueErr].
func DecodeStreamValue(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}

	return v
}

// DecodeStreamValueErr decodes a stored value, returning the decode error
// instead of falling back to the raw string. It does not know whether the row
// was written by [EncodeStreamValueErr] or holds a plain string, so callers
// should use it only for rows they know are JSON-encoded.
func DecodeStreamValueErr(s string) (any, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("metaengine.DecodeStreamValue: decode stored value: %w", err)
	}

	return v, nil
}
