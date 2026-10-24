package metaengine

import (
	"encoding/json/v2"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
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
// JSON-representable (a channel, a function, a cyclic structure, ...). The
// failure is Rejection-classified: the caller supplied input no engine can
// store, and retrying the same value cannot succeed.
func EncodeStreamValueErr(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return "", errorfamily.WrapRejection(err, "metaengine.encode_stream_value",
			fmt.Sprintf("encode stream value %T", v))
	}

	return string(data), nil
}

// EncodeStreamValuesErr encodes a batch of values for storage in a
// StreamLogBackend. It encodes EVERY value before returning, so an append that
// cannot represent one of its values fails before any row reaches the engine —
// the per-row loops in the SQL engines would otherwise leave a partial append
// behind on an unrepresentable value.
func EncodeStreamValuesErr(values []any) ([]string, error) {
	encoded := make([]string, len(values))

	for i, v := range values {
		s, err := EncodeStreamValueErr(v)
		if err != nil {
			return nil, err
		}

		encoded[i] = s
	}

	return encoded, nil
}

// DecodeStreamValue decodes a value from its stored representation.
// Falls back to returning the raw string if JSON decoding fails.
// This mirrors the encoding logic in [EncodeStreamValue].
//
// The fallback is intentional and permanent at this layer: strings are stored
// unquoted, so a failed JSON decode cannot be distinguished from a legitimate
// plain-string row here. There is deliberately no error-returning twin for
// this path — it would report false corruption for every plain string. The
// inverse also holds: a STRING value that happens to be valid JSON reads back
// as the parsed value, not as a string (the row carries no type tag). Fixing
// that needs the stamped engine-value envelope (ADR-0066 reversal), not a
// read-side change; see the metaengine COOKBOOK "Stored values" section.
//
// Callers that already know the row was JSON-encoded and must reject
// corruption use [DecodeStreamValueErr].
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
