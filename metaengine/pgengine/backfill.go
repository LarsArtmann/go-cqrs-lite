package pgengine

import (
	"context"

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
	limit = metaengine.ScanLimit(limit)

	return metaengine.ScanKeyValuesPage(ctx, e.conn(ctx),
		`SELECT key, value::text FROM meta_map
		 WHERE collection = $1 AND ($2::text IS NULL OR key > $2)
		 ORDER BY key
		 LIMIT $3`,
		[]any{collection, metaengine.CursorArg(cursor), limit},
		limit, "pgengine.MapScanKeyValues")
}
