package pgengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScanKeyValues implements metaengine.KeyScanBackend: a paged key+value
// read over the BASE meta_map table (never the planned table), in
// deterministic key order — the read primitive for planned-table backfill.
// cursor is the last key of the previous page (nil/"" for the first page).
func (e *pgEngine) MapScanKeyValues(
	ctx context.Context,
	collection string,
	cursor any,
	limit int,
) ([]any, []any, bool, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if limit <= 0 {
		limit = 500
	}

	var cursorArg any
	if cursor != nil {
		cursorArg = fmt.Sprint(cursor)
	}

	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT key, value::text FROM meta_map
		 WHERE collection = $1 AND ($2::text IS NULL OR key > $2)
		 ORDER BY key
		 LIMIT $3`,
		collection, cursorArg, limit,
	)
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if err != nil {
		return nil, nil, false, fmt.Errorf("pgengine.MapScanKeyValues: %w", err)
	}

	defer metaengine.DeferClose(rows)

	keys := make([]any, 0, limit)
	values := make([]any, 0, limit)

	for rows.Next() {
		var key string

		var raw []byte

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, nil, false, fmt.Errorf("pgengine.MapScanKeyValues: scan: %w", err)
		}

		var val any

		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, nil, false, fmt.Errorf("pgengine.MapScanKeyValues: unmarshal: %w", err)
		}

		keys = append(keys, key)
		values = append(values, val)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, false, fmt.Errorf("pgengine.MapScanKeyValues: rows: %w", err)
	}

	return keys, values, len(keys) == limit, nil
}
