package sqliteengine

// Key/value encoding helpers for meta_map rows: key stringification, value
// serialization (DecodeStreamValue's inverse), and JSON path extraction for
// pushdown filters.

import (
	"encoding/json/v2"
	"fmt"
	"strconv"
	"strings"
)

func encodeKey(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case uint32:
		return strconv.FormatUint(uint64(k), 10)
	default:
		b, err := json.Marshal(key)
		if err != nil {
			// Keys are identifiers, not caller payload: a non-JSON-representable
			// key falls back to its %v form, matching the memory engine's key
			// stringification. Values never take this path (see encodeValue).
			return fmt.Sprintf("%v", key)
		}

		return string(b)
	}
}

// encodeValue serializes a map/collection value for its TEXT column. Unlike
// [encodeKey], it does NOT fall back to fmt.Sprintf("%v"): a value that is not
// JSON-representable (channel, function, cyclic structure) is caller data, so
// writing a Go-syntax placeholder would silently corrupt the read model. The
// error surfaces instead, before any row is written.
func encodeValue(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("sqliteengine: encode value %T: %w", value, err)
	}

	return string(b), nil
}

// jsonPath converts a field name to a JSON path for json_extract.
// E.g. "status" → "$.status". Single quotes are escaped to prevent
// breaking out of the SQL string literal that wraps the path.
func jsonPath(field string) string {
	escaped := strings.ReplaceAll(field, "'", "''")

	return "$." + escaped
}
