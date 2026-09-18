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
		return encodeJSON(key)
	}
}

func encodeValue(value any) string {
	return encodeJSON(value)
}

// encodeJSON marshals v to a JSON string, falling back to fmt.Sprintf("%v", v)
// if v is not JSON-serializable. Centralized so encodeKey/encodeValue stay
// in sync — both are the same operation on different conceptual inputs.
func encodeJSON(v any) string {
	// art-dupl:accept marshal-with-fallback helper; badgerengine graphNodeKeyJSON twin is dep-isolated
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(b)
}

// jsonPath converts a field name to a JSON path for json_extract.
// E.g. "status" → "$.status". Single quotes are escaped to prevent
// breaking out of the SQL string literal that wraps the path.
func jsonPath(field string) string {
	escaped := strings.ReplaceAll(field, "'", "''")

	return "$." + escaped
}
