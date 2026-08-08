package metaengine

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// DecodeFloat converts a database driver scan value to float64.
//
// SQL engines return different Go types for numeric aggregate results:
//   - float64 for SUM/AVG over DOUBLE columns
//   - *big.Int for HUGEINT (DuckDB SUM over INTEGER columns)
//   - int64 for COUNT results
//   - nil for empty sets (COUNT of zero rows returns nil, not 0)
//   - []byte for some driver encodings
//
// This function normalizes all of these to float64, which is what the
// aggregate reader interfaces return. It is used by the DuckDB, SQLite,
// and Postgres engine implementations.
func DecodeFloat(raw any) (float64, error) {
	if raw == nil {
		return 0, nil
	}

	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case *big.Int:
		f, _ := v.Float64()
		return f, nil
	case []byte:
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			return 0, fmt.Errorf("metaengine DecodeFloat: %w", err)
		}

		return f, nil
	default:
		return 0, fmt.Errorf("metaengine DecodeFloat: unexpected type %T", raw)
	}
}
